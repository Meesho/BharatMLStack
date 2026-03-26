package controller

import (
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/controller"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type PermissionController struct {
	PermissionHandler *handler.PermissionHandler
	Authenticator     handler.Authenticator // Added to avoid hacky NewController() calls
}

func NewPermissionController() *PermissionController {
	return &PermissionController{
		PermissionHandler: handler.InitPermissionHandler(),
		Authenticator:     handler.InitAuthHandler(), // Inject authenticator to avoid hacky pattern
	}
}

// GetAllPermissions retrieves all permissions (super_admin only)
func (p *PermissionController) GetAllPermissions(ctx *gin.Context) {
	_, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}
	
	permissions, err := p.PermissionHandler.GetAllPermissions()
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, permissions)
}

// GetPermissionsByRole retrieves permissions for a specific role (super_admin only)
func (p *PermissionController) GetPermissionsByRole(ctx *gin.Context) {
	_, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}
	
	roleParam := ctx.Param("role")
	if roleParam == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrRoleParameterRequired})
		return
	}
	
	permissions, err := p.PermissionHandler.GetPermissionsByRole(roleParam)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, permissions)
}

// CreatePermission creates a new permission (super_admin only)
func (p *PermissionController) CreatePermission(ctx *gin.Context) {
	email, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}
	
	var request handler.PermissionRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user ID from email
	authUser, err := p.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}
	
	permission, err := p.PermissionHandler.CreatePermission(&request, authUser.ID, authUser.ID)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusCreated, permission)
}

// UpdatePermission updates an existing permission (super_admin only)
func (p *PermissionController) UpdatePermission(ctx *gin.Context) {
	email, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}
	
	permissionID := ctx.Param("id")
	if permissionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrPermissionIDRequired})
		return
	}
	
	id, err := strconv.ParseUint(permissionID, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidPermissionID})
		return
	}
	
	var request handler.PermissionRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user ID from email
	authUser, err := p.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}
	
	permission, err := p.PermissionHandler.UpdatePermission(uint(id), &request, authUser.ID)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, permission)
}

// DeletePermission deletes a permission (super_admin only)
func (p *PermissionController) DeletePermission(ctx *gin.Context) {
	_, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}
	
	permissionID := ctx.Param("id")
	if permissionID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrPermissionIDRequired})
		return
	}
	
	id, err := strconv.ParseUint(permissionID, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidPermissionID})
		return
	}
	
	err = p.PermissionHandler.DeletePermission(uint(id))
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{"message": constants.MsgPermissionDeleted})
}

// BulkUpdatePermissionsByRole updates all permissions for a role (super_admin only)
func (p *PermissionController) BulkUpdatePermissionsByRole(ctx *gin.Context) {
	email, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}
	
	roleParam := ctx.Param("role")
	if roleParam == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "role parameter is required"})
		return
	}
	
	var request []handler.PermissionRequest
	if err := ctx.BindJSON(&request); err != nil {
		log.Error().Err(err).Msg("Error in binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user ID from email
	authUser, err := p.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}
	
	err = p.PermissionHandler.BulkUpdatePermissionsByRole(roleParam, request, authUser.ID)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{"message": constants.MsgPermissionsUpdated})
}

// GetPermissionsByCurrentUserRole retrieves permissions for the authenticated user's role
// This endpoint is used by the frontend after login to get user permissions
func (p *PermissionController) GetPermissionsByCurrentUserRole(ctx *gin.Context) {
	_, role, err := controller.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}
	
	if role == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrRoleNotFoundInToken})
		return
	}
	
	// Get formatted permissions for the user's role
	permissions, err := p.PermissionHandler.GetPermissionsByRoleFormatted(role)
	if err != nil {
		ctx.Error(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, permissions)
}

