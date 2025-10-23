package odata

import (
	"github.com/fitlcarlos/go-data/pkg/auth"
	"github.com/fitlcarlos/go-data/pkg/auth/basic"
	"github.com/fitlcarlos/go-data/pkg/auth/jwt"
	"github.com/gofiber/fiber/v3"
)

// =======================================================================================
// COMPATIBILIDADE RETROATIVA
// =======================================================================================
// NOTA: Os tipos e funções abaixo são re-exports para compatibilidade retroativa.
// É recomendado usar os pacotes pkg/auth, pkg/auth/jwt e pkg/auth/basic diretamente.
// =======================================================================================

// AuthProvider interface para provedores de autenticação
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.AuthProvider
type AuthProvider = auth.AuthProvider

// UserIdentity representa a identidade do usuário autenticado
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.UserIdentity
type UserIdentity = auth.UserIdentity

// JWTConfig configurações para JWT
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/jwt.JWTConfig
type JWTConfig = jwt.JWTConfig

// JWTClaims representa os claims do token JWT
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/jwt.JWTClaims
type JWTClaims = jwt.JWTClaims

// JwtAuth implementação de AuthProvider usando JWT
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/jwt.JwtAuth
type JwtAuth = jwt.JwtAuth

// BasicAuthConfig configurações para Basic Authentication
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/basic.BasicAuthConfig
type BasicAuthConfig = basic.BasicAuthConfig

// BasicAuth implementação de AuthProvider usando Basic Authentication
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/basic.BasicAuth
type BasicAuth = basic.BasicAuth

// UserValidator função que valida credenciais e retorna UserIdentity
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/basic.UserValidator
type UserValidator = basic.UserValidator

// Constants
const (
	// UserContextKey chave para armazenar usuário no contexto
	// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.UserContextKey
	UserContextKey = auth.UserContextKey
)

// DefaultJWTConfig retorna configuração padrão para JWT
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/jwt.DefaultJWTConfig
func DefaultJWTConfig() *jwt.JWTConfig {
	return jwt.DefaultJWTConfig()
}

// NewJwtAuth cria uma nova instância de JwtAuth
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/jwt.NewJwtAuth
func NewJwtAuth(config *jwt.JWTConfig) *jwt.JwtAuth {
	return jwt.NewJwtAuth(config)
}

// NewBasicAuth cria uma nova instância de BasicAuth
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/basic.NewBasicAuth
func NewBasicAuth(config *basic.BasicAuthConfig, validator basic.UserValidator) *basic.BasicAuth {
	return basic.NewBasicAuth(config, validator)
}

// AuthMiddleware cria um middleware de autenticação obrigatória usando AuthProvider
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.AuthMiddleware
func AuthMiddleware(authProvider auth.AuthProvider) fiber.Handler {
	return auth.AuthMiddleware(authProvider)
}

// OptionalAuthMiddleware cria um middleware de autenticação opcional usando AuthProvider
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.OptionalAuthMiddleware
func OptionalAuthMiddleware(authProvider auth.AuthProvider) fiber.Handler {
	return auth.OptionalAuthMiddleware(authProvider)
}

// RequireAuth middleware que requer autenticação
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.RequireAuth
func RequireAuth() fiber.Handler {
	return auth.RequireAuth()
}

// RequireRole middleware que requer uma role específica
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.RequireRole
func RequireRole(role string) fiber.Handler {
	return auth.RequireRole(role)
}

// RequireAnyRole middleware que requer pelo menos uma das roles
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.RequireAnyRole
func RequireAnyRole(roles ...string) fiber.Handler {
	return auth.RequireAnyRole(roles...)
}

// RequireScope middleware que requer um scope específico
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.RequireScope
func RequireScope(scope string) fiber.Handler {
	return auth.RequireScope(scope)
}

// RequireAnyScope middleware que requer pelo menos um dos scopes
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.RequireAnyScope
func RequireAnyScope(scopes ...string) fiber.Handler {
	return auth.RequireAnyScope(scopes...)
}

// RequireAdmin middleware que requer privilégios de administrador
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.RequireAdmin
func RequireAdmin() fiber.Handler {
	return auth.RequireAdmin()
}

// GetCurrentUser obtém o usuário atual do contexto
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.GetCurrentUser
func GetCurrentUser(c fiber.Ctx) *auth.UserIdentity {
	return auth.GetCurrentUser(c)
}

// GetUserFromContext obtém o usuário do contexto (alias para GetCurrentUser)
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.GetUserFromContext
func GetUserFromContext(c fiber.Ctx) *auth.UserIdentity {
	return auth.GetUserFromContext(c)
}

// IsAuthenticated verifica se o usuário está autenticado
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.IsAuthenticated
func IsAuthenticated(c fiber.Ctx) bool {
	return auth.IsAuthenticated(c)
}

// HasRole verifica se o usuário atual possui uma role específica
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.HasRole
func HasRole(c fiber.Ctx, role string) bool {
	return auth.HasRole(c, role)
}

// HasScope verifica se o usuário atual possui um scope específico
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.HasScope
func HasScope(c fiber.Ctx, scope string) bool {
	return auth.HasScope(c, scope)
}

// IsAdmin verifica se o usuário atual é administrador
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth.IsAdmin
func IsAdmin(c fiber.Ctx) bool {
	return auth.IsAdmin(c)
}

// BasicAuthMiddleware middleware de autenticação Basic
// Deprecated: Use github.com/fitlcarlos/go-data/pkg/auth/basic.BasicAuthMiddleware
func BasicAuthMiddleware(basicAuth *basic.BasicAuth) fiber.Handler {
	return basic.BasicAuthMiddleware(basicAuth)
}

// =======================================================================================
// ENTITY OPTIONS SYSTEM
// =======================================================================================

// EntityConfig configuração de uma entidade
type EntityConfig struct {
	Name     string
	Entity   interface{}
	Auth     auth.AuthProvider
	ReadOnly bool
	// Futuras opções podem ser adicionadas aqui
}

// EntityOption função que modifica a configuração de uma entidade
type EntityOption func(*EntityConfig)

// WithAuth configura autenticação para uma entidade
func WithAuth(authProvider auth.AuthProvider) EntityOption {
	return func(config *EntityConfig) {
		config.Auth = authProvider
	}
}

// WithReadOnly configura uma entidade como somente leitura
func WithReadOnly(readOnly bool) EntityOption {
	return func(config *EntityConfig) {
		config.ReadOnly = readOnly
	}
}
