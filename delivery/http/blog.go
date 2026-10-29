package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type BlogHandler struct {
	UseCase usecase.BlogUseCase
	Bundle  *goi18n.Bundle
}

type createBlogBody struct {
	Title   string          `json:"title"`
	Content json.RawMessage `json:"content"`
}

type updateBlogBody struct {
	Title   string          `json:"title"`
	Content json.RawMessage `json:"content"`
}

type publicBlogItemJSON struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	AuthorName string `json:"authorName"`
	CreatedAt  string `json:"createdAt"`
}

type publicBlogDetailJSON struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	AuthorName string `json:"authorName"`
	CreatedAt  string `json:"createdAt"`
}

type publicBlogPageJSON struct {
	Items    []publicBlogItemJSON `json:"items"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
	Total    int                  `json:"total"`
}

type adminBlogItemJSON struct {
	ID         string `json:"id"`
	OwnerID    string `json:"ownerId"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	AuthorName string `json:"authorName"`
	CreatedAt  string `json:"createdAt"`
	DeletedAt  any    `json:"deletedAt"`
}

type adminBlogPageJSON struct {
	Items    []adminBlogItemJSON `json:"items"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
	Total    int                 `json:"total"`
}

type managedBlogJSON struct {
	ID         string `json:"id"`
	OwnerID    string `json:"ownerId"`
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	AuthorName string `json:"authorName"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	DeletedAt  any    `json:"deletedAt"`
}

func pathBlogID(r *http.Request) (uuid.UUID, error) {
	raw := mux.Vars(r)["id"]
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errs.New(errs.CodeInvalidBlogIdentifier, errs.KindInvalidInput, err)
	}
	return id, nil
}

func parseBlogContent(raw json.RawMessage, required bool) (string, error) {
	if len(raw) == 0 {
		if required {
			return "", errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, io.ErrUnexpectedEOF)
		}
		return "", nil
	}
	if string(raw) == "null" {
		return "", errs.New(errs.CodeMalformedRequest, errs.KindInvalidInput, io.ErrUnexpectedEOF)
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", errs.New(errs.CodeMalformedRequest, errs.KindInvalidInput, err)
	}
	return text, nil
}

func optionalQueryInt(r *http.Request, key string) (*int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, err)
	}
	return &value, nil
}

func blogActorFromRequest(r *http.Request) (dto.BlogActor, error) {
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		return dto.BlogActor{}, unauthenticated()
	}
	return dto.BlogActor{ID: actor.ID, Role: actor.Role}, nil
}

func toPublicItemJSON(item dto.PublicBlogItem) publicBlogItemJSON {
	return publicBlogItemJSON{
		ID:         item.ID.String(),
		Slug:       item.Slug,
		Title:      item.Title,
		AuthorName: item.AuthorName,
		CreatedAt:  formatTime(item.CreatedAt),
	}
}

func toPublicDetailJSON(view dto.PublicBlogView) publicBlogDetailJSON {
	return publicBlogDetailJSON{
		ID:         view.ID.String(),
		Slug:       view.Slug,
		Title:      view.Title,
		Content:    view.Content,
		Status:     view.Status,
		AuthorName: view.AuthorName,
		CreatedAt:  formatTime(view.CreatedAt),
	}
}

func toAdminItemJSON(item dto.AdminBlogItem) adminBlogItemJSON {
	return adminBlogItemJSON{
		ID:         item.ID.String(),
		OwnerID:    item.OwnerID.String(),
		Slug:       item.Slug,
		Title:      item.Title,
		Status:     item.Status,
		AuthorName: item.AuthorName,
		CreatedAt:  formatTime(item.CreatedAt),
		DeletedAt:  formatTimePtr(item.DeletedAt),
	}
}

func toManagedJSON(view dto.BlogView) managedBlogJSON {
	return managedBlogJSON{
		ID:         view.ID.String(),
		OwnerID:    view.OwnerID.String(),
		Slug:       view.Slug,
		Title:      view.Title,
		Content:    view.Content,
		Status:     view.Status,
		AuthorName: view.AuthorName,
		CreatedAt:  formatTime(view.CreatedAt),
		UpdatedAt:  formatTime(view.UpdatedAt),
		DeletedAt:  formatTimePtr(view.DeletedAt),
	}
}

func (h BlogHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	page, err := optionalQueryInt(r, "page")
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	pageSize, err := optionalQueryInt(r, "pageSize")
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	result, err := h.UseCase.ListPublic(r.Context(), dto.ListBlogsInput{Page: page, PageSize: pageSize})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	items := make([]publicBlogItemJSON, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toPublicItemJSON(item))
	}
	writeSuccess(w, http.StatusOK, publicBlogPageJSON{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (h BlogHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	view, err := h.UseCase.GetPublic(r.Context(), slug)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toPublicDetailJSON(view))
}

func (h BlogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createBlogBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	content, err := parseBlogContent(body.Content, false)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, err := blogActorFromRequest(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	view, err := h.UseCase.Create(r.Context(), actor, dto.CreateBlogInput{Title: body.Title, Content: content})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusCreated, toManagedJSON(view))
}

func (h BlogHandler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	actor, err := blogActorFromRequest(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	page, err := optionalQueryInt(r, "page")
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	pageSize, err := optionalQueryInt(r, "pageSize")
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	result, err := h.UseCase.ListAdmin(r.Context(), actor, dto.ListAdminBlogsInput{
		Page:     page,
		PageSize: pageSize,
		Deleted:  r.URL.Query().Get("deleted"),
	})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	items := make([]adminBlogItemJSON, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAdminItemJSON(item))
	}
	writeSuccess(w, http.StatusOK, adminBlogPageJSON{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (h BlogHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathBlogID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, err := blogActorFromRequest(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	view, err := h.UseCase.Get(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toManagedJSON(view))
}

func (h BlogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathBlogID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	var body updateBlogBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	content, err := parseBlogContent(body.Content, true)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, err := blogActorFromRequest(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	view, err := h.UseCase.Update(r.Context(), actor, dto.UpdateBlogInput{ID: id, Title: body.Title, Content: content})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toManagedJSON(view))
}

func (h BlogHandler) Publish(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, func(actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
		return h.UseCase.Publish(r.Context(), actor, id)
	})
}

func (h BlogHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, func(actor dto.BlogActor, id uuid.UUID) (dto.BlogView, error) {
		return h.UseCase.Unpublish(r.Context(), actor, id)
	})
}

func (h BlogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathBlogID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, err := blogActorFromRequest(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	if r.ContentLength > 0 {
		var body map[string]any
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, r, h.Bundle, err)
			return
		}
		if len(body) != 0 {
			writeError(w, r, h.Bundle, errs.New(errs.CodeMalformedRequest, errs.KindInvalidInput, io.ErrUnexpectedEOF))
			return
		}
	}
	if err := h.UseCase.Delete(r.Context(), actor, id); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{})
}

func (h BlogHandler) mutate(w http.ResponseWriter, r *http.Request, fn func(dto.BlogActor, uuid.UUID) (dto.BlogView, error)) {
	id, err := pathBlogID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, err := blogActorFromRequest(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	if r.ContentLength > 0 {
		var body map[string]any
		if err := decodeJSON(r, &body); err != nil {
			writeError(w, r, h.Bundle, err)
			return
		}
		if len(body) != 0 {
			writeError(w, r, h.Bundle, errs.New(errs.CodeMalformedRequest, errs.KindInvalidInput, io.ErrUnexpectedEOF))
			return
		}
	}
	view, err := fn(actor, id)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toManagedJSON(view))
}
