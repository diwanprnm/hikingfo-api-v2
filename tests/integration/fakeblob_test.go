package integration

import (
	"context"
	"encoding/base64"
	"io"
	"sync"
	"time"

	"hikingfo/backend/internal/platform/storage"
)

// fakeBlobStore is the in-memory BlobStore stand-in for integration tests: no
// Docker/MinIO needed (tests/integration harness spins up postgres only).
type fakeBlobStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	deletes []string
}

func newFakeBlobStore() *fakeBlobStore {
	return &fakeBlobStore{objects: map[string][]byte{}}
}

func (f *fakeBlobStore) PresignURL(_ context.Context, key storage.Key, _ storage.PresignPolicy, ttl time.Duration) (storage.ObjectURL, error) {
	return storage.ObjectURL("http://fake.local/bucket/" + string(key) + "?ttl=" + ttl.String()), nil
}

func (f *fakeBlobStore) Delete(_ context.Context, key storage.Key) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes = append(f.deletes, string(key))
	delete(f.objects, string(key))
	return nil
}

func (f *fakeBlobStore) Put(_ context.Context, key storage.Key, r io.Reader, _ string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[string(key)] = b
	return nil
}

func (f *fakeBlobStore) has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.objects[key]
	return ok
}

func (f *fakeBlobStore) deletedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deletes...)
}

var _ storage.BlobStore = (*fakeBlobStore)(nil)

// tinyPNG is a 1x1 valid PNG (magic bytes detectable by http.DetectContentType).
var tinyPNG = func() []byte {
	const b64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic("tinyPNG fixture: " + err.Error())
	}
	return b
}()
