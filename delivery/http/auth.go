package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/application/dto"
	"github.com/rfanazhari/go-boilerplate/application/usecase"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/repository"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type contextKey string

const actorContextKey contextKey = "httpapi.actor"

type Actor struct {
	ID   uuid.UUID
	Role enum.Role
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	value := ctx.Value(actorContextKey)
	if value == nil {
		return Actor{}, false
	}
	actor, ok := value.(Actor)
	return actor, ok
}

type Auth struct {
	Accounts repository.AccountRepository
	Tokens   repository.TokenRepository
}

type LoginHandler struct {
	UseCase usecase.LoginUseCase
	Bundle  *goi18n.Bundle
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a Auth) Middleware(bundle *goi18n.Bundle) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, err := a.authenticate(r)
			if err != nil {
				writeError(w, r, bundle, err)
				return
			}
			ctx := context.WithValue(r.Context(), actorContextKey, actor)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(bundle *goi18n.Bundle, want enum.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := ActorFromContext(r.Context())
			if !ok || actor.Role != want {
				writeError(w, r, bundle, forbidden())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a Auth) authenticate(r *http.Request) (Actor, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return Actor{}, unauthenticated()
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return Actor{}, unauthenticated()
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if raw == "" {
		return Actor{}, unauthenticated()
	}
	hash := usecase.HashToken(raw)
	accountID, err := a.Tokens.GetByHash(r.Context(), hash)
	if err != nil {
		var appErr *errs.Error
		if errors.As(err, &appErr) && appErr.Kind == errs.KindUnauthenticated {
			return Actor{}, err
		}
		return Actor{}, unauthenticated()
	}
	account, err := a.Accounts.GetByID(r.Context(), accountID)
	if err != nil {
		return Actor{}, unauthenticated()
	}
	if account.IsSoftDeleted() || account.Status() != enum.AccountActive {
		_ = a.Tokens.DeleteByAccountID(r.Context(), accountID)
		return Actor{}, unauthenticated()
	}
	return Actor{ID: account.ID(), Role: account.Role()}, nil
}

func (h LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	result, err := h.UseCase.Login(r.Context(), dto.LoginInput{Email: body.Email, Password: body.Password})
	if err != nil {
		writeError(w, r, h.Bundle, err)
		return
	}
	writeSuccess(w, http.StatusOK, map[string]string{
		"token":     result.Token,
		"tokenType": result.TokenType,
	})
}

func unauthenticated() error {
	return errs.New(errs.CodeUnauthenticated, errs.KindUnauthenticated, errors.New("authentication required"))
}

func forbidden() error {
	return errs.New(errs.CodeForbidden, errs.KindForbidden, errors.New("forbidden"))
}
