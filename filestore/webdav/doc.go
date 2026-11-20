// Package webdav is the WebDAV Driver for package filestore, a client over
// net/http that needs only PROPFIND, GET, PUT, MKCOL and DELETE.
//
// Put creates the missing parent collections. Delete removes the file but
// leaves empty collections behind; List never returns collections, so they
// do not show. Many servers (Nextcloud, x/net/webdav) ignore the
// Content-Type sent with PUT and derive the type from the file name.
//
// Unlike S3, a key cannot be both a file and a collection (a and a/b).
package webdav
