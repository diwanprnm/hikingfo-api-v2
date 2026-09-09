package application

import (
	"context"

	"hikingfo/backend/internal/identity/domain"
	"hikingfo/backend/internal/shared/ids"
	"hikingfo/backend/internal/shared/kerr"
)

// AdminUserView is the admin-facing projection of a user (contracts §9
// GET /admin/users). Serves status/role/join date; omits contact channels.
type AdminUserView struct {
	ID            ids.ID `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	DisplayName   string `json:"display_name"`
	HomeRegion    string `json:"home_region"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	JoinedAt      string `json:"joined_at"`
}

// AdminGetUser loads one user for the admin panel.
func (s *Service) AdminGetUser(ctx context.Context, id ids.ID) (*AdminUserView, error) {
	u, err := s.d.Users.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("user not found")
	}
	return adminUserView(u), nil
}

// AdminSetUserStatus suspends, bans or reactivates a user and revokes every
// active session on a restrictive change (contracts §9).
func (s *Service) AdminSetUserStatus(ctx context.Context, id ids.ID, status string) (*AdminUserView, error) {
	st := domain.Status(status)
	switch st {
	case domain.StatusActive, domain.StatusSuspended, domain.StatusBanned:
	default:
		return nil, kerr.Validation("status must be active, suspended or banned")
	}
	u, err := s.d.Users.FindByID(ctx, id)
	if err != nil {
		return nil, kerr.NotFound("user not found")
	}
	if u.Role == domain.RoleAdmin && st != domain.StatusActive {
		return nil, kerr.Forbidden("cannot suspend or ban an admin")
	}
	if err := s.d.Users.SetStatus(ctx, id, st); err != nil {
		return nil, kerr.WrapInternal("could not set user status", err)
	}
	if st != domain.StatusActive {
		if err := s.d.Sessions.RevokeAllForUser(ctx, id); err != nil {
			return nil, kerr.WrapInternal("could not revoke sessions", err)
		}
	}
	u.Status = st
	return adminUserView(u), nil
}

func adminUserView(u *domain.User) *AdminUserView {
	return &AdminUserView{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified(),
		DisplayName:   u.DisplayName,
		HomeRegion:    u.HomeRegion,
		Role:          string(u.Role),
		Status:        string(u.Status),
		JoinedAt:      u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
