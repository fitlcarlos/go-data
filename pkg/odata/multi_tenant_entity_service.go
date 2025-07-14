package odata

import (
	"context"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
)

// MultiTenantEntityService encapsula o BaseEntityService com suporte multi-tenant
type MultiTenantEntityService struct {
	*BaseEntityService
	server *Server
}

// NewMultiTenantEntityService cria um novo serviço multi-tenant
func NewMultiTenantEntityService(metadata EntityMetadata, server *Server) *MultiTenantEntityService {
	// Cria o serviço base com provider nil (será resolvido dinamicamente)
	baseService := &BaseEntityService{
		metadata:      metadata,
		server:        server,
		computeParser: NewComputeParser(),
		searchParser:  NewSearchParser(),
	}

	return &MultiTenantEntityService{
		BaseEntityService: baseService,
		server:            server,
	}
}

// getProviderForContext retorna o provider apropriado para o contexto
func (s *MultiTenantEntityService) getProviderForContext(ctx context.Context) DatabaseProvider {
	// Tenta extrair o Fiber Context do contexto
	if fiberCtx, ok := ctx.Value(FiberContextKey).(fiber.Ctx); ok {
		return s.server.getCurrentProvider(fiberCtx)
	}

	// Tenta extrair o tenant ID diretamente do contexto
	if tenantID, ok := ctx.Value(TenantContextKey).(string); ok {
		if s.server.multiTenantPool != nil {
			return s.server.multiTenantPool.GetProvider(tenantID)
		}
	}

	// Fallback para provider padrão
	if s.server.multiTenantPool != nil {
		return s.server.multiTenantPool.GetProvider("default")
	}

	return s.server.provider
}

// logTenantOperation registra operação com informações do tenant
func (s *MultiTenantEntityService) logTenantOperation(ctx context.Context, operation string, details string) {
	tenantID := "default"

	// Tenta extrair tenant ID do contexto
	if fiberCtx, ok := ctx.Value(FiberContextKey).(fiber.Ctx); ok {
		tenantID = GetCurrentTenant(fiberCtx)
	} else if tid, ok := ctx.Value(TenantContextKey).(string); ok {
		tenantID = tid
	}

	s.server.logger.Printf("🏢 [%s] %s - %s: %s", tenantID, s.metadata.Name, operation, details)
}

// Query executa uma consulta usando o provider apropriado
func (s *MultiTenantEntityService) Query(ctx context.Context, options QueryOptions) (*ODataResponse, error) {
	// Resolve o provider dinamicamente
	originalProvider := s.provider
	s.provider = s.getProviderForContext(ctx)

	// Log da operação
	s.logTenantOperation(ctx, "Query", fmt.Sprintf("Options: %+v", options))

	// Verifica se o provider está disponível
	if s.provider == nil {
		return nil, fmt.Errorf("provider não disponível para o tenant")
	}

	// Chama o método original
	response, err := s.BaseEntityService.Query(ctx, options)

	// Restaura o provider original
	s.provider = originalProvider

	if err != nil {
		s.logTenantOperation(ctx, "Query", fmt.Sprintf("Error: %v", err))
		return nil, err
	}

	s.logTenantOperation(ctx, "Query", "Success")
	return response, nil
}

// Get executa uma consulta de entidade específica usando o provider apropriado
func (s *MultiTenantEntityService) Get(ctx context.Context, keys map[string]any) (any, error) {
	// Resolve o provider dinamicamente
	originalProvider := s.provider
	s.provider = s.getProviderForContext(ctx)

	// Log da operação
	s.logTenantOperation(ctx, "Get", fmt.Sprintf("Keys: %+v", keys))

	// Verifica se o provider está disponível
	if s.provider == nil {
		return nil, fmt.Errorf("provider não disponível para o tenant")
	}

	// Chama o método original
	result, err := s.BaseEntityService.Get(ctx, keys)

	// Restaura o provider original
	s.provider = originalProvider

	if err != nil {
		s.logTenantOperation(ctx, "Get", fmt.Sprintf("Error: %v", err))
		return nil, err
	}

	s.logTenantOperation(ctx, "Get", "Success")
	return result, nil
}

// Create cria uma nova entidade usando o provider apropriado
func (s *MultiTenantEntityService) Create(ctx context.Context, entity any) (any, error) {
	// Resolve o provider dinamicamente
	originalProvider := s.provider
	s.provider = s.getProviderForContext(ctx)

	// Log da operação
	s.logTenantOperation(ctx, "Create", fmt.Sprintf("Entity: %T", entity))

	// Verifica se o provider está disponível
	if s.provider == nil {
		return nil, fmt.Errorf("provider não disponível para o tenant")
	}

	// Chama o método original
	result, err := s.BaseEntityService.Create(ctx, entity)

	// Restaura o provider original
	s.provider = originalProvider

	if err != nil {
		s.logTenantOperation(ctx, "Create", fmt.Sprintf("Error: %v", err))
		return nil, err
	}

	s.logTenantOperation(ctx, "Create", "Success")
	return result, nil
}

// Update atualiza uma entidade usando o provider apropriado
func (s *MultiTenantEntityService) Update(ctx context.Context, keys map[string]any, entity any) (any, error) {
	// Resolve o provider dinamicamente
	originalProvider := s.provider
	s.provider = s.getProviderForContext(ctx)

	// Log da operação
	s.logTenantOperation(ctx, "Update", fmt.Sprintf("Keys: %+v, Entity: %T", keys, entity))

	// Verifica se o provider está disponível
	if s.provider == nil {
		return nil, fmt.Errorf("provider não disponível para o tenant")
	}

	// Chama o método original
	result, err := s.BaseEntityService.Update(ctx, keys, entity)

	// Restaura o provider original
	s.provider = originalProvider

	if err != nil {
		s.logTenantOperation(ctx, "Update", fmt.Sprintf("Error: %v", err))
		return nil, err
	}

	s.logTenantOperation(ctx, "Update", "Success")
	return result, nil
}

