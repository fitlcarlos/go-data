package auth

import (
	"github.com/gofiber/fiber/v3"
)

// AuthProvider interface para provedores de autenticação
// Permite que o usuário defina sua própria lógica de autenticação
type AuthProvider interface {
	// ValidateToken valida um token e retorna a identidade do usuário
	ValidateToken(token string) (*UserIdentity, error)

	// GenerateToken gera um token para a identidade do usuário
	GenerateToken(user *UserIdentity) (string, error)

	// ExtractToken extrai o token do contexto Fiber
	ExtractToken(c fiber.Ctx) string
}

// UserIdentity representa a identidade do usuário autenticado
type UserIdentity struct {
	Username string                 `json:"username"`
	Roles    []string               `json:"roles"`
	Scopes   []string               `json:"scopes"`
	Admin    bool                   `json:"admin"`
	Custom   map[string]interface{} `json:"custom"`
}

// HasRole verifica se o usuário possui uma role específica
func (u *UserIdentity) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasScope verifica se o usuário possui um scope específico
func (u *UserIdentity) HasScope(scope string) bool {
	for _, s := range u.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyRole verifica se o usuário possui pelo menos uma das roles
func (u *UserIdentity) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAnyScope verifica se o usuário possui pelo menos um dos scopes
func (u *UserIdentity) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if u.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsAdmin verifica se o usuário é administrador
func (u *UserIdentity) IsAdmin() bool {
	return u.Admin
}

// GetCustomClaim obtém um claim customizado
func (u *UserIdentity) GetCustomClaim(key string) (interface{}, bool) {
	if u.Custom == nil {
		return nil, false
	}
	value, exists := u.Custom[key]
	return value, exists
}

// Constants
const (
	// UserContextKey chave para armazenar usuário no contexto
	UserContextKey = "user"
)

// GetCurrentUser obtém o usuário autenticado do contexto
func GetCurrentUser(c fiber.Ctx) *UserIdentity {
	user := c.Locals(UserContextKey)
	if user == nil {
		return nil
	}
	if u, ok := user.(*UserIdentity); ok {
		return u
	}
	return nil
}

// GetUserFromContext obtém o usuário do contexto (alias para GetCurrentUser)
func GetUserFromContext(c fiber.Ctx) *UserIdentity {
	return GetCurrentUser(c)
}
