package httpapi

import (
	"net/http"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/usecase"
)

type AccountHandler struct {
	UseCase usecase.AccountUseCase
	Bundle  *goi18n.Bundle
}

func (h AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h AccountHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (h AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
