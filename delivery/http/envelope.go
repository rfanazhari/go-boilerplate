package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

type Envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *ErrorBody      `json:"error,omitempty"`
}

type ErrorBody struct {
	Message string       `json:"message"`
	Code    string       `json:"code"`
	Errors  []FieldError `json:"errors"`
}

type FieldError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func writeSuccess(w http.ResponseWriter, statusCode int, data any) {
	payload, _ := json.Marshal(data)
	writeJSON(w, statusCode, Envelope{Status: "success", Data: payload})
}

func writeError(w http.ResponseWriter, r *http.Request, bundle *goi18n.Bundle, err error) {
	var appErr *errs.Error
	if !errors.As(err, &appErr) {
		appErr = errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	status := http.StatusInternalServerError
	switch appErr.Kind {
	case errs.KindInvalidInput:
		status = http.StatusBadRequest
	case errs.KindNotFound:
		status = http.StatusNotFound
	case errs.KindConflict:
		status = http.StatusConflict
	case errs.KindUnavailable:
		status = http.StatusServiceUnavailable
	case errs.KindUnauthenticated:
		status = http.StatusUnauthorized
	case errs.KindForbidden:
		status = http.StatusForbidden
	}
	if appErr.Code == errs.CodeMethodNotAllowed {
		status = http.StatusMethodNotAllowed
	}
	msg := localize(bundle, r.Header.Get("Accept-Language"), appErr.Code)
	writeJSON(w, status, Envelope{
		Status: "error",
		Data:   json.RawMessage("{}"),
		Error: &ErrorBody{
			Message: msg,
			Code:    appErr.Code,
			Errors:  []FieldError{{Message: msg, Code: appErr.Code}},
		},
	})
}

func localize(bundle *goi18n.Bundle, acceptLanguage, code string) string {
	localizer := goi18n.NewLocalizer(bundle, acceptLanguage, "en")
	msg, err := localizer.Localize(&goi18n.LocalizeConfig{MessageID: code})
	if err != nil {
		return code
	}
	return msg
}

func writeJSON(w http.ResponseWriter, statusCode int, body Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
