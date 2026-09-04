package application

import (
	"context"
	"strings"
	"time"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
	"hikingfo/backend/internal/shared/langtext"
)

// Me returns the requester's own full view (session user). It is the identity
// context's own profile read (contracts §1 GET /auth/me). The distinct-mountain
// count and experience level are passed in by the caller, which composes the
// achievement context via port — keeping identity free of achievement imports.
type MeView struct {
	ID              ids.ID
	Email           string
	EmailVerified   bool
	DisplayName     string
	AvatarKey       string
	HomeRegion      string
	Bio             langtext.Text
	Role            domain.Role
	Status          domain.Status
	JoinedAt        time.Time
	DistinctCount   int
	ExperienceLevel domain.ExperienceLevel
}

// Me returns the requester's profile view.
func (s *Service) Me(ctx context.Context, userID ids.ID, distinctCount int, level domain.ExperienceLevel) (*MeView, error) {
	u, err := s.d.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, kerr.WrapNotFound("account not found", err)
	}
	return &MeView{
		ID:              u.ID,
		Email:           u.Email,
		EmailVerified:   u.EmailVerified(),
		DisplayName:     u.DisplayName,
		AvatarKey:       u.AvatarKey,
		HomeRegion:      u.HomeRegion,
		Bio:             u.Bio,
		Role:            u.Role,
		Status:          u.Status,
		JoinedAt:        u.CreatedAt,
		DistinctCount:   distinctCount,
		ExperienceLevel: level,
	}, nil
}

// UpdateProfileParams carries the mutable public profile fields.
type UpdateProfileParams struct {
	DisplayName string
	AvatarKey   string
	BioID       string
	BioEN       string
	HomeRegion  string
}

// UpdateProfile applies public-profile edits (contracts §5 PATCH /me/profile).
func (s *Service) UpdateProfile(ctx context.Context, userID ids.ID, p UpdateProfileParams) (*domain.User, error) {
	u, err := s.d.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, kerr.WrapNotFound("account not found", err)
	}
	if err := domain.ValidateDisplayName(p.DisplayName); err != nil {
		return nil, err
	}
	u.DisplayName = strings.TrimSpace(p.DisplayName)
	u.AvatarKey = p.AvatarKey
	u.HomeRegion = strings.TrimSpace(p.HomeRegion)
	u.Bio = langtext.From(strings.TrimSpace(p.BioID), strings.TrimSpace(p.BioEN))
	if err := s.d.Users.Update(ctx, u); err != nil {
		return nil, kerr.WrapInternal("could not update profile", err)
	}
	return u, nil
}

// UpdateContacts stores the private channels (contracts §5 PUT /me/contacts).
// They are never returned except to a mutually-matched counterpart.
func (s *Service) UpdateContacts(ctx context.Context, userID ids.ID, c domain.ContactChannel) error {
	if err := s.d.Contacts.Upsert(ctx, userID, c); err != nil {
		return kerr.WrapInternal("could not update contacts", err)
	}
	return nil
}

// AuthorizeContactReveal reports whether requester may see owner's contact
// channels, and if so returns them. The partner context calls this application
// service (via the composition root) after confirming a mutual match — this is
// the ONLY code path that ever serves contact channels (FR-013/SC-006).
func (s *Service) AuthorizeContactReveal(ctx context.Context, ownerID ids.ID) (domain.ContactChannel, bool, error) {
	c, err := s.d.Contacts.Get(ctx, ownerID)
	if err != nil {
		return domain.ContactChannel{}, false, kerr.WrapInternal("could not read contacts", err)
	}
	return c, c.Has(), nil
}

// RevealContacts returns the private contact data of a matched user: the email
// (users row) plus the phone/WhatsApp/Instagram channels. It is invoked ONLY by
// the partner context's revealer after that context has confirmed a mutual
// accepted match — the match check is partner-owned (partner_requests.status),
// identity simply supplies the already-authorised private data (FR-013/SC-006).
type RevealContactsResult struct {
	Email     string
	Phone     string
	WhatsApp  string
	Instagram string
}

func (s *Service) RevealContacts(ctx context.Context, ownerID ids.ID) (*RevealContactsResult, error) {
	u, err := s.d.Users.FindByID(ctx, ownerID)
	if err != nil {
		return nil, kerr.WrapNotFound("user not found", err)
	}
	c, err := s.d.Contacts.Get(ctx, ownerID)
	if err != nil {
		return nil, kerr.WrapInternal("could not read contacts", err)
	}
	return &RevealContactsResult{
		Email:     u.Email,
		Phone:     c.Phone,
		WhatsApp:  c.WhatsApp,
		Instagram: c.Instagram,
	}, nil
}

// LimitedProfile returns the public-safe subset of a user's profile used by the
// partner context's candidate search (US5). It NEVER includes contact channels
// (email/phone/whatsapp/instagram are private until a mutual match). The partner
// domain declares this port; identity provides the adapter (DDD rule: partner
// reads identity via a declared port, never by importing identity internals).
func (s *Service) LimitedProfile(ctx context.Context, userID ids.ID) (*domain.LimitedProfile, error) {
	u, err := s.d.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, kerr.WrapNotFound("profile not found", err)
	}
	if u.Status != domain.StatusActive {
		return nil, kerr.NotFound("profile not available")
	}
	return &domain.LimitedProfile{
		UserID:      u.ID,
		DisplayName: u.DisplayName,
		AvatarKey:   u.AvatarKey,
		HomeRegion:  u.HomeRegion,
		Bio:         u.Bio,
	}, nil
}
