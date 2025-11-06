package odata

import (
	"database/sql"

	"github.com/gofiber/fiber/v3"
)

// getServerFromContext recupera o servidor armazenado no contexto Fiber
func getServerFromContext(c fiber.Ctx) *Server {
	if srv := c.Locals("odata_server"); srv != nil {
		if server, ok := srv.(*Server); ok {
			return server
		}
	}
	return nil
}

// GetObjectManager retorna o ObjectManager do fiber context
// Útil para endpoints customizados (JWT, handlers manuais)
func GetObjectManager(c fiber.Ctx) *ObjectManager {
	server := getServerFromContext(c)
	if server == nil {
		return nil
	}
	provider := server.getCurrentProvider(c)
	return NewObjectManager(provider, c.Context())
}

// GetConnection retorna a conexão SQL do fiber context
func GetConnection(c fiber.Ctx) *sql.DB {
	server := getServerFromContext(c)
	if server == nil {
		return nil
	}
	provider := server.getCurrentProvider(c)
	if provider == nil {
		return nil
	}
	return provider.GetConnection()
}

// GetProvider retorna o DatabaseProvider do fiber context
func GetProvider(c fiber.Ctx) DatabaseProvider {
	server := getServerFromContext(c)
	if server == nil {
		return nil
	}
	return server.getCurrentProvider(c)
}

// GetConnectionPool retorna o pool multi-tenant
func GetConnectionPool(c fiber.Ctx) *MultiTenantProviderPool {
	server := getServerFromContext(c)
	if server == nil {
		return nil
	}
	return server.multiTenantPool
}

// CreateObjectManager cria um novo ObjectManager
func CreateObjectManager(c fiber.Ctx) *ObjectManager {
	server := getServerFromContext(c)
	if server == nil {
		return nil
	}
	provider := server.getCurrentProvider(c)
	if provider == nil {
		return nil
	}
	return NewObjectManager(provider, c.Context())
}
