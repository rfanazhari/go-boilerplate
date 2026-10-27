package httpapi

import (
	"net/http"

	"github.com/google/uuid"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
)

type AccountHandler struct {
	UseCase usecase.AccountUseCase
	Bundle  *goi18n.Bundle
}

type createAccountBody struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type setAccountStatusBody struct {
	Status string `json:"status"`
}

type accountJSON struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func toAccountJSON(view dto.AccountView) accountJSON {
	return accountJSON{
		ID:        view.ID.String(),
		Email:     view.Email,
		Name:      view.Name,
		Role:      view.Role,
		Status:    view.Status,
		CreatedAt: formatTime(view.CreatedAt),
		UpdatedAt: formatTime(view.UpdatedAt),
	}
}

func (h AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createAccountBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	view, err := h.UseCase.CreateAccount(r.Context(), dto.CreateAccountInput{
		Email:    body.Email,
		Name:     body.Name,
		Password: body.Password,
		Role:     body.Role,
	})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusCreated, toAccountJSON(view))
}

func (h AccountHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	accountID, err := pathAccountID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	var body setAccountStatusBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	view, err := h.UseCase.SetAccountStatus(r.Context(), actor.ID, accountID, body.Status)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toAccountJSON(view))
}

func (h AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	accountID, err := pathAccountID(r)
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	if err := h.UseCase.SoftDeleteAccount(r.Context(), actor.ID, accountID); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{})
}

func pathAccountID(r *http.Request) (uuid.UUID, error) {
	return pathID(r)
}
