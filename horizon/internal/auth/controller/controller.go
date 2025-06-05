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
