package storage

import (
	"context"
	"io"
)

type FileSystem interface {
	Upload(ctx context.Context, body io.Reader, bucket string, key string) error
	Init() error
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, bucket, key string) error
	IsExist(ctx context.Context, bucket, key string) (bool, error)
	Copy(ctx context.Context, bucket, key, destBucket, destKey string) error
}
