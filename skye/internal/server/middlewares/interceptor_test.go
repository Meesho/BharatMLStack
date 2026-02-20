package middlewares

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAuthorized_ValidToken(t *testing.T) {
	// Set package-level authTokens for testing
	authTokens = "token1,token2,token3"

	assert.True(t, isAuthorized([]string{"token1"}))
	assert.True(t, isAuthorized([]string{"token2"}))
	assert.True(t, isAuthorized([]string{"token3"}))
}

func TestIsAuthorized_InvalidToken(t *testing.T) {
	authTokens = "token1,token2"

	assert.False(t, isAuthorized([]string{"bad_token"}))
	assert.False(t, isAuthorized([]string{""}))
}

func TestIsAuthorized_SingleToken(t *testing.T) {
	authTokens = "single_token"

	assert.True(t, isAuthorized([]string{"single_token"}))
	assert.False(t, isAuthorized([]string{"other"}))
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "skye-caller-id", CallerIdHeader)
	assert.Equal(t, "skye-auth-token", AuthTokenHeader)
}
