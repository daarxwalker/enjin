package minio

import (
	"context"
	"errors"
	"io"

	"github.com/daarxwalker/enjin/filestore"

	miniosdk "github.com/minio/minio-go/v7"
)

var _ filestore.Driver = (*Driver)(nil)

type Driver struct {
	client *miniosdk.Client
	bucket string
}

func NewDriver(client *miniosdk.Client, bucket string) *Driver {
	return &Driver{
		client: client,
		bucket: bucket,
	}
}

func (d *Driver) List(ctx context.Context, prefix string) ([]filestore.File, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var files []filestore.File
	for info := range d.client.ListObjectsIter(ctx, d.bucket, miniosdk.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}) {
		if info.Err != nil {
			return nil, info.Err
		}
		files = append(files, d.infoToFile(info))
	}
	return files, nil
}

func (d *Driver) Stat(ctx context.Context, key string) (filestore.File, bool, error) {
	var zero filestore.File
	info, err := d.client.StatObject(ctx, d.bucket, key, miniosdk.StatObjectOptions{})
	if d.isErrNoSuchKey(err) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	return d.infoToFile(info), true, nil
}

func (d *Driver) Get(ctx context.Context, key string) (*filestore.Object, bool, error) {
	object, err := d.client.GetObject(ctx, d.bucket, key, miniosdk.GetObjectOptions{})
	if err != nil {
		return nil, false, err
	}
	info, err := object.Stat()
	if d.isErrNoSuchKey(err) {
		if closeErr := object.Close(); closeErr != nil {
			return nil, false, closeErr
		}
		return nil, false, nil
	}
	if err != nil {
		if closeErr := object.Close(); closeErr != nil {
			return nil, false, closeErr
		}
		return nil, false, err
	}
	return &filestore.Object{Body: object, File: d.infoToFile(info)}, true, nil
}

func (d *Driver) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := d.client.PutObject(ctx, d.bucket, key, body, size, miniosdk.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (d *Driver) Delete(ctx context.Context, key string) error {
	return d.client.RemoveObject(ctx, d.bucket, key, miniosdk.RemoveObjectOptions{})
}

func (d *Driver) isErrNoSuchKey(err error) bool {
	res, ok := errors.AsType[miniosdk.ErrorResponse](err)
	return ok && res.Code == miniosdk.NoSuchKey
}

func (d *Driver) infoToFile(info miniosdk.ObjectInfo) filestore.File {
	return filestore.File{
		ModifiedAt:  info.LastModified,
		Key:         info.Key,
		ContentType: info.ContentType,
		ETag:        info.ETag,
		Size:        info.Size,
	}
}
