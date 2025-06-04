package middlewares

import (
	"net/http"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
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
	tokenRepo token.Repository
}

func NewMiddleware() Middleware {
	middlewareOnce.Do(func() {
		connection, _ := infra.SQL.GetConnection()
		sqlConn := connection.(*infra.SQLConnection)
		tokenRepo, err := token.NewRepository(sqlConn)
		if err != nil {
			log.Error().Msgf("Error in creating token repository")
		}
		middleware = &MiddlewareHandler{
			tokenRepo: tokenRepo,
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
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true

	middlewares = append(middlewares, cors.New(corsConfig))
	return middlewares
}

// AuthMiddleware checks for a valid JWT token except on login and register routes
func (m *MiddlewareHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bypass authentication for login and register routes
		if strings.HasPrefix(c.Request.URL.Path, "/login") ||
			strings.HasPrefix(c.Request.URL.Path, "/register") ||
			strings.HasPrefix(c.Request.URL.Path, "/health") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/v1/online-feature-store/get-source-mapping") {
			c.Next()
			return
		}

		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if the header is in the correct format (e.g., "Bearer <token>")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token must be Bearer <token>"})
			c.Abort()
			return
		}

		valid, err := m.tokenRepo.IsTokenValid(tokenString)
		if err != nil || !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Parse and validate the JWT token
		claims := &handler.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return handler.JwtKey, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set claims in the context for later use
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}
