package basic

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/fitlcarlos/go-data/pkg/auth"
	"github.com/gofiber/fiber/v3"
)

// UserValidator função que valida credenciais e retorna UserIdentity
type UserValidator func(username, password string) (*auth.UserIdentity, error)

// UserValidatorWithContext função que valida credenciais com acesso ao contexto enriquecido
// Fornece acesso ao ObjectManager, Connection, Provider e Pool durante validação
type UserValidatorWithContext func(ctx AuthContext, username, password string) (*auth.UserIdentity, error)

// AuthContext representa o contexto enriquecido (importado para evitar import cycle)
// A implementação real está em pkg/odata/auth_context.go
type AuthContext interface {
	GetManager() interface{}
	GetConnection() interface{}
	GetProvider() interface{}
	GetPool() interface{}
	CreateObjectManager() interface{}
	GetTenantID() string
	Body() []byte
	GetHeader(key string) string
	IP() string
	Query(key string) string
}

// BasicAuth implementação de AuthProvider usando Basic Authentication
type BasicAuth struct {
	config *BasicAuthConfig

	// UserValidator função customizável para validar usuário/senha
	// Deve retornar UserIdentity se credenciais forem válidas
	UserValidator UserValidator

	// UserValidatorWithContext função que recebe contexto enriquecido (opcional)
	// Se definida, tem prioridade sobre UserValidator
	UserValidatorWithContext UserValidatorWithContext

	// TokenExtractor função customizável para extrair credenciais do contexto
	// Se nil, usa extração padrão (Basic Auth header)
	TokenExtractor func(fiber.Ctx) string

	// server referência ao servidor para criar AuthContext (opcional)
	server interface{}
}

// NewBasicAuth cria uma nova instância de BasicAuth
func NewBasicAuth(config *BasicAuthConfig, validator UserValidator) *BasicAuth {
	if config == nil {
		config = DefaultBasicAuthConfig()
	}

	if validator == nil {
		panic("BasicAuth requires a UserValidator function")
	}

	auth := &BasicAuth{
		config:        config,
		UserValidator: validator,
	}

	// Define extrator padrão
	auth.TokenExtractor = auth.DefaultExtractToken

	return auth
}

// NewBasicAuthWithContext cria BasicAuth com suporte a contexto enriquecido
// O server deve ser do tipo *odata.Server para criar AuthContext
func NewBasicAuthWithContext(server interface{}, config *BasicAuthConfig, validator UserValidatorWithContext) *BasicAuth {
	if config == nil {
		config = DefaultBasicAuthConfig()
	}

	if validator == nil {
		panic("BasicAuth requires a UserValidatorWithContext function")
	}

	auth := &BasicAuth{
		config:                   config,
		UserValidatorWithContext: validator,
		server:                   server,
	}

	// Define extrator padrão
	auth.TokenExtractor = auth.DefaultExtractToken

	return auth
}

// ValidateToken implementa AuthProvider.ValidateToken
// Para Basic Auth, o "token" são as credenciais base64
func (b *BasicAuth) ValidateToken(token string) (*auth.UserIdentity, error) {
	// Decodifica credenciais base64
	credentials, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.New("credenciais inválidas")
	}

	// Separa username:password
	parts := strings.SplitN(string(credentials), ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("formato de credenciais inválido")
	}

	username := parts[0]
	password := parts[1]

	// Valida usando a função customizada
	if b.UserValidator == nil {
		return nil, errors.New("user validator não configurado")
	}

	return b.UserValidator(username, password)
}

// GenerateToken implementa AuthProvider.GenerateToken
// Para Basic Auth, não geramos tokens (stateless)
func (b *BasicAuth) GenerateToken(user *auth.UserIdentity) (string, error) {
	return "", errors.New("Basic Auth não gera tokens - use credenciais no header")
}

// ExtractToken implementa AuthProvider.ExtractToken
func (b *BasicAuth) ExtractToken(c fiber.Ctx) string {
	if b.TokenExtractor != nil {
		return b.TokenExtractor(c)
	}
	return b.DefaultExtractToken(c)
}

// DefaultExtractToken extração padrão de credenciais do header Authorization
func (b *BasicAuth) DefaultExtractToken(c fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		return ""
	}

	return strings.TrimPrefix(authHeader, "Basic ")
}

// GetRealm retorna o realm configurado
func (b *BasicAuth) GetRealm() string {
	return b.config.Realm
}

// GetConfig retorna a configuração
func (b *BasicAuth) GetConfig() *BasicAuthConfig {
	return b.config
}

// SendUnauthorizedResponse envia resposta 401 com WWW-Authenticate header
func (b *BasicAuth) SendUnauthorizedResponse(c fiber.Ctx) error {
	c.Set("WWW-Authenticate", `Basic realm="`+b.config.Realm+`"`)
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Autenticação requerida",
	})
}

// BasicAuthMiddleware middleware de autenticação Basic
// Similar ao AuthMiddleware mas envia WWW-Authenticate header
func BasicAuthMiddleware(basicAuth *BasicAuth) fiber.Handler {
	return func(c fiber.Ctx) error {
		if basicAuth == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Auth provider não configurado")
		}

		token := basicAuth.ExtractToken(c)
		if token == "" {
			return basicAuth.SendUnauthorizedResponse(c)
		}

		var user *auth.UserIdentity
		var err error

		// Decodifica credenciais base64
		credentials, decodeErr := base64.StdEncoding.DecodeString(token)
		if decodeErr != nil {
			return basicAuth.SendUnauthorizedResponse(c)
		}

		parts := strings.SplitN(string(credentials), ":", 2)
		if len(parts) != 2 {
			return basicAuth.SendUnauthorizedResponse(c)
		}

		username := parts[0]
		password := parts[1]

		// Usa validator com contexto se disponível
		if basicAuth.UserValidatorWithContext != nil && basicAuth.server != nil {
			// Cria AuthContext através do server
			// Usando type assertion para o método createAuthContext
			if serverWithAuth, ok := basicAuth.server.(interface {
				createAuthContext(fiber.Ctx) AuthContext
			}); ok {
				authCtx := serverWithAuth.createAuthContext(c)
				user, err = basicAuth.UserValidatorWithContext(authCtx, username, password)
			} else {
				// Fallback se o server não tiver o método
				if basicAuth.UserValidator != nil {
					user, err = basicAuth.UserValidator(username, password)
				} else {
					return basicAuth.SendUnauthorizedResponse(c)
				}
			}
		} else if basicAuth.UserValidator != nil {
			// Usa validator tradicional
			user, err = basicAuth.UserValidator(username, password)
		} else {
			return basicAuth.SendUnauthorizedResponse(c)
		}

		if err != nil || user == nil {
			return basicAuth.SendUnauthorizedResponse(c)
		}

		// Armazenar usuário no contexto
		c.Locals(auth.UserContextKey, user)
		return c.Next()
	}
}
