package httpx

import (
	"encoding/json"
	"mime"
	"net/http"
)

func Status(w http.ResponseWriter, status int) error {
	w.WriteHeader(status)
	return nil
}

func HTML(w http.ResponseWriter, status int, v string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, writeErr := w.Write([]byte(v)); writeErr != nil {
		return Error(http.StatusInternalServerError, writeErr)
	}
	return nil
}

func NoContent(w http.ResponseWriter, status int) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	return nil
}

func String(w http.ResponseWriter, status int, v string) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	if _, writeErr := w.Write([]byte(v)); writeErr != nil {
		return Error(http.StatusInternalServerError, writeErr)
	}
	return nil
}

func JSON(w http.ResponseWriter, status int, v any) error {
	bytes, marshalErr := json.Marshal(v)
	if marshalErr != nil {
		return Error(http.StatusInternalServerError, marshalErr)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, writeErr := w.Write(bytes); writeErr != nil {
		return Error(http.StatusInternalServerError, writeErr)
	}
	return nil
}

func JSONBytes(w http.ResponseWriter, status int, bts []byte) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, writeErr := w.Write(bts); writeErr != nil {
		return Error(http.StatusInternalServerError, writeErr)
	}
	return nil
}

func Stream(w http.ResponseWriter, name string, bts []byte) error {
	w.Header().Set("Content-Type", "application/octet-stream")
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	if disposition == "" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", disposition)
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write(bts); writeErr != nil {
		return Error(http.StatusInternalServerError, writeErr)
	}
	return nil
}

func Redirect(w http.ResponseWriter, r *http.Request, url string, status ...int) error {
	var statusCode int
	if len(status) > 0 {
		statusCode = status[0]
	} else {
		statusCode = http.StatusSeeOther
	}
	http.Redirect(w, r, url, statusCode)
	return nil
}

func Refresh(w http.ResponseWriter, r *http.Request) error {
	ctxv := From(r.Context())
	if ctxv == nil {
		return nil
	}
	http.Redirect(w, r, ctxv.URL, http.StatusSeeOther)
	return nil
}

func StatusError(status int) error {
	return &Err{
		Status: status,
	}
}

func Error(status int, err error) error {
	if err == nil {
		return nil
	}
	return &Err{
		Status: status,
		Raw:    err,
	}
}
