package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type TaskHandler struct {
	UseCase usecase.TaskUseCase
	Bundle  *goi18n.Bundle
}

type createBody struct {
	Title   string          `json:"title"`
	DueDate json.RawMessage `json:"dueDate"`
}

type rescheduleBody struct {
	DueDate json.RawMessage `json:"dueDate"`
}

type taskJSON struct {
	ID          string `json:"id"`
	OwnerID     string `json:"ownerId"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	DueDate     any    `json:"dueDate"`
	Overdue     bool   `json:"overdue"`
	CreatedAt   string `json:"createdAt"`
	CompletedAt any    `json:"completedAt"`
}

func decodeJSON(r *http.Request, dest any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		if err == io.EOF {
			return nil
		}
		return errs.New(errs.CodeMalformedRequest, errs.KindInvalidInput, err)
	}
	if decoder.More() {
		return errs.New(errs.CodeMalformedRequest, errs.KindInvalidInput, io.ErrUnexpectedEOF)
	}
	return nil
}

func parseOptionalDue(raw json.RawMessage, required bool) (*time.Time, error) {
	if len(raw) == 0 {
		if required {
			return nil, errs.New(errs.CodeValidationFailed, errs.KindInvalidInput, io.ErrUnexpectedEOF)
		}
		return nil, nil
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, errs.New(errs.CodeInvalidDueDate, errs.KindInvalidInput, err)
	}
	parsed, err := parseRFC3339Millis(text)
	if err != nil {
		return nil, errs.New(errs.CodeInvalidDueDate, errs.KindInvalidInput, err)
	}
	return &parsed, nil
}

func pathID(r *http.Request) (uuid.UUID, error) {
	raw := mux.Vars(r)["id"]
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, errs.New(errs.CodeInvalidIdentifier, errs.KindInvalidInput, err)
	}
	return id, nil
}

func toJSON(view dto.TaskView) taskJSON {
	return taskJSON{
		ID:          view.ID.String(),
		OwnerID:     view.OwnerID.String(),
		Title:       view.Title,
		Status:      view.Status,
		DueDate:     formatTimePtr(view.DueDate),
		Overdue:     view.Overdue,
		CreatedAt:   formatTime(view.CreatedAt),
		CompletedAt: formatTimePtr(view.CompletedAt),
	}
}

func (h TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	due, err := parseOptionalDue(body.DueDate, false)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	view, err := h.UseCase.Create(r.Context(), dto.TaskActor{ID: actor.ID, Role: actor.Role}, dto.CreateTaskInput{Title: body.Title, DueDate: due})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusCreated, toJSON(view))
}

func (h TaskHandler) Start(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, func(actor dto.TaskActor, id uuid.UUID) (dto.TaskView, error) {
		return h.UseCase.Start(r.Context(), actor, id)
	})
}

func (h TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, func(actor dto.TaskActor, id uuid.UUID) (dto.TaskView, error) {
		return h.UseCase.Complete(r.Context(), actor, id)
	})
}

func (h TaskHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	var body rescheduleBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	due, err := parseOptionalDue(body.DueDate, true)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	view, err := h.UseCase.Reschedule(r.Context(), dto.TaskActor{ID: actor.ID, Role: actor.Role}, dto.RescheduleTaskInput{ID: id, DueDate: due})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toJSON(view))
}

func (h TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	view, err := h.UseCase.Get(r.Context(), dto.TaskActor{ID: actor.ID, Role: actor.Role}, id)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toJSON(view))
}

func (h TaskHandler) mutate(w http.ResponseWriter, r *http.Request, fn func(dto.TaskActor, uuid.UUID) (dto.TaskView, error)) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	taskActor := dto.TaskActor{ID: actor.ID, Role: actor.Role}
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
	view, err := fn(taskActor, id)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toJSON(view))
}
