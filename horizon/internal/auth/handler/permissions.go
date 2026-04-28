package handler

import (
	"encoding/json"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/auth/constants"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/metadata"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/permissions"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type PermissionHandler struct {
	permissionRepo permissions.Repository
	metadataRepo   metadata.MetadataRepository
}

func InitPermissionHandler() *PermissionHandler {
	connection, _ := infra.SQL.GetConnection()
	sqlConn := connection.(*infra.SQLConnection)
	permissionRepo, err := permissions.NewRepository(sqlConn)
	if err != nil {
		log.Error().Msgf("Error in creating permission repository: %v", err)
		return nil
	}
	metadataRepo, err := metadata.NewRepository(sqlConn)
	if err != nil {
		log.Error().Msgf("Error in creating metadata repository: %v", err)
		return nil
	}
	return &PermissionHandler{
		permissionRepo: permissionRepo,
		metadataRepo:   metadataRepo,
	}
}

type PermissionRequest struct {
	Role           string   `json:"role" binding:"required"`
	ServiceID      uint     `json:"service_id" binding:"required"`
	ScreenTypeID   uint     `json:"screen_type_id" binding:"required"`
	AllowedActions []uint   `json:"allowed_actions" binding:"required"` // Array of action IDs
}

type PermissionResponse struct {
	ID             uint     `json:"id"`
	Role           string   `json:"role"`
	ServiceID      uint     `json:"service_id"`
	ServiceName    string   `json:"service_name"`
	ScreenTypeID   uint     `json:"screen_type_id"`
	ScreenTypeName string   `json:"screen_type_name"`
	AllowedActions []uint   `json:"allowed_actions"` // Array of action IDs
	AllowedActionNames []string `json:"allowed_action_names"` // Array of action names for convenience
	CreatedBy      uint     `json:"created_by"`
	UpdatedBy      uint     `json:"updated_by"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// GetAllPermissions retrieves all permissions
func (p *PermissionHandler) GetAllPermissions() ([]PermissionResponse, error) {
	perms, err := p.permissionRepo.GetAllPermissions()
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	return p.convertPermissionsToResponse(perms)
}

// GetPermissionsByRole retrieves permissions for a specific role
func (p *PermissionHandler) GetPermissionsByRole(role string) ([]PermissionResponse, error) {
	perms, err := p.permissionRepo.GetPermissionsByRole(role)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	return p.convertPermissionsToResponse(perms)
}

// CreatePermission creates a new permission with validation
func (p *PermissionHandler) CreatePermission(req *PermissionRequest, createdBy, updatedBy uint) (*PermissionResponse, error) {
	// Validate service exists and is active
	service, err := p.metadataRepo.GetServiceByID(req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("invalid service_id: %w", err)
	}
	if !service.IsActive {
		return nil, fmt.Errorf("service is not active")
	}

	// Validate screen type exists, belongs to service, and is active
	screenType, err := p.metadataRepo.GetScreenTypeByID(req.ScreenTypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid screen_type_id: %w", err)
	}
	if screenType.ServiceID != req.ServiceID {
		return nil, fmt.Errorf("screen type does not belong to the specified service")
	}
	if !screenType.IsActive {
		return nil, fmt.Errorf("screen type is not active")
	}

	// Validate all actions exist and are active
	actions, err := p.metadataRepo.GetActionsByIDs(req.AllowedActions)
	if err != nil {
		return nil, fmt.Errorf("failed to validate actions: %w", err)
	}
	if len(actions) != len(req.AllowedActions) {
		return nil, fmt.Errorf("some action IDs are invalid or inactive")
	}
	for _, action := range actions {
		if !action.IsActive {
			return nil, fmt.Errorf("action '%s' is not active", action.Name)
		}
	}

	// Convert allowed_actions to JSON
	allowedActionsJSON, err := json.Marshal(req.AllowedActions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allowed_actions: %w", err)
	}

	permission := &permissions.Permission{
		Role:           req.Role,
		ServiceID:      req.ServiceID,
		ScreenTypeID:   req.ScreenTypeID,
		AllowedActions: string(allowedActionsJSON),
		CreatedBy:      createdBy,
		UpdatedBy:      updatedBy,
	}

	id, err := p.permissionRepo.CreatePermission(permission)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	permission.ID = id
	return p.convertPermissionToResponse(permission)
}

// UpdatePermission updates an existing permission with validation
func (p *PermissionHandler) UpdatePermission(id uint, req *PermissionRequest, updatedBy uint) (*PermissionResponse, error) {
	// Validate service exists and is active
	service, err := p.metadataRepo.GetServiceByID(req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("invalid service_id: %w", err)
	}
	if !service.IsActive {
		return nil, fmt.Errorf("service is not active")
	}

	// Validate screen type exists, belongs to service, and is active
	screenType, err := p.metadataRepo.GetScreenTypeByID(req.ScreenTypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid screen_type_id: %w", err)
	}
	if screenType.ServiceID != req.ServiceID {
		return nil, fmt.Errorf("screen type does not belong to the specified service")
	}
	if !screenType.IsActive {
		return nil, fmt.Errorf("screen type is not active")
	}

	// Validate all actions exist and are active
	actions, err := p.metadataRepo.GetActionsByIDs(req.AllowedActions)
	if err != nil {
		return nil, fmt.Errorf("failed to validate actions: %w", err)
	}
	if len(actions) != len(req.AllowedActions) {
		return nil, fmt.Errorf("some action IDs are invalid or inactive")
	}
	for _, action := range actions {
		if !action.IsActive {
			return nil, fmt.Errorf("action '%s' is not active", action.Name)
		}
	}

	// Convert allowed_actions to JSON
	allowedActionsJSON, err := json.Marshal(req.AllowedActions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal allowed_actions: %w", err)
	}

	permission := &permissions.Permission{
		ID:             id,
		Role:           req.Role,
		ServiceID:      req.ServiceID,
		ScreenTypeID:   req.ScreenTypeID,
		AllowedActions: string(allowedActionsJSON),
		UpdatedBy:      updatedBy,
	}

	err = p.permissionRepo.UpdatePermission(id, permission)
	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	// Get updated permission
	perm, err := p.permissionRepo.GetPermission(req.Role, req.ServiceID, req.ScreenTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated permission: %w", err)
	}

	return p.convertPermissionToResponse(perm)
}

// DeletePermission deletes a permission
func (p *PermissionHandler) DeletePermission(id uint) error {
	return p.permissionRepo.DeletePermission(id)
}

// BulkUpdatePermissionsByRole updates all permissions for a role
func (p *PermissionHandler) BulkUpdatePermissionsByRole(role string, permissionList []PermissionRequest, updatedBy uint) error {
	perms := make([]permissions.Permission, len(permissionList))

	for i, req := range permissionList {
		// Validate each permission
		_, err := p.metadataRepo.GetServiceByID(req.ServiceID)
		if err != nil {
			return fmt.Errorf("invalid service_id for permission %d: %w", i, err)
		}

		_, err = p.metadataRepo.GetScreenTypeByID(req.ScreenTypeID)
		if err != nil {
			return fmt.Errorf("invalid screen_type_id for permission %d: %w", i, err)
		}

		_, err = p.metadataRepo.GetActionsByIDs(req.AllowedActions)
		if err != nil {
			return fmt.Errorf("invalid action IDs for permission %d: %w", i, err)
		}

		allowedActionsJSON, err := json.Marshal(req.AllowedActions)
		if err != nil {
			return fmt.Errorf("failed to marshal allowed_actions for permission %d: %w", i, err)
		}

		perms[i] = permissions.Permission{
			Role:           role,
			ServiceID:      req.ServiceID,
			ScreenTypeID:   req.ScreenTypeID,
			AllowedActions: string(allowedActionsJSON),
			CreatedBy:      updatedBy,
			UpdatedBy:      updatedBy,
		}
	}

	return p.permissionRepo.BulkUpdatePermissionsByRole(role, perms)
}

// Helper functions
func (p *PermissionHandler) convertPermissionsToResponse(perms []permissions.Permission) ([]PermissionResponse, error) {
	responses := make([]PermissionResponse, len(perms))
	for i, perm := range perms {
		response, err := p.convertPermissionToResponse(&perm)
		if err != nil {
			return nil, err
		}
		responses[i] = *response
	}
	return responses, nil
}

func (p *PermissionHandler) convertPermissionToResponse(perm *permissions.Permission) (*PermissionResponse, error) {
	// Parse allowed_actions JSON array (array of action IDs)
	var allowedActionIDs []uint
	if err := json.Unmarshal([]byte(perm.AllowedActions), &allowedActionIDs); err != nil {
		allowedActionIDs = []uint{}
	}

	// Get service and screen type names
	service, err := p.metadataRepo.GetServiceByID(perm.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	screenType, err := p.metadataRepo.GetScreenTypeByID(perm.ScreenTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get screen type: %w", err)
	}

	// Get action names
	actions, err := p.metadataRepo.GetActionsByIDs(allowedActionIDs)
	if err != nil {
		// Log error but continue with empty action names
		log.Warn().Err(err).Msg("Failed to get action names")
	}

	actionNames := make([]string, len(actions))
	for i, action := range actions {
		actionNames[i] = action.Name
	}

	return &PermissionResponse{
		ID:                perm.ID,
		Role:              perm.Role,
		ServiceID:         perm.ServiceID,
		ServiceName:       service.Name,
		ScreenTypeID:      perm.ScreenTypeID,
		ScreenTypeName:    screenType.Name,
		AllowedActions:    allowedActionIDs,
		AllowedActionNames: actionNames,
		CreatedBy:         perm.CreatedBy,
		UpdatedBy:         perm.UpdatedBy,
		CreatedAt:         perm.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         perm.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// GetPermissionsByRoleFormatted retrieves permissions for a role and formats them for frontend
// Returns format: { role: string, permissions: [{ service: string, screens: [{ screenType: string, allowedActions: [] }] }] }
// For super_admin, returns all unique service/screen combinations with all actions
func (p *PermissionHandler) GetPermissionsByRoleFormatted(role string) (map[string]interface{}, error) {
	// Super admin has all permissions - get all unique service/screen combinations
	if role == constants.RoleSuperAdmin {
		return p.getAllPermissionsForSuperAdmin()
	}

	perms, err := p.permissionRepo.GetPermissionsByRole(role)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}

	// Group permissions by service
	serviceMap := make(map[string]map[string][]string) // service -> screenType -> allowedActionNames

	for _, perm := range perms {
		// Get service and screen type names
		service, err := p.metadataRepo.GetServiceByID(perm.ServiceID)
		if err != nil {
			continue // Skip if service not found
		}

		screenType, err := p.metadataRepo.GetScreenTypeByID(perm.ScreenTypeID)
		if err != nil {
			continue // Skip if screen type not found
		}

		// Parse allowed_actions JSON array (action IDs)
		var allowedActionIDs []uint
		if err := json.Unmarshal([]byte(perm.AllowedActions), &allowedActionIDs); err != nil {
			continue
		}

		// Get action names
		actions, err := p.metadataRepo.GetActionsByIDs(allowedActionIDs)
		if err != nil {
			continue
		}

		actionNames := make([]string, len(actions))
		for i, action := range actions {
			actionNames[i] = action.Name
		}

		if serviceMap[service.Name] == nil {
			serviceMap[service.Name] = make(map[string][]string)
		}
		serviceMap[service.Name][screenType.Name] = actionNames
	}

	// Convert to frontend expected format
	permissionsList := make([]map[string]interface{}, 0)
	for service, screens := range serviceMap {
		screensList := make([]map[string]interface{}, 0)
		for screenType, allowedActions := range screens {
			screensList = append(screensList, map[string]interface{}{
				"screenType":     screenType,
				"allowedActions": allowedActions,
			})
		}
		permissionsList = append(permissionsList, map[string]interface{}{
			"service": service,
			"screens": screensList,
		})
	}

	return map[string]interface{}{
		"role":        role,
		"permissions": permissionsList,
	}, nil
}

// getAllPermissionsForSuperAdmin returns all permissions for super_admin role from database
func (p *PermissionHandler) getAllPermissionsForSuperAdmin() (map[string]interface{}, error) {
	// Get super_admin permissions from database
	perms, err := p.permissionRepo.GetPermissionsByRole(constants.RoleSuperAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to get super_admin permissions: %w", err)
	}

	// Group permissions by service
	serviceMap := make(map[string]map[string][]string) // service -> screenType -> allowedActionNames

	for _, perm := range perms {
		// Get service and screen type names
		service, err := p.metadataRepo.GetServiceByID(perm.ServiceID)
		if err != nil {
			continue // Skip if service not found
		}

		screenType, err := p.metadataRepo.GetScreenTypeByID(perm.ScreenTypeID)
		if err != nil {
			continue // Skip if screen type not found
		}

		// Parse allowed_actions JSON array (action IDs)
		var allowedActionIDs []uint
		if err := json.Unmarshal([]byte(perm.AllowedActions), &allowedActionIDs); err != nil {
			continue
		}

		// Get action names
		actions, err := p.metadataRepo.GetActionsByIDs(allowedActionIDs)
		if err != nil {
			continue
		}

		actionNames := make([]string, len(actions))
		for i, action := range actions {
			actionNames[i] = action.Name
		}

		if serviceMap[service.Name] == nil {
			serviceMap[service.Name] = make(map[string][]string)
		}
		serviceMap[service.Name][screenType.Name] = actionNames
	}

	// Convert to frontend expected format
	permissionsList := make([]map[string]interface{}, 0)
	for service, screens := range serviceMap {
		screensList := make([]map[string]interface{}, 0)
		for screenType, allowedActions := range screens {
			screensList = append(screensList, map[string]interface{}{
				"screenType":     screenType,
				"allowedActions": allowedActions,
			})
		}
		permissionsList = append(permissionsList, map[string]interface{}{
			"service": service,
			"screens": screensList,
		})
	}

	return map[string]interface{}{
		"role":        constants.RoleSuperAdmin,
		"permissions": permissionsList,
	}, nil
}