// Delete remove uma entidade usando o provider apropriado
func (s *MultiTenantEntityService) Delete(ctx context.Context, keys map[string]any) error {
	// Resolve o provider dinamicamente
	originalProvider := s.provider
	s.provider = s.getProviderForContext(ctx)

	// Log da operação
	s.logTenantOperation(ctx, "Delete", fmt.Sprintf("Keys: %+v", keys))

	// Verifica se o provider está disponível
	if s.provider == nil {
		return fmt.Errorf("provider não disponível para o tenant")
	}

	// Chama o método original
	err := s.BaseEntityService.Delete(ctx, keys)

	// Restaura o provider original
	s.provider = originalProvider

	if err != nil {
		s.logTenantOperation(ctx, "Delete", fmt.Sprintf("Error: %v", err))
		return err
	}

	s.logTenantOperation(ctx, "Delete", "Success")
	return nil
}

// GetTenantProvider retorna o provider para um tenant específico
func (s *MultiTenantEntityService) GetTenantProvider(tenantID string) DatabaseProvider {
	if s.server.multiTenantPool != nil {
		return s.server.multiTenantPool.GetProvider(tenantID)
	}
	return s.server.provider
}

// IsMultiTenantEnabled verifica se o multi-tenant está habilitado
func (s *MultiTenantEntityService) IsMultiTenantEnabled() bool {
	return s.server.multiTenantConfig != nil && s.server.multiTenantConfig.Enabled
}

// GetAvailableTenants retorna lista de tenants disponíveis
func (s *MultiTenantEntityService) GetAvailableTenants() []string {
	if s.server.multiTenantPool != nil {
		return s.server.multiTenantPool.GetTenantList()
	}
	return []string{"default"}
}

// ExecuteWithTenant executa uma operação com um tenant específico
func (s *MultiTenantEntityService) ExecuteWithTenant(tenantID string, operation func(provider DatabaseProvider) error) error {
	if s.server.multiTenantPool == nil {
		return fmt.Errorf("multi-tenant não está habilitado")
	}

	provider := s.server.multiTenantPool.GetProvider(tenantID)
	if provider == nil {
		return fmt.Errorf("provider não encontrado para tenant: %s", tenantID)
	}

	return operation(provider)
}

// GetTenantStats retorna estatísticas do tenant para esta entidade
func (s *MultiTenantEntityService) GetTenantStats(tenantID string) map[string]interface{} {
	stats := make(map[string]interface{})
	stats["entity_name"] = s.metadata.Name
	stats["tenant_id"] = tenantID

	if s.server.multiTenantPool != nil {
		// Adiciona estatísticas do pool
		poolStats := s.server.multiTenantPool.GetTenantStats(tenantID)
		stats["pool_stats"] = poolStats
	}

	return stats
}

// ValidateTenantAccess valida se o tenant pode acessar esta entidade
func (s *MultiTenantEntityService) ValidateTenantAccess(ctx context.Context, operation string) error {
	tenantID := "default"

	// Extrai tenant ID do contexto
	if fiberCtx, ok := ctx.Value(FiberContextKey).(fiber.Ctx); ok {
		tenantID = GetCurrentTenant(fiberCtx)
	} else if tid, ok := ctx.Value(TenantContextKey).(string); ok {
		tenantID = tid
	}

	// Verifica se o tenant existe
	if s.server.multiTenantConfig != nil && s.server.multiTenantConfig.Enabled {
		if !s.server.multiTenantConfig.TenantExists(tenantID) {
			return fmt.Errorf("tenant '%s' não encontrado", tenantID)
		}
	}

	// Verifica se o provider está disponível
	provider := s.getProviderForContext(ctx)
	if provider == nil {
		return fmt.Errorf("provider não disponível para tenant '%s'", tenantID)
	}

	// Verifica se a conexão está saudável
	if db := provider.GetConnection(); db != nil {
		if err := db.Ping(); err != nil {
			return fmt.Errorf("conexão não saudável para tenant '%s': %w", tenantID, err)
		}
	}

	return nil
}

// WithTenantContext cria um novo contexto com tenant específico
func (s *MultiTenantEntityService) WithTenantContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantContextKey, tenantID)
}

// GetCurrentTenantFromContext extrai o tenant ID do contexto
func (s *MultiTenantEntityService) GetCurrentTenantFromContext(ctx context.Context) string {
	if fiberCtx, ok := ctx.Value(FiberContextKey).(fiber.Ctx); ok {
		return GetCurrentTenant(fiberCtx)
	}

	if tenantID, ok := ctx.Value(TenantContextKey).(string); ok {
		return tenantID
	}

	return "default"
}

// LogMultiTenantInfo registra informações sobre o estado multi-tenant
func (s *MultiTenantEntityService) LogMultiTenantInfo() {
	if s.server.multiTenantConfig != nil && s.server.multiTenantConfig.Enabled {
		log.Printf("🏢 Entity %s - Multi-tenant habilitado", s.metadata.Name)
		log.Printf("   Tenants disponíveis: %v", s.GetAvailableTenants())
		log.Printf("   Modo de identificação: %s", s.server.multiTenantConfig.IdentificationMode)
	} else {
		log.Printf("🏢 Entity %s - Single-tenant mode", s.metadata.Name)
	}
}
