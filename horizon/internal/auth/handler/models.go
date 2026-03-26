package handler

import "github.com/dgrijalva/jwt-go"

type User struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Email        string `json:"email"`
	Role         string `json:"role"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	AuthProvider string `json:"auth_provider,omitempty"`
	IsNewUser    bool   `json:"is_new_user,omitempty"`
	IsActive     bool   `json:"is_active,omitempty"`
}

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.StandardClaims
}

type UpdateUserAccessAndRole struct {
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
	Role     string `json:"role"`
}

type UserListingResponse struct {
	ID            uint   `json:"id,omitempty"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	Email         string `json:"email"`
	IsActive      bool   `json:"is_active"`
	Role          string `json:"role"`
	AuthProvider  string `json:"auth_provider,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	LastLoginAt   string `json:"last_login_at,omitempty"`
}

type SSOStatusResponse struct {
	SSOEnabled    bool     `json:"sso_enabled"`
	Providers     []string `json:"providers"`
	AllowPassword bool     `json:"allow_password"`
}

type RefreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}
