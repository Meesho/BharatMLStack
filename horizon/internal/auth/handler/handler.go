package handler

import (
	"os"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/auth"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	authOnce      sync.Once
	authenticator Authenticator
	JwtKey        = getJWTKey()
)

// getJWTKey retrieves JWT secret key from environment variable or uses default
func getJWTKey() []byte {
	key := os.Getenv("JWT_SECRET_KEY")
	if key == "" {
		// Try viper as fallback
		if viper.IsSet("JWT_SECRET_KEY") {
			key = viper.GetString("JWT_SECRET_KEY")
		}
	}
	if key == "" {
		// Default key for development only - should be set in production
		log.Warn().Msg("JWT_SECRET_KEY not set, using default key. This should be changed in production!")
		return []byte(constants.DefaultJWTSecret)
	}
	return []byte(key)
}

type Authenticator interface {
	Register(user *User) error
	Login(user *Login) (*LoginResponse, error)
	Logout(token string) error
	GetAllUsers() ([]UserListingResponse, error)
	UpdateUserAccessAndRole(email string, isActive bool, role string) error
	// SSO methods
	GetSSOStatus() (*SSOStatusResponse, error)
	InitiateGoogleOAuth() (string, string, error)
	LoginWithGoogle(code, state string) (*LoginResponse, error)
	// Token refresh
	RefreshToken(refreshToken string) (*RefreshTokenResponse, error)
	// User management
	GetUserByEmail(email string) (*auth.User, error)
	UpdateUserRole(id uint, role string, updatedBy uint) error
	UpdateUserStatus(id uint, isActive bool, updatedBy uint) error
}
