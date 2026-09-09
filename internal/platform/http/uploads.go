// uploads.go is the shared POST /uploads pipeline (002 §8 — the journey
// placeholder becomes real). It is mounted by the journey context (auth +
// rate limit) and lives in platform because catalogue photos, hike evidence,
// and avatars all land in the same staging prefix.
//
// Contract (specs/002-mountain-photo-routes/contracts/api.md §8): multipart
// field "file"; sniff magic bytes with http.DetectContentType; accept only
// jpeg/png/webp (400 otherwise); cap 10 MB (413); store at
// uploads/<uuid>.<ext>; respond 201 {key, url} with a 24 h presigned GET.
package http

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hikingfo/backend/internal/platform/storage"
)

// maxUploadBytes is the 10 MB hard cap (FR-007).
const maxUploadBytes = 10 << 20

// uploadURLTTL matches the catalogue's presigned-read TTL (24 h).
const uploadURLTTL = 24 * time.Hour

// uploadExts maps sniffed content types to stored file extensions.
var uploadExts = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// UploadHandler returns the authed upload handler over a blob store. A nil
// store degrades to the pre-002 placeholder response (minio-less tests).
func UploadHandler(store storage.BlobStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if store == nil {
			c.JSON(http.StatusCreated, gin.H{"key": "uploads/placeholder.jpg", "url": ""})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)
		fh, err := c.FormFile("file")
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": gin.H{"code": "validation", "message": "file exceeds 10 MB limit"},
				})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "validation", "message": "field \"file\" is required"},
			})
			return
		}
		if fh.Size > maxUploadBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{"code": "validation", "message": "file exceeds 10 MB limit"},
			})
			return
		}
		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "validation", "message": "file could not be read"},
			})
			return
		}
		defer f.Close()

		head := make([]byte, 512)
		n, err := io.ReadFull(f, head)
		if n > 0 {
			head = head[:n]
		}
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "validation", "message": "file could not be read"},
			})
			return
		}
		ct := http.DetectContentType(head)
		ext, ok := uploadExts[ct]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "validation", "message": "file is not an image (jpeg, png, or webp)"},
			})
			return
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "internal", "message": "upload failed"},
			})
			return
		}
		key := storage.Key("uploads/" + uuid.NewString() + "." + ext)
		if err := store.Put(c.Request.Context(), key, f, ct); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "internal", "message": "upload failed"},
			})
			return
		}
		url, err := store.PresignURL(c.Request.Context(), key, storage.Download, uploadURLTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "internal", "message": "upload failed"},
			})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"key": string(key), "url": string(url)})
	}
}
