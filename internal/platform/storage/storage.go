// Package storage declares the platform blob-store port used by contexts that
// hold uploaded media (avatars in identity, hike evidence + photos in journey).
//
// A concrete MinIO/S3 adapter (via github.com/minio/minio-go or aws-sdk-go-v2)
// lands together with the journey upload flow in US2, where it is first
// exercised end-to-end. Until then the port + key type keep every context that
// stores an object key compiling against the contract without dragging an S3
// dependency into the kernel (research.md §1 kernel-light rule).
package storage

import (
	"context"
	"io"
	"time"
)

// Key is a namespaced object key, e.g. "avatars/<user-id>/<file>".
// The platform never assumes a storage layout beyond this opaque string.
type Key string

// ObjectURL is a pre-signed, time-limited URL a browser can use to upload or
// download an object without platform credentials.
type ObjectURL string

// PresignPolicy describes what a signed URL will be used for.
type PresignPolicy string

const (
	// Upload is a PUT policy (browser → storage).
	Upload PresignPolicy = "upload"
	// Download is a GET policy (storage → browser).
	Download PresignPolicy = "download"
)

// BlobStore is the platform port for object storage. Contexts depend on it via
// the composition root; they never talk to MinIO/S3 directly.
type BlobStore interface {
	// PresignURL returns a time-limited URL to upload or download the object.
	PresignURL(ctx context.Context, key Key, policy PresignPolicy, ttl time.Duration) (ObjectURL, error)
	// Delete removes an object (best-effort; returns nil when absent).
	Delete(ctx context.Context, key Key) error
	// Put streams raw bytes to a key (used by server-side copy/resize flows).
	Put(ctx context.Context, key Key, r io.Reader, contentType string) error
}
