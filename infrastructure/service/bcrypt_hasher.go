package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/rfanazhari/go-boilerplate/domain/service"
	"github.com/rfanazhari/go-boilerplate/pkg/errs"
)

const BcryptCost = 12

type BcryptHasher struct{}

func (BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return string(hash), nil
}

func (BcryptHasher) Compare(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return err
		}
		return errs.New(errs.CodeInternalError, errs.KindInternal, err)
	}
	return nil
}

var _ service.PasswordHasher = BcryptHasher{}
