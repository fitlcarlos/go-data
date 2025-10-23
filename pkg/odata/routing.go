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

	// Configurar middlewares de autenticação se JWT estiver habilitado
	var authMiddleware fiber.Handler
	if s.config.EnableJWT {
		// Usar middleware de autenticação opcional para permitir acesso sem token se configurado
		authMiddleware = s.OptionalAuthMiddleware()
	}

	// Middleware para verificar autenticação específica da entidade
	entityAuthMiddleware := s.RequireEntityAuth(entityName)

	// Aplicar middlewares nas rotas
	var middlewares []fiber.Handler
	if authMiddleware != nil {
		middlewares = append(middlewares, authMiddleware)
	}
	middlewares = append(middlewares, entityAuthMiddleware)

	// Rota para coleção de entidades (GET, POST)
	getHandlers := append(middlewares, s.handleEntityCollection)
	s.router.Get(prefix+"/"+entityName, getHandlers[0], getHandlers[1:]...)

	postHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "POST"), s.handleEntityCollection)
	s.router.Post(prefix+"/"+entityName, postHandlers[0], postHandlers[1:]...)

	// Rota para entidade individual (GET, PUT, PATCH, DELETE)
	// Usando padrão wildcard para capturar URLs como /odata/FabTarefa(53)
	getByIdHandlers := append(middlewares, s.handleEntityById)
	s.router.Get(prefix+"/"+entityName+"(*)", getByIdHandlers[0], getByIdHandlers[1:]...)

	putHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PUT"), s.handleEntityById)
	s.router.Put(prefix+"/"+entityName+"(*)", putHandlers[0], putHandlers[1:]...)

	patchHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PATCH"), s.handleEntityById)
	s.router.Patch(prefix+"/"+entityName+"(*)", patchHandlers[0], patchHandlers[1:]...)

	deleteHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "DELETE"), s.handleEntityById)
	s.router.Delete(prefix+"/"+entityName+"(*)", deleteHandlers[0], deleteHandlers[1:]...)

	// Rota para count da coleção
	countHandlers := append(middlewares, s.handleEntityCount)
	s.router.Get(prefix+"/"+entityName+"/$count", countHandlers[0], countHandlers[1:]...)

	// Rota OPTIONS para CORS se habilitado
	if s.config.EnableCORS {
		s.router.Options(prefix+"/"+entityName, s.handleOptions)
		s.router.Options(prefix+"/"+entityName+"(*)", s.handleOptions)
	}
}

// setupEntityRoutesWithAuth configura as rotas para uma entidade com autenticação customizada
func (s *Server) setupEntityRoutesWithAuth(entityName string, auth AuthProvider, readOnly bool) {
	prefix := s.config.RoutePrefix

	// Usa o middleware de autenticação customizado
	authMiddleware := AuthMiddleware(auth)

	// Criar middleware de readonly se necessário
	var readOnlyMiddleware fiber.Handler
	if readOnly {
		readOnlyMiddleware = func(c fiber.Ctx) error {
			method := c.Method()
			if method != "GET" && method != "OPTIONS" {
				return fiber.NewError(fiber.StatusForbidden, "Entidade "+entityName+" é apenas leitura")
			}
			return c.Next()
		}
	}

	// Rota para coleção de entidades (GET sempre permitido, POST se não readOnly)
	s.router.Get(prefix+"/"+entityName, authMiddleware, s.handleEntityCollection)

	if !readOnly {
		s.router.Post(prefix+"/"+entityName, authMiddleware, s.handleEntityCollection)
	} else if readOnlyMiddleware != nil {
		s.router.Post(prefix+"/"+entityName, authMiddleware, readOnlyMiddleware, s.handleEntityCollection)
	}

	// Rota para entidade individual (GET sempre permitido, PUT/PATCH/DELETE se não readOnly)
	s.router.Get(prefix+"/"+entityName+"(*)", authMiddleware, s.handleEntityById)

	if !readOnly {
		s.router.Put(prefix+"/"+entityName+"(*)", authMiddleware, s.handleEntityById)
		s.router.Patch(prefix+"/"+entityName+"(*)", authMiddleware, s.handleEntityById)
		s.router.Delete(prefix+"/"+entityName+"(*)", authMiddleware, s.handleEntityById)
	} else if readOnlyMiddleware != nil {
		s.router.Put(prefix+"/"+entityName+"(*)", authMiddleware, readOnlyMiddleware, s.handleEntityById)
		s.router.Patch(prefix+"/"+entityName+"(*)", authMiddleware, readOnlyMiddleware, s.handleEntityById)
		s.router.Delete(prefix+"/"+entityName+"(*)", authMiddleware, readOnlyMiddleware, s.handleEntityById)
	}

	// Rota para count da coleção
	s.router.Get(prefix+"/"+entityName+"/$count", authMiddleware, s.handleEntityCount)

	// Rota OPTIONS para CORS se habilitado
	if s.config.EnableCORS {
		s.router.Options(prefix+"/"+entityName, s.handleOptions)
		s.router.Options(prefix+"/"+entityName+"(*)", s.handleOptions)
	}
}
