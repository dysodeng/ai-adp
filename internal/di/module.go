package di

import (
	"github.com/google/wire"

	tenantappsvc "github.com/dysodeng/ai-adp/internal/application/tenant/service"
	appdomainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	modeldomainrepo "github.com/dysodeng/ai-adp/internal/domain/model/repository"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
	apprepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/app"
	modelrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/model"
	tenantrepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/tenant"
	tenanthandler "github.com/dysodeng/ai-adp/internal/interfaces/http/handler"
)

// TenantModuleSet wires the complete tenant bounded context
var TenantModuleSet = wire.NewSet(
	tenantrepo.NewTenantRepository,
	wire.Bind(new(domainrepo.TenantRepository), new(*tenantrepo.TenantRepositoryImpl)),
	tenantappsvc.NewTenantAppService,
	wire.Bind(new(tenantappsvc.TenantService), new(*tenantappsvc.TenantAppService)),
	tenanthandler.NewTenantHandler,
)

// AppModuleSet wires the app bounded context
var AppModuleSet = wire.NewSet(
	apprepo.NewAppRepository,
	wire.Bind(new(appdomainrepo.AppRepository), new(*apprepo.AppRepositoryImpl)),
)

// ModelModuleSet wires the model bounded context
var ModelModuleSet = wire.NewSet(
	modelrepo.NewModelConfigRepository,
	wire.Bind(new(modeldomainrepo.ModelConfigRepository), new(*modelrepo.ModelConfigRepositoryImpl)),
)

// ModulesSet aggregates all bounded context Wire sets
var ModulesSet = wire.NewSet(
	TenantModuleSet,
	AppModuleSet,
	ModelModuleSet,
)
