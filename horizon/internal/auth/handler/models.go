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
	Email string `json:"email"`
	Role  string `json:"role"`
	Token string `json:"token"`
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
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	Role      string `json:"role"`
}
