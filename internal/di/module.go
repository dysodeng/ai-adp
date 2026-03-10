package di

import (
	"github.com/google/wire"

	tenantappsvc "github.com/dysodeng/ai-adp/internal/application/tenant/service"
	apprepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/app"
	modelrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/model"
	tenantrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/tenant"
	tenanthandler "github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
)

// TenantModuleSet wires the complete tenant bounded context
var TenantModuleSet = wire.NewSet(
	tenantrepo.NewTenantRepository,
	tenantappsvc.NewTenantAppService,
	tenanthandler.NewTenantHandler,
)

// AppModuleSet wires the app bounded context
var AppModuleSet = wire.NewSet(
	apprepo.NewAppRepository,
)

// ModelModuleSet wires the model bounded context
var ModelModuleSet = wire.NewSet(
	modelrepo.NewModelConfigRepository,
)

// ModulesSet aggregates all bounded context Wire sets
var ModulesSet = wire.NewSet(
	TenantModuleSet,
	AppModuleSet,
	ModelModuleSet,
	ChatModuleSet,
)
