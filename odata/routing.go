package odata

import (
	"github.com/gofiber/fiber/v3"
)

// setupBaseRoutes configura as rotas base do servidor OData
func (s *Server) setupBaseRoutes() {
	prefix := s.config.RoutePrefix

	// Rota para metadados
	s.router.Get(prefix+"/$metadata", s.handleMetadata)

	// Rota para service document
	s.router.Get(prefix+"/", s.handleServiceDocument)

	// Rota para $batch (OData v4)
	s.router.Post(prefix+"/$batch", s.HandleBatch)

	// Rota para health check
	s.router.Get("/health", s.handleHealth)

	// Rota para server info
	s.router.Get("/info", s.handleServerInfo)

	// Rotas específicas para multi-tenant
	if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
		// Rota para informações dos tenants
		s.router.Get("/tenants", s.handleTenantList)

		// Rota para estatísticas dos tenants
		s.router.Get("/tenants/stats", s.handleTenantStats)

		// Rota para health check específico de tenant
		s.router.Get("/tenants/:tenantId/health", s.handleTenantHealth)
	}
}

// setupEntityRoutes configura as rotas para uma entidade
func (s *Server) setupEntityRoutes(entityName string) {
	prefix := s.config.RoutePrefix

	// Obter configuração de autenticação da entidade (se houver)
	entityAuth, hasAuth := s.GetEntityAuth(entityName)

	// Preparar middlewares customizados da entidade
	// Usar []any desde o início para evitar conversão posterior
	var middlewares []any
	if hasAuth && len(entityAuth.Middlewares) > 0 {
		// Converter []fiber.Handler para []any ao adicionar
		for _, m := range entityAuth.Middlewares {
			middlewares = append(middlewares, m)
		}
	}

	// Adicionar middleware de readonly se necessário
	if hasAuth && entityAuth.ReadOnly {
		readOnlyMiddleware := func(c fiber.Ctx) error {
			method := c.Method()
			if method != "GET" && method != "OPTIONS" {
				return fiber.NewError(fiber.StatusForbidden, "Entidade "+entityName+" é apenas leitura")
			}
			return c.Next()
		}
		middlewares = append(middlewares, readOnlyMiddleware)
	}

	// Função helper para verificar se operação é permitida
	isOperationAllowed := func(operation string) bool {
		if !hasAuth || len(entityAuth.Permissions) == 0 {
			return true // Se não tem permissions definidas, permite todas
		}
		for _, perm := range entityAuth.Permissions {
			if perm == operation {
				return true
			}
		}
		return false
	}

	// Rota para coleção de entidades (GET, POST)
	if isOperationAllowed("GET") {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Get(prefix+"/"+entityName, s.handleEntityCollection, middlewares...)
		} else {
			s.router.Get(prefix+"/"+entityName, s.handleEntityCollection)
		}
	}

	if isOperationAllowed("POST") && !(hasAuth && entityAuth.ReadOnly) {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Post(prefix+"/"+entityName, s.handleEntityCollection, middlewares...)
		} else {
			s.router.Post(prefix+"/"+entityName, s.handleEntityCollection)
		}
	}

	// Rota para entidade individual (GET, PUT, PATCH, DELETE)
	if isOperationAllowed("GET") {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Get(prefix+"/"+entityName+"(*)", s.handleEntityById, middlewares...)
		} else {
			s.router.Get(prefix+"/"+entityName+"(*)", s.handleEntityById)
		}
	}

	if isOperationAllowed("PUT") && !(hasAuth && entityAuth.ReadOnly) {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Put(prefix+"/"+entityName+"(*)", s.handleEntityById, middlewares...)
		} else {
			s.router.Put(prefix+"/"+entityName+"(*)", s.handleEntityById)
		}
	}

	if isOperationAllowed("PATCH") && !(hasAuth && entityAuth.ReadOnly) {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Patch(prefix+"/"+entityName+"(*)", s.handleEntityById, middlewares...)
		} else {
			s.router.Patch(prefix+"/"+entityName+"(*)", s.handleEntityById)
		}
	}

	if isOperationAllowed("DELETE") && !(hasAuth && entityAuth.ReadOnly) {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Delete(prefix+"/"+entityName+"(*)", s.handleEntityById, middlewares...)
		} else {
			s.router.Delete(prefix+"/"+entityName+"(*)", s.handleEntityById)
		}
	}

	// Rota para count da coleção (sempre GET)
	if isOperationAllowed("GET") {
		if len(middlewares) > 0 {
			// Fiber v3: handler PRIMEIRO, depois middlewares
			s.router.Get(prefix+"/"+entityName+"/$count", s.handleEntityCount, middlewares...)
		} else {
			s.router.Get(prefix+"/"+entityName+"/$count", s.handleEntityCount)
		}
	}

	// Rota OPTIONS para CORS se habilitado
	if s.config.EnableCORS {
		s.router.Options(prefix+"/"+entityName, s.handleOptions)
		s.router.Options(prefix+"/"+entityName+"(*)", s.handleOptions)
	}
}
