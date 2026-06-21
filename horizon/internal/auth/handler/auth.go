package handler

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/auth"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/rolepermission"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/token"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// maxLoginFailures is the number of consecutive failed logins for a single
	// account before it is temporarily locked.
	maxLoginFailures = 5
	// loginLockDuration is how long an account stays locked after hitting the
	// failure threshold.
	loginLockDuration = 15 * time.Minute
	// loginFailureRetention bounds how long an idle failure-tracking entry is
	// kept. Entries not touched within this window are evicted so that an
	// attacker spraying many distinct (often non-existent) emails cannot grow
	// the in-memory map without bound.
	loginFailureRetention = loginLockDuration
)

// loginLockState tracks consecutive login failures for an account.
//
// NOTE: this is in-memory and therefore per-replica. For a multi-replica
// deployment move this counter to a shared store (Redis/DB) so the lockout is
// enforced globally. It is still a useful defence-in-depth layer alongside the
// per-IP rate limiter on the /login route.
//
// All fields are guarded by the package-level loginFailuresMu. They are read
// and written from concurrent request goroutines, so the mutex is required to
// avoid a data race.
type loginLockState struct {
	failures int
	until    time.Time
	// updatedAt is the last time this entry was modified; used for eviction.
	updatedAt time.Time
}

var (
	loginFailuresMu sync.Mutex
	loginFailures   = make(map[string]*loginLockState) // email -> state
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
	// Enforce per-account lockout before doing any expensive work.
	if locked, retryAfter := isAccountLocked(user.Email); locked {
		log.Warn().Msgf("Login blocked for %s: account temporarily locked", user.Email)
		// Round up to whole seconds so a sub-second remainder is never reported
		// as "0s" (which would be a confusing instruction to the user).
		wait := retryAfter.Round(time.Second)
		if wait < time.Second {
			wait = time.Second
		}
		return nil, fmt.Errorf("account temporarily locked due to too many failed attempts, try again in %s", wait)
	}

	// Fetch user from the repository using email
	authUser, err := a.authRepo.GetUserByEmailId(user.Email)
	if err != nil {
		// Only count *authentication* failures (no such user) toward the
		// lockout. A transient backend error (DB down, timeout, etc.) is not an
		// attacker's failed guess, so locking on it would let infra blips lock
		// out legitimate users and would surface as a confusing error.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			recordLoginFailure(user.Email)
			log.Error().Msgf("User not found with email: %s", user.Email)
			return nil, fmt.Errorf("invalid email or password")
		}
		log.Error().Err(err).Msgf("Error looking up user %s", user.Email)
		return nil, fmt.Errorf("login temporarily unavailable, please try again")
	}

	// Compare the provided password with the stored password hash
	err = bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(user.Password))
	if err != nil {
		recordLoginFailure(user.Email)
		log.Error().Msg("Password mismatch")
		return nil, fmt.Errorf("invalid email or password")
	}
	if !authUser.IsActive {
		log.Error().Msgf("User %s is not active, Please contact admin to activate your account", authUser.Email)
		return nil, fmt.Errorf("User is not active, Please contact admin to activate your account")
	}

	// Successful authentication resets the failure counter.
	resetLoginFailures(user.Email)

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

// isAccountLocked reports whether the account is currently locked out and, if
// so, how long until it is unlocked.
func isAccountLocked(email string) (bool, time.Duration) {
	loginFailuresMu.Lock()
	defer loginFailuresMu.Unlock()

	st, ok := loginFailures[email]
	if !ok {
		return false, 0
	}
	if st.until.IsZero() || time.Now().After(st.until) {
		return false, 0
	}
	return true, time.Until(st.until)
}

// recordLoginFailure increments the failure counter for the account and locks
// it once the threshold is reached.
//
// Unknown/non-existent emails are recorded exactly like real ones. This is
// deliberate: the lockout check short-circuits before the user lookup, so a
// known and an unknown email that cross the threshold produce the identical
// "locked" response. Recording both keeps that behaviour symmetric and avoids
// turning the lockout into an account-existence oracle.
func recordLoginFailure(email string) {
	loginFailuresMu.Lock()
	defer loginFailuresMu.Unlock()

	evictExpiredLocked()

	st, ok := loginFailures[email]
	if !ok {
		st = &loginLockState{}
		loginFailures[email] = st
	}
	st.failures++
	st.updatedAt = time.Now()
	if st.failures >= maxLoginFailures {
		st.until = time.Now().Add(loginLockDuration)
		st.failures = 0
	}
}

// resetLoginFailures clears any failure/lock state for the account.
func resetLoginFailures(email string) {
	loginFailuresMu.Lock()
	defer loginFailuresMu.Unlock()
	delete(loginFailures, email)
}

// evictExpiredLocked removes stale entries to bound memory growth from an
// attacker spraying many distinct emails. An entry is safe to drop once it is
// not actively locked and has not been touched within the retention window.
// Caller must hold loginFailuresMu.
func evictExpiredLocked() {
	now := time.Now()
	for email, st := range loginFailures {
		locked := !st.until.IsZero() && now.Before(st.until)
		if locked {
			continue
		}
		if now.Sub(st.updatedAt) >= loginFailureRetention {
			delete(loginFailures, email)
		}
	}
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
