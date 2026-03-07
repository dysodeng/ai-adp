package di

import (
	"github.com/google/wire"

	tenantappsvc "github.com/dysodeng/ai-adp/internal/application/tenant/service"
	appdomainrepo "github.com/dysodeng/ai-adp/internal/domain/app/repository"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
	apprepo "github.com/dysodeng/ai-adp/internal/infrastructure/persistence/repository/app"
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

// ModulesSet aggregates all bounded context Wire sets
var ModulesSet = wire.NewSet(
	TenantModuleSet,
	AppModuleSet,
)
