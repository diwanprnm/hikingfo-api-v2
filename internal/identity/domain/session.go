package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// Session is a server-side opaque session (research.md §2): the cookie carries
// only a random token; its sha256 hash is stored, so a DB leak does not expose
// live tokens.
type Session struct {
	ID        ids.ID
	UserID    ids.ID
	TokenHash string
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Active reports whether the session is not revoked and not past its expiry.
func (s Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}

// IssueSessionToken mints an opaque token and returns both the token (to put
// in the httpOnly cookie) and its hash (to persist). One call pairs them.
func IssueSessionToken() (token, tokenHash string, err error) {
	return mintOpaqueToken()
}

// IssueCSRFToken mints the per-session CSRF token the SPA sends back on
// mutating requests (research.md §2).
func IssueCSRFToken() (string, error) {
	tok, _, err := mintOpaqueToken()
	return tok, err
}

// HashToken computes the persisted form of an opaque token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func mintOpaqueToken() (token, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("identity: read randomness: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, HashToken(token), nil
}
