package domain

import (
	"context"
	"time"

	"hikingfo/backend/internal/shared/ids"
)

// Revision is a full-row snapshot of a mountain captured before every admin
// edit (data-model → mountain_revisions; spec Decisions → Versioning).
type Revision struct {
	ID         ids.ID
	MountainID ids.ID
	Snapshot   map[string]any
	RevisedBy  *ids.ID
	Reason     string
	CreatedAt  time.Time
}

// EditLogEntry is one append-only attribution row for photo/route/basecamp
// changes (002 data-model E-AdminEditLog). Mountain-profile edits keep using
// Revision snapshots; sub-object changes are attributed here (FR-013).
type EditLogEntry struct {
	ID         ids.ID        `json:"id"`
	AdminID    ids.ID        `json:"admin_id"`
	EntityType string        `json:"entity_type"` // mountain | route | basecamp
	EntityID   ids.ID        `json:"entity_id"`
	MountainID ids.ID        `json:"mountain_id"`
	Action     string        `json:"action"` // create | update | delete | photo_set | photo_remove
	Reason     string        `json:"reason"`
	Detail     map[string]any `json:"detail"`
	CreatedAt  time.Time     `json:"created_at"`
}

// Edit-log entity/action constants.
const (
	EditEntityMountain = "mountain"
	EditEntityRoute    = "route"
	EditEntityBasecamp = "basecamp"

	EditActionCreate      = "create"
	EditActionUpdate      = "update"
	EditActionDelete      = "delete"
	EditActionPhotoSet    = "photo_set"
	EditActionPhotoRemove = "photo_remove"
)

// MountainWriter is the admin persistence port for curated mountains
// (contracts §9). Reads stay on MountainRepository; writes run in a single
// transaction that snapshots the prior row into mountain_revisions.
type MountainWriter interface {
	// Create inserts a new curated mountain (draft or published).
	Create(ctx context.Context, m *Mountain) error
	// Update persists mutable fields and atomically appends a revision snapshot
	// of the row as it was BEFORE the edit.
	Update(ctx context.Context, m *Mountain, revisedBy *ids.ID, reason string) error
	// Delete removes a mountain (routes/basecamps/revisions cascade).
	Delete(ctx context.Context, id ids.ID) error
	// ListAll returns every mountain regardless of publish status.
	ListAll(ctx context.Context) ([]Mountain, error)
	// Revisions returns the edit history of a mountain, newest first.
	Revisions(ctx context.Context, mountainID ids.ID) ([]Revision, error)
	// Rollback restores a prior snapshot (itself snapshotting the current row).
	Rollback(ctx context.Context, mountainID, revisionID ids.ID, revisedBy *ids.ID) error
	// RouteWriter manages the sub-objects of a mountain. Sub-object mutations
	// atomically append an admin_edit_log row attributing the change (FR-013).
	CreateRoute(ctx context.Context, mountainID, adminID ids.ID, r *Route, reason string) error
	UpdateRoute(ctx context.Context, adminID ids.ID, r *Route, reason string) error
	DeleteRoute(ctx context.Context, id, adminID ids.ID, reason string) error
	CreateBasecamp(ctx context.Context, mountainID, adminID ids.ID, b *Basecamp, reason string) error
	UpdateBasecamp(ctx context.Context, adminID ids.ID, b *Basecamp, reason string) error
	DeleteBasecamp(ctx context.Context, id, adminID ids.ID, reason string) error
	// SetPhoto writes the cover photo key and returns the PRIOR key ("" when
	// there was none) so the caller can best-effort delete the old object
	// (002 contracts §Photo). It also snapshots a revision + appends an
	// edit-log row (photo_set | photo_remove) in one transaction.
	SetPhoto(ctx context.Context, mountainID, adminID ids.ID, key, reason string, deleted bool) (string, error)
	// AppendEditLog writes one attribution row.
	AppendEditLog(ctx context.Context, e EditLogEntry) error
	// EditLog returns a mountain's change trail, newest first.
	EditLog(ctx context.Context, mountainID ids.ID) ([]EditLogEntry, error)
}
