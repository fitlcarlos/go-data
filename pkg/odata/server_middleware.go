package odata

import (
	"database/sql"
	"os"
	"strings"

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

	// Middleware de rate limit se habilitado
	if s.rateLimiter != nil {
		s.router.Use(s.RateLimitMiddleware())
	}
}

// =======================================================================================
// AUTHENTICATION MIDDLEWARES (DEPRECATED - mantidos para compatibilidade)
// =======================================================================================

// AuthMiddleware middleware de autenticação obrigatória (DEPRECATED: use auth.AuthMiddleware)
// Mantido para compatibilidade retroativa com código legado
func (s *Server) AuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if s.jwtService == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "JWT não configurado")
		}

		token := extractTokenFromContext(c)
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Token de acesso requerido")
		}

		user, err := s.jwtService.ValidateAndExtractUser(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Token inválido")
		}

		// Armazenar usuário no contexto
		c.Locals(UserContextKey, user)
		return c.Next()
	}
}

// OptionalAuthMiddleware middleware de autenticação opcional (DEPRECATED: use auth.OptionalAuthMiddleware)
// Mantido para compatibilidade retroativa com código legado
func (s *Server) OptionalAuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if s.jwtService == nil {
			return c.Next()
		}

		token := extractTokenFromContext(c)
		if token == "" {
			return c.Next()
		}

		user, err := s.jwtService.ValidateAndExtractUser(token)
		if err != nil {
			// Token inválido, mas não bloqueia a requisição
			return c.Next()
		}

		// Armazenar usuário no contexto
		c.Locals(UserContextKey, user)
		return c.Next()
	}
}

// extractTokenFromContext extrai o token do cabeçalho Authorization
func extractTokenFromContext(c fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}

	return strings.TrimPrefix(authHeader, "Bearer ")
}

// =======================================================================================
// ENTITY-SPECIFIC MIDDLEWARES
// =======================================================================================

// RequireEntityAuth aplica middleware de autenticação baseado na configuração da entidade
func (s *Server) RequireEntityAuth(entityName string) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Se JWT não estiver habilitado, pular verificação
		if !s.config.EnableJWT {
			return c.Next()
		}

		// Obter configuração da entidade
		authConfig, exists := s.GetEntityAuth(entityName)
		if !exists {
			// Se não há configuração específica, usar configuração global
			if s.config.RequireAuth {
				return RequireAuth()(c)
			}
			return c.Next()
		}

		// Verificar se autenticação é necessária
		if authConfig.RequireAuth {
			user := GetCurrentUser(c)
			if user == nil {
				return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida para acessar "+entityName)
			}

			// Verificar se é admin
			if authConfig.RequireAdmin && !user.IsAdmin() {
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
		// Obter provider (já existe a lógica)
		provider := s.getCurrentProvider(c)
		if provider != nil {
			// Obter conexão do pool
			if conn := provider.GetConnection(); conn != nil {
				// Armazenar conexão no contexto
				c.Locals("db_conn", conn)

				// Garantir fechamento da conexão ao final da requisição
				defer func() {
					// A conexão será fechada automaticamente pelo pool
					// quando não estiver mais em uso
				}()
			}
		}

		return c.Next()
	}
}

// GetDBFromContext obtém a conexão de banco de dados do contexto
func GetDBFromContext(c fiber.Ctx) *sql.DB {
	if conn, ok := c.Locals("db_conn").(*sql.DB); ok {
		return conn
	}
	return nil
}
