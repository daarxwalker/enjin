package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
)

func LoadManifest(fs fs.FS, path string) (m map[string]string, err error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open manifest %s failed: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close manifest %s failed: %w", path, closeErr))
		}
	}()
	filebts, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s failed: %w", path, err)
	}
	var manifest map[string]string
	if err = json.Unmarshal(filebts, &manifest); err != nil {
		return nil, fmt.Errorf("deserialize manifest %s failed: %w", path, err)
	}
	return manifest, nil
}
