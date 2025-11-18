package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/constant"
	"github.com/Meesho/BharatMLStack/horizon/internal/middleware/resolver"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/apiresolver"
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
	rolePermissionRepo rolepermission.Repository
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

		middleware = &MiddlewareHandler{
			tokenRepo:          tokenRepo,
			apiResolverRepo:    apiResolverRepo,
			rolePermissionRepo: rolePermissionRepo,
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
	corsConfig.AllowOrigins = []string{"*"} // Adjust to specific origins if needed
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true

	middlewares = append(middlewares, cors.New(corsConfig))
	return middlewares
}

// AuthMiddleware checks for a valid JWT token except on login and register routes
func (m *MiddlewareHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bypass authentication for login, register, and specific routes
		if strings.HasPrefix(c.Request.URL.Path, "/login") ||
			strings.HasPrefix(c.Request.URL.Path, "/register") ||
			strings.HasPrefix(c.Request.URL.Path, "/health") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/1.0/fs-config") {
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

		m.CheckScreenPermission(c, claims)

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

	apiResolver, err := m.apiResolverRepo.GetResolver(method, path)

	if &apiResolver == nil || apiResolver.ResolverFn == "" {
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constant.Error: "Unable to resolve API"})
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
	isPermit, err := m.rolePermissionRepo.CheckPermission(claims.Role, screenModule.Service, screenModule.ScreenType, screenModule.Module)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constant.Error: "Error checking permission"})
		c.Abort()
		return
	}
	if !isPermit {
		c.JSON(http.StatusForbidden, gin.H{constant.Error: "Permission Denied"})
		c.Abort()
		return
	}
	return
}

func cloneRequestBody(c *gin.Context) ([]byte, bool) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, false
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, true
}
