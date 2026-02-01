package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Tag struct {
	ID   string
	Name string
}

type Guide struct {
	ID        string
	CreatorID string
	Title     string
	Content   string
	Status    string
	Tags      []Tag

	CreatedAt time.Time
	UpdatedAt time.Time
}

type GuideStore struct {
	db *pgxpool.Pool
}

func NewGuideStore(db *pgxpool.Pool) *GuideStore {
	return &GuideStore{db: db}
}

func (s *GuideStore) CreateGuide(ctx context.Context, creatorID, title, content string, tags []string) (string, error) {
	if creatorID == "" {
		return "", errors.New("creatorID is required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errors.New("title is required")
	}
	if content == "" {
		return "", errors.New("content is required")
	}

	tagNames := normalizeTagNames(tags)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var guideID string
	err = tx.QueryRow(ctx, `
		insert into public.guides (creator_id, title, content, status)
		values ($1, $2, $3, 'draft')
		returning id;
	`, creatorID, title, content).Scan(&guideID)
	if err != nil {
		return "", err
	}

	if len(tagNames) > 0 {
		if err := upsertTags(ctx, tx, tagNames); err != nil {
			return "", err
		}
		if err := replaceGuideTags(ctx, tx, guideID, tagNames); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return guideID, nil
}

func (s *GuideStore) UpdateGuide(ctx context.Context, guideID, creatorID, title, content string) error {
	if guideID == "" || creatorID == "" {
		return errors.New("guideID and creatorID are required")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title is required")
	}
	if content == "" {
		return errors.New("content is required")
	}

	ct, err := s.db.Exec(ctx, `
		update public.guides
		set title = $3,
		    content = $4,
		    updated_at = now()
		where id = $1
		  and creator_id = $2;
	`, guideID, creatorID, title, content)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *GuideStore) ChangeStatus(ctx context.Context, guideID, creatorID, status string) error {
	if guideID == "" || creatorID == "" {
		return errors.New("guideID and creatorID are required")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "draft" && status != "published" {
		return errors.New("invalid status: must be 'draft' or 'published'")
	}

	ct, err := s.db.Exec(ctx, `
		update public.guides
		set status = $3,
		    updated_at = now()
		where id = $1
		  and creator_id = $2;
	`, guideID, creatorID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *GuideStore) DeleteGuide(ctx context.Context, guideID, creatorID string) error {
	if guideID == "" || creatorID == "" {
		return errors.New("guideID and creatorID are required")
	}

	ct, err := s.db.Exec(ctx, `
		delete from public.guides
		where id = $1
		  and creator_id = $2;
	`, guideID, creatorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *GuideStore) ReplaceTags(ctx context.Context, guideID, creatorID string, tags []string) error {
	if guideID == "" || creatorID == "" {
		return errors.New("guideID and creatorID are required")
	}

	tagNames := normalizeTagNames(tags)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var one int
	err = tx.QueryRow(ctx, `
		select 1
		from public.guides
		where id = $1 and creator_id = $2
		for update;
	`, guideID, creatorID).Scan(&one)
	if err != nil {
		return err
	}

	if len(tagNames) > 0 {
		if err := upsertTags(ctx, tx, tagNames); err != nil {
			return err
		}
	}

	if err := replaceGuideTags(ctx, tx, guideID, tagNames); err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		update public.guides
		set updated_at = now()
		where id = $1 and creator_id = $2;
	`, guideID, creatorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *GuideStore) GetGuideByID(ctx context.Context, guideID string) (Guide, error) {
	var g Guide
	if guideID == "" {
		return g, errors.New("guideID is required")
	}

	err := s.db.QueryRow(ctx, `
		select id, creator_id, title, content, status, created_at, updated_at
		from public.guides
		where id = $1;
	`, guideID).Scan(
		&g.ID,
		&g.CreatorID,
		&g.Title,
		&g.Content,
		&g.Status,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if err != nil {
		return g, err
	}

	rows, err := s.db.Query(ctx, `
		select t.id, t.name
		from public.tags t
		join public.guide_tags gt on gt.tag_id = t.id
		where gt.guide_id = $1
		order by t.name asc;
	`, guideID)
	if err != nil {
		return g, err
	}
	defer rows.Close()

	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return g, err
		}
		g.Tags = append(g.Tags, t)
	}
	if err := rows.Err(); err != nil {
		return g, err
	}

	return g, nil
}

func (s *GuideStore) ListPublishedGuides(ctx context.Context, tagFilter string, titleSearch string, limit, offset int) ([]Guide, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	tagFilter = strings.ToLower(strings.TrimSpace(tagFilter))
	titleSearch = strings.TrimSpace(titleSearch)
	var tagParam *string
	var searchParam *string
	if tagFilter != "" {
		tagParam = &tagFilter
	}
	if titleSearch != "" {
		searchParam = &titleSearch
	}

	rows, err := s.db.Query(ctx, `
		select
		  g.id,
		  g.creator_id,
		  g.title,
		  g.content,
		  g.status,
		  g.created_at,
		  g.updated_at
		from public.guides g
		where g.status = 'published'
		  and ($2::text is null or g.title ilike ('%' || $2 || '%'))
		  and (
		    $1::text is null
		    or exists (
		      select 1
		      from public.guide_tags gt
		      join public.tags t on t.id = gt.tag_id
		      where gt.guide_id = g.id
		        and t.name = $1
		    )
		  )
		order by g.created_at desc
		limit $3 offset $4;
	`, tagParam, searchParam, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Guide
	for rows.Next() {
		var g Guide
		if err := rows.Scan(&g.ID, &g.CreatorID, &g.Title, &g.Content, &g.Status, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeTagNames(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))

	for _, t := range tags {
		n := strings.ToLower(strings.TrimSpace(t))
		if n == "" {
			continue
		}
		if len(n) > 50 {
			n = n[:50]
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func upsertTags(ctx context.Context, tx pgx.Tx, tagNames []string) error {
	_, err := tx.Exec(ctx, `
		insert into public.tags (name)
		select distinct unnest($1::text[])
		on conflict (name) do nothing;
	`, tagNames)
	return err
}

func replaceGuideTags(ctx context.Context, tx pgx.Tx, guideID string, tagNames []string) error {
	_, err := tx.Exec(ctx, `delete from public.guide_tags where guide_id = $1;`, guideID)
	if err != nil {
		return err
	}

	if len(tagNames) == 0 {
		return nil
	}

	_, err = tx.Exec(ctx, `
		insert into public.guide_tags (guide_id, tag_id)
		select $1, t.id
		from public.tags t
		where t.name = any($2::text[])
		on conflict do nothing;
	`, guideID, tagNames)
	return err
}
