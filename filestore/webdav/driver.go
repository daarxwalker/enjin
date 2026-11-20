package webdav

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/daarxwalker/enjin/filestore"
)

var _ filestore.Driver = (*Driver)(nil)

type Driver struct {
	client        *http.Client
	collections   map[string]struct{}
	authorization string
	base          url.URL
	collectionsMu sync.Mutex
}

const (
	methodPropfind = "PROPFIND"
	methodMkcol    = "MKCOL"
	propfindBody   = `<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:"><d:prop>
<d:resourcetype/><d:getcontentlength/><d:getcontenttype/><d:getetag/><d:getlastmodified/>
</d:prop></d:propfind>`
	// maxDrainSize is how much of an unread response body is drained so the
	// connection can be reused; a longer body only costs a new connection.
	maxDrainSize = 1 << 20
	// maxMultistatusSize stops a misbehaving server from streaming an endless
	// PROPFIND answer into memory.
	maxMultistatusSize = 64 << 20
)

var (
	ErrInvalidKey       = errors.New("invalid filestore key")
	ErrUnexpectedStatus = errors.New("unexpected webdav response status")
)

// NewDriver stores files under baseURL, a collection on a WebDAV server such
// as https://cloud.example.com/remote.php/dav/files/app/.
func NewDriver(baseURL string, options ...Option) (*Driver, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse webdav url failed: %w", err)
	}
	if (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("check webdav url %s failed: an http or https url with a host is required", base.Redacted())
	}
	if base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("check webdav url %s failed: query and fragment are not allowed", base.Redacted())
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
		if base.RawPath != "" {
			base.RawPath += "/"
		}
	}
	d := &Driver{
		client:      new(http.Client),
		collections: make(map[string]struct{}),
		base:        *base,
	}
	for _, option := range options {
		option(d)
	}
	return d, nil
}

func (d *Driver) List(ctx context.Context, prefix string) ([]filestore.File, error) {
	var (
		files   []filestore.File
		pending = []string{d.findListDir(prefix)}
	)
	for len(pending) > 0 {
		collection := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		responses, found, loadErr := d.loadResponses(ctx, collection, true, "1")
		if loadErr != nil {
			return nil, fmt.Errorf("list files with prefix %q failed: %w", prefix, loadErr)
		}
		if !found {
			continue
		}
		for _, response := range responses {
			key, keyErr := d.createKey(response.Href)
			if keyErr != nil {
				return nil, fmt.Errorf("list files with prefix %q failed: %w", prefix, keyErr)
			}
			props, ok := d.findOKProps(response)
			if !ok || key == collection {
				continue
			}
			if props.ResourceType.Collection.Local != "" {
				if !d.skipsListDir(key, prefix) {
					pending = append(pending, key)
				}
				continue
			}
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			file, fileErr := d.propDAVRowToFile(key, props)
			if fileErr != nil {
				return nil, fmt.Errorf("list files with prefix %q failed: %w", prefix, fileErr)
			}
			files = append(files, file)
		}
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
	props, found, err := d.loadFileProps(ctx, key)
	if err != nil {
		return zero, false, fmt.Errorf("stat file %q failed: %w", key, err)
	}
	if !found || props.ResourceType.Collection.Local != "" {
		return zero, false, nil
	}
	file, err := d.propDAVRowToFile(key, props)
	if err != nil {
		return zero, false, fmt.Errorf("stat file %q failed: %w", key, err)
	}
	return file, true, nil
}

func (d *Driver) Get(ctx context.Context, key string) (*filestore.Object, bool, error) {
	if err := d.checkKey(key); err != nil {
		return nil, false, fmt.Errorf("get file failed: %w", err)
	}
	req, err := d.createRequest(ctx, http.MethodGet, d.createURL(key, false), nil)
	if err != nil {
		return nil, false, fmt.Errorf("get file %q failed: %w", key, err)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("send get of %q failed: %w", key, err)
	}
	if res.StatusCode == http.StatusOK {
		object, objectErr := d.createObject(key, res)
		if objectErr == nil {
			return object, true, nil
		}
		getErr := fmt.Errorf("get file %q failed: %w", key, objectErr)
		if closeErr := d.closeResponse(res); closeErr != nil {
			return nil, false, errors.Join(getErr, fmt.Errorf("finish get of %q failed: %w", key, closeErr))
		}
		return nil, false, getErr
	}
	var statusErr error
	if res.StatusCode != http.StatusNotFound {
		statusErr = fmt.Errorf("get file %q answered %s: %w", key, res.Status, ErrUnexpectedStatus)
	}
	if closeErr := d.closeResponse(res); closeErr != nil {
		return nil, false, errors.Join(statusErr, fmt.Errorf("finish get of %q failed: %w", key, closeErr))
	}
	return nil, false, statusErr
}

// Put sends the body once. When the server answers that a parent collection
// is missing (someone removed it since the driver created it), Put creates
// the collections again and resends a body that can seek back.
func (d *Driver) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	var (
		start  int64
		status int
		err    error
	)
	if err = d.checkKey(key); err != nil {
		return fmt.Errorf("put file failed: %w", err)
	}
	if err = d.ensureCollections(ctx, path.Dir(key)); err != nil {
		return fmt.Errorf("put file %q failed: %w", key, err)
	}
	seeker, rewindable := body.(io.Seeker)
	if rewindable {
		if start, err = seeker.Seek(0, io.SeekCurrent); err != nil {
			return fmt.Errorf("find body offset of %q failed: %w", key, err)
		}
	}
	if status, err = d.sendFile(ctx, key, body, size, contentType); err != nil {
		return fmt.Errorf("put file %q failed: %w", key, err)
	}
	if d.signalsMissingParent(status) {
		d.collectionsMu.Lock()
		d.collections = make(map[string]struct{})
		d.collectionsMu.Unlock()
		if !rewindable {
			return fmt.Errorf(
				"put file %q answered %d, parent collections are gone, retry the put: %w",
				key,
				status,
				ErrUnexpectedStatus,
			)
		}
		if err = d.ensureCollections(ctx, path.Dir(key)); err != nil {
			return fmt.Errorf("put file %q failed: %w", key, err)
		}
		if _, err = seeker.Seek(start, io.SeekStart); err != nil {
			return fmt.Errorf("rewind body of %q failed: %w", key, err)
		}
		if status, err = d.sendFile(ctx, key, body, size, contentType); err != nil {
			return fmt.Errorf("put file %q again failed: %w", key, err)
		}
	}
	if status != http.StatusCreated && status != http.StatusNoContent && status != http.StatusOK {
		return fmt.Errorf("put file %q answered %d: %w", key, status, ErrUnexpectedStatus)
	}
	return nil
}

