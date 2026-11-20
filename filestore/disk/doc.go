// Package disk is the local disk Driver for package filestore.
//
// Every file lives at <dir>/<key>, its content type and ETag in
// <dir>/.meta/<key>.json. All access goes through os.Root, so neither a key
// nor a symlink can reach outside dir. Writes are atomic (temporary file and
// rename) and serialized per key within one process.
//
// Unlike S3, a key cannot be both a file and a directory: after "a" is stored,
// "a/b" fails, and the other way round.
package disk
