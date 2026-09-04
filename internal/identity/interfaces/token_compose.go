package interfaces

import (
	"context"
	"time"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
)

// tokenCompose merges two infrastructure TokenStores (email-verification +
// password-reset) into a single domain.TokenStore so the application layer
// depends on one port. Each method routes to the correct backing store.
type tokenCompose struct {
	email    domain.TokenStore
	password domain.TokenStore
}

// NewTokenCompose merges two stores into one domain.TokenStore.
func NewTokenCompose(email, password domain.TokenStore) domain.TokenStore {
	return &tokenCompose{email: email, password: password}
}

func (t *tokenCompose) SaveEmailVerification(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error {
	return t.email.SaveEmailVerification(ctx, userID, tokenHash, expiresAt)
}

func (t *tokenCompose) ConsumeEmailVerification(ctx context.Context, tokenHash string) (ids.ID, error) {
	return t.email.ConsumeEmailVerification(ctx, tokenHash)
}

func (t *tokenCompose) SavePasswordReset(ctx context.Context, userID ids.ID, tokenHash string, expiresAt time.Time) error {
	return t.password.SavePasswordReset(ctx, userID, tokenHash, expiresAt)
}

func (t *tokenCompose) ConsumePasswordReset(ctx context.Context, tokenHash string) (ids.ID, error) {
	return t.password.ConsumePasswordReset(ctx, tokenHash)
}
