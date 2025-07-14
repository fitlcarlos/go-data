package odata

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// TenantContext chave para armazenar tenant no contexto
// TenantContextKeyType define um tipo customizado para chaves de contexto
type TenantContextKeyType struct{}

var TenantContextKey = TenantContextKeyType{}

// FiberContextKeyType define um tipo customizado para chave do Fiber Context
type FiberContextKeyType struct{}

var FiberContextKey = FiberContextKeyType{}

// TenantMiddleware middleware para identificação de tenant
func (s *Server) TenantMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if s.multiTenantPool == nil || !s.multiTenantConfig.Enabled {
			// Se não está em modo multi-tenant, continua normalmente
			return c.Next()
		}

		tenantID := s.identifyTenant(c)

		// Valida se o tenant existe
		if !s.multiTenantConfig.TenantExists(tenantID) {
			return fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("Tenant '%s' não encontrado", tenantID))
		}

		// Armazena o tenant no contexto
		c.Locals(TenantContextKey, tenantID)

		s.logger.Printf("🏢 Tenant identificado: %s", tenantID)
		return c.Next()
	}
}

// identifyTenant identifica o tenant baseado na configuração
func (s *Server) identifyTenant(c fiber.Ctx) string {
	if s.multiTenantConfig == nil || !s.multiTenantConfig.Enabled {
		return "default"
	}

	switch s.multiTenantConfig.IdentificationMode {
	case "header":
		return s.identifyTenantByHeader(c)
	case "subdomain":
		return s.identifyTenantBySubdomain(c)
	case "path":
		return s.identifyTenantByPath(c)
	case "jwt":
		return s.identifyTenantByJWT(c)
	default:
		return s.multiTenantConfig.DefaultTenant
	}
}

// identifyTenantByHeader identifica tenant via header
func (s *Server) identifyTenantByHeader(c fiber.Ctx) string {
	headerName := s.multiTenantConfig.HeaderName
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}

	tenantID := c.Get(headerName)
	if tenantID == "" {
		tenantID = s.multiTenantConfig.DefaultTenant
	}

	return tenantID
}

// identifyTenantBySubdomain identifica tenant via subdomain
func (s *Server) identifyTenantBySubdomain(c fiber.Ctx) string {
	hostname := c.Hostname()
	parts := strings.Split(hostname, ".")

	if len(parts) > 2 {
		// Assume que o subdomain é o tenant
		subdomain := parts[0]
		// Filtra subdomains comuns que não são tenants
		if subdomain != "www" && subdomain != "api" && subdomain != "app" {
			return subdomain
		}
	}

	return s.multiTenantConfig.DefaultTenant
}

// identifyTenantByPath identifica tenant via path
func (s *Server) identifyTenantByPath(c fiber.Ctx) string {
	path := c.Path()

	// Ex: /tenant/empresa_a/odata/Entidade
	if strings.HasPrefix(path, "/tenant/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			return parts[2]
		}
	}

	// Ex: /api/empresa_a/odata/Entidade
	if strings.HasPrefix(path, "/api/") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			// Verifica se o segundo segmento não é "odata"
			if parts[2] != "odata" {
				return parts[2]
			}
		}
	}

	return s.multiTenantConfig.DefaultTenant
}

// identifyTenantByJWT identifica tenant via JWT
func (s *Server) identifyTenantByJWT(c fiber.Ctx) string {
	user := GetCurrentUser(c)
	if user != nil {
		if tenantID, exists := user.GetCustomClaim("tenant_id"); exists {
			if tenantStr, ok := tenantID.(string); ok {
				return tenantStr
			}
		}
	}

	return s.multiTenantConfig.DefaultTenant
}

// GetCurrentTenant retorna o tenant atual do contexto
func GetCurrentTenant(c fiber.Ctx) string {
	if tenantID, ok := c.Locals(TenantContextKey).(string); ok {
		return tenantID
	}
	return "default"
}

// RequireTenant middleware que requer um tenant específico
func (s *Server) RequireTenant(allowedTenants ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		currentTenant := GetCurrentTenant(c)

		// Se não especificou tenants permitidos, permite qualquer um
		if len(allowedTenants) == 0 {
			return c.Next()
		}

		// Verifica se o tenant atual está na lista permitida
		for _, allowed := range allowedTenants {
			if currentTenant == allowed {
				return c.Next()
			}
		}

		return fiber.NewError(fiber.StatusForbidden,
			fmt.Sprintf("Acesso negado para tenant '%s'", currentTenant))
	}
}

