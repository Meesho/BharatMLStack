package bulkdeletestrategy

import "github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/servicedeployableconfig"

type BulkDeleteStrategy interface {
	ProcessBulkDelete(serviceDeployable servicedeployableconfig.ServiceDeployableConfig) error
}
