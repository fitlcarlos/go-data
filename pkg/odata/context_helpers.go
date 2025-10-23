package odata

import (
	"database/sql"

	"github.com/gofiber/fiber/v3"
)

// GetObjectManager retorna o ObjectManager do fiber context
// Útil para endpoints customizados (JWT, handlers manuais)
func GetObjectManager(c fiber.Ctx, server *Server) *ObjectManager {
	provider := server.getCurrentProvider(c)
	return NewObjectManager(provider, c.Context())
}

// GetConnection retorna a conexão SQL do fiber context
func GetConnection(c fiber.Ctx, server *Server) *sql.DB {
	provider := server.getCurrentProvider(c)
	if provider == nil {
		return nil
	}
	return provider.GetConnection()
}

// GetProvider retorna o DatabaseProvider do fiber context
func GetProvider(c fiber.Ctx, server *Server) DatabaseProvider {
	return server.getCurrentProvider(c)
}

// GetConnectionPool retorna o pool multi-tenant
func GetConnectionPool(server *Server) *MultiTenantProviderPool {
	return server.multiTenantPool
}

// CreateObjectManager cria um novo ObjectManager
func CreateObjectManager(c fiber.Ctx, server *Server) *ObjectManager {
	provider := server.getCurrentProvider(c)
	if provider == nil {
		return nil
	}
	return NewObjectManager(provider, c.Context())
}
