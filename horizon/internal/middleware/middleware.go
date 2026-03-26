package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"github.com/Meesho/BharatMLStack/horizon/internal/middleware/resolver"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/apiresolver"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/metadata"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/permissions"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/rolepermission"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/token"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var (
	middlewareOnce sync.Once
	middleware     Middleware
)

type Middleware interface {
	GetMiddleWares() []gin.HandlerFunc
}

type MiddlewareHandler struct {
	tokenRepo          token.Repository
	apiResolverRepo    apiresolver.Repository
	rolePermissionRepo rolepermission.Repository // Keep for backward compatibility
	permissionRepo     permissions.Repository    // New permissions repository
	metadataRepo       metadata.MetadataRepository // Metadata repository for lookups
	mwhandler          *resolver.Handler
}

func NewMiddleware() Middleware {
	middlewareOnce.Do(func() {
		connection, _ := infra.SQL.GetConnection()
		sqlConn := connection.(*infra.SQLConnection)
		tokenRepo, err := token.NewRepository(sqlConn)
		if err != nil {
			log.Error().Msgf("Error in creating token repository: %v", err)
		}

		mwhandler, err := resolver.NewHandler()
		if err != nil {
			log.Error().Msgf("Error in creating middleware resolver handler: %v", err)
		}

		apiResolverRepo, err := apiresolver.NewRepository(sqlConn)
		if err != nil {
			log.Error().Msgf("Error in creating api resolver repository: %v", err)
		}

		rolePermissionRepo, err := rolepermission.NewRepository(sqlConn)
		if err != nil {
			log.Error().Msgf("Error in creating role permission repository: %v", err)
		}
<<<<<<< HEAD:horizon/internal/middlewares/middleware.go
		permissionRepo, err := permissions.NewRepository(sqlConn)
		if err != nil {
			log.Error().Msgf("Error in creating permission repository")
		}
		metadataRepo, err := metadata.NewRepository(sqlConn)
		if err != nil {
			log.Error().Msgf("Error in creating metadata repository")
		}
=======

>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/middleware.go
		middleware = &MiddlewareHandler{
			tokenRepo:          tokenRepo,
			apiResolverRepo:    apiResolverRepo,
			rolePermissionRepo: rolePermissionRepo,
			permissionRepo:     permissionRepo,
			metadataRepo:       metadataRepo,
			mwhandler:          mwhandler,
		}
	})
	return middleware
}

func (m *MiddlewareHandler) GetMiddleWares() []gin.HandlerFunc {
	var middlewares []gin.HandlerFunc
	middlewares = append(middlewares, m.Cors()...)
	middlewares = append(middlewares, m.AuthMiddleware())

	return middlewares
}

func (m *MiddlewareHandler) Cors() []gin.HandlerFunc {
	var middlewares []gin.HandlerFunc
	corsConfig := cors.DefaultConfig()
	// WARNING: CORS allowing all origins is a security risk in production
	// Should be configured via environment variable
	corsConfig.AllowOrigins = []string{constants.CORSAllowAllOrigins} // TODO: Make configurable via env var
	corsConfig.AllowMethods = strings.Split(constants.CORSAllowedMethods, ",")
	corsConfig.AllowHeaders = strings.Split(constants.CORSAllowedHeaders, ",")
	corsConfig.AllowCredentials = true

	middlewares = append(middlewares, cors.New(corsConfig))
	return middlewares
}

