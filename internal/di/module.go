package di

import (
	"github.com/google/wire"

	tenantappsvc "github.com/dysodeng/ai-adp/internal/application/tenant/service"
	domainrepo "github.com/dysodeng/ai-adp/internal/domain/tenant/repository"
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

// ModulesSet aggregates all bounded context Wire sets
var ModulesSet = wire.NewSet(
	TenantModuleSet,
)
