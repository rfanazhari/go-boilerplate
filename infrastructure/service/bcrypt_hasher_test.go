package service_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	infraservice "github.com/rfanazhari/go-boilerplate/infrastructure/service"
)

func TestBcryptHasherHashAndCompare(t *testing.T) {
	hasher := infraservice.BcryptHasher{}
	hash, err := hasher.Hash("secret-password")
	require.NoError(t, err)
	require.NotEqual(t, "secret-password", hash)
	require.NoError(t, hasher.Compare(hash, "secret-password"))
	require.Error(t, hasher.Compare(hash, "wrong-password"))
}
