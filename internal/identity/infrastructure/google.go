package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"hikingfo/backend/internal/identity/application"
)

// GoogleVerifier validates a Google sign-in credential (a Google ID token,
// typically from the One Tap / Sign In With Google JS SDK) via Google's
// tokeninfo endpoint, then checks the audience and issuer.
//
// Phase-2 hardening (research.md §2): swap this for coreos/go-oidc, which
// fetches the discovery document, pins the JWKS, and verifies the token's
// signature and exp locally instead of calling tokeninfo per login. The
// application port (application.GoogleVerifier) is unchanged by that swap.
type GoogleVerifier struct {
	// ClientID is this app's Google OAuth client ID (audience).
	ClientID string
	// HTTPClient is optional; a default with a short timeout is used otherwise.
	HTTPClient *http.Client
}

// NewGoogleVerifier wires the verifier for a client ID.
func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{ClientID: clientID}
}

var _ application.GoogleVerifier = (*GoogleVerifier)(nil)

func (g *GoogleVerifier) client() *http.Client {
	if g.HTTPClient != nil {
		return g.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// Verify validates the credential and returns the verified identity.
func (g *GoogleVerifier) Verify(ctx context.Context, idToken string) (*application.GoogleIdentity, error) {
	if g.ClientID == "" {
		return nil, errors.New("infra/identity: google client id not configured")
	}
	u := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("infra/identity: google verify: %w", err)
	}
	resp, err := g.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("infra/identity: google verify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, errors.New("infra/identity: google rejected token")
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Aud           string `json:"aud"`
		Iss           string `json:"iss"`
		Exp           int64  `json:"exp"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&claims); err != nil {
		return nil, fmt.Errorf("infra/identity: google verify: decode: %w", err)
	}

	// audience pinning prevents a token minted for another client from passing.
	if claims.Aud != g.ClientID {
		return nil, errors.New("infra/identity: google token audience mismatch")
	}
	if claims.Iss != "accounts.google.com" && claims.Iss != "https://accounts.google.com" {
		return nil, errors.New("infra/identity: google token issuer mismatch")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("infra/identity: google token expired")
	}
	if claims.Sub == "" {
		return nil, errors.New("infra/identity: google token missing subject")
	}

	return &application.GoogleIdentity{
		Sub:           claims.Sub,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		DisplayName:   claims.Name,
		PictureURL:    claims.Picture,
	}, nil
}
