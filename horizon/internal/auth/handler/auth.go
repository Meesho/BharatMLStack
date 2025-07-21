package handler

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/auth"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/token"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	authRepo  auth.Repository
	tokenRepo token.Repository
}

func InitAuthHandler() Authenticator {
	if authenticator == nil {
		authOnce.Do(func() {
			connection, _ := infra.SQL.GetConnection()
			sqlConn := connection.(*infra.SQLConnection)
			authRepo, err := auth.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating auth repository")
			}
			tokenRepo, err := token.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating token repository")
			}
			authenticator = &AuthHandler{
				authRepo:  authRepo,
				tokenRepo: tokenRepo,
			}
		})
	}
	return authenticator
}

// validatePassword performs comprehensive password validation
func (a *AuthHandler) validatePassword(password string) error {
	var failedRules []string

	// Check minimum length (8 characters)
	if len(password) < 8 {
		failedRules = append(failedRules, "At least 8 characters")
	}

	// Check for uppercase letter
	if matched, _ := regexp.MatchString(`[A-Z]`, password); !matched {
		failedRules = append(failedRules, "One uppercase letter (A-Z)")
	}

	// Check for lowercase letter
	if matched, _ := regexp.MatchString(`[a-z]`, password); !matched {
		failedRules = append(failedRules, "One lowercase letter (a-z)")
	}

	// Check for number
	if matched, _ := regexp.MatchString(`\d`, password); !matched {
		failedRules = append(failedRules, "One number (0-9)")
	}

	// Check for special character
	if matched, _ := regexp.MatchString(`[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]`, password); !matched {
		failedRules = append(failedRules, "One special character (!@#$%^&*...)")
	}

	// Check for spaces
	if strings.Contains(password, " ") {
		failedRules = append(failedRules, "No spaces allowed")
	}

	// Check for common passwords
	commonPasswords := []string{"password", "123456", "qwerty", "abc123", "admin", "user"}
	for _, common := range commonPasswords {
		if strings.ToLower(password) == common {
			failedRules = append(failedRules, "Not a common password")
			break
		}
	}

	if len(failedRules) > 0 {
		return fmt.Errorf("password validation failed: %s", strings.Join(failedRules, ", "))
	}

	return nil
}

// Register handler
func (a *AuthHandler) Register(user *User) error {

	// Validate password before hashing
	if err := a.validatePassword(user.Password); err != nil {
		log.Error().Msgf("Password validation failed: %v", err)
		return err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Msgf("Failed to hash password: %v", err)
		return err
	}

	// Map User struct to auth.User
	authUser := auth.User{
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user", // By default onboard everyone with role user
	}

	// Create user in the repository
	_, err = a.authRepo.CreateUser(&authUser)
	if err != nil {
		log.Error().Msgf("Failed to register user: %v", err)
		return err
	}

	log.Info().Msgf("User %s registered successfully", user.Email)
	return nil
}

// Login method
func (a *AuthHandler) Login(user *Login) (*LoginResponse, error) {
	// Fetch user from the repository using email
	authUser, err := a.authRepo.GetUserByEmailId(user.Email)
	if err != nil {
		log.Error().Msgf("User not found with email: %s", user.Email)
		return nil, fmt.Errorf("invalid email or password")
	}

	// Compare the provided password with the stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(user.Password))
	if err != nil {
		log.Error().Msg("Password mismatch")
		return nil, fmt.Errorf("invalid email or password")
	}
	if !authUser.IsActive {
		log.Error().Msgf("User %s is not active, Please contact admin to activate your account", authUser.Email)
		return nil, fmt.Errorf("User is not active, Please contact admin to activate your account")
	}

	// Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: authUser.Email,
		Role:  authUser.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JwtKey)
	if err != nil {
		log.Error().Msgf("Failed to generate JWT token: %v", err)
		return nil, fmt.Errorf("failed to generate token")
	}
	saveTokenErr := a.saveToken(authUser.Email, tokenString, expirationTime)
	if saveTokenErr != nil {
		log.Error().Msgf("Failed to save token: %v", saveTokenErr)
		return nil, fmt.Errorf("failed to save token")
	}
	log.Info().Msgf("User %s logged in successfully", authUser.Email)
	return &LoginResponse{
		Email: authUser.Email,
		Role:  authUser.Role,
		Token: tokenString,
	}, nil
}

func (a *AuthHandler) Logout(token string) error {
	err := a.tokenRepo.InvalidateToken(token)
	if err != nil {
		log.Error().Msgf("Failed to invalidate token: %v", err)
		return err
	}
	return err
}

func (a *AuthHandler) saveToken(email, token string, expiration time.Time) error {
	err := a.tokenRepo.SaveToken(email, token, expiration)
	return err
}

func (a *AuthHandler) GetAllUsers() ([]UserListingResponse, error) {
	users, err := a.authRepo.GetAllUsers()
	if err != nil {
		log.Error().Msgf("Error Retrieving Users")
		return nil, err
	}
	userListingResponse := make([]UserListingResponse, len(users))
	for i, user := range users {
		userListingResponse[i] = UserListingResponse{
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
			IsActive:  user.IsActive,
			Role:      user.Role,
		}
	}
	return userListingResponse, nil
}

func (a *AuthHandler) UpdateUserAccessAndRole(email string, isActive bool, role string) error {
	err := a.authRepo.UpdateUserAccessAndRole(email, isActive, role)
	if err != nil {
		log.Error().Msgf("Error Toggling User Access")
		return err
	}
	return nil
}
