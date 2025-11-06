package odata

import (
	"github.com/gofiber/fiber/v3"
)

// =======================================================================================
// ENTITY OPTIONS SYSTEM
// =======================================================================================

// EntityConfig configuração de uma entidade
type EntityConfig struct {
	Name        string
	Entity      interface{}
	Middlewares []fiber.Handler // Middlewares aplicados às rotas da entidade
	ReadOnly    bool
	Permissions []string // GET, POST, PUT, DELETE, PATCH - se vazio, permite todos
}

// EntityOption função que modifica a configuração de uma entidade
type EntityOption func(*EntityConfig)

// WithMiddleware adiciona middlewares à entidade (ex: JWT, Basic Auth, etc)
func WithMiddleware(middlewares ...fiber.Handler) EntityOption {
	return func(config *EntityConfig) {
		config.Middlewares = append(config.Middlewares, middlewares...)
	}
}

// WithReadOnly configura uma entidade como somente leitura
func WithReadOnly(readOnly bool) EntityOption {
	return func(config *EntityConfig) {
		config.ReadOnly = readOnly
	}
}

// WithPermissions define quais operações HTTP são permitidas na entidade
// Exemplo: WithPermissions("GET", "POST") - permite apenas GET e POST
// Se não especificado, permite todas as operações
func WithPermissions(permissions ...string) EntityOption {
	return func(config *EntityConfig) {
		config.Permissions = permissions
	}
}

// GetCurrentUser obtém o usuário atual do contexto
func GetCurrentUser(c fiber.Ctx) *UserIdentity {
	if user := c.Locals(UserContextKey); user != nil {
		if u, ok := user.(*UserIdentity); ok {
			return u
		}
	}
	return nil
}

// IsAuthenticated verifica se o usuário está autenticado
func IsAuthenticated(c fiber.Ctx) bool {
	return GetCurrentUser(c) != nil
}

// HasRole verifica se o usuário atual possui uma role específica
func HasRole(c fiber.Ctx, role string) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}
	return user.HasRole(role)
}

// HasScope verifica se o usuário atual possui um scope específico
func HasScope(c fiber.Ctx, scope string) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}
	return user.HasScope(scope)
}

// IsAdmin verifica se o usuário atual é administrador
func IsAdmin(c fiber.Ctx) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}
	return user.Admin
}
