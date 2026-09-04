// Package infrastructure provides the identity context's adapters: Postgres
// repositories (pgx), argon2id hashing, SMTP mailer, Google OIDC verifier.
// It implements the ports declared in identity/domain (plan.md → DDD).
package infrastructure

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// argon2id parameters (research.md §2). OWASP-recognised defaults; memory in
// KiB. Cost is chosen for an interactive login on a modest single host.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

// Argon2Hasher implements identity/domain.PasswordHasher with argon2id.
type Argon2Hasher struct{}

// NewArgon2Hasher returns an argon2id hasher.
func NewArgon2Hasher() *Argon2Hasher { return &Argon2Hasher{} }

// Hash encodes password as PHC string:
// $argon2id$v=19$m=65536,t=1,p=4$<salt b64>$<hash b64>
func (h *Argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("infra/password: read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify checks a PHC-encoded argon2id hash against a password.
func (h *Argon2Hasher) Verify(hash, password string) (bool, error) {
	params, salt, key, err := decodePHC(hash)
	if err != nil {
		return false, err
	}
	other := argon2.IDKey([]byte(password), salt, params.time, params.memory, uint8(params.threads), uint32(len(key)))
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

type argonParams struct{ time, memory, threads uint32 }

func decodePHC(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=65536,t=1,p=4", salt, hash]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, errors.New("infra/password: malformed hash")
	}
	var p argonParams
	var version int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return argonParams{}, nil, nil, errors.New("infra/password: malformed params")
	}
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, errors.New("infra/password: unsupported version")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, errors.New("infra/password: bad salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, errors.New("infra/password: bad key")
	}
	return p, salt, key, nil
}
