package service_test

import (
	"github.com/google/uuid"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlogPublicVisible(t *testing.T) {
	type testCase struct {
		name         string
		exists       bool
		status       enum.BlogStatus
		deleted      bool
		ownerStatus  enum.AccountStatus
		ownerDeleted bool
		expected     bool
	}
	tests := []testCase{
		{name: "eligible", exists: true, status: enum.BlogPublished, ownerStatus: enum.AccountActive, expected: true},
		{name: "missing", status: enum.BlogPublished, ownerStatus: enum.AccountActive},
		{name: "draft", exists: true, status: enum.BlogDraft, ownerStatus: enum.AccountActive},
		{name: "deleted blog", exists: true, status: enum.BlogPublished, deleted: true, ownerStatus: enum.AccountActive},
		{name: "inactive owner", exists: true, status: enum.BlogPublished, ownerStatus: enum.AccountNonActive},
		{name: "deleted owner", exists: true, status: enum.BlogPublished, ownerStatus: enum.AccountActive, ownerDeleted: true},
		{name: "unknown status", exists: true, status: "unknown", ownerStatus: enum.AccountActive},
		{name: "unknown owner status", exists: true, status: enum.BlogPublished},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := service.BlogVisibility{Exists: tt.exists, Status: tt.status, Deleted: tt.deleted, OwnerStatus: tt.ownerStatus, OwnerDeleted: tt.ownerDeleted}
			assert.Equal(t, tt.expected, service.BlogPublicVisible(facts))
		})
	}
}

func TestAuthorizeBlogCreate(t *testing.T) {
	type testCase struct {
		name    string
		id      uuid.UUID
		role    enum.Role
		status  enum.AccountStatus
		deleted bool
		wantErr bool
		code    string
		kind    errs.Kind
	}
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	tests := []testCase{
		{name: "active user", id: id, role: enum.RoleUser, status: enum.AccountActive},
		{name: "admin cannot author", id: id, role: enum.RoleAdmin, status: enum.AccountActive, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "missing identity", role: enum.RoleUser, status: enum.AccountActive, wantErr: true, code: errs.CodeUnauthenticated, kind: errs.KindUnauthenticated},
		{name: "inactive admin before role gate", id: id, role: enum.RoleAdmin, status: enum.AccountNonActive, wantErr: true, code: errs.CodeUnauthenticated, kind: errs.KindUnauthenticated},
		{name: "deleted user", id: id, role: enum.RoleUser, status: enum.AccountActive, deleted: true, wantErr: true, code: errs.CodeUnauthenticated, kind: errs.KindUnauthenticated},
		{name: "unknown role", id: id, role: "unknown", status: enum.AccountActive, wantErr: true, code: errs.CodeUnauthenticated, kind: errs.KindUnauthenticated},
		{name: "unknown account status", id: id, role: enum.RoleUser, wantErr: true, code: errs.CodeUnauthenticated, kind: errs.KindUnauthenticated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := service.BlogActor{ID: tt.id, Role: tt.role, Status: tt.status, Deleted: tt.deleted}
			err := service.AuthorizeBlogCreate(actor)
			assertBlogDenial(t, err, tt.wantErr, tt.code, tt.kind)
		})
	}
}

