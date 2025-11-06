package odata

import (
	"database/sql"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v3/middleware/recover"
)

// =======================================================================================
// SETUP DE MIDDLEWARES
// =======================================================================================

// setupMultiTenantMiddlewares configura middlewares específicos para multi-tenant
func (s *Server) setupMultiTenantMiddlewares() {
	// Middleware de identificação de tenant (deve ser o primeiro)
	s.router.Use(s.TenantMiddleware())

	// Middleware de informações do tenant
	s.router.Use(s.TenantInfo())

	// Middleware de conexão de banco de dados (transparente)
	s.router.Use(s.DatabaseMiddleware())

	// Demais middlewares...
	if s.config.EnableCORS {
		s.router.Use(cors.New(cors.Config{
			AllowOrigins:     s.config.AllowedOrigins,
			AllowMethods:     s.config.AllowedMethods,
			AllowHeaders:     s.config.AllowedHeaders,
			ExposeHeaders:    s.config.ExposedHeaders,
			AllowCredentials: s.config.AllowCredentials,
		}))
	}

	if s.config.EnableLogging {
		s.router.Use(fiberlogger.New(fiberlogger.Config{
			Format: "${time} ${method} ${path} ${status} ${latency} [${locals:tenant_id}]\n",
			Output: os.Stdout,
		}))
	}

	s.router.Use(fiberrecover.New())

	// Middleware que injeta o servidor no contexto Fiber
	s.router.Use(func(c fiber.Ctx) error {
		c.Locals("odata_server", s)
		return c.Next()
	})

	// Middleware de rate limit se habilitado
	if s.rateLimiter != nil {
		s.router.Use(s.RateLimitMiddleware())
	}
}

// =======================================================================================
// AUTHENTICATION MIDDLEWARES (DEPRECATED - removidos - use NewRouterJWTAuth)
// =======================================================================================

// =======================================================================================
// ENTITY-SPECIFIC MIDDLEWARES
// =======================================================================================

// RequireEntityAuth aplica middleware de autenticação baseado na configuração da entidade
func (s *Server) RequireEntityAuth(entityName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Obter configuração da entidade
		authConfig, exists := s.GetEntityAuth(entityName)
		if !exists {
			// Se não há configuração específica, permitir acesso
			return c.Next()
		}

		// Verificar se autenticação é necessária
		if authConfig.RequireAuth {
			user := GetCurrentUser(c)
			if user == nil {
				return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida para acessar "+entityName)
			}

			// Verificar se é admin
			if authConfig.RequireAdmin && !user.Admin {
				return fiber.NewError(fiber.StatusForbidden, "Privilégios de administrador requeridos para acessar "+entityName)
			}

			// Verificar roles
			if len(authConfig.RequiredRoles) > 0 && !user.HasAnyRole(authConfig.RequiredRoles...) {
				return fiber.NewError(fiber.StatusForbidden, "Role necessária para acessar "+entityName)
			}

			// Verificar scopes
			if len(authConfig.RequiredScopes) > 0 && !user.HasAnyScope(authConfig.RequiredScopes...) {
				return fiber.NewError(fiber.StatusForbidden, "Scope necessário para acessar "+entityName)
			}
		}

		return c.Next()
	}
}

// CheckEntityReadOnly verifica se a entidade é apenas leitura
func (s *Server) CheckEntityReadOnly(entityName string, method string) fiber.Handler {
	return func(c fiber.Ctx) error {
		authConfig, exists := s.GetEntityAuth(entityName)
		if !exists {
			return c.Next()
		}

		// Se é read-only e método não é GET, bloquear
		if authConfig.ReadOnly && method != "GET" {
			return fiber.NewError(fiber.StatusForbidden, "Entidade "+entityName+" é apenas leitura")
		}

		return c.Next()
	}
}

// =======================================================================================
// DATABASE MIDDLEWARE
// =======================================================================================

// DatabaseMiddleware middleware que adiciona conexão de banco no contexto
func (s *Server) DatabaseMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Middleware apenas para garantir que o provider está disponível
		// A conexão não é armazenada no contexto para evitar garbage collection
		s.getCurrentProvider(c)
		return c.Next()
	}
}

// GetDBFromContext obtém a conexão de banco de dados do contexto
// DEPRECADO: Use GetConnection() do context_helpers.go que retorna o pool global
func GetDBFromContext(c fiber.Ctx) *sql.DB {
	// Retorna o pool global diretamente
	return GetConnection(c)
}
