package disk

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/daarxwalker/enjin/filestore"
)

var _ filestore.Driver = (*Driver)(nil)

type Driver struct {
	root     *os.Root
	lockSeed maphash.Seed
	locks    [lockStripes]sync.Mutex
	fileMode fs.FileMode
	dirMode  fs.FileMode
}

const (
	metaDir     = ".meta"
	tempPrefix  = ".enjin-tmp-"
	lockStripes = 64
	// tempAttempts covers a Delete pruning the directory of a Put between its
	// MkdirAll and the creation of the temporary file.
	tempAttempts = 3
)

var (
	ErrInvalidKey   = errors.New("invalid filestore key")
	ErrSizeMismatch = errors.New("body size does not match the given size")
)

// NewDriver stores files under dir, creating it when missing. Close the
// driver when the application stops.
func NewDriver(dir string, options ...Option) (*Driver, error) {
	d := &Driver{
		lockSeed: maphash.MakeSeed(),
		fileMode: 0o644,
		dirMode:  0o755,
	}
	for _, option := range options {
		option(d)
	}
	if err := os.MkdirAll(dir, d.dirMode); err != nil {
		return nil, fmt.Errorf("create filestore dir %s failed: %w", dir, err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open filestore dir %s failed: %w", dir, err)
	}
	d.root = root
	return d, nil
}

func (d *Driver) Close() error {
	if err := d.root.Close(); err != nil {
		return fmt.Errorf("close filestore dir %s failed: %w", d.root.Name(), err)
	}
	return nil
}

func (d *Driver) List(ctx context.Context, prefix string) ([]filestore.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list files with prefix %q stopped: %w", prefix, err)
	}
	var files []filestore.File
	walkErr := fs.WalkDir(d.root.FS(), d.findListDir(prefix), func(name string, entry fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("walk %s failed: %w", name, err)
		}
		if entry.IsDir() {
			if d.skipsListDir(name, prefix) {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || strings.HasPrefix(entry.Name(), tempPrefix) || !strings.HasPrefix(name, prefix) {
			return nil
		}
		info, infoErr := entry.Info()
		if errors.Is(infoErr, fs.ErrNotExist) {
			return nil
		}
		if infoErr != nil {
			return fmt.Errorf("stat %s failed: %w", name, infoErr)
		}
		meta, metaErr := d.loadMetadata(name, info)
		if metaErr != nil {
			return fmt.Errorf("load file %s failed: %w", name, metaErr)
		}
		files = append(files, d.createFile(name, info, meta))
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("list files with prefix %q failed: %w", prefix, walkErr)
	}
	slices.SortFunc(files, func(a, b filestore.File) int {
		return strings.Compare(a.Key, b.Key)
	})
	return files, nil
}

func (d *Driver) Stat(ctx context.Context, key string) (filestore.File, bool, error) {
	var zero filestore.File
	if err := d.checkKey(key); err != nil {
		return zero, false, fmt.Errorf("stat file failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return zero, false, fmt.Errorf("stat file %q stopped: %w", key, err)
	}
	info, err := d.root.Stat(filepath.FromSlash(key))
	if errors.Is(err, fs.ErrNotExist) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("stat file %q failed: %w", key, err)
	}
	if !info.Mode().IsRegular() {
		return zero, false, nil
	}
	meta, err := d.loadMetadata(key, info)
	if err != nil {
		return zero, false, fmt.Errorf("stat file %q failed: %w", key, err)
	}
	return d.createFile(key, info, meta), true, nil
}

func (d *Driver) Get(ctx context.Context, key string) (*filestore.Object, bool, error) {
	if err := d.checkKey(key); err != nil {
		return nil, false, fmt.Errorf("get file failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, false, fmt.Errorf("get file %q stopped: %w", key, err)
	}
	file, err := d.root.Open(filepath.FromSlash(key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("open file %q failed: %w", key, err)
	}
	object, exists, objectErr := d.createObject(key, file)
	if objectErr == nil && exists {
		return object, true, nil
	}
	var getErr error
	if objectErr != nil {
		getErr = fmt.Errorf("get file %q failed: %w", key, objectErr)
	}
	if closeErr := file.Close(); closeErr != nil {
		return nil, false, errors.Join(getErr, fmt.Errorf("close file %q failed: %w", key, closeErr))
	}
	return nil, false, getErr
}

func (d *Driver) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if err := d.checkKey(key); err != nil {
		return fmt.Errorf("put file failed: %w", err)
	}
	lock := d.lockKey(key)
	lock.Lock()
	defer lock.Unlock()

	hasher := sha256.New()
	written, err := d.writeFileAtomically(ctx, key, func(file *os.File) (int64, error) {
		copied, copyErr := io.Copy(io.MultiWriter(file, hasher), contextReader{ctx: ctx, reader: body})
		if copyErr != nil {
			return copied, fmt.Errorf("copy body failed: %w", copyErr)
		}
		if size >= 0 && copied != size {
			return copied, fmt.Errorf("copy body of %d bytes, expected %d, failed: %w", copied, size, ErrSizeMismatch)
		}
		return copied, nil
	})
	if err != nil {
		return fmt.Errorf("put file %q failed: %w", key, err)
	}
	saveErr := d.saveMetadata(ctx, key, metadata{
		ContentType: contentType,
		ETag:        hex.EncodeToString(hasher.Sum(nil)),
		Size:        written,
	})
	if saveErr == nil {
		return nil
	}
	// Metadata of the previous content must not describe the new one.
	removeErr := d.root.Remove(filepath.FromSlash(d.createMetadataPath(key)))
	if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return errors.Join(
			fmt.Errorf("put file %q failed: %w", key, saveErr),
			fmt.Errorf("remove stale metadata of %q failed: %w", key, removeErr),
		)
	}
	return fmt.Errorf("put file %q failed: %w", key, saveErr)
}

func (d *Driver) Delete(ctx context.Context, key string) error {
	if err := d.checkKey(key); err != nil {
		return fmt.Errorf("delete file failed: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("delete file %q stopped: %w", key, err)
	}
	lock := d.lockKey(key)
	lock.Lock()
	defer lock.Unlock()
	if err := d.removeKeyFiles(key); err != nil {
		return fmt.Errorf("delete file %q failed: %w", key, err)
	}
	return nil
}

func (d *Driver) checkKey(key string) error {
	if !fs.ValidPath(key) || key == "." || strings.Contains(key, `\`) {
		return fmt.Errorf("check key %q failed: %w", key, ErrInvalidKey)
	}
	segments := strings.Split(key, "/")
	if segments[0] == metaDir {
		return fmt.Errorf("check key %q failed, %s is reserved: %w", key, metaDir, ErrInvalidKey)
	}
	for _, segment := range segments {
		if strings.HasPrefix(segment, tempPrefix) {
			return fmt.Errorf("check key %q failed, %s is reserved: %w", key, tempPrefix, ErrInvalidKey)
		}
	}
	return nil
}

func (d *Driver) lockKey(key string) *sync.Mutex {
	return &d.locks[maphash.String(d.lockSeed, key)%lockStripes]
}

// findListDir returns the deepest directory every key with prefix lives in.
func (d *Driver) findListDir(prefix string) string {
	slash := strings.LastIndex(prefix, "/")
	if slash <= 0 || !fs.ValidPath(prefix[:slash]) {
		return "."
	}
	return prefix[:slash]
}

func (d *Driver) skipsListDir(name, prefix string) bool {
	if name == "." {
		return false
	}
	if name == metaDir || strings.HasPrefix(name, metaDir+"/") {
		return true
	}
	dirPrefix := name + "/"
	return !strings.HasPrefix(dirPrefix, prefix) && !strings.HasPrefix(prefix, dirPrefix)
}

func (d *Driver) createObject(key string, file *os.File) (*filestore.Object, bool, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("stat open file failed: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, false, nil
	}
	meta, err := d.loadMetadata(key, info)
	if err != nil {
		return nil, false, fmt.Errorf("load file metadata failed: %w", err)
	}
	return &filestore.Object{Body: file, File: d.createFile(key, info, meta)}, true, nil
}

// loadMetadata falls back to the extension and the modification time when the
// file was stored without the driver or changed behind its back. Metadata is
// written atomically, so unreadable JSON is an error, not a fallback.
func (d *Driver) loadMetadata(key string, info fs.FileInfo) (metadata, error) {
	var (
		zero metadata
		meta metadata
	)
	fallback := metadata{
		ContentType: d.createContentType(key),
		ETag:        fmt.Sprintf("%x-%x", info.ModTime().UnixNano(), info.Size()),
		Size:        info.Size(),
	}
	raw, err := d.root.ReadFile(filepath.FromSlash(d.createMetadataPath(key)))
	if errors.Is(err, fs.ErrNotExist) {
		return fallback, nil
	}
	if err != nil {
		return zero, fmt.Errorf("read metadata of %q failed: %w", key, err)
	}
	if unmarshalErr := json.Unmarshal(raw, &meta); unmarshalErr != nil {
		return zero, fmt.Errorf("deserialize metadata of %q failed: %w", key, unmarshalErr)
	}
	if meta.Size != info.Size() {
		return fallback, nil
	}
	if meta.ContentType == "" {
		meta.ContentType = fallback.ContentType
	}
	return meta, nil
}

func (d *Driver) saveMetadata(ctx context.Context, key string, meta metadata) error {
	raw, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("serialize metadata of %q failed: %w", key, err)
	}
	_, err = d.writeFileAtomically(ctx, d.createMetadataPath(key), func(file *os.File) (int64, error) {
		written, writeErr := file.Write(raw)
		if writeErr != nil {
			return int64(written), fmt.Errorf("write metadata failed: %w", writeErr)
		}
		return int64(written), nil
	})
	if err != nil {
		return fmt.Errorf("save metadata of %q failed: %w", key, err)
	}
	return nil
}

// writeFileAtomically writes into a temporary file next to name and renames
// it over name, so a reader never sees a half written file.
func (d *Driver) writeFileAtomically(
	ctx context.Context,
	name string,
	write func(file *os.File) (int64, error),
) (written int64, err error) {
	var (
		zero     int64
		file     *os.File
		tempName string
	)
	file, tempName, err = d.createTempFile(path.Dir(name))
	if err != nil {
		return zero, fmt.Errorf("create temporary file for %s failed: %w", name, err)
	}
	renamed := false
	defer func() {
		if renamed {
			return
		}
		if removeErr := d.root.Remove(tempName); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("remove temporary file %s failed: %w", tempName, removeErr))
		}
	}()

	written, err = write(file)
	if err != nil {
		return zero, errors.Join(fmt.Errorf("write %s failed: %w", name, err), d.closeFile(file))
	}
	if err = file.Sync(); err != nil {
		return zero, errors.Join(fmt.Errorf("sync %s failed: %w", name, err), d.closeFile(file))
	}
	if err = d.closeFile(file); err != nil {
		return zero, fmt.Errorf("finish %s failed: %w", name, err)
	}
	if err = ctx.Err(); err != nil {
		return zero, fmt.Errorf("write %s stopped: %w", name, err)
	}
	if err = d.root.Rename(tempName, filepath.FromSlash(name)); err != nil {
		return zero, fmt.Errorf("rename temporary file to %s failed: %w", name, err)
	}
	renamed = true
	return written, nil
}

func (d *Driver) createTempFile(dir string) (*os.File, string, error) {
	var (
		zero    string
		lastErr error
	)
	for range tempAttempts {
		if dir != "." {
			if err := d.root.MkdirAll(filepath.FromSlash(dir), d.dirMode); err != nil {
				return nil, zero, fmt.Errorf("create dir %s failed: %w", dir, err)
			}
		}
		tempName := filepath.FromSlash(path.Join(dir, tempPrefix+rand.Text()))
		file, err := d.root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, d.fileMode)
		if err == nil {
			return file, tempName, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, zero, fmt.Errorf("open temporary file %s failed: %w", tempName, err)
		}
		lastErr = fmt.Errorf("open temporary file %s failed: %w", tempName, err)
	}
	return nil, zero, lastErr
}

func (d *Driver) closeFile(file *os.File) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s failed: %w", file.Name(), err)
	}
	return nil
}

func (d *Driver) removeKeyFiles(key string) error {
	info, err := d.root.Lstat(filepath.FromSlash(key))
	if err == nil && info.IsDir() {
		return nil
	}
	var errs []error
	for _, name := range []string{key, d.createMetadataPath(key)} {
		removeErr := d.root.Remove(filepath.FromSlash(name))
		if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("remove %s failed: %w", name, removeErr))
			continue
		}
		if pruneErr := d.removeEmptyDirs(path.Dir(name)); pruneErr != nil {
			errs = append(errs, pruneErr)
		}
	}
	return errors.Join(errs...)
}

// removeEmptyDirs walks up from dir until it meets a directory that is not
// empty. A Put of another key may fill a directory between the check and the
// removal; the directory then stays, which is the right outcome.
func (d *Driver) removeEmptyDirs(dir string) error {
	for dir != "." && dir != metaDir {
		empty, err := d.hasNoEntries(dir)
		if err != nil {
			return fmt.Errorf("check dir %s before removal failed: %w", dir, err)
		}
		if !empty {
			return nil
		}
		removeErr := d.root.Remove(filepath.FromSlash(dir))
		if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			stillEmpty, recheckErr := d.hasNoEntries(dir)
			if recheckErr != nil {
				return fmt.Errorf("check dir %s after failed removal failed: %w", dir, recheckErr)
			}
			if !stillEmpty {
				return nil
			}
			return fmt.Errorf("remove empty dir %s failed: %w", dir, removeErr)
		}
		dir = path.Dir(dir)
	}
	return nil
}

func (d *Driver) hasNoEntries(dir string) (bool, error) {
	var zero bool

	entries, err := fs.ReadDir(d.root.FS(), dir)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return zero, fmt.Errorf("read dir %s failed: %w", dir, err)
	}
	return len(entries) == 0, nil
}

func (d *Driver) createMetadataPath(key string) string {
	return path.Join(metaDir, key+".json")
}

func (d *Driver) createContentType(key string) string {
	if contentType := mime.TypeByExtension(path.Ext(key)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func (d *Driver) createFile(key string, info fs.FileInfo, meta metadata) filestore.File {
	return filestore.File{
		ModifiedAt:  info.ModTime(),
		Key:         key,
		ContentType: meta.ContentType,
		ETag:        meta.ETag,
		Size:        info.Size(),
	}
}
