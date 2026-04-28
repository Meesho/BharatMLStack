package handler

import (
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/metadata"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"github.com/rs/zerolog/log"
)

type MetadataHandler struct {
	metadataRepo metadata.MetadataRepository
}

func InitMetadataHandler() *MetadataHandler {
	connection, _ := infra.SQL.GetConnection()
	sqlConn := connection.(*infra.SQLConnection)
	metadataRepo, err := metadata.NewRepository(sqlConn)
	if err != nil {
		log.Error().Msgf("Error in creating metadata repository: %v", err)
		return nil
	}
	return &MetadataHandler{
		metadataRepo: metadataRepo,
	}
}

// ==================== Service Models ====================

type ServiceRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

type ServiceResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ==================== Screen Type Models ====================

type ScreenTypeRequest struct {
	ServiceID   uint   `json:"service_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

type ScreenTypeResponse struct {
	ID          uint   `json:"id"`
	ServiceID   uint   `json:"service_id"`
	ServiceName string `json:"service_name,omitempty"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ==================== Action Models ====================

type ActionRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Category    string `json:"category"` // 'crud', 'approval', 'testing', 'management'
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

type ActionResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ==================== Service Handlers ====================

func (h *MetadataHandler) GetAllServices() ([]ServiceResponse, error) {
	services, err := h.metadataRepo.GetAllServices()
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	responses := make([]ServiceResponse, len(services))
	for i, s := range services {
		responses[i] = ServiceResponse{
			ID:          s.ID,
			Name:        s.Name,
			DisplayName: s.DisplayName,
			Description: s.Description,
			IsActive:    s.IsActive,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return responses, nil
}

func (h *MetadataHandler) GetServiceByID(id uint) (*ServiceResponse, error) {
	service, err := h.metadataRepo.GetServiceByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &ServiceResponse{
		ID:          service.ID,
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		IsActive:    service.IsActive,
		CreatedAt:   service.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   service.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (h *MetadataHandler) CreateService(req *ServiceRequest, createdBy, updatedBy uint) (*ServiceResponse, error) {
	// Check if service with same name already exists
	existing, err := h.metadataRepo.GetServiceByName(req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("service with name '%s' already exists", req.Name)
	}

	service := &metadata.Service{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsActive:    req.IsActive,
	}
	if createdBy > 0 {
		service.CreatedBy = &createdBy
	}
	if updatedBy > 0 {
		service.UpdatedBy = &updatedBy
	}

	id, err := h.metadataRepo.CreateService(service)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return h.GetServiceByID(id)
}

func (h *MetadataHandler) UpdateService(id uint, req *ServiceRequest, updatedBy uint) (*ServiceResponse, error) {
	service := &metadata.Service{
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsActive:    req.IsActive,
	}
	if updatedBy > 0 {
		service.UpdatedBy = &updatedBy
	}

	err := h.metadataRepo.UpdateService(id, service)
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	return h.GetServiceByID(id)
}

func (h *MetadataHandler) DeleteService(id uint) error {
	return h.metadataRepo.DeleteService(id)
}

// ==================== Screen Type Handlers ====================

func (h *MetadataHandler) GetAllScreenTypes() ([]ScreenTypeResponse, error) {
	screenTypes, err := h.metadataRepo.GetAllScreenTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get screen types: %w", err)
	}

	responses := make([]ScreenTypeResponse, len(screenTypes))
	for i, st := range screenTypes {
		responses[i] = ScreenTypeResponse{
			ID:          st.ID,
			ServiceID:   st.ServiceID,
			ServiceName: st.Service.Name,
			Name:        st.Name,
			DisplayName: st.DisplayName,
			Description: st.Description,
			IsActive:    st.IsActive,
			CreatedAt:   st.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return responses, nil
}

func (h *MetadataHandler) GetScreenTypesByServiceID(serviceID uint) ([]ScreenTypeResponse, error) {
	screenTypes, err := h.metadataRepo.GetScreenTypesByServiceID(serviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get screen types: %w", err)
	}

	responses := make([]ScreenTypeResponse, len(screenTypes))
	for i, st := range screenTypes {
		responses[i] = ScreenTypeResponse{
			ID:          st.ID,
			ServiceID:   st.ServiceID,
			Name:        st.Name,
			DisplayName: st.DisplayName,
			Description: st.Description,
			IsActive:    st.IsActive,
			CreatedAt:   st.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return responses, nil
}

func (h *MetadataHandler) GetScreenTypeByID(id uint) (*ScreenTypeResponse, error) {
	screenType, err := h.metadataRepo.GetScreenTypeByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get screen type: %w", err)
	}

	return &ScreenTypeResponse{
		ID:          screenType.ID,
		ServiceID:   screenType.ServiceID,
		ServiceName: screenType.Service.Name,
		Name:        screenType.Name,
		DisplayName: screenType.DisplayName,
		Description: screenType.Description,
		IsActive:    screenType.IsActive,
		CreatedAt:   screenType.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   screenType.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (h *MetadataHandler) CreateScreenType(req *ScreenTypeRequest, createdBy, updatedBy uint) (*ScreenTypeResponse, error) {
	// Validate service exists
	_, err := h.metadataRepo.GetServiceByID(req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("invalid service_id: %w", err)
	}

	// Check if screen type with same name already exists for this service
	existing, err := h.metadataRepo.GetScreenTypeByServiceAndName(req.ServiceID, req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("screen type with name '%s' already exists for this service", req.Name)
	}

	screenType := &metadata.ScreenType{
		ServiceID:   req.ServiceID,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsActive:    req.IsActive,
	}
	if createdBy > 0 {
		screenType.CreatedBy = &createdBy
	}
	if updatedBy > 0 {
		screenType.UpdatedBy = &updatedBy
	}

	id, err := h.metadataRepo.CreateScreenType(screenType)
	if err != nil {
		return nil, fmt.Errorf("failed to create screen type: %w", err)
	}

	return h.GetScreenTypeByID(id)
}

func (h *MetadataHandler) UpdateScreenType(id uint, req *ScreenTypeRequest, updatedBy uint) (*ScreenTypeResponse, error) {
	// Validate service exists if service_id is being updated
	if req.ServiceID > 0 {
		_, err := h.metadataRepo.GetServiceByID(req.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("invalid service_id: %w", err)
		}
	}

	screenType := &metadata.ScreenType{
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsActive:    req.IsActive,
	}
	if req.ServiceID > 0 {
		screenType.ServiceID = req.ServiceID
	}
	if updatedBy > 0 {
		screenType.UpdatedBy = &updatedBy
	}

	err := h.metadataRepo.UpdateScreenType(id, screenType)
	if err != nil {
		return nil, fmt.Errorf("failed to update screen type: %w", err)
	}

	return h.GetScreenTypeByID(id)
}

func (h *MetadataHandler) DeleteScreenType(id uint) error {
	return h.metadataRepo.DeleteScreenType(id)
}

// ==================== Action Handlers ====================

func (h *MetadataHandler) GetAllActions() ([]ActionResponse, error) {
	actions, err := h.metadataRepo.GetAllActions()
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	responses := make([]ActionResponse, len(actions))
	for i, a := range actions {
		responses[i] = ActionResponse{
			ID:          a.ID,
			Name:        a.Name,
			DisplayName: a.DisplayName,
			Category:    a.Category,
			Description: a.Description,
			IsActive:    a.IsActive,
			CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return responses, nil
}

func (h *MetadataHandler) GetActionByID(id uint) (*ActionResponse, error) {
	action, err := h.metadataRepo.GetActionByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}

	return &ActionResponse{
		ID:          action.ID,
		Name:        action.Name,
		DisplayName: action.DisplayName,
		Category:    action.Category,
		Description: action.Description,
		IsActive:    action.IsActive,
		CreatedAt:   action.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   action.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (h *MetadataHandler) CreateAction(req *ActionRequest, createdBy, updatedBy uint) (*ActionResponse, error) {
	// Check if action with same name already exists
	existing, err := h.metadataRepo.GetActionByName(req.Name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("action with name '%s' already exists", req.Name)
	}

	action := &metadata.Action{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Category:    req.Category,
		Description: req.Description,
		IsActive:    req.IsActive,
	}
	if createdBy > 0 {
		action.CreatedBy = &createdBy
	}
	if updatedBy > 0 {
		action.UpdatedBy = &updatedBy
	}

	id, err := h.metadataRepo.CreateAction(action)
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	return h.GetActionByID(id)
}

func (h *MetadataHandler) UpdateAction(id uint, req *ActionRequest, updatedBy uint) (*ActionResponse, error) {
	action := &metadata.Action{
		DisplayName: req.DisplayName,
		Category:    req.Category,
		Description: req.Description,
		IsActive:    req.IsActive,
	}
	if updatedBy > 0 {
		action.UpdatedBy = &updatedBy
	}

	err := h.metadataRepo.UpdateAction(id, action)
	if err != nil {
		return nil, fmt.Errorf("failed to update action: %w", err)
	}

	return h.GetActionByID(id)
}

func (h *MetadataHandler) DeleteAction(id uint) error {
	return h.metadataRepo.DeleteAction(id)
}

// GetActionsByIDs returns actions by their IDs (for validation)
func (h *MetadataHandler) GetActionsByIDs(ids []uint) ([]ActionResponse, error) {
	actions, err := h.metadataRepo.GetActionsByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get actions: %w", err)
	}

	responses := make([]ActionResponse, len(actions))
	for i, a := range actions {
		responses[i] = ActionResponse{
			ID:          a.ID,
			Name:        a.Name,
			DisplayName: a.DisplayName,
			Category:    a.Category,
			Description: a.Description,
			IsActive:    a.IsActive,
			CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	return responses, nil
}