// Delete checks first that key is no collection: DELETE on a collection would
// remove everything below it.
func (d *Driver) Delete(ctx context.Context, key string) error {
	if err := d.checkKey(key); err != nil {
		return fmt.Errorf("delete file failed: %w", err)
	}
	props, found, err := d.loadFileProps(ctx, key)
	if err != nil {
		return fmt.Errorf("delete file %q failed: %w", key, err)
	}
	if !found || props.ResourceType.Collection.Local != "" {
		return nil
	}
	req, err := d.createRequest(ctx, http.MethodDelete, d.createURL(key, false), nil)
	if err != nil {
		return fmt.Errorf("delete file %q failed: %w", key, err)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send delete of %q failed: %w", key, err)
	}
	var statusErr error
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound:
	default:
		statusErr = fmt.Errorf("delete file %q answered %s: %w", key, res.Status, ErrUnexpectedStatus)
	}
	if closeErr := d.closeResponse(res); closeErr != nil {
		return errors.Join(statusErr, fmt.Errorf("finish delete of %q failed: %w", key, closeErr))
	}
	return statusErr
}

func (d *Driver) checkKey(key string) error {
	if !fs.ValidPath(key) || key == "." || strings.Contains(key, `\`) {
		return fmt.Errorf("check key %q failed: %w", key, ErrInvalidKey)
	}
	return nil
}

// createURL escapes every segment on its own, so "#", "?" and "%" in a key
// stay part of the name.
func (d *Driver) createURL(name string, collection bool) string {
	target := d.base
	if name != "" {
		segments := strings.Split(name, "/")
		escaped := make([]string, 0, len(segments))
		for _, segment := range segments {
			escaped = append(escaped, url.PathEscape(segment))
		}
		target.RawPath = d.base.EscapedPath() + strings.Join(escaped, "/")
		target.Path = d.base.Path + name
		if collection {
			target.RawPath += "/"
			target.Path += "/"
		}
	}
	return target.String()
}

// createKey turns an href (a path or an absolute URL, percent-encoded) back
// into the key it names.
func (d *Driver) createKey(href string) (string, error) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("parse href %q failed: %w", href, err)
	}
	if parsed.Path == strings.TrimSuffix(d.base.Path, "/") {
		return "", nil
	}
	if !strings.HasPrefix(parsed.Path, d.base.Path) {
		return "", fmt.Errorf("read href %q failed: it lies outside %s", href, d.base.Path)
	}
	return strings.TrimSuffix(strings.TrimPrefix(parsed.Path, d.base.Path), "/"), nil
}

func (d *Driver) createRequest(ctx context.Context, method, target string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("create %s request failed: %w", method, err)
	}
	if d.authorization != "" {
		req.Header.Set("Authorization", d.authorization)
	}
	return req, nil
}

func (d *Driver) createObject(key string, res *http.Response) (*filestore.Object, error) {
	var modifiedAt time.Time
	if lastModified := res.Header.Get("Last-Modified"); lastModified != "" {
		parsed, err := http.ParseTime(lastModified)
		if err != nil {
			return nil, fmt.Errorf("parse last-modified %q failed: %w", lastModified, err)
		}
		modifiedAt = parsed
	}
	file := filestore.File{
		ModifiedAt:  modifiedAt,
		Key:         key,
		ContentType: d.createContentType(key, res.Header.Get("Content-Type")),
		ETag:        d.createETag(res.Header.Get("ETag")),
		Size:        res.ContentLength,
	}
	return &filestore.Object{Body: res.Body, File: file}, nil
}

func (d *Driver) loadFileProps(ctx context.Context, key string) (propDAVRow, bool, error) {
	var zero propDAVRow
	responses, found, err := d.loadResponses(ctx, key, false, "0")
	if err != nil {
		return zero, false, fmt.Errorf("load properties failed: %w", err)
	}
	if !found {
		return zero, false, nil
	}
	if len(responses) == 0 {
		return zero, false, fmt.Errorf("load properties failed: %w: empty multistatus", ErrUnexpectedStatus)
	}
	props, ok := d.findOKProps(responses[0])
	if !ok {
		return zero, false, fmt.Errorf("load properties failed: %w: no successful propstat", ErrUnexpectedStatus)
	}
	return props, true, nil
}

func (d *Driver) loadResponses(
	ctx context.Context,
	name string,
	collection bool,
	depth string,
) (responses []responseDAVRow, found bool, err error) {
	target := d.createURL(name, collection)
	req, err := d.createRequest(ctx, methodPropfind, target, strings.NewReader(propfindBody))
	if err != nil {
		return nil, false, fmt.Errorf("load properties of %q failed: %w", name, err)
	}
	req.Header.Set("Depth", depth)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	res, err := d.client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("send propfind of %q failed: %w", name, err)
	}
	defer func() {
		if closeErr := d.closeResponse(res); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("finish propfind of %q failed: %w", name, closeErr))
		}
	}()
	if res.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if res.StatusCode != http.StatusMultiStatus {
		return nil, false, fmt.Errorf("propfind %q answered %s: %w", name, res.Status, ErrUnexpectedStatus)
	}
	var multistatus multistatusDAVRow
	if decodeErr := xml.NewDecoder(io.LimitReader(res.Body, maxMultistatusSize)).Decode(&multistatus); decodeErr != nil {
		return nil, false, fmt.Errorf("decode propfind answer for %q failed: %w", name, decodeErr)
	}
	return multistatus.Responses, true, nil
}

func (d *Driver) findOKProps(response responseDAVRow) (propDAVRow, bool) {
	var zero propDAVRow
	for _, propstat := range response.Propstats {
		if strings.Contains(propstat.Status, " 200 ") || strings.HasSuffix(propstat.Status, " 200") {
			return propstat.Prop, true
		}
	}
	return zero, false
}

func (d *Driver) propDAVRowToFile(key string, props propDAVRow) (filestore.File, error) {
	var (
		zero       filestore.File
		size       int64
		modifiedAt time.Time
		err        error
	)
	if props.ContentLength != "" {
		if size, err = strconv.ParseInt(props.ContentLength, 10, 64); err != nil {
			return zero, fmt.Errorf("parse content length of %q failed: %w", key, err)
		}
	}
	if props.LastModified != "" {
		if modifiedAt, err = http.ParseTime(props.LastModified); err != nil {
			return zero, fmt.Errorf("parse last modified of %q failed: %w", key, err)
		}
	}
	return filestore.File{
		ModifiedAt:  modifiedAt,
		Key:         key,
		ContentType: d.createContentType(key, props.ContentType),
		ETag:        d.createETag(props.ETag),
		Size:        size,
	}, nil
}

func (d *Driver) createContentType(key, served string) string {
	if served != "" {
		return served
	}
	if contentType := mime.TypeByExtension(path.Ext(key)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}

func (d *Driver) createETag(etag string) string {
	return strings.Trim(strings.TrimPrefix(etag, "W/"), `"`)
}

// ensureCollections creates every collection of dir from the top down. The
// collections the driver created or found are remembered, so later puts into
// the same directories send no MKCOL.
func (d *Driver) ensureCollections(ctx context.Context, dir string) error {
	if dir == "." {
		return nil
	}
	segments := strings.Split(dir, "/")
	for i := range segments {
		collection := strings.Join(segments[:i+1], "/")
		if d.knowsCollection(collection) {
			continue
		}
		if err := d.createCollection(ctx, collection); err != nil {
			return fmt.Errorf("ensure collection %q failed: %w", collection, err)
		}
		d.collectionsMu.Lock()
		d.collections[collection] = struct{}{}
		d.collectionsMu.Unlock()
	}
	return nil
}

func (d *Driver) knowsCollection(collection string) bool {
	d.collectionsMu.Lock()
	defer d.collectionsMu.Unlock()
	_, known := d.collections[collection]
	return known
}

// createCollection treats 405 as success: RFC 4918 answers MKCOL on an
// existing resource with 405 Method Not Allowed.
func (d *Driver) createCollection(ctx context.Context, collection string) error {
	req, err := d.createRequest(ctx, methodMkcol, d.createURL(collection, true), nil)
	if err != nil {
		return fmt.Errorf("create collection failed: %w", err)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send mkcol failed: %w", err)
	}
	var statusErr error
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusMethodNotAllowed {
		statusErr = fmt.Errorf("mkcol answered %s: %w", res.Status, ErrUnexpectedStatus)
	}
	if closeErr := d.closeResponse(res); closeErr != nil {
		return errors.Join(statusErr, fmt.Errorf("finish mkcol failed: %w", closeErr))
	}
	return statusErr
}

// sendFile hides the Close of body from net/http, which would otherwise close
// a caller's *os.File, and sets the length itself for the same reason.
func (d *Driver) sendFile(ctx context.Context, key string, body io.Reader, size int64, contentType string) (int, error) {
	req, err := d.createRequest(ctx, http.MethodPut, d.createURL(key, false), io.NopCloser(body))
	if err != nil {
		return 0, fmt.Errorf("send file failed: %w", err)
	}
	switch {
	case size == 0:
		req.Body = http.NoBody
		req.ContentLength = 0
	case size > 0:
		req.ContentLength = size
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("send put failed: %w", err)
	}
	if closeErr := d.closeResponse(res); closeErr != nil {
		return res.StatusCode, fmt.Errorf("finish put failed: %w", closeErr)
	}
	return res.StatusCode, nil
}

// signalsMissingParent covers both answers servers give to a PUT below a
// missing collection: 409 Conflict from RFC 4918 and 404 Not Found.
func (d *Driver) signalsMissingParent(status int) bool {
	return status == http.StatusConflict || status == http.StatusNotFound
}

func (d *Driver) closeResponse(res *http.Response) error {
	var errs []error
	if _, err := io.Copy(io.Discard, io.LimitReader(res.Body, maxDrainSize)); err != nil {
		errs = append(errs, fmt.Errorf("drain webdav response failed: %w", err))
	}
	if err := res.Body.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close webdav response failed: %w", err))
	}
	return errors.Join(errs...)
}

// findListDir returns the deepest collection every key with prefix lives in.
func (d *Driver) findListDir(prefix string) string {
	slash := strings.LastIndex(prefix, "/")
	if slash <= 0 || !fs.ValidPath(prefix[:slash]) {
		return ""
	}
	return prefix[:slash]
}

func (d *Driver) skipsListDir(name, prefix string) bool {
	dirPrefix := name + "/"
	return !strings.HasPrefix(dirPrefix, prefix) && !strings.HasPrefix(prefix, dirPrefix)
}
