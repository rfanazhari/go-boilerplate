package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"
	"github.com/SALT-Indonesia/salt-pkg/logmanager/integrations/lmgorilla"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Server struct {
	Handler TaskHandler
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
	router.HandleFunc("/accounts", s.Account.Create).Methods(http.MethodPost)
	router.HandleFunc("/profile", s.Profile.Update).Methods(http.MethodPatch)
	router.HandleFunc("/tasks", s.Handler.Create).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{id}/start", s.Handler.Start).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{id}/complete", s.Handler.Complete).Methods(http.MethodPost)
	router.HandleFunc("/tasks/{id}/due-date", s.Handler.Reschedule).Methods(http.MethodPatch)
	router.HandleFunc("/tasks/{id}", s.Handler.Get).Methods(http.MethodGet)
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
