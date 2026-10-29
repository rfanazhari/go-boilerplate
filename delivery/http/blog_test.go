package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/rfanazhari/go-boilerplate/test/testdata"
)

type blogHTTPCase struct {
	name       string
	setupToken func(t *testing.T, handler http.Handler) string
	method     string
	path       string
	body       any
	wantStatus int
	wantCode   string
}

func TestBlogHTTPRoutes(t *testing.T) {
	handler, _ := testRouter(t)

	cases := []blogHTTPCase{
		{
			name:       "public list without token succeeds",
			method:     http.MethodGet,
			path:       "/blogs",
			wantStatus: http.StatusOK,
		},
		{
			name:       "management list without token is unauthenticated",
			method:     http.MethodGet,
			path:       "/blog-records",
			wantStatus: http.StatusUnauthorized,
			wantCode:   errs.CodeUnauthenticated,
		},
		{
			name:       "create without token is unauthenticated",
			method:     http.MethodPost,
			path:       "/blog-records",
			body:       map[string]string{"title": testdata.BlogTitle()},
			wantStatus: http.StatusUnauthorized,
			wantCode:   errs.CodeUnauthenticated,
		},
		{
			name: "admin cannot create",
			setupToken: func(t *testing.T, handler http.Handler) string {
				return loginToken(t, handler, "admin@example.com", "admin-pass")
			},
			method:     http.MethodPost,
			path:       "/blog-records",
			body:       map[string]string{"title": testdata.BlogTitle()},
			wantStatus: http.StatusForbidden,
			wantCode:   errs.CodeForbidden,
		},
		{
			name: "invalid blog id is INVALID_BLOG_IDENTIFIER",
			setupToken: func(t *testing.T, handler http.Handler) string {
				return loginToken(t, handler, "user@example.com", "user-pass")
			},
			method:     http.MethodGet,
			path:       "/blog-records/not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantCode:   errs.CodeInvalidBlogIdentifier,
		},
		{
			name: "user create returns 201",
			setupToken: func(t *testing.T, handler http.Handler) string {
				return loginToken(t, handler, "user@example.com", "user-pass")
			},
			method:     http.MethodPost,
			path:       "/blog-records",
			body:       map[string]string{"title": testdata.BlogTitle()},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			bearer := ""
			if tt.setupToken != nil {
				bearer = tt.setupToken(t, handler)
			}
			status, env := doJSON(t, handler, tt.method, tt.path, tt.body, bearer)
			assert.Equal(t, tt.wantStatus, status)
			if tt.wantCode != "" {
				require.NotNil(t, env.Error)
				assert.Equal(t, tt.wantCode, env.Error.Code)
				return
			}
			assert.Equal(t, "success", env.Status)
		})
	}
}

func loginToken(t *testing.T, handler http.Handler, email, password string) string {
	t.Helper()
	_, loginEnv := doJSON(t, handler, http.MethodPost, "/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	var loginData struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(loginEnv.Data, &loginData))
	require.NotEmpty(t, loginData.Token)
	return loginData.Token
}
