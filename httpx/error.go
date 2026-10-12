package httpx

import (
	"net/http"
	"strconv"
)

type Err struct {
	Raw    error
	Status int
}

func (e *Err) Error() string {
	if e == nil {
		return ""
	}
	if e.Raw != nil {
		return e.Raw.Error()
	}
	if text := http.StatusText(e.Status); text != "" {
		return text
	}
	return "http status " + strconv.Itoa(e.Status)
}

func (e *Err) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Raw
}
