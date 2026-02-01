package services

import (
	"context"
	"errors"
	"strings"

	"skyhow/internal/store"

	"github.com/jackc/pgx/v5"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrInvalidInput    = errors.New("invalid input")
)

type GuideService struct {
	Guides *store.GuideStore
}

func NewGuideService(guides *store.GuideStore) *GuideService {
	return &GuideService{Guides: guides}
}

func (s *GuideService) CreateGuide(ctx context.Context, currentUser *store.User, title, content string, tags []string) (string, error) {
	if s.Guides == nil {
		return "", errors.New("guide service not configured")
	}
	if !isAuthedActive(currentUser) {
		return "", ErrUnauthenticated
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return "", ErrInvalidInput
	}
	if content == "" {
		return "", ErrInvalidInput
	}

	guideID, err := s.Guides.CreateGuide(ctx, currentUser.ID, title, content, tags)
	if err != nil {
		return "", err
	}
	return guideID, nil
}

func (s *GuideService) UpdateGuide(ctx context.Context, currentUser *store.User, guideID, title, content string, tags *[]string) error {
	if s.Guides == nil {
		return errors.New("guide service not configured")
	}
	if !isAuthedActive(currentUser) {
		return ErrUnauthenticated
	}
	if strings.TrimSpace(guideID) == "" {
		return ErrInvalidInput
	}

	g, err := s.Guides.GetGuideByID(ctx, guideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if !canEditGuide(currentUser, g.CreatorID) {
		return ErrForbidden
	}

	title = strings.TrimSpace(title)
	if title == "" || content == "" {
		return ErrInvalidInput
	}

	if currentUser.Role != "editor" && currentUser.Role != "admin" {
		if err := s.Guides.UpdateGuide(ctx, guideID, currentUser.ID, title, content); err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
	} else {
		if currentUser.ID != g.CreatorID {
			return errors.New("editor/admin updates require a store method (UpdateGuideAsEditor)")
		}
		if err := s.Guides.UpdateGuide(ctx, guideID, currentUser.ID, title, content); err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
	}

	if tags != nil {
		if currentUser.ID != g.CreatorID {
			return errors.New("editor/admin tag changes require a store method (ReplaceTagsAsEditor)")
		}
		if err := s.Guides.ReplaceTags(ctx, guideID, currentUser.ID, *tags); err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return err
		}
	}

	return nil
}

func (s *GuideService) PublishGuide(ctx context.Context, currentUser *store.User, guideID string) error {
	return s.setStatus(ctx, currentUser, guideID, "published")
}

func (s *GuideService) UnpublishGuide(ctx context.Context, currentUser *store.User, guideID string) error {
	return s.setStatus(ctx, currentUser, guideID, "draft")
}

func (s *GuideService) setStatus(ctx context.Context, currentUser *store.User, guideID, status string) error {
	if s.Guides == nil {
		return errors.New("guide service not configured")
	}
	if !isAuthedActive(currentUser) {
		return ErrUnauthenticated
	}
	if strings.TrimSpace(guideID) == "" {
		return ErrInvalidInput
	}

	g, err := s.Guides.GetGuideByID(ctx, guideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if !canEditGuide(currentUser, g.CreatorID) {
		return ErrForbidden
	}

	if currentUser.ID != g.CreatorID {
		return errors.New("editor/admin status changes require a store method (ChangeStatusAsEditor)")
	}

	if err := s.Guides.ChangeStatus(ctx, guideID, currentUser.ID, status); err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *GuideService) DeleteGuide(ctx context.Context, currentUser *store.User, guideID string) error {
	if s.Guides == nil {
		return errors.New("guide service not configured")
	}
	if !isAuthedActive(currentUser) {
		return ErrUnauthenticated
	}
	if strings.TrimSpace(guideID) == "" {
		return ErrInvalidInput
	}

	g, err := s.Guides.GetGuideByID(ctx, guideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	}

	if !canEditGuide(currentUser, g.CreatorID) {
		return ErrForbidden
	}

	if currentUser.ID != g.CreatorID {
		return errors.New("editor/admin deletes require a store method (DeleteGuideAsEditor)")
	}

	if err := s.Guides.DeleteGuide(ctx, guideID, currentUser.ID); err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *GuideService) GetGuide(ctx context.Context, currentUser *store.User, guideID string) (store.Guide, error) {
	if s.Guides == nil {
		return store.Guide{}, errors.New("guide service not configured")
	}
	if strings.TrimSpace(guideID) == "" {
		return store.Guide{}, ErrInvalidInput
	}

	g, err := s.Guides.GetGuideByID(ctx, guideID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return store.Guide{}, ErrNotFound
		}
		return store.Guide{}, err
	}

	if g.Status == "published" {
		return g, nil
	}

	if !isAuthedActive(currentUser) {
		return store.Guide{}, ErrUnauthenticated
	}
	if !canEditGuide(currentUser, g.CreatorID) {
		return store.Guide{}, ErrForbidden
	}

	return g, nil
}

func (s *GuideService) ListPublishedGuides(ctx context.Context, tagFilter, titleSearch string, limit, offset int) ([]store.Guide, error) {
	if s.Guides == nil {
		return nil, errors.New("guide service not configured")
	}
	return s.Guides.ListPublishedGuides(ctx, tagFilter, titleSearch, limit, offset)
}

func isAuthedActive(u *store.User) bool {
	return u != nil && u.ID != "" && u.IsActive
}

func canEditGuide(u *store.User, creatorID string) bool {
	if u == nil {
		return false
	}
	if u.ID == creatorID {
		return true
	}
	return u.Role == "editor" || u.Role == "admin"
}
