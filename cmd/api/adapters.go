package main

// Cross-context adapter types used by the composition root. Each bridge
// implements a port declared in one context's domain by calling another
// context's application service — no context ever imports another's domain
// types directly (DDD rule from plan.md). The composition root is the one
// place allowed to know every context, so adapters live here.

import (
	"context"

	identityApp "hikingfo/backend/internal/identity/application"
	partnerDomain "hikingfo/backend/internal/partner/domain"
	"hikingfo/backend/internal/shared/ids"
)

// ---- identity → partner: limited profile reader ---------------------------

// profileReader implements partner/domain.ProfileReader by delegating to the
// identity application service. Partner search reads identity through this
// port and never receives private contact channels (FR-013/SC-006).
type profileReader struct{ svc *identityApp.Service }

// LimitedProfile adapts identity's LimitedProfile into the partner candidate shape.
func (a profileReader) LimitedProfile(ctx context.Context, userID ids.ID) (*partnerDomain.CandidateProfile, error) {
	p, err := a.svc.LimitedProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &partnerDomain.CandidateProfile{
		UserID:      p.UserID,
		DisplayName: p.DisplayName,
		AvatarURL:   p.AvatarKey, // blob key; interfaces layer may presign
		HomeRegion:  p.HomeRegion,
		Bio:         p.Bio.ID, // UGC bio is ID-only in v1
	}, nil
}

// ---- identity → partner: matched-contact revealer --------------------------

// contactRevealer implements partner/domain.ContactRevealer. It must be called
// only after the partner context has confirmed a mutual accepted match — this
// adapter performs no match check of its own (identity cannot see partner state).
type contactRevealer struct{ svc *identityApp.Service }

// RevealContacts returns the matched counterpart's private channels.
func (a contactRevealer) RevealContacts(ctx context.Context, ownerID ids.ID) (*partnerDomain.RevealedContact, error) {
	c, err := a.svc.RevealContacts(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return &partnerDomain.RevealedContact{
		Email:     c.Email,
		Phone:     c.Phone,
		WhatsApp:  c.WhatsApp,
		Instagram: c.Instagram,
	}, nil
}
