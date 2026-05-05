package handler

import (
	"sync"
)

var (
	authOnce      sync.Once
	authenticator Authenticator
	JwtKey        = []byte("horizon-admin-secret") // Replace with a secure secret key
)

type Authenticator interface {
	Register(user *User) error
	Login(user *Login) (*LoginResponse, error)
	Logout(token string) error
	GetAllUsers() ([]UserListingResponse, error)
	UpdateUserAccessAndRole(email string, isActive bool, role string) error
	GetPermissionByRole(role string) PermissionResponse
}
