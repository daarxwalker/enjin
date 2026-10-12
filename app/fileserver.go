package app

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

func ServeFS(r *Router, root http.FileSystem, path string, strip string, maxAge time.Duration) error {
	return serveFS(r, root, path, strip, fmt.Sprintf("public, max-age=%d", int(maxAge.Seconds())))
}

func ServeImmutableFS(r *Router, root http.FileSystem, path string, strip string) error {
	return serveFS(r, root, path, strip, "public, max-age=31536000, immutable")
}

func serveFS(r *Router, root http.FileSystem, path string, strip string, cacheControl string) error {
	if strings.ContainsAny(path, "{}*") {
		return errors.New("files serve does not permit any url parameters")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Raw().Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}

	prefix := strip
	if prefix == "" {
		prefix = strings.TrimSuffix(path, "/")
	}

	fileHandler := http.StripPrefix(prefix, http.FileServer(root))

	r.Raw().Get(path+"*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		fileHandler.ServeHTTP(w, r)
	})

	return nil
}

func ServeFile(r *Router, root fs.FS, path string, name string, maxAge time.Duration) error {
	if strings.ContainsAny(path, "{}*") {
		return errors.New("file serve does not permit any url parameters")
	}

	if _, err := fs.Stat(root, name); err != nil {
		return fmt.Errorf("stat %s failed: %w", name, err)
	}

	cacheControl := fmt.Sprintf("public, max-age=%d", int(maxAge.Seconds()))

	r.Raw().Get(path, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		http.ServeFileFS(w, req, root, name)
	})

	return nil
}
