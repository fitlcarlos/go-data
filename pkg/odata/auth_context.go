package odata

import (
	"database/sql"

	"github.com/gofiber/fiber/v3"
)

// AuthContext fornece contexto enriquecido para autenticação
// Similar ao ServiceContext, mas específico para handlers de autenticação
// Reaproveita os context_helpers existentes como fachada orientada a objetos
//
// USO:
// Para usar AuthContext, implemente a interface ContextAuthenticator:
//
//	type MyAuthenticator struct {}
//
//	// Método de login (com validação de senha)
//	func (a *MyAuthenticator) AuthenticateWithContext(ctx *odata.AuthContext, username, password string) (*odata.UserIdentity, error) {
//	    manager := ctx.GetManager()
//	    conn := ctx.GetConnection()
//
//	    // Validar credenciais no banco
//	    var dbPassword string
//	    err := conn.QueryRow("SELECT password FROM users WHERE email = ?", username).Scan(&dbPassword)
//	    if err != nil || !checkPassword(password, dbPassword) {
//	        return nil, errors.New("credenciais inválidas")
//	    }
//
//	    // Retornar identidade do usuário
//	    return &odata.UserIdentity{Username: username, ...}, nil
//	}
//
//	// Método de refresh token (sem senha, apenas recarrega dados)
//	func (a *MyAuthenticator) RefreshToken(ctx *odata.AuthContext, username string) (*odata.UserIdentity, error) {
//	    // O contexto está disponível para validar no banco, mas não é obrigatório usar
//	    // Você pode simplesmente retornar os dados do token se preferir
//
//	    manager := ctx.GetManager()
//
//	    // Buscar dados atualizados do usuário (opcional - para validar se está ativo)
//	    user, err := manager.Find("Users", username)
//	    if err != nil {
//	        return nil, errors.New("usuário não encontrado")
//	    }
//
//	    // Validar se usuário ainda está ativo
//	    if !user["is_active"].(bool) {
//	        return nil, errors.New("usuário inativo")
//	    }
//
//	    return &odata.UserIdentity{
//	        Username: username,
//	        Roles:    user["roles"].([]string),
//	        // ... outros campos atualizados do banco
//	    }, nil
//	}
type AuthContext struct {
	FiberContext fiber.Ctx
	server       *Server
}

// GetManager retorna o ObjectManager para operações ORM
// Equivale ao TXDataOperationContext.Current.GetManager() do XData
func (ctx *AuthContext) GetManager() *ObjectManager {
	return GetObjectManager(ctx.FiberContext, ctx.server)
}

// GetConnection retorna a conexão SQL direta
// Útil para queries SQL customizadas ou transações manuais
func (ctx *AuthContext) GetConnection() *sql.DB {
	return GetConnection(ctx.FiberContext, ctx.server)
}

// GetProvider retorna o DatabaseProvider do tenant corrente
func (ctx *AuthContext) GetProvider() DatabaseProvider {
	return GetProvider(ctx.FiberContext, ctx.server)
}

// GetPool retorna o pool de conexões multi-tenant (se habilitado)
// Retorna nil se multi-tenant não estiver configurado
func (ctx *AuthContext) GetPool() *MultiTenantProviderPool {
	return GetConnectionPool(ctx.server)
}

// CreateObjectManager cria um novo ObjectManager isolado
// Útil para operações paralelas ou contextos de persistência separados
func (ctx *AuthContext) CreateObjectManager() *ObjectManager {
	return CreateObjectManager(ctx.FiberContext, ctx.server)
}

// GetTenantID retorna o ID do tenant corrente
func (ctx *AuthContext) GetTenantID() string {
	return GetCurrentTenant(ctx.FiberContext)
}

// Body retorna o corpo bruto da requisição
func (ctx *AuthContext) Body() []byte {
	return ctx.FiberContext.Body()
}

// GetHeader retorna o valor de um header da requisição
func (ctx *AuthContext) GetHeader(key string) string {
	return ctx.FiberContext.Get(key)
}

// IP retorna o endereço IP do cliente
func (ctx *AuthContext) IP() string {
	return ctx.FiberContext.IP()
}

// Query retorna o valor de um parâmetro de query string
func (ctx *AuthContext) Query(key string) string {
	return ctx.FiberContext.Query(key)
}
