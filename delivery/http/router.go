package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/SALT-Indonesia/salt-pkg/logmanager/integrations/lmgorilla"

	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Server struct {
	Handler TaskHandler
	Blog    BlogHandler
	Login   LoginHandler
	Account AccountHandler
	Profile ProfileHandler
	Auth    Auth
	Bundle  *goi18n.Bundle
	Log     *logmanager.Application
}

func (s Server) Router() http.Handler {
	router := mux.NewRouter()
	router.Use(lmgorilla.Middleware(s.Log))
	router.HandleFunc("/health", s.health).Methods(http.MethodGet)
	router.HandleFunc("/login", s.Login.Login).Methods(http.MethodPost)
	router.HandleFunc("/blogs", s.Blog.ListPublic).Methods(http.MethodGet)
	router.HandleFunc("/blogs/{slug}", s.Blog.GetPublic).Methods(http.MethodGet)

	auth := s.Auth.Middleware(s.Bundle)
	protected := router.NewRoute().Subrouter()
	protected.Use(auth)

	protected.HandleFunc("/blog-records/{id}/publish", s.Blog.Publish).Methods(http.MethodPost)
	protected.HandleFunc("/blog-records/{id}/unpublish", s.Blog.Unpublish).Methods(http.MethodPost)
	protected.HandleFunc("/blog-records/{id}", s.Blog.Get).Methods(http.MethodGet)
	protected.HandleFunc("/blog-records/{id}", s.Blog.Update).Methods(http.MethodPatch)
	protected.HandleFunc("/blog-records/{id}", s.Blog.Delete).Methods(http.MethodDelete)
	protected.HandleFunc("/tasks/{id}", s.Handler.Get).Methods(http.MethodGet)
	protected.HandleFunc("/tasks/{id}/start", s.Handler.Start).Methods(http.MethodPost)
	protected.HandleFunc("/tasks/{id}/complete", s.Handler.Complete).Methods(http.MethodPost)
	protected.HandleFunc("/tasks/{id}/due-date", s.Handler.Reschedule).Methods(http.MethodPatch)

	admin := protected.NewRoute().Subrouter()
	admin.Use(RequireRole(s.Bundle, enum.RoleAdmin))
	admin.HandleFunc("/blog-records", s.Blog.ListAdmin).Methods(http.MethodGet)
	admin.HandleFunc("/accounts", s.Account.Create).Methods(http.MethodPost)
	admin.HandleFunc("/accounts/{id}/status", s.Account.SetStatus).Methods(http.MethodPatch)
	admin.HandleFunc("/accounts/{id}", s.Account.Delete).Methods(http.MethodDelete)

	user := protected.NewRoute().Subrouter()
	user.Use(RequireRole(s.Bundle, enum.RoleUser))
	user.HandleFunc("/blog-records", s.Blog.Create).Methods(http.MethodPost)
	user.HandleFunc("/tasks", s.Handler.Create).Methods(http.MethodPost)
	user.HandleFunc("/profile", s.Profile.Update).Methods(http.MethodPatch)

	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, s.Bundle, errs.New(errs.CodeMethodNotAllowed, errs.KindInvalidInput, nil))
	})
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, s.Bundle, errs.New(errs.CodeRouteNotFound, errs.KindNotFound, nil))
	})
	return router
}

func (s Server) health(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, http.StatusOK, map[string]any{})
}
