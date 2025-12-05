package resolver

import (
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
)

const (
	// Screen types and services
	screenTypeDeployable = "deployable"
	serviceAll           = "all"

	// Modules
	moduleDeployableOnBoard = "onboard"
	moduleDeployableEdit    = "edit"
	moduleDeployableView    = "view"

	// Resolvers
	resolverDeployableRefresh  = "DeployableRefreshResolver"
	resolverDeployableMetadata = "DeployableMetadataViewResolver"
	resolverDeployableCreate   = "DeployableCreateResolver"
	resolverDeployableUpdate   = "DeployableUpdateResolver"
	resolverDeployableView     = "DeployableViewResolver"
	resolverDeployableTune     = "DeployableTuneResolver"
)

type DeployableResolver struct {
	repo servicedeployableconfig.ServiceDeployableRepository
}

func NewDeployableServiceResolver() (ServiceResolver, error) {
	conn, err := infra.SQL.GetConnection()
	if err != nil {
		return nil, err
	}

	sqlConn := conn.(*infra.SQLConnection)
	repo, err := servicedeployableconfig.NewRepository(sqlConn)
	if err != nil {
		return nil, err
	}

	return &DeployableResolver{repo: repo}, nil
}

func (d *DeployableResolver) GetResolvers() map[string]Func {
	return map[string]Func{
		resolverDeployableCreate:   StaticResolver(screenTypeDeployable, moduleDeployableOnBoard, serviceAll),
		resolverDeployableUpdate:   StaticResolver(screenTypeDeployable, moduleDeployableEdit, serviceAll),
		resolverDeployableView:     StaticResolver(screenTypeDeployable, moduleDeployableView, serviceAll),
		resolverDeployableTune:     StaticResolver(screenTypeDeployable, moduleDeployableEdit, serviceAll),
		resolverDeployableRefresh:  StaticResolver(screenTypeDeployable, moduleDeployableView, serviceAll),
		resolverDeployableMetadata: StaticResolver(screenTypeDeployable, moduleDeployableView, serviceAll),
	}
}
