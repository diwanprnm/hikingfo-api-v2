package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"hikingfo/backend/internal/journey/domain"
	"hikingfo/backend/internal/shared/ids"
)

// JourneyPostRepository is the pgx adapter for domain.JourneyPostRepository.
type JourneyPostRepository struct{ pool *pgxpool.Pool }

// NewJourneyPostRepository wires the adapter over a shared pool.
func NewJourneyPostRepository(pool *pgxpool.Pool) *JourneyPostRepository {
	return &JourneyPostRepository{pool: pool}
}

var _ domain.JourneyPostRepository = (*JourneyPostRepository)(nil)

func (r *JourneyPostRepository) Create(ctx context.Context, p *domain.JourneyPost) error {
	summary, err := json.Marshal(p.Summary)
	if err != nil {
		return err
	}
	narrative, err := json.Marshal(p.Narrative)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO journey_posts
		   (id, user_id, mountain_id, route_id, title, summary, narrative,
		    photo_keys, visibility, moderation_status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		string(p.ID), string(p.UserID), string(p.MountainID),
		nullableID(p.RouteID), p.Title, summary, narrative,
		coalesceStrings(p.PhotoKeys), string(p.Visibility), string(p.ModerationStatus),
	)
	return err
}

func (r *JourneyPostRepository) FindByID(ctx context.Context, id ids.ID) (*domain.JourneyPost, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, mountain_id, route_id, title, summary, narrative,
		        photo_keys, visibility, moderation_status, published_at, updated_at
		   FROM journey_posts WHERE id = $1`, string(id))
	return scanJourneyPost(row)
}

func (r *JourneyPostRepository) Update(ctx context.Context, p *domain.JourneyPost) error {
	summary, err := json.Marshal(p.Summary)
	if err != nil {
		return err
	}
	narrative, err := json.Marshal(p.Narrative)
	if err != nil {
		return err
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE journey_posts
		   SET title = $1, summary = $2, narrative = $3, photo_keys = $4,
		       visibility = $5
		 WHERE id = $6 AND user_id = $7`,
		p.Title, summary, narrative, p.PhotoKeys,
		string(p.Visibility), string(p.ID), string(p.UserID),
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.JourneyPostNotFound{}
	}
	return nil
}

func (r *JourneyPostRepository) Delete(ctx context.Context, id, userID ids.ID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM journey_posts WHERE id = $1 AND user_id = $2`,
		string(id), string(userID))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.JourneyPostNotFound{}
	}
	return nil
}

func (r *JourneyPostRepository) Publish(ctx context.Context, id, userID ids.ID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE journey_posts
		   SET visibility = 'published', published_at = now()
		 WHERE id = $1 AND user_id = $2 AND visibility = 'draft'`,
		string(id), string(userID))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return domain.JourneyPostNotFound{}
	}
	return nil
}

