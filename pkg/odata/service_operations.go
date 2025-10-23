package odata

import (
	"database/sql"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// ServiceContext representa o contexto de uma operação de service
// Equivale funcionalmente ao TXDataOperationContext do XData
type ServiceContext struct {
	Manager      *ObjectManager // Equivale ao TXDataOperationContext.Current.GetManager()
	FiberContext fiber.Ctx      // Contexto do Fiber (já tem TenantID via GetCurrentTenant())
	User         *UserIdentity  // Usuário autenticado (só se JWT habilitado)

	// Campos internos (privados)
	provider DatabaseProvider // Provider do tenant corrente
	server   *Server          // Referência ao servidor
}

// ServiceHandler representa uma função handler para service operations
type ServiceHandler func(ctx *ServiceContext) error

// Service representa uma operação de service
type Service struct {
	Name     string
	Method   string
	Endpoint string
	Handler  ServiceHandler
	Auth     *AuthConfig
}

// AuthConfig configurações de autenticação para services
type AuthConfig struct {
	Required bool
	Roles    []string
	Scopes   []string
}

// Service registra um service operation sem autenticação
func (s *Server) Service(method, endpoint string, handler ServiceHandler) {
	s.ServiceWithAuth(method, endpoint, handler, false)
}

// ServiceWithAuth registra um service operation com controle de autenticação
func (s *Server) ServiceWithAuth(method, endpoint string, handler ServiceHandler, requireAuth bool) {
	route := fmt.Sprintf("%s%s", s.config.RoutePrefix, endpoint)

	s.router.Add([]string{method}, route, func(c fiber.Ctx) error {
		// Obter provider
		provider := s.getCurrentProvider(c)

		// Criar ServiceContext com todos os campos
		serviceContext := &ServiceContext{
			Manager:      NewObjectManager(provider, c.Context()),
			FiberContext: c,
			provider:     provider,
			server:       s,
		}

		// Adicionar User apenas se JWT habilitado E auth requerido
		if s.config.EnableJWT && requireAuth {
			user := GetCurrentUser(c)
			if user == nil {
				return c.Status(401).JSON(map[string]string{"error": "Unauthorized"})
			}
			serviceContext.User = user
		}

		// Executar handler do service
		return handler(serviceContext)
	})
}

// ServiceWithRoles registra um service operation com verificação de roles
func (s *Server) ServiceWithRoles(method, endpoint string, handler ServiceHandler, roles ...string) {
	route := fmt.Sprintf("%s%s", s.config.RoutePrefix, endpoint)

	s.router.Add([]string{method}, route, func(c fiber.Ctx) error {
		// Obter provider
		provider := s.getCurrentProvider(c)

		serviceContext := &ServiceContext{
			Manager:      NewObjectManager(provider, c.Context()),
			FiberContext: c,
			provider:     provider,
			server:       s,
		}

		// Verificar auth e roles se JWT habilitado
		if s.config.EnableJWT && len(roles) > 0 {
			user := GetCurrentUser(c)
			if user == nil || !user.HasAnyRole(roles...) {
				return c.Status(401).JSON(map[string]string{"error": "Unauthorized"})
			}
			serviceContext.User = user
		}

		return handler(serviceContext)
	})
}

// ServiceGroup representa um grupo de services para organização
type ServiceGroup struct {
	server *Server
	prefix string
}

// ServiceGroup cria um grupo de services
func (s *Server) ServiceGroup(prefix string) *ServiceGroup {
	return &ServiceGroup{
		server: s,
		prefix: prefix,
	}
}

// Service registra um service no grupo
func (sg *ServiceGroup) Service(method, name string, handler ServiceHandler) {
	endpoint := fmt.Sprintf("/Service/%s/%s", sg.prefix, name)
	sg.server.Service(method, endpoint, handler)
}

// ServiceWithAuth registra um service com auth no grupo
func (sg *ServiceGroup) ServiceWithAuth(method, name string, handler ServiceHandler, requireAuth bool) {
	endpoint := fmt.Sprintf("/Service/%s/%s", sg.prefix, name)
	sg.server.ServiceWithAuth(method, endpoint, handler, requireAuth)
}

// ServiceWithRoles registra um service com roles no grupo
func (sg *ServiceGroup) ServiceWithRoles(method, name string, handler ServiceHandler, roles ...string) {
	endpoint := fmt.Sprintf("/Service/%s/%s", sg.prefix, name)
	sg.server.ServiceWithRoles(method, endpoint, handler, roles...)
}

// Métodos utilitários do ServiceContext

// GetManager retorna o ObjectManager (equivale ao TXDataOperationContext.Current.GetManager())
func (ctx *ServiceContext) GetManager() *ObjectManager {
	return ctx.Manager
}

// GetUser retorna o usuário autenticado
func (ctx *ServiceContext) GetUser() *UserIdentity {
	return ctx.User
}

// GetTenantID retorna o tenant atual
func (ctx *ServiceContext) GetTenantID() string {
	return GetCurrentTenant(ctx.FiberContext)
}

// IsAuthenticated verifica se o usuário está autenticado
func (ctx *ServiceContext) IsAuthenticated() bool {
	return ctx.User != nil
}

// HasRole verifica se o usuário tem uma role específica
func (ctx *ServiceContext) HasRole(role string) bool {
	if ctx.User == nil {
		return false
	}
	return ctx.User.HasRole(role)
}

// HasAnyRole verifica se o usuário tem pelo menos uma das roles
func (ctx *ServiceContext) HasAnyRole(roles ...string) bool {
	if ctx.User == nil {
		return false
	}
	return ctx.User.HasAnyRole(roles...)
}

// IsAdmin verifica se o usuário é administrador
func (ctx *ServiceContext) IsAdmin() bool {
	if ctx.User == nil {
		return false
	}
	return ctx.User.IsAdmin()
}

// GetConnection retorna a conexão SQL do contexto corrente
// Útil para queries SQL diretas ou transações manuais
func (ctx *ServiceContext) GetConnection() *sql.DB {
	if ctx.provider == nil {
		return nil
	}
	return ctx.provider.GetConnection()
}

// GetProvider retorna o DatabaseProvider do tenant corrente
func (ctx *ServiceContext) GetProvider() DatabaseProvider {
	return ctx.provider
}

// GetPool retorna o pool de conexões multi-tenant (se habilitado)
// Retorna nil se multi-tenant não estiver configurado
func (ctx *ServiceContext) GetPool() *MultiTenantProviderPool {
	if ctx.server == nil {
		return nil
	}
	return ctx.server.multiTenantPool
}

// CreateObjectManager cria um novo ObjectManager para operações isoladas
// Útil para transações paralelas ou contextos de persistência separados
func (ctx *ServiceContext) CreateObjectManager() *ObjectManager {
	if ctx.provider == nil {
		return nil
	}
	return NewObjectManager(ctx.provider, ctx.FiberContext.Context())
}

// QueryParams retorna os parâmetros da query string
func (ctx *ServiceContext) QueryParams() map[string]string {
	params := make(map[string]string)
	ctx.FiberContext.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		params[string(key)] = string(value)
	})
	return params
}

