package service

import (
	"github.com/google/uuid"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

// BlogActor contains current account facts, supplied after account lookup.
type BlogActor struct {
	ID      uuid.UUID
	Role    enum.Role
	Status  enum.AccountStatus
	Deleted bool
}

// BlogVisibility contains resource and current owner facts; it performs no I/O.
type BlogVisibility struct {
	Exists       bool
	OwnerID      uuid.UUID
	Status       enum.BlogStatus
	Deleted      bool
	OwnerStatus  enum.AccountStatus
	OwnerDeleted bool
}

type BlogOperation string

const (
	BlogReadPrivate BlogOperation = "read_private"
	BlogReadAdmin   BlogOperation = "read_admin"
	BlogEdit        BlogOperation = "edit"
	BlogPublish     BlogOperation = "publish"
	BlogUnpublish   BlogOperation = "unpublish"
	BlogDelete      BlogOperation = "delete"
)

func authenticateBlogActor(actor BlogActor) error {
	if actor.ID == uuid.Nil || actor.Status != enum.AccountActive || actor.Deleted {
		return errs.New(errs.CodeUnauthenticated, errs.KindUnauthenticated, nil)
	}
	if _, ok := enum.ParseRole(string(actor.Role)); !ok {
		return errs.New(errs.CodeUnauthenticated, errs.KindUnauthenticated, nil)
	}
	return nil
}

// BlogPublicVisible expresses the shared read-time blog/owner eligibility rule.
func BlogPublicVisible(blog BlogVisibility) bool {
	return blog.Exists && blog.Status == enum.BlogPublished && !blog.Deleted &&
		blog.OwnerStatus == enum.AccountActive && !blog.OwnerDeleted
}

// AuthorizeBlogCreate permits only an eligible user to author a new blog.
func AuthorizeBlogCreate(actor BlogActor) error {
	if err := authenticateBlogActor(actor); err != nil {
		return err
	}
	if actor.Role != enum.RoleUser {
		return errs.New(errs.CodeForbidden, errs.KindForbidden, nil)
	}
	return nil
}

// AuthorizeBlogAccess checks disclosure before operation permission or state.
// Call this before validation and before treating a mutation as an idempotent no-op.
// Facts must come from current account data and the loaded (locked for writes) blog.
func AuthorizeBlogAccess(actor BlogActor, blog BlogVisibility, operation BlogOperation) error {
	if err := authenticateBlogActor(actor); err != nil {
		return err
	}
	admin := actor.Role == enum.RoleAdmin
	if operation == BlogReadAdmin && !admin {
		return errs.New(errs.CodeForbidden, errs.KindForbidden, nil)
	}
	if !blog.Exists {
		return errs.New(errs.CodeBlogNotFound, errs.KindNotFound, nil)
	}
	if blog.Deleted {
		if !admin || operation == BlogReadPrivate {
			return errs.New(errs.CodeBlogNotFound, errs.KindNotFound, nil)
		}
		switch operation {
		case BlogReadAdmin, BlogDelete:
			return nil
		case BlogEdit, BlogPublish, BlogUnpublish:
			return errs.New(errs.CodeBlogDeleted, errs.KindConflict, nil)
		default:
			return errs.New(errs.CodeForbidden, errs.KindForbidden, nil)
		}
	}
	owner := actor.ID == blog.OwnerID
	if operation == BlogReadPrivate {
		if owner || admin {
			return nil
		}
		return errs.New(errs.CodeBlogNotFound, errs.KindNotFound, nil)
	}
	if !owner && !admin && !BlogPublicVisible(blog) {
		return errs.New(errs.CodeBlogNotFound, errs.KindNotFound, nil)
	}
	switch operation {
	case BlogReadAdmin:
		return nil // The route-wide admin gate has already passed.
	case BlogEdit, BlogPublish:
		if owner {
			return nil
		}
	case BlogUnpublish, BlogDelete:
		if owner || admin {
			return nil
		}
	}
	return errs.New(errs.CodeForbidden, errs.KindForbidden, nil)
}
