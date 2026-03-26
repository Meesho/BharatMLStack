package controller

import (
	"net/http"
	"strconv"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
	"github.com/Meesho/BharatMLStack/horizon/internal/auth/handler"
	ofsController "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/controller"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type MetadataController struct {
	MetadataHandler *handler.MetadataHandler
	Authenticator   handler.Authenticator
}

func NewMetadataController() *MetadataController {
	return &MetadataController{
		MetadataHandler: handler.InitMetadataHandler(),
		Authenticator:   handler.InitAuthHandler(),
	}
}

// ==================== Service Endpoints ====================

// GetAllServices retrieves all services (authenticated users)
func (m *MetadataController) GetAllServices(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	services, err := m.MetadataHandler.GetAllServices()
	if err != nil {
		log.Error().Err(err).Msg("Error getting services")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"services": services})
}

// GetServiceByID retrieves a service by ID (authenticated users)
func (m *MetadataController) GetServiceByID(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid service ID"})
		return
	}

	service, err := m.MetadataHandler.GetServiceByID(uint(id))
	if err != nil {
		log.Error().Err(err).Msg("Error getting service")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}

	ctx.JSON(http.StatusOK, service)
}

// CreateService creates a new service (super_admin only)
func (m *MetadataController) CreateService(ctx *gin.Context) {
	email, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	var req handler.ServiceRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authUser, err := m.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}

	service, err := m.MetadataHandler.CreateService(&req, authUser.ID, authUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error creating service")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, service)
}

// UpdateService updates a service (super_admin only)
func (m *MetadataController) UpdateService(ctx *gin.Context) {
	email, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid service ID"})
		return
	}

	var req handler.ServiceRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authUser, err := m.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}

	service, err := m.MetadataHandler.UpdateService(uint(id), &req, authUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error updating service")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, service)
}

// DeleteService deletes a service (super_admin only)
func (m *MetadataController) DeleteService(ctx *gin.Context) {
	_, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid service ID"})
		return
	}

	err = m.MetadataHandler.DeleteService(uint(id))
	if err != nil {
		log.Error().Err(err).Msg("Error deleting service")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Service deleted successfully"})
}

// ==================== Screen Type Endpoints ====================

// GetAllScreenTypes retrieves all screen types (authenticated users)
func (m *MetadataController) GetAllScreenTypes(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	screenTypes, err := m.MetadataHandler.GetAllScreenTypes()
	if err != nil {
		log.Error().Err(err).Msg("Error getting screen types")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"screen_types": screenTypes})
}

// GetScreenTypesByServiceID retrieves screen types for a service (authenticated users)
func (m *MetadataController) GetScreenTypesByServiceID(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	serviceIDStr := ctx.Query("service_id")
	if serviceIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "service_id parameter required"})
		return
	}

	serviceID, err := strconv.ParseUint(serviceIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid service_id"})
		return
	}

	screenTypes, err := m.MetadataHandler.GetScreenTypesByServiceID(uint(serviceID))
	if err != nil {
		log.Error().Err(err).Msg("Error getting screen types")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"screen_types": screenTypes})
}

// GetScreenTypeByID retrieves a screen type by ID (authenticated users)
func (m *MetadataController) GetScreenTypeByID(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid screen type ID"})
		return
	}

	screenType, err := m.MetadataHandler.GetScreenTypeByID(uint(id))
	if err != nil {
		log.Error().Err(err).Msg("Error getting screen type")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "screen type not found"})
		return
	}

	ctx.JSON(http.StatusOK, screenType)
}

// CreateScreenType creates a new screen type (super_admin only)
func (m *MetadataController) CreateScreenType(ctx *gin.Context) {
	email, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	var req handler.ScreenTypeRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authUser, err := m.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}

	screenType, err := m.MetadataHandler.CreateScreenType(&req, authUser.ID, authUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error creating screen type")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, screenType)
}

// UpdateScreenType updates a screen type (super_admin only)
func (m *MetadataController) UpdateScreenType(ctx *gin.Context) {
	email, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid screen type ID"})
		return
	}

	var req handler.ScreenTypeRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authUser, err := m.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}

	screenType, err := m.MetadataHandler.UpdateScreenType(uint(id), &req, authUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error updating screen type")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, screenType)
}

// DeleteScreenType deletes a screen type (super_admin only)
func (m *MetadataController) DeleteScreenType(ctx *gin.Context) {
	_, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid screen type ID"})
		return
	}

	err = m.MetadataHandler.DeleteScreenType(uint(id))
	if err != nil {
		log.Error().Err(err).Msg("Error deleting screen type")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Screen type deleted successfully"})
}

// ==================== Action Endpoints ====================

// GetAllActions retrieves all actions (authenticated users)
func (m *MetadataController) GetAllActions(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	actions, err := m.MetadataHandler.GetAllActions()
	if err != nil {
		log.Error().Err(err).Msg("Error getting actions")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"actions": actions})
}

// GetActionByID retrieves an action by ID (authenticated users)
func (m *MetadataController) GetActionByID(ctx *gin.Context) {
	_, _, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid action ID"})
		return
	}

	action, err := m.MetadataHandler.GetActionByID(uint(id))
	if err != nil {
		log.Error().Err(err).Msg("Error getting action")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
		return
	}

	ctx.JSON(http.StatusOK, action)
}

// CreateAction creates a new action (super_admin only)
func (m *MetadataController) CreateAction(ctx *gin.Context) {
	email, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	var req handler.ActionRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authUser, err := m.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}

	action, err := m.MetadataHandler.CreateAction(&req, authUser.ID, authUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error creating action")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, action)
}

// UpdateAction updates an action (super_admin only)
func (m *MetadataController) UpdateAction(ctx *gin.Context) {
	email, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid action ID"})
		return
	}

	var req handler.ActionRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Error().Err(err).Msg("Error binding request body")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authUser, err := m.Authenticator.GetUserByEmail(email)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": constants.ErrUserNotFound})
		return
	}

	action, err := m.MetadataHandler.UpdateAction(uint(id), &req, authUser.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error updating action")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, action)
}

// DeleteAction deletes an action (super_admin only)
func (m *MetadataController) DeleteAction(ctx *gin.Context) {
	_, role, err := ofsController.ParseAuthenticationHeader(ctx)
	if err != nil {
		return
	}

	if role != constants.RoleSuperAdmin {
		ctx.JSON(http.StatusForbidden, gin.H{"error": constants.ErrOnlySuperAdmin})
		return
	}

	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid action ID"})
		return
	}

	err = m.MetadataHandler.DeleteAction(uint(id))
	if err != nil {
		log.Error().Err(err).Msg("Error deleting action")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Action deleted successfully"})
}

