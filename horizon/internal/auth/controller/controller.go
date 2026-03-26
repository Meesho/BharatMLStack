package controller

import (
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/controller"
	"github.com/Meesho/BharatMLStack/horizon/pkg/api"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Auth interface {
	Register(ctx *gin.Context)
	Login(ctx *gin.Context)
	Logout(ctx *gin.Context)
	GetAllUsers(ctx *gin.Context)
	UpdateUserAccessAndRole(ctx *gin.Context)
	GetSSOStatus(ctx *gin.Context)
	InitiateGoogleOAuth(ctx *gin.Context)
	GoogleOAuthCallback(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	TrackSession(ctx *gin.Context)
}

var (
	auth Auth
	once sync.Once
)

type AuthController struct {
	Authenticator handler.Authenticator
}

func NewController() Auth {
	if auth == nil {
		once.Do(func() {
			auth = &AuthController{
				Authenticator: handler.NewAuthenticator(1),
			}
		})
	}
	return auth
}

func (a *AuthController) Register(ctx *gin.Context) {
	var request handler.User
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	apiError := a.Authenticator.Register(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "User Registered Successfully"})
}

func (a *AuthController) Login(ctx *gin.Context) {
	var request handler.Login
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, apiError := a.Authenticator.Login(&request)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, token)
}

func (a *AuthController) Logout(ctx *gin.Context) {
	token := strings.TrimPrefix(ctx.GetHeader("Authorization"), "Bearer ")
	apiError := a.Authenticator.Logout(token)
	if apiError != nil {
		ctx.Error(apiError)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": apiError.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "User Logged out successfully"})
}

func (a *AuthController) GetAllUsers(ctx *gin.Context) {
	_, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	users, err := a.Authenticator.GetAllUsers()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, users)
}

func (a *AuthController) UpdateUserAccessAndRole(ctx *gin.Context) {
	var request handler.UpdateUserAccessAndRole
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		_ = ctx.Error(api.NewBadRequestError(err.Error()))
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	if role != "admin" {
		err = errors.New("not authorized to process request")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Role != "admin" && request.Role != "user" {
		err = errors.New("invalid role")
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = a.Authenticator.UpdateUserAccessAndRole(request.Email, request.IsActive, request.Role)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "User info updated successfully"})
}

// GetSSOStatus returns SSO configuration status (public endpoint)
func (a *AuthController) GetSSOStatus(ctx *gin.Context) {
	status, err := a.Authenticator.GetSSOStatus()
	if err != nil {
		log.Error().Err(err).Msg("Error getting SSO status")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, status)
}

// InitiateGoogleOAuth initiates Google OAuth flow (public endpoint)
func (a *AuthController) InitiateGoogleOAuth(ctx *gin.Context) {
	authURL, state, err := a.Authenticator.InitiateGoogleOAuth()
	if err != nil {
		log.Error().Err(err).Msg("Error initiating Google OAuth")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"auth_url": authURL,
		"state":    state,
	})
}

// GoogleOAuthCallback handles Google OAuth callback (public endpoint)
func (a *AuthController) GoogleOAuthCallback(ctx *gin.Context) {
	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" || state == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "code and state parameters are required"})
		return
	}

	loginResponse, err := a.Authenticator.LoginWithGoogle(code, state)
	if err != nil {
		log.Error().Err(err).Msg("Error in Google OAuth callback")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, loginResponse)
}

// RefreshToken refreshes access token using refresh token (public endpoint)
func (a *AuthController) RefreshToken(ctx *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := a.Authenticator.RefreshToken(request.RefreshToken)
	if err != nil {
		log.Error().Err(err).Msg("Error refreshing token")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// TrackSession tracks user session (authenticated endpoint)
func (a *AuthController) TrackSession(ctx *gin.Context) {
	var request struct {
		Email           string `json:"email"`
		UserID          string `json:"user_id,omitempty"`
		Role            string `json:"role"`
		SessionStartTime string `json:"session_start_time"`
		UserAgent       string `json:"user_agent"`
	}

	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For now, just return success - session tracking can be implemented later
	// This endpoint is called by frontend but may not need full implementation
	ctx.JSON(http.StatusOK, gin.H{
		"message":    "Session tracked successfully",
		"session_id": ctx.GetString("session_id"), // Can be generated if needed
	})
}
