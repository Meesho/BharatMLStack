package resolver

import "github.com/gin-gonic/gin"

type ScreenModule struct {
	ScreenType string
	Module     string
	Service    string
}

type Func func(c *gin.Context) ScreenModule

type ServiceResolver interface {
	GetResolvers() map[string]Func
}

func StaticResolver(screenType, module, service string) Func {
	return func(c *gin.Context) ScreenModule {
		return ScreenModule{
			ScreenType: screenType,
			Module:     module,
			Service:    service,
		}
	}
}
