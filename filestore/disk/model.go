package disk

// metadata keeps what a filesystem cannot. Size ties it to the content it was
// written with: a mismatch means the file changed without the driver.
type metadata struct {
	ContentType string `json:"contentType"`
	ETag        string `json:"etag"`
	Size        int64  `json:"size"`
}
