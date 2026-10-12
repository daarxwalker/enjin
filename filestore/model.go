package filestore

import (
	"io"
	"time"
)

type File struct {
	ModifiedAt  time.Time
	Key         string
	ContentType string
	ETag        string
	Size        int64
}

type Object struct {
	Body io.ReadCloser
	File
}
