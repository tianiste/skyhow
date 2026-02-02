package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"skyhow/internal/services"
	"skyhow/internal/store"

	"github.com/gin-gonic/gin"
)

type GuideHandler struct {
	Guides *services.GuideService
}

func NewGuideHandler(guides *services.GuideService) *GuideHandler {
	return &GuideHandler{Guides: guides}
}

type createGuideRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type updateGuideRequest struct {
	Title   string    `json:"title"`
	Content string    `json:"content"`
	Tags    *[]string `json:"tags"`
}

type guideResponse struct {
	ID        string   `json:"id"`
	CreatorID string   `json:"creator_id"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Status    string   `json:"status"`
	Tags      []tagDTO `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type tagDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type guideListItemResponse struct {
	ID        string   `json:"id"`
	CreatorID string   `json:"creator_id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Tags      []tagDTO `json:"tags"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func (h *GuideHandler) Create(c *gin.Context) {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var req createGuideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	guideID, err := h.Guides.CreateGuide(c.Request.Context(), &currentUser, req.Title, req.Content, req.Tags)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": guideID})
}

func (h *GuideHandler) Update(c *gin.Context) {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	guideID := strings.TrimSpace(c.Param("id"))
	if guideID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing guide id"})
		return
	}

	var req updateGuideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	err := h.Guides.UpdateGuide(c.Request.Context(), &currentUser, guideID, req.Title, req.Content, req.Tags)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *GuideHandler) Publish(c *gin.Context) {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	guideID := strings.TrimSpace(c.Param("id"))
	if guideID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing guide id"})
		return
	}

	if err := h.Guides.PublishGuide(c.Request.Context(), &currentUser, guideID); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *GuideHandler) Unpublish(c *gin.Context) {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	guideID := strings.TrimSpace(c.Param("id"))
	if guideID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing guide id"})
		return
	}

	if err := h.Guides.UnpublishGuide(c.Request.Context(), &currentUser, guideID); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *GuideHandler) Delete(c *gin.Context) {
	currentUser, ok := getCurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	guideID := strings.TrimSpace(c.Param("id"))
	if guideID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing guide id"})
		return
	}

	if err := h.Guides.DeleteGuide(c.Request.Context(), &currentUser, guideID); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *GuideHandler) Get(c *gin.Context) {
	guideID := strings.TrimSpace(c.Param("id"))
	if guideID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing guide id"})
		return
	}

	var userPtr *store.User
	if u, ok := getCurrentUser(c); ok {
		userPtr = &u
	}

	g, err := h.Guides.GetGuide(c.Request.Context(), userPtr, guideID)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGuideResponse(g))
}

func (h *GuideHandler) ListPublished(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	q := strings.TrimSpace(c.Query("q"))

	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)

	guides, err := h.Guides.ListPublishedGuides(c.Request.Context(), tag, q, limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	out := make([]guideListItemResponse, 0, len(guides))
	for _, g := range guides {
		out = append(out, toGuideListItemResponse(g))
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  out,
		"limit":  limit,
		"offset": offset,
	})
}

func getCurrentUser(c *gin.Context) (store.User, bool) {
	uAny, ok := c.Get("user")
	if !ok || uAny == nil {
		return store.User{}, false
	}
	u, ok := uAny.(store.User)
	if !ok {
		return store.User{}, false
	}
	if u.ID == "" || !u.IsActive {
		return store.User{}, false
	}
	return u, true
}

func writeServiceError(c *gin.Context, err error) {
	switch err {
	case services.ErrUnauthenticated:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
	case services.ErrForbidden:
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case services.ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case services.ErrInvalidInput:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func parseIntDefault(v string, def int) int {
	if strings.TrimSpace(v) == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func toGuideResponse(g store.Guide) guideResponse {
	tags := make([]tagDTO, 0, len(g.Tags))
	for _, t := range g.Tags {
		tags = append(tags, tagDTO{ID: t.ID, Name: t.Name})
	}

	return guideResponse{
		ID:        g.ID,
		CreatorID: g.CreatorID,
		Title:     g.Title,
		Content:   g.Content,
		Status:    g.Status,
		Tags:      tags,
		CreatedAt: g.CreatedAt.Format(timeRFC3339()),
		UpdatedAt: g.UpdatedAt.Format(timeRFC3339()),
	}
}

func toGuideListItemResponse(g store.Guide) guideListItemResponse {
	tags := make([]tagDTO, 0, len(g.Tags))
	for _, t := range g.Tags {
		tags = append(tags, tagDTO{ID: t.ID, Name: t.Name})
	}

	return guideListItemResponse{
		ID:        g.ID,
		CreatorID: g.CreatorID,
		Title:     g.Title,
		Status:    g.Status,
		Tags:      tags,
		CreatedAt: g.CreatedAt.Format(timeRFC3339()),
		UpdatedAt: g.UpdatedAt.Format(timeRFC3339()),
	}
}

func timeRFC3339() string {
	return time.RFC3339
}
