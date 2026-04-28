package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/auth"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/rolepermission"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/token"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	authRepo       auth.Repository
	tokenRepo      token.Repository
	rolePermission rolepermission.Repository
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
			rolePermission, err := rolepermission.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating role permission repository")
			}
			authenticator = &AuthHandler{
				authRepo:       authRepo,
				tokenRepo:      tokenRepo,
				rolePermission: rolePermission,
			}
		})
	}
	return authenticator
}

// validatePassword performs comprehensive password validation
func (a *AuthHandler) validatePassword(password string) error {
	var failedRules []string

	// Check minimum length
	if len(password) < constants.MinPasswordLength {
		failedRules = append(failedRules, fmt.Sprintf("At least %d characters", constants.MinPasswordLength))
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
	for _, common := range constants.CommonPasswords {
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
	// Check if password registration is allowed
	cfg := config.GetOAuthConfig()
	if cfg.SSOProvider != constants.AuthProviderPassword {
		return fmt.Errorf("password registration is not enabled. This system uses Google SSO only. Please use Google to create an account")
	}

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
		Role:         constants.DefaultUserRole, // By default onboard everyone with role user
		IsActive:     constants.DefaultIsActive, // New users are active by default
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
	// Check if password authentication is allowed
	cfg := config.GetOAuthConfig()
	if cfg.SSOProvider != constants.AuthProviderPassword {
		return nil, fmt.Errorf("password authentication is not enabled. This system uses Google SSO only")
	}

	// Fetch user from the repository using email
	authUser, err := a.authRepo.GetUserByEmailId(user.Email)
	if err != nil {
		log.Error().Msgf("User not found with email: %s", user.Email)
		return nil, fmt.Errorf(constants.ErrInvalidCredentials)
	}

	// Check if user has password authentication
	if authUser.PasswordHash == "" {
		return nil, fmt.Errorf(constants.ErrPasswordAuthNotAvailable)
	}

	// Compare the provided password with the stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(user.Password))
	if err != nil {
		log.Error().Msg("Password mismatch")
		return nil, fmt.Errorf(constants.ErrInvalidCredentials)
	}
	if !authUser.IsActive {
		log.Error().Msgf("User %s is not active, Please contact admin to activate your account", authUser.Email)
		return nil, fmt.Errorf(constants.ErrUserNotActive)
	}

	// Generate tokens
	cfg = config.GetOAuthConfig()
	accessTokenExpiry := time.Duration(cfg.AccessTokenExpiry) * time.Hour
	refreshTokenExpiry := time.Duration(cfg.RefreshTokenExpiry) * 24 * time.Hour

	accessToken, refreshToken, err := a.generateTokens(authUser.Email, authUser.Role, accessTokenExpiry, refreshTokenExpiry)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("User %s logged in successfully", authUser.Email)
	return &LoginResponse{
		Email:        authUser.Email,
		Role:         authUser.Role,
		Token:        accessToken,
		RefreshToken: refreshToken,
		AuthProvider: constants.DefaultAuthProvider,
		IsActive:     authUser.IsActive,
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

// generateTokens generates both access and refresh tokens
func (a *AuthHandler) generateTokens(email, role string, accessExpiry, refreshExpiry time.Duration) (string, string, error) {
	// Generate access token
	accessExpirationTime := time.Now().Add(accessExpiry)
	accessClaims := &Claims{
		Email: email,
		Role:  role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: accessExpirationTime.Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(JwtKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (simple random string, can be enhanced)
	refreshTokenBytes := make([]byte, constants.RefreshTokenSize)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenString := base64.URLEncoding.EncodeToString(refreshTokenBytes)

	// Save access token
	if err := a.saveToken(email, accessTokenString, accessExpirationTime); err != nil {
		return "", "", fmt.Errorf("failed to save access token: %w", err)
	}

	// Save refresh token
	refreshExpirationTime := time.Now().Add(refreshExpiry)
	if err := a.tokenRepo.SaveRefreshToken(email, refreshTokenString, refreshExpirationTime); err != nil {
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return accessTokenString, refreshTokenString, nil
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
			ID:           user.ID,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			Email:        user.Email,
			IsActive:     user.IsActive,
			Role:         user.Role,
			AuthProvider: constants.DefaultAuthProvider,
		}
		if !user.CreatedAt.IsZero() {
			userListingResponse[i].CreatedAt = user.CreatedAt.Format(time.RFC3339)
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

// GetSSOStatus returns SSO configuration status
func (a *AuthHandler) GetSSOStatus() (*SSOStatusResponse, error) {
	cfg := config.GetOAuthConfig()

	providers := []string{}
	if cfg.SSOEnabled && cfg.GoogleClientID != "" && cfg.SSOProvider == constants.AuthProviderGoogle {
		providers = append(providers, constants.AuthProviderGoogle)
	}

	allowPassword := cfg.SSOProvider == constants.AuthProviderPassword

	return &SSOStatusResponse{
		SSOEnabled:    cfg.SSOEnabled && len(providers) > 0,
		Providers:     providers,
		AllowPassword: allowPassword,
	}, nil
}

// InitiateGoogleOAuth initiates Google OAuth flow
func (a *AuthHandler) InitiateGoogleOAuth() (string, string, error) {
	return InitiateGoogleOAuth()
}

// LoginWithGoogle handles Google OAuth callback and logs in/creates user
func (a *AuthHandler) LoginWithGoogle(code, state string) (*LoginResponse, error) {
	// Check if Google authentication is allowed
	cfg := config.GetOAuthConfig()
	if cfg.SSOProvider != constants.AuthProviderGoogle {
		return nil, fmt.Errorf("google SSO is not enabled. This system uses password authentication only")
	}

	// Validate CSRF state
	if !ValidateCSRFState(state) {
		return nil, fmt.Errorf(constants.ErrInvalidCSRFState)
	}

	// Exchange code for token
	tokenResp, err := ExchangeGoogleCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info from Google
	userInfo, err := GetGoogleUserInfo(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Check if user exists by email
	var authUser *auth.User
	var isNewUser bool
	existingUser, err := a.authRepo.GetUserByEmailId(userInfo.Email)
	if err == nil {
		// User exists - allow login
		authUser = existingUser
		isNewUser = false
	} else {
		// Create new user (Google SSO)
		names := strings.SplitN(userInfo.Name, " ", 2)
		firstName := names[0]
		lastName := ""
		if len(names) > 1 {
			lastName = names[1]
		}

		// Create user without password (empty password hash indicates SSO-only user)
		newUser := auth.User{
			FirstName:    firstName,
			LastName:     lastName,
			Email:        userInfo.Email,
			PasswordHash: "", // Empty password hash for SSO-only users
			Role:         constants.DefaultUserRole,
			IsActive:     constants.DefaultIsActive,
		}

		userID, err := a.authRepo.CreateUser(&newUser)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		authUser, err = a.authRepo.GetUserByID(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve created user: %w", err)
		}
		isNewUser = true
	}

	if !authUser.IsActive {
		return nil, fmt.Errorf("user account is not active")
	}

	// Generate tokens
	cfg = config.GetOAuthConfig()
	accessTokenExpiry := time.Duration(cfg.AccessTokenExpiry) * time.Hour
	refreshTokenExpiry := time.Duration(cfg.RefreshTokenExpiry) * 24 * time.Hour

	accessToken, refreshToken, err := a.generateTokens(authUser.Email, authUser.Role, accessTokenExpiry, refreshTokenExpiry)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Email:        authUser.Email,
		Role:         authUser.Role,
		Token:        accessToken,
		RefreshToken: refreshToken,
		AuthProvider: constants.AuthProviderGoogle,
		IsNewUser:    isNewUser,
		IsActive:     authUser.IsActive,
	}, nil
}

// RefreshToken refreshes access token using refresh token
func (a *AuthHandler) RefreshToken(refreshToken string) (*RefreshTokenResponse, error) {
	// Get refresh token from database
	tokenRecord, err := a.tokenRepo.GetRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf(constants.ErrInvalidRefreshToken)
	}

	// Get user by email
	authUser, err := a.authRepo.GetUserByEmailId(tokenRecord.UserEmail)
	if err != nil {
		return nil, fmt.Errorf(constants.ErrUserNotFound)
	}

	if !authUser.IsActive {
		return nil, fmt.Errorf(constants.ErrUserNotActive)
	}

	// Invalidate old refresh token
	_ = a.tokenRepo.InvalidateRefreshToken(refreshToken)

	// Generate new tokens
	cfg := config.GetOAuthConfig()
	accessTokenExpiry := time.Duration(cfg.AccessTokenExpiry) * time.Hour
	refreshTokenExpiry := time.Duration(cfg.RefreshTokenExpiry) * 24 * time.Hour

	accessToken, newRefreshToken, err := a.generateTokens(authUser.Email, authUser.Role, accessTokenExpiry, refreshTokenExpiry)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// UpdateUserRole updates a user's role (super_admin only)
func (a *AuthHandler) UpdateUserRole(id uint, role string, updatedBy uint) error {
	// Validate role
	validRole := false
	for _, valid := range constants.ValidRoles {
		if role == valid {
			validRole = true
			break
		}
	}
	if !validRole {
		return fmt.Errorf("%s: %s", constants.ErrInvalidRole, role)
	}

	// Get user to check current role
	user, err := a.authRepo.GetUserByID(id)
	if err != nil {
		return fmt.Errorf("%s: %w", constants.ErrUserNotFound, err)
	}

	// Check if trying to change super_admin role
	if user.Role == constants.RoleSuperAdmin && role != constants.RoleSuperAdmin {
		// Check if this is the last super_admin
		allUsers, err := a.authRepo.GetAllUsers()
		if err != nil {
			return fmt.Errorf("failed to check super_admin count: %w", err)
		}

		superAdminCount := 0
		for _, u := range allUsers {
			if u.Role == constants.RoleSuperAdmin {
				superAdminCount++
			}
		}

		if superAdminCount <= 1 {
			return fmt.Errorf(constants.ErrCannotDemoteLastSuperAdmin)
		}
	}

	return a.authRepo.UpdateUserRole(id, role, updatedBy)
}

// GetUserByEmail is a helper method to get user by email (used by controller)
func (a *AuthHandler) GetUserByEmail(email string) (*auth.User, error) {
	return a.authRepo.GetUserByEmailId(email)
}

// UpdateUserStatus updates a user's active status (admin/super_admin)
func (a *AuthHandler) UpdateUserStatus(id uint, isActive bool, updatedBy uint) error {
	// Get user to check if trying to deactivate self
	user, err := a.authRepo.GetUserByID(id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get updater user
	updater, err := a.authRepo.GetUserByID(updatedBy)
	if err != nil {
		return fmt.Errorf("updater not found: %w", err)
	}

	// Prevent self-deactivation
	if user.ID == updater.ID && !isActive {
		return fmt.Errorf(constants.ErrCannotDeactivateSelf)
	}

	return a.authRepo.UpdateUserStatus(id, isActive, updatedBy)
}

func (a *AuthHandler) GetPermissionByRole(role string) PermissionResponse {
	permissions, err := a.rolePermission.GetPermissionsByRole(role)
	if err != nil {
		log.Warn().Msgf("Error fetching permissions for role %s: %v", role, err)
		return PermissionResponse{
			Role:        role,
			Permissions: []ServiceSet{},
		}
	}

	serviceMap := make(map[string]map[string][]string)

	for _, perm := range permissions {
		if _, ok := serviceMap[perm.Service]; !ok {
			serviceMap[perm.Service] = make(map[string][]string)
		}
		serviceMap[perm.Service][perm.ScreenType] = append(serviceMap[perm.Service][perm.ScreenType], perm.Module)
	}

	var serviceSets []ServiceSet
	for service, screenMap := range serviceMap {
		var screens []ScreenInfo
		for screenType, actions := range screenMap {
			screens = append(screens, ScreenInfo{
				ScreenType:     screenType,
				AllowedActions: unique(actions),
			})
		}
		serviceSets = append(serviceSets, ServiceSet{
			Service: service,
			Screens: screens,
		})
	}

	return PermissionResponse{
		Role:        role,
		Permissions: serviceSets,
	}
}

func unique(input []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, v := range input {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}
