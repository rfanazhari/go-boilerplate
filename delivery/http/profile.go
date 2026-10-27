package httpapi

import (
	"net/http"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
)

type ProfileHandler struct {
	UseCase usecase.AccountUseCase
	Bundle  *goi18n.Bundle
}

type profileBody struct {
	Name     *string `json:"name"`
	Password *string `json:"password"`
}

func (h ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	actor, ok := ActorFromContext(r.Context())
	if !ok {
		writeError(w, r, h.Bundle, unauthenticated())
		return
	}
	var body profileBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	view, err := h.UseCase.UpdateProfile(r.Context(), actor.ID, dto.UpdateProfileInput{
		Name:     body.Name,
		Password: body.Password,
	})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, toAccountJSON(view))
}