func (r *JourneyPostRepository) ListFeed(ctx context.Context, filter domain.FeedFilter) ([]domain.FeedItem, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	offset := (filter.Page - 1) * filter.PageSize

	where := []string{
		"jp.visibility = 'published'",
		"jp.moderation_status = 'visible'",
		"u.status = 'active'",
	}
	args := []any{}
	argN := 1

	if !filter.MountainID.IsNil() {
		where = append(where, fmt.Sprintf("jp.mountain_id = $%d", argN))
		args = append(args, string(filter.MountainID))
		argN++
	}
	if filter.Region != "" {
		where = append(where, fmt.Sprintf("m.region = $%d::mountain_region", argN))
		args = append(args, filter.Region)
		argN++
	}

	whereClause := strings.Join(where, " AND ")

	// Total count
	var total int
	countQ := fmt.Sprintf(
		`SELECT count(*) FROM journey_posts jp
		  JOIN mountains m ON m.id = jp.mountain_id
		  JOIN users u ON u.id = jp.user_id
		 WHERE %s`, whereClause)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	q := fmt.Sprintf(`
		SELECT jp.id, jp.title, jp.summary,
		       m.id, m.name, m.slug, m.region,
		       u.id, u.display_name, u.avatar_key,
		       jp.published_at, jp.updated_at
		  FROM journey_posts jp
		  JOIN mountains m ON m.id = jp.mountain_id
		  JOIN users u ON u.id = jp.user_id
		 WHERE %s
		 ORDER BY jp.published_at DESC
		 LIMIT $%d OFFSET $%d`, whereClause, argN, argN+1)
	args = append(args, filter.PageSize, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []domain.FeedItem
	for rows.Next() {
		var fi domain.FeedItem
		var summaryJSON []byte
		var region string
		var avatar *string
		if err := rows.Scan(
			&fi.ID, &fi.Title, &summaryJSON,
			&fi.Mountain.ID, &fi.Mountain.Name, &fi.Mountain.Slug, &region,
			&fi.Author.ID, &fi.Author.DisplayName, &avatar,
			&fi.PublishedAt, &fi.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		fi.Mountain.Region = region
		if avatar != nil {
			fi.Author.AvatarURL = *avatar
		}
		fi.Summary = make(map[string]any)
		_ = json.Unmarshal(summaryJSON, &fi.Summary)
		out = append(out, fi)
	}
	return out, total, rows.Err()
}

func (r *JourneyPostRepository) GetFeedItem(ctx context.Context, id ids.ID) (*domain.FeedDetail, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT jp.id, jp.user_id, jp.mountain_id, jp.route_id, jp.title,
		        jp.summary, jp.narrative, jp.photo_keys, jp.visibility,
		        jp.moderation_status, jp.published_at, jp.updated_at,
		        m.id, m.name, m.slug, m.region,
		        u.id, u.display_name, u.avatar_key
		  FROM journey_posts jp
		  JOIN mountains m ON m.id = jp.mountain_id
		  JOIN users u ON u.id = jp.user_id
		 WHERE jp.id = $1`, string(id))

	var d domain.FeedDetail
	var routeID *string
	var summaryJSON, narrativeJSON []byte
	var region string
	var avatar *string
	if err := row.Scan(
		&d.ID, &d.UserID, &d.MountainID, &routeID, &d.Title,
		&summaryJSON, &narrativeJSON, &d.PhotoKeys, &d.Visibility,
		&d.ModerationStatus, &d.PublishedAt, &d.UpdatedAt,
		&d.Mountain.ID, &d.Mountain.Name, &d.Mountain.Slug, &region,
		&d.Author.ID, &d.Author.DisplayName, &avatar,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.JourneyPostNotFound{}
		}
		return nil, err
	}
	d.Mountain.Region = region
	if avatar != nil {
		d.Author.AvatarURL = *avatar
	}
	d.Summary = make(map[string]any)
	_ = json.Unmarshal(summaryJSON, &d.Summary)
	_ = json.Unmarshal(narrativeJSON, &d.Narrative)
	if routeID != nil {
		id := ids.ID(*routeID)
		d.RouteID = &id
	}
	return &d, nil
}

// ---- scan helpers ---------------------------------------------------------

func scanJourneyPost(row scannable) (*domain.JourneyPost, error) {
	var p domain.JourneyPost
	var routeID *string
	var summaryJSON, narrativeJSON []byte
	var visibility, moderation string
	if err := row.Scan(
		&p.ID, &p.UserID, &p.MountainID, &routeID, &p.Title,
		&summaryJSON, &narrativeJSON, &p.PhotoKeys, &visibility,
		&moderation, &p.PublishedAt, &p.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.JourneyPostNotFound{}
		}
		return nil, err
	}
	p.Visibility = domain.PostVisibility(visibility)
	p.ModerationStatus = domain.ModerationStatus(moderation)
	p.Summary = make(map[string]any)
	_ = json.Unmarshal(summaryJSON, &p.Summary)
	_ = json.Unmarshal(narrativeJSON, &p.Narrative)
	if routeID != nil {
		id := ids.ID(*routeID)
		p.RouteID = &id
	}
	return &p, nil
}

// coalesceStrings maps a nil slice to an empty slice (photo_keys NOT NULL).
func coalesceStrings(s []string) any {
	if s == nil {
		return []string{}
	}
	return s
}

// SetModerationStatus flips moderation_status directly (admin path).
func (r *JourneyPostRepository) SetModerationStatus(ctx context.Context, id ids.ID, status domain.ModerationStatus) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE journey_posts SET moderation_status = $2 WHERE id = $1`,
		string(id), string(status))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.JourneyPostNotFound{}
	}
	return nil
}