// TenantInfo middleware que adiciona informações do tenant no contexto
func (s *Server) TenantInfo() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID := GetCurrentTenant(c)

		// Adiciona informações do tenant no contexto
		if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
			if tenantConfig := s.multiTenantConfig.GetTenantConfig(tenantID); tenantConfig != nil {
				c.Locals("tenant_config", tenantConfig)
				c.Locals("tenant_db_type", tenantConfig.DBType)
				c.Locals("tenant_db_host", tenantConfig.DBHost)
			}
		}

		return c.Next()
	}
}

// GetCurrentTenantConfig retorna a configuração do tenant atual
func GetCurrentTenantConfig(c fiber.Ctx) *TenantConfig {
	if config, ok := c.Locals("tenant_config").(*TenantConfig); ok {
		return config
	}
	return nil
}

// TenantStatsMiddleware middleware para coletar estatísticas por tenant
func (s *Server) TenantStatsMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		tenantID := GetCurrentTenant(c)

		// Registra a requisição para o tenant
		s.logger.Printf("📊 Requisição para tenant %s: %s %s", tenantID, c.Method(), c.Path())

		// Prossegue com a requisição
		err := c.Next()

		// Registra o status da resposta
		s.logger.Printf("📊 Resposta para tenant %s: %d", tenantID, c.Response().StatusCode())

		return err
	}
}

// MultiTenantHealthCheck middleware para verificar saúde do tenant
func (s *Server) MultiTenantHealthCheck() fiber.Handler {
	return func(c fiber.Ctx) error {
		if s.multiTenantPool == nil || !s.multiTenantConfig.Enabled {
			return c.Next()
		}

		tenantID := GetCurrentTenant(c)

		// Verifica se o provider do tenant está saudável
		provider := s.multiTenantPool.GetProvider(tenantID)
		if provider == nil {
			return fiber.NewError(fiber.StatusServiceUnavailable,
				fmt.Sprintf("Provider não disponível para tenant '%s'", tenantID))
		}

		// Testa a conexão
		if db := provider.GetConnection(); db != nil {
			if err := db.Ping(); err != nil {
				return fiber.NewError(fiber.StatusServiceUnavailable,
					fmt.Sprintf("Conexão com banco não disponível para tenant '%s': %v", tenantID, err))
			}
		}

		return c.Next()
	}
}

// TenantSwitchMiddleware middleware para permitir mudança de tenant em tempo de execução
func (s *Server) TenantSwitchMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Verifica se há um switch de tenant na query string
		switchTenant := c.Query("switch_tenant")
		if switchTenant != "" {
			// Verifica se o tenant de destino existe
			if s.multiTenantConfig.TenantExists(switchTenant) {
				// Substitui o tenant no contexto
				c.Locals(TenantContextKey, switchTenant)
				s.logger.Printf("🔄 Tenant alternado para: %s", switchTenant)
			} else {
				return fiber.NewError(fiber.StatusBadRequest,
					fmt.Sprintf("Tenant de destino '%s' não encontrado", switchTenant))
			}
		}

		return c.Next()
	}
}

// TenantRateLimitMiddleware middleware para limitar requests por tenant
func (s *Server) TenantRateLimitMiddleware(requestsPerMinute int) fiber.Handler {
	// Mapa para armazenar contadores por tenant
	tenantCounters := make(map[string]int)

	return func(c fiber.Ctx) error {
		tenantID := GetCurrentTenant(c)

		// Incrementa contador para o tenant
		tenantCounters[tenantID]++

		// Verifica se excedeu o limite
		if tenantCounters[tenantID] > requestsPerMinute {
			return fiber.NewError(fiber.StatusTooManyRequests,
				fmt.Sprintf("Limite de requisições excedido para tenant '%s'", tenantID))
		}

		// Reset periódico dos contadores seria implementado aqui
		// Por simplicidade, não implementamos a lógica de reset temporal

		return c.Next()
	}
}
