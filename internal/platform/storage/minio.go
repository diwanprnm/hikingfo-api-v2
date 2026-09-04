// Package storage provides the concrete MinIO (S3-compatible) adapter for the
// BlobStore port. It is the only file that imports the minio-go SDK; every
// other package depends on the BlobStore interface declared in storage.go.
package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOConfig carries the connection details for a MinIO instance.
type MinIOConfig struct {
	Endpoint  string // host:port, e.g. "minio:9000"
	AccessKey string
	SecretKey string
	Secure    bool // use HTTPS to MinIO
}

// MinIOStore is the concrete BlobStore backed by MinIO / S3.
type MinIOStore struct {
	client *minio.Client
	bucket string
}

// NewMinIOStore creates a MinIOStore scoped to a single bucket.
func NewMinIOStore(cfg MinIOConfig, bucket string) (*MinIOStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOStore{client: client, bucket: bucket}, nil
}

// EnsureBucket creates the bucket if it does not already exist.
func (s *MinIOStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func (s *MinIOStore) PresignURL(ctx context.Context, key Key, policy PresignPolicy, ttl time.Duration) (ObjectURL, error) {
	switch policy {
	case Upload:
		u, err := s.client.PresignedPutObject(ctx, s.bucket, string(key), ttl)
		if err != nil {
			return "", err
		}
		return ObjectURL(u.String()), nil
	case Download:
		u, err := s.client.PresignedGetObject(ctx, s.bucket, string(key), ttl, nil)
		if err != nil {
			return "", err
		}
		return ObjectURL(u.String()), nil
	default:
		return "", io.ErrNoProgress
	}
}

func (s *MinIOStore) Delete(ctx context.Context, key Key) error {
	return s.client.RemoveObject(ctx, s.bucket, string(key), minio.RemoveObjectOptions{})
}

func (s *MinIOStore) Put(ctx context.Context, key Key, r io.Reader, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, string(key), r, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}