// AuthMiddleware checks for a valid JWT token except on login and register routes
func (m *MiddlewareHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
<<<<<<< HEAD:horizon/internal/middlewares/middleware.go
		// Bypass authentication for public routes
		isPublicRoute := false
		for _, publicRoute := range constants.PublicRoutes {
			if strings.HasPrefix(c.Request.URL.Path, publicRoute) {
				isPublicRoute = true
				break
			}
		}
		if isPublicRoute {
=======
		// Bypass authentication for login, register, and specific routes
		if strings.HasPrefix(c.Request.URL.Path, "/login") ||
			strings.HasPrefix(c.Request.URL.Path, "/register") ||
			strings.HasPrefix(c.Request.URL.Path, "/health") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/1.0/fs-config") {
>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/middleware.go
			c.Next()
			return
		}

		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Error().
				Str("reason", "Authorization header required").
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("unauthorized request blocked by auth middleware")
			c.JSON(http.StatusUnauthorized, gin.H{constant.Error: "Authorization header required"})
			c.Abort()
			return
		}

		// Check if the header is in the correct format (e.g., "Bearer <token>")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			log.Error().
				Str("reason", "Authorization token must be Bearer <token>").
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("unauthorized request blocked by auth middleware")
			c.JSON(http.StatusUnauthorized, gin.H{constant.Error: "Authorization token must be Bearer <token>"})
			c.Abort()
			return
		}

		valid, err := m.tokenRepo.IsTokenValid(tokenString)
		if err != nil || !valid {
			log.Error().
				Str("reason", "Invalid token").
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("unauthorized request blocked by auth middleware")
			c.JSON(http.StatusUnauthorized, gin.H{constant.Error: "Invalid token"})
			c.Abort()
			return
		}

		// Parse and validate the JWT token
		claims := &handler.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return handler.JwtKey, nil
		})
		if err != nil || !token.Valid {
			log.Error().
				Str("reason", "Invalid or expired token").
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("unauthorized request blocked by auth middleware")
			c.JSON(http.StatusUnauthorized, gin.H{constant.Error: "Invalid or expired token"})
			c.Abort()
			return
		}

		// m.CheckScreenPermission(c, claims)

		// Set claims in the context for later use
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func (m *MiddlewareHandler) CheckScreenPermission(c *gin.Context, claims *handler.Claims) {
	method := c.Request.Method
	path := c.FullPath()

	// If FullPath is empty (static routes), use the actual request path
	if path == "" {
		path = c.Request.URL.Path
	}

	if strings.HasPrefix(path, "/api/v1/online-feature-store") {
		return
	}

	if path == "/logout" ||
		path == "/users" ||
		path == "/update-user" ||
		path == "/api/v1/horizon/permission-by-role" {
		return
	}

	apiResolver, err := m.apiResolverRepo.GetResolver(method, path)

	if err != nil || apiResolver.ResolverFn == "" {
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{constant.Error: "Unable to resolve API"})
		}
		c.Abort()
		return
	}
	bodyBytes, ok := cloneRequestBody(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{constant.Error: "Invalid request body"})
		return
	}
	var bodyMap map[string]interface{}
	if len(bodyBytes) != 0 {
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{constant.Error: "Malformed JSON"})
			return
		}
	}
	c.Set("requestBody", bodyMap)

	resolver, exists := m.mwhandler.ResolverRegistry[apiResolver.ResolverFn]
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{constant.Error: "Resolver function not found"})
		c.Abort()
		return
	}
	screenModule := resolver(c)
	
	// Super admin bypass - has all permissions
	if claims.Role == constants.RoleSuperAdmin {
		return
	}
	
	// Look up service_id from service name
	service, err := m.metadataRepo.GetServiceByName(screenModule.Service)
	if err != nil {
<<<<<<< HEAD:horizon/internal/middlewares/middleware.go
		log.Warn().Err(err).Str("service", screenModule.Service).Msg("Service not found in metadata")
		c.JSON(http.StatusForbidden, gin.H{"error": constants.ErrPermissionDenied})
		c.Abort()
		return
	}
	
	// Look up screen_type_id from screen type name and service_id
	screenType, err := m.metadataRepo.GetScreenTypeByServiceAndName(service.ID, screenModule.ScreenType)
	if err != nil {
		log.Warn().Err(err).Str("screenType", screenModule.ScreenType).Msg("Screen type not found in metadata")
		c.JSON(http.StatusForbidden, gin.H{"error": constants.ErrPermissionDenied})
		c.Abort()
		return
	}
	
	// Look up action_id from action name
	action, err := m.metadataRepo.GetActionByName(screenModule.Module)
	if err != nil {
		log.Warn().Err(err).Str("action", screenModule.Module).Msg("Action not found in metadata")
		c.JSON(http.StatusForbidden, gin.H{"error": constants.ErrPermissionDenied})
		c.Abort()
		return
	}
	
	// Check permission using new permissions system with IDs
	isPermit, err := m.permissionRepo.CheckPermission(claims.Role, service.ID, screenType.ID, action.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error checking permission")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking permission"})
=======
		c.JSON(http.StatusInternalServerError, gin.H{constant.Error: "Error checking permission"})
>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/middleware.go
		c.Abort()
		return
	}
	if !isPermit {
<<<<<<< HEAD:horizon/internal/middlewares/middleware.go
		c.JSON(http.StatusForbidden, gin.H{"error": constants.ErrPermissionDenied})
=======
		c.JSON(http.StatusForbidden, gin.H{constant.Error: "Permission Denied"})
>>>>>>> 719e1f68b6c4710e883a4d61b281c16133c167a5:horizon/internal/middleware/middleware.go
		c.Abort()
	}
}

// RequireSuperAdmin middleware ensures only super_admin can access
func (m *MiddlewareHandler) RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found in context"})
			c.Abort()
			return
		}
		
		if role != constants.RoleSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// RequireAdminOrSuperAdmin middleware ensures admin or super_admin can access
func (m *MiddlewareHandler) RequireAdminOrSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Role not found in context"})
			c.Abort()
			return
		}
		
		if role != constants.RoleAdmin && role != constants.RoleSuperAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlyAdminOrSuperAdmin})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

func cloneRequestBody(c *gin.Context) ([]byte, bool) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, false
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, true
}
