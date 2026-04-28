package constants

// User roles
const (
	RoleUser       = "user"
	RoleAdmin      = "admin"
	RoleSuperAdmin = "super_admin"
)

// Auth providers
const (
	AuthProviderPassword = "password"
	AuthProviderGoogle   = "google"
)

// Password validation constants
const (
	MinPasswordLength    = 8
	PasswordMinUppercase = 1
	PasswordMinLowercase = 1
	PasswordMinNumbers   = 1
	PasswordMinSpecial   = 1
)

// Common passwords that should be rejected
var CommonPasswords = []string{
	"password", "123456", "qwerty", "abc123", "admin", "user",
	"password123", "12345678", "welcome", "monkey", "1234567890",
}

// Bcrypt cost (should be configurable, but DefaultCost is acceptable)
// bcrypt.DefaultCost = 10, which is industry standard
const BcryptCost = 10 // This matches bcrypt.DefaultCost

// JWT configuration
const (
	// JWT signing method
	JWTSigningMethod = "HS256"

	// Default JWT secret (should NEVER be used in production)
	// This is only for development - production MUST set JWT_SECRET_KEY env var
	DefaultJWTSecret = "horizon-admin-secret"
)

// Token generation constants
const (
	RefreshTokenSize = 32 // bytes for refresh token generation
)

// CSRF state management
const (
	CSRFStateExpiryMinutes = 10 // CSRF state token expiry in minutes
	CSRFStateSize          = 32 // bytes for CSRF state generation
)

// OAuth configuration
const (
	// Google OAuth endpoints (these are standard and unlikely to change)
	GoogleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL    = "https://oauth2.googleapis.com/token"
	GoogleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

	// OAuth scopes
	GoogleOAuthScopes = "openid email profile"

	// OAuth parameters
	OAuthResponseType = "code"
	OAuthAccessType   = "offline"
	OAuthPrompt       = "consent"
)

// HTTP client configuration
const (
	GoogleAPITimeoutSeconds = 10 // Timeout for Google API calls
)

// CORS configuration (should be configurable via env vars)
const (
	// Default CORS settings (should be overridden in production)
	CORSAllowAllOrigins = "*" // WARNING: Should be restricted in production

	// Allowed HTTP methods
	CORSAllowedMethods = "GET,POST,PUT,DELETE,OPTIONS,PATCH"

	// Allowed headers
	CORSAllowedHeaders = "Origin,Content-Length,Content-Type,Authorization"
)

// Default user configuration
const (
	DefaultUserRole     = RoleUser
	DefaultIsActive     = true
	DefaultAuthProvider = AuthProviderPassword
)

// Error messages (standardized)
const (
	ErrInvalidCredentials         = "invalid email or password"
	ErrUserNotFound               = "user not found"
	ErrUserNotActive              = "user is not active, Please contact admin to activate your account"
	ErrPasswordAuthNotAvailable   = "password authentication not available for this account"
	ErrInvalidRole                = "invalid role"
	ErrCannotDemoteLastSuperAdmin = "cannot demote the last super_admin"
	ErrCannotDeactivateSelf       = "cannot deactivate yourself"
	ErrInvalidRefreshToken        = "invalid or expired refresh token"
	ErrInvalidCSRFState           = "invalid or expired CSRF state"
	ErrSSONotEnabled              = "SSO is not enabled"
	ErrOAuthConfigIncomplete      = "OAuth configuration is incomplete"
	ErrOnlySuperAdmin             = "only super_admin can access this resource"
	ErrOnlyAdminOrSuperAdmin      = "only admin or super_admin can access this resource"
	ErrPermissionDenied           = "Permission Denied"
	ErrRoleParameterRequired      = "role parameter is required"
	ErrPermissionIDRequired       = "permission id is required"
	ErrInvalidPermissionID        = "invalid permission id"
	ErrRoleNotFoundInToken        = "role not found in token"
)

// Success messages
const (
	MsgRegistrationSuccessful = "Registration successful. Your account is active."
	MsgLoginSuccessful        = "User logged in successfully"
	MsgLogoutSuccessful       = "User Logged out successfully"
	MsgUserUpdated            = "User info updated successfully"
	MsgRoleUpdated            = "User role updated successfully"
	MsgStatusUpdated          = "User status updated successfully"
	MsgPermissionDeleted      = "Permission deleted successfully"
	MsgPermissionsUpdated     = "Permissions updated successfully"
)

// Route paths (for middleware bypass)
var PublicRoutes = []string{
	"/login",
	"/register",
	"/health",
	"/auth/sso/status",
	"/auth/google/initiate",
	"/auth/google/callback",
	"/auth/refresh",
	"/api/1.0/fs-config",
	"/api/v1/online-feature-store/get-source-mapping",
	"/api/v1/online-feature-store/get-online-features-mapping",
	"/api/v1/online-feature-store/retrieve-feature-groups",
}

// Valid roles for validation
var ValidRoles = []string{
	RoleUser,
	RoleAdmin,
	RoleSuperAdmin,
}

// Valid auth providers
var ValidAuthProviders = []string{
	AuthProviderPassword,
	AuthProviderGoogle,
}
