package service

import (
	"github.com/google/uuid"
	"github.com/rfanazhari/go-boilerplate/domain/enum"
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

func BlogPublicVisible(blog BlogVisibility) bool { return false }
func AuthorizeBlogCreate(actor BlogActor) error  { return nil }
func AuthorizeBlogAccess(actor BlogActor, blog BlogVisibility, operation BlogOperation) error {
	return nil
}
