package httpapi

import (
	"net/http"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
)

type Auth struct {
	Accounts repository.AccountRepository
	Tokens   repository.TokenRepository
}

type LoginHandler struct {
	UseCase usecase.LoginUseCase
	Bundle  *goi18n.Bundle
}

func (Auth) Middleware(next http.Handler) http.Handler {
	return next
}

func RequireRole(_ enum.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}

func (h LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
