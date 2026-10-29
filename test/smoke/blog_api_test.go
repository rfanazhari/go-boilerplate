package smoke_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startBlogSmokeAPI(t *testing.T) (base string, adminID, userID, strangerID uuid.UUID) {
	t.Helper()
	return startSmokeAPI(t),
		uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
}

func blogContentHTML(n int) string {
	return `"<p>` + strings.Repeat("a", n) + `</p>"`
}

func TestBlogAPISmoke(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	base, _, userID, _ := startBlogSmokeAPI(t)

	t.Run("post blog-records without token returns 401", func(t *testing.T) {
		postJSON(t, ctx, base+"/blog-records", `{"title":"Catatan Go","content":""}`, http.StatusUnauthorized)
	})

	adminToken := loginToken(t, ctx, base, "admin@example.com", "admin-pass")
	userToken := loginToken(t, ctx, base, "user@example.com", "user-pass")

	t.Run("admin create blog returns 403", func(t *testing.T) {
		postJSON(t, ctx, base+"/blog-records", `{"title":"Catatan Go","content":""}`, http.StatusForbidden, adminToken)
	})

	t.Run("user list blog-records returns 403", func(t *testing.T) {
		getJSON(t, ctx, base+"/blog-records", http.StatusForbidden, userToken)
	})

	created := postJSON(t, ctx, base+"/blog-records", `{"title":"Catatan Go","content":""}`, http.StatusCreated, userToken)
	blogID, _ := created.Data["id"].(string)
	slug, _ := created.Data["slug"].(string)
	require.NotEmpty(t, blogID)
	require.NotEmpty(t, slug)
	assert.Equal(t, "draft", created.Data["status"])
	assert.Equal(t, userID.String(), created.Data["ownerId"])

	t.Run("draft slug is absent from public list and detail is not found", func(t *testing.T) {
		list := getJSON(t, ctx, base+"/blogs", http.StatusOK)
		items, _ := list.Data["items"].([]any)
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			assert.NotEqual(t, slug, item["slug"])
		}
		notFound := getJSON(t, ctx, base+"/blogs/"+slug, http.StatusNotFound)
		assert.Equal(t, "error", notFound.Status)
		assert.Equal(t, "BLOG_NOT_FOUND", notFound.Error["code"])
	})

	patchJSON(t, ctx, base+"/blog-records/"+blogID, `{"title":"Catatan Go","content":`+blogContentHTML(100)+`}`, http.StatusOK, userToken)
	published := postJSON(t, ctx, base+"/blog-records/"+blogID+"/publish", `{}`, http.StatusOK, userToken)
	assert.Equal(t, "published", published.Data["status"])
	getJSON(t, ctx, base+"/blogs/"+slug, http.StatusOK)

	t.Run("public reads ignore garbage authorization when blog is eligible", func(t *testing.T) {
		withToken := getJSON(t, ctx, base+"/blogs/"+slug, http.StatusOK, "not-a-real-token")
		assert.Equal(t, "published", withToken.Data["status"])
		getJSON(t, ctx, base+"/blogs", http.StatusOK, "not-a-real-token")
	})

	postJSON(t, ctx, base+"/blog-records/"+blogID+"/unpublish", `{}`, http.StatusOK, userToken)
	patchJSON(t, ctx, base+"/blog-records/"+blogID, `{"title":"Catatan Go","content":`+blogContentHTML(99)+`}`, http.StatusOK, userToken)

	t.Run("publish with content length 99 is rejected and stays off public channel", func(t *testing.T) {
		fail := postJSON(t, ctx, base+"/blog-records/"+blogID+"/publish", `{}`, http.StatusBadRequest, userToken)
		assert.Equal(t, "error", fail.Status)
		getJSON(t, ctx, base+"/blogs/"+slug, http.StatusNotFound)
	})

	patchJSON(t, ctx, base+"/blog-records/"+blogID, `{"title":"Catatan Go","content":`+blogContentHTML(100)+`}`, http.StatusOK, userToken)
	postJSON(t, ctx, base+"/blog-records/"+blogID+"/publish", `{}`, http.StatusOK, userToken)

	t.Run("admin cannot patch or publish owner blog", func(t *testing.T) {
		patchJSON(t, ctx, base+"/blog-records/"+blogID, `{"title":"Catatan Go","content":`+blogContentHTML(100)+`}`, http.StatusForbidden, adminToken)
		postJSON(t, ctx, base+"/blog-records/"+blogID+"/publish", `{}`, http.StatusForbidden, adminToken)
	})

	t.Run("admin unpublish hides blog from public detail", func(t *testing.T) {
		postJSON(t, ctx, base+"/blog-records/"+blogID+"/unpublish", `{}`, http.StatusOK, adminToken)
		getJSON(t, ctx, base+"/blogs/"+slug, http.StatusNotFound)
	})

	t.Run("owner publish again restores public detail", func(t *testing.T) {
		postJSON(t, ctx, base+"/blog-records/"+blogID+"/publish", `{}`, http.StatusOK, userToken)
		detail := getJSON(t, ctx, base+"/blogs/"+slug, http.StatusOK)
		assert.Equal(t, "published", detail.Data["status"])
	})

	t.Run("inactive author hides published blog from public reads", func(t *testing.T) {
		patchJSON(t, ctx, base+"/accounts/"+userID.String()+"/status", `{"status":"non_active"}`, http.StatusOK, adminToken)
		getJSON(t, ctx, base+"/blogs/"+slug, http.StatusNotFound)
		patchJSON(t, ctx, base+"/accounts/"+userID.String()+"/status", `{"status":"active"}`, http.StatusOK, adminToken)
		again := getJSON(t, ctx, base+"/blogs/"+slug, http.StatusOK)
		assert.Equal(t, "published", again.Data["status"])
	})
	userToken = loginToken(t, ctx, base, "user@example.com", "user-pass")

	draft := postJSON(t, ctx, base+"/blog-records", `{"title":"Catatan Go Dua","content":""}`, http.StatusCreated, userToken)
	draftID, _ := draft.Data["id"].(string)
	strangerToken := loginToken(t, ctx, base, "stranger@example.com", "stranger-pass")
	t.Run("stranger gets forbidden on a published blog and not found on a draft", func(t *testing.T) {
		getJSON(t, ctx, base+"/blog-records/"+blogID, http.StatusForbidden, strangerToken)
		getJSON(t, ctx, base+"/blog-records/"+draftID, http.StatusNotFound, strangerToken)
	})

	t.Run("owner delete then owner get is not found while admin still sees it", func(t *testing.T) {
		deleted := deleteJSON(t, ctx, base+"/blog-records/"+blogID, http.StatusOK, userToken)
		assert.Equal(t, "success", deleted.Status)
		assert.Empty(t, deleted.Data)
		getJSON(t, ctx, base+"/blog-records/"+blogID, http.StatusNotFound, userToken)
		getJSON(t, ctx, base+"/blogs/"+slug, http.StatusNotFound)
		adminView := getJSON(t, ctx, base+"/blog-records/"+blogID, http.StatusOK, adminToken)
		assert.NotNil(t, adminView.Data["deletedAt"])
		list := getJSON(t, ctx, base+"/blog-records?deleted=only", http.StatusOK, adminToken)
		items, _ := list.Data["items"].([]any)
		var found bool
		for _, raw := range items {
			item, _ := raw.(map[string]any)
			if item["id"] == blogID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("admin patch on deleted blog returns conflict", func(t *testing.T) {
		conflict := patchJSON(t, ctx, base+"/blog-records/"+blogID, `{"title":"Catatan Go","content":`+blogContentHTML(100)+`}`, http.StatusConflict, adminToken)
		assert.Equal(t, "BLOG_DELETED", conflict.Error["code"])
	})

	t.Run("task routes still require bearer and succeed for owner", func(t *testing.T) {
		postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusUnauthorized)
		task := postJSON(t, ctx, base+"/tasks", `{"title":"Prepare report"}`, http.StatusCreated, userToken)
		taskID, _ := task.Data["id"].(string)
		require.NotEmpty(t, taskID)
		getJSON(t, ctx, base+"/tasks/"+taskID, http.StatusOK, userToken)
	})

	getJSON(t, ctx, base+"/health", http.StatusOK)
}

func patchJSON(t *testing.T, ctx context.Context, url, body string, want int, bearer ...string) envelope {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if len(bearer) > 0 && bearer[0] != "" {
		req.Header.Set("Authorization", "Bearer "+bearer[0])
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, want, resp.StatusCode)
	return decodeEnvelope(t, resp.Body)
}

func deleteJSON(t *testing.T, ctx context.Context, url string, want int, bearer ...string) envelope {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	require.NoError(t, err)
	if len(bearer) > 0 && bearer[0] != "" {
		req.Header.Set("Authorization", "Bearer "+bearer[0])
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	assert.Equal(t, want, resp.StatusCode)
	return decodeEnvelope(t, resp.Body)
}
