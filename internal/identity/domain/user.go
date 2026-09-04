// Package domain holds the identity bounded context's entities, value objects
// and repository ports (plan.md → DDD). Pure Go — no framework, no DB driver.
package domain

import (
	"time"

	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/langtext"
)

// Role distinguishes the single platform admin from regular members.
type Role string

const (
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
)

// Status is a user's moderation lifecycle state (data-model → users).
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusBanned    Status = "banned"
)

// IsActive reports whether the user may use the platform.
func (s Status) IsActive() bool { return s == StatusActive }

// User is a registered person (data-model → users, context identity).
type User struct {
	ID              ids.ID
	Email           string
	PasswordHash    string // empty → Google-only account
	GoogleSub       string // empty → email/password account
	EmailVerifiedAt *time.Time
	DisplayName     string
	AvatarKey       string // blob-storage key (MinIO avatars)
	Bio             langtext.Text
	HomeRegion      string
	Role            Role
	Status          Status
	LastSeenAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsAdmin reports whether the user holds the admin role.
func (u User) IsAdmin() bool { return u.Role == RoleAdmin }

// CanAccess reports whether the user is allowed past the session guard.
func (u User) CanAccess() bool { return u.Status.IsActive() }

// EmailVerified reports whether the primary email was verified.
func (u User) EmailVerified() bool { return u.EmailVerifiedAt != nil }

// ContactChannel is the set of private ways to reach a hiker. Never served in
// public/limited responses; only revealed to a mutually-matched counterpart
// (data-model → users → Private; contracts §5 reveal rule).
type ContactChannel struct {
	Phone     string
	WhatsApp  string
	Instagram string
}

// Has reports whether any private channel is set.
func (c ContactChannel) Has() bool {
	return c.Phone != "" || c.WhatsApp != "" || c.Instagram != ""
}

// LimitedProfile is the public-safe subset of a user's profile that partner
// search (US5) and profile reads may serve. It deliberately omits every private
// contact channel — those leave the identity context only through
// AuthorizeContactReveal after a mutual match (FR-013/SC-006).
type LimitedProfile struct {
	UserID      ids.ID
	DisplayName string
	AvatarKey   string // blob key; may be presigned by the interfaces layer
	HomeRegion  string
	Bio         langtext.Text
}