// Query retorna um parâmetro específico da query string
func (ctx *ServiceContext) Query(key string) string {
	return ctx.FiberContext.Query(key)
}

// Body retorna o corpo da requisição
func (ctx *ServiceContext) Body() []byte {
	return ctx.FiberContext.Body()
}

// JSON retorna JSON da requisição
func (ctx *ServiceContext) JSON(v interface{}) error {
	return ctx.FiberContext.JSON(v)
}

// Status define o status code da resposta
func (ctx *ServiceContext) Status(code int) *ServiceContext {
	ctx.FiberContext.Status(code)
	return ctx
}

// SetHeader define um header da resposta
func (ctx *ServiceContext) SetHeader(key, value string) *ServiceContext {
	ctx.FiberContext.Set(key, value)
	return ctx
}

// GetHeader obtém um header da requisição
func (ctx *ServiceContext) GetHeader(key string) string {
	return ctx.FiberContext.Get(key)
}

// Params retorna os parâmetros da rota
func (ctx *ServiceContext) Params() map[string]string {
	params := make(map[string]string)
	// Para Fiber v3, usar método mais simples
	route := ctx.FiberContext.Route()
	if route != nil {
		for i, param := range route.Params {
			if i%2 == 0 && i+1 < len(route.Params) {
				params[param] = route.Params[i+1]
			}
		}
	}
	return params
}

// Param retorna um parâmetro específico da rota
func (ctx *ServiceContext) Param(key string) string {
	return ctx.FiberContext.Params(key)
}
