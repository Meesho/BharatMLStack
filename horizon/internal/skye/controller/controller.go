package controller

import (
	"sync"

	"github.com/Meesho/BharatMLStack/horizon/internal/configs"
	"github.com/Meesho/BharatMLStack/horizon/internal/skye/handler"
	"github.com/gin-gonic/gin"
)

type Config interface {
	RegisterStore(ctx *gin.Context)
	ApproveStoreRequest(ctx *gin.Context)
	GetStores(ctx *gin.Context)
	GetAllStoreRequests(ctx *gin.Context)
	RegisterEntity(ctx *gin.Context)
	ApproveEntityRequest(ctx *gin.Context)
	GetEntities(ctx *gin.Context)
	GetAllEntityRequests(ctx *gin.Context)
	RegisterModel(ctx *gin.Context)
	ApproveModelRequest(ctx *gin.Context)
	GetModels(ctx *gin.Context)
	GetAllModelRequests(ctx *gin.Context)
	RegisterVariant(ctx *gin.Context)
	ApproveVariantRequest(ctx *gin.Context)
	GetVariants(ctx *gin.Context)
	GetAllVariantRequests(ctx *gin.Context)
	RegisterFilter(ctx *gin.Context)
	ApproveFilterRequest(ctx *gin.Context)
	GetFilters(ctx *gin.Context)
	GetAllFilters(ctx *gin.Context)
	GetAllFilterRequests(ctx *gin.Context)
	RegisterJobFrequency(ctx *gin.Context)
	ApproveJobFrequencyRequest(ctx *gin.Context)
	GetJobFrequencies(ctx *gin.Context)
	GetAllJobFrequencyRequests(ctx *gin.Context)
	GetMQIdTopics(ctx *gin.Context)
	GetVariantsList(ctx *gin.Context)
	// OnboardVariant(ctx *gin.Context)
	// ApproveVariantOnboardingRequest(ctx *gin.Context)
	// GetAllVariantOnboardingRequests(ctx *gin.Context)
	GetVariantOnboardingTasks(ctx *gin.Context)
	ScaleUpVariant(ctx *gin.Context)
	ApproveVariantScaleUpRequest(ctx *gin.Context)
	GetAllVariantScaleUpRequests(ctx *gin.Context)
	GetVariantScaleUpTasks(ctx *gin.Context)
	TestVariant(ctx *gin.Context)
	GenerateTestRequest(ctx *gin.Context)
}

var (
	configController Config
	once             sync.Once
)

type V1 struct {
	Config handler.Config
}

func NewConfigController(appConfig configs.Configs) Config {
	once.Do(func() {
		configController = &V1{
			Config: handler.NewConfigHandler(1, appConfig),
		}
	})
	return configController
}

// Store Operations
func (c *V1) RegisterStore(ctx *gin.Context) {
	handleRegister(ctx, c.Config.RegisterStore)
}

func (c *V1) ApproveStoreRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveStoreRequest)
}

func (c *V1) GetStores(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetStores)
}

func (c *V1) GetAllStoreRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllStoreRequests)
}

// Entity Operations
func (c *V1) RegisterEntity(ctx *gin.Context) {
	handleRegister(ctx, c.Config.RegisterEntity)
}

func (c *V1) ApproveEntityRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveEntityRequest)
}

func (c *V1) GetEntities(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetEntities)
}

func (c *V1) GetAllEntityRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllEntityRequests)
}

// Model Operations
func (c *V1) RegisterModel(ctx *gin.Context) {
	handleRegister(ctx, c.Config.RegisterModel)
}

func (c *V1) ApproveModelRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveModelRequest)
}

func (c *V1) GetModels(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetModels)
}

func (c *V1) GetAllModelRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllModelRequests)
}

// Variant Operations
func (c *V1) RegisterVariant(ctx *gin.Context) {
	handleRegister(ctx, c.Config.RegisterVariant)
}

func (c *V1) ApproveVariantRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveVariantRequest)
}

func (c *V1) GetVariants(ctx *gin.Context) {
	entity := ctx.Query("entity")
	model := ctx.Query("model")
	response, err := c.Config.GetVariants(entity, model)
	if err != nil {
		respondInternalError(ctx, err)
		return
	}
	respondSuccess(ctx, response)
}

func (c *V1) GetAllVariantRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllVariantRequests)
}

// Filter Operations
func (c *V1) RegisterFilter(ctx *gin.Context) {
	handleRegister(ctx, c.Config.RegisterFilter)
}

func (c *V1) ApproveFilterRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveFilterRequest)
}

func (c *V1) GetFilters(ctx *gin.Context) {
	entity := ctx.Query("entity")
	response, err := c.Config.GetFilters(entity)
	if err != nil {
		respondInternalError(ctx, err)
		return
	}
	respondSuccess(ctx, response)
}

func (c *V1) GetAllFilters(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllFilters)
}

func (c *V1) GetAllFilterRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllFilterRequests)
}

// Job Frequency Operations
func (c *V1) RegisterJobFrequency(ctx *gin.Context) {
	handleRegister(ctx, c.Config.RegisterJobFrequency)
}

func (c *V1) ApproveJobFrequencyRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveJobFrequencyRequest)
}

func (c *V1) GetJobFrequencies(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetJobFrequencies)
}

func (c *V1) GetAllJobFrequencyRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllJobFrequencyRequests)
}

// Miscellaneous Operations
func (c *V1) GetMQIdTopics(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetMQIdTopics)
}

func (c *V1) GetVariantsList(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetVariantsList)
}

// Variant Onboarding Operations
// func (c *V1) OnboardVariant(ctx *gin.Context) {
// 	handleRegister(ctx, c.Config.OnboardVariant)
// }

// func (c *V1) ApproveVariantOnboardingRequest(ctx *gin.Context) {
// 	handleApprove(ctx, c.Config.ApproveVariantOnboardingRequest)
// }

// func (c *V1) GetAllVariantOnboardingRequests(ctx *gin.Context) {
// 	handleSimpleGet(ctx, c.Config.GetAllVariantOnboardingRequests)
// }

func (c *V1) GetVariantOnboardingTasks(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetVariantOnboardingTasks)
}

// Variant Scale-Up Operations
func (c *V1) ScaleUpVariant(ctx *gin.Context) {
	handleRegister(ctx, c.Config.ScaleUpVariant)
}

func (c *V1) ApproveVariantScaleUpRequest(ctx *gin.Context) {
	handleApprove(ctx, c.Config.ApproveVariantScaleUpRequest)
}

func (c *V1) GetAllVariantScaleUpRequests(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetAllVariantScaleUpRequests)
}

func (c *V1) GetVariantScaleUpTasks(ctx *gin.Context) {
	handleSimpleGet(ctx, c.Config.GetVariantScaleUpTasks)
}

func (c *V1) TestVariant(ctx *gin.Context) {
	handleTest(ctx, c.Config.TestVariant)
}

func (c *V1) GenerateTestRequest(ctx *gin.Context) {
	handleRegister(ctx, c.Config.GenerateTestRequest)
}
