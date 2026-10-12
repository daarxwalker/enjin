package assets

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
)

func LoadManifest(fs fs.FS, path string) (m map[string]string, err error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("close file: %w", closeErr)
		}
	}()
	filebts, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var manifest map[string]string
	err = json.Unmarshal(filebts, &manifest)
	if err != nil {
		return nil, fmt.Errorf("deserialize content: %w", err)
	}
	return manifest, err
}