func TestAuthorizeBlogAccess(t *testing.T) {
	type testCase struct {
		name          string
		role          enum.Role
		owner         bool
		exists        bool
		published     bool
		deleted       bool
		ownerInactive bool
		ownerDeleted  bool
		operation     service.BlogOperation
		wantErr       bool
		code          string
		kind          errs.Kind
	}
	tests := []testCase{
		{name: "owner draft private", role: enum.RoleUser, owner: true, exists: true, operation: service.BlogReadPrivate},
		{name: "admin draft private", role: enum.RoleAdmin, exists: true, operation: service.BlogReadPrivate},
		{name: "other cannot private read public blog", role: enum.RoleUser, exists: true, published: true, operation: service.BlogReadPrivate, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "owner edits draft", role: enum.RoleUser, owner: true, exists: true, operation: service.BlogEdit},
		{name: "owner publishes with inactive owner facts", role: enum.RoleUser, owner: true, exists: true, ownerInactive: true, operation: service.BlogPublish},
		{name: "admin historical owner publishes", role: enum.RoleAdmin, owner: true, exists: true, operation: service.BlogPublish},
		{name: "admin cannot edit draft", role: enum.RoleAdmin, exists: true, operation: service.BlogEdit, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "admin cannot publish", role: enum.RoleAdmin, exists: true, published: true, operation: service.BlogPublish, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "other hidden edit", role: enum.RoleUser, exists: true, operation: service.BlogEdit, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "other visible edit", role: enum.RoleUser, exists: true, published: true, operation: service.BlogEdit, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "other visible publish", role: enum.RoleUser, exists: true, published: true, operation: service.BlogPublish, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "other inactive owner", role: enum.RoleUser, exists: true, published: true, ownerInactive: true, operation: service.BlogDelete, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "other deleted owner", role: enum.RoleUser, exists: true, published: true, ownerDeleted: true, operation: service.BlogUnpublish, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "owner unpublish", role: enum.RoleUser, owner: true, exists: true, published: true, operation: service.BlogUnpublish},
		{name: "admin unpublish draft", role: enum.RoleAdmin, exists: true, operation: service.BlogUnpublish},
		{name: "owner delete", role: enum.RoleUser, owner: true, exists: true, operation: service.BlogDelete},
		{name: "admin delete", role: enum.RoleAdmin, exists: true, operation: service.BlogDelete},
		{name: "other visible unpublish", role: enum.RoleUser, exists: true, published: true, operation: service.BlogUnpublish, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "other visible delete", role: enum.RoleUser, exists: true, published: true, operation: service.BlogDelete, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "other hidden unpublish", role: enum.RoleUser, exists: true, operation: service.BlogUnpublish, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "other hidden delete", role: enum.RoleUser, exists: true, operation: service.BlogDelete, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "admin reads deleted on admin surface", role: enum.RoleAdmin, exists: true, deleted: true, operation: service.BlogReadAdmin},
		{name: "admin private hides deleted", role: enum.RoleAdmin, exists: true, deleted: true, operation: service.BlogReadPrivate, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "owner private hides deleted", role: enum.RoleUser, owner: true, exists: true, deleted: true, operation: service.BlogReadPrivate, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "owner repeat delete hidden", role: enum.RoleUser, owner: true, exists: true, deleted: true, operation: service.BlogDelete, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "owner deleted edit hidden", role: enum.RoleUser, owner: true, exists: true, deleted: true, operation: service.BlogEdit, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "other deleted hidden", role: enum.RoleUser, exists: true, deleted: true, operation: service.BlogPublish, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "admin deleted edit conflicts", role: enum.RoleAdmin, exists: true, deleted: true, operation: service.BlogEdit, wantErr: true, code: "BLOG_DELETED", kind: errs.KindConflict},
		{name: "admin deleted publish conflicts", role: enum.RoleAdmin, exists: true, deleted: true, operation: service.BlogPublish, wantErr: true, code: "BLOG_DELETED", kind: errs.KindConflict},
		{name: "admin deleted unpublish conflicts", role: enum.RoleAdmin, exists: true, deleted: true, operation: service.BlogUnpublish, wantErr: true, code: "BLOG_DELETED", kind: errs.KindConflict},
		{name: "admin repeat delete allowed", role: enum.RoleAdmin, exists: true, deleted: true, operation: service.BlogDelete},
		{name: "nonadmin route gate before missing", role: enum.RoleUser, operation: service.BlogReadAdmin, wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
		{name: "missing admin read", role: enum.RoleAdmin, operation: service.BlogReadAdmin, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "missing owner edit", role: enum.RoleUser, owner: true, operation: service.BlogEdit, wantErr: true, code: "BLOG_NOT_FOUND", kind: errs.KindNotFound},
		{name: "unknown operation fails closed", role: enum.RoleAdmin, owner: true, exists: true, operation: "unknown", wantErr: true, code: errs.CodeForbidden, kind: errs.KindForbidden},
	}
	ownerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	otherID := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := service.BlogActor{ID: otherID, Role: tt.role, Status: enum.AccountActive}
			if tt.owner {
				actor.ID = ownerID
			}
			blog := service.BlogVisibility{Exists: tt.exists, OwnerID: ownerID, Status: enum.BlogDraft, Deleted: tt.deleted, OwnerStatus: enum.AccountActive, OwnerDeleted: tt.ownerDeleted}
			if tt.published {
				blog.Status = enum.BlogPublished
			}
			if tt.ownerInactive {
				blog.OwnerStatus = enum.AccountNonActive
			}
			before := blog
			err := service.AuthorizeBlogAccess(actor, blog, tt.operation)
			assertBlogDenial(t, err, tt.wantErr, tt.code, tt.kind)
			assert.Equal(t, before, blog)
		})
	}
}

func TestAuthorizeBlogAccessRejectsIneligibleActorsFirst(t *testing.T) {
	type testCase struct {
		name    string
		id      uuid.UUID
		role    enum.Role
		status  enum.AccountStatus
		deleted bool
		wantErr bool
	}
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	tests := []testCase{
		{name: "anonymous", wantErr: true},
		{name: "inactive", id: id, role: enum.RoleAdmin, status: enum.AccountNonActive, wantErr: true},
		{name: "deleted", id: id, role: enum.RoleAdmin, status: enum.AccountActive, deleted: true, wantErr: true},
		{name: "unknown role", id: id, role: "unknown", status: enum.AccountActive, wantErr: true},
		{name: "unknown status", id: id, role: enum.RoleUser, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := service.BlogActor{ID: tt.id, Role: tt.role, Status: tt.status, Deleted: tt.deleted}
			for _, operation := range []service.BlogOperation{service.BlogReadPrivate, service.BlogReadAdmin, service.BlogEdit, service.BlogPublish, service.BlogUnpublish, service.BlogDelete} {
				err := service.AuthorizeBlogAccess(actor, service.BlogVisibility{}, operation)
				assertBlogDenial(t, err, tt.wantErr, errs.CodeUnauthenticated, errs.KindUnauthenticated)
			}
		})
	}
}

func assertBlogDenial(t *testing.T, err error, wantErr bool, code string, kind errs.Kind) {
	t.Helper()
	if !wantErr {
		require.NoError(t, err)
		return
	}
	require.Error(t, err)
	var publicErr *errs.Error
	require.ErrorAs(t, err, &publicErr)
	assert.Equal(t, code, publicErr.Code)
	assert.Equal(t, kind, publicErr.Kind)
	assert.Equal(t, code, publicErr.Error())
}
