package odata

import (
	"fmt"
	"log"
	"sync"
)

// MultiTenantProviderPool gerencia pools de conexões para múltiplos tenants
type MultiTenantProviderPool struct {
	providers       map[string]DatabaseProvider
	config          *MultiTenantConfig
	mu              sync.RWMutex
	logger          *log.Logger
	defaultProvider DatabaseProvider
}

// NewMultiTenantProviderPool cria um novo pool multi-tenant
func NewMultiTenantProviderPool(config *MultiTenantConfig, logger *log.Logger) *MultiTenantProviderPool {
	return &MultiTenantProviderPool{
		providers: make(map[string]DatabaseProvider),
		config:    config,
		logger:    logger,
	}
}

// InitializeProviders inicializa todos os providers configurados
func (p *MultiTenantProviderPool) InitializeProviders() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Inicializa provider padrão se existe configuração base
	if p.config.EnvConfig != nil {
		defaultProvider := p.config.EnvConfig.CreateProviderFromConfig()
		if defaultProvider != nil {
			p.defaultProvider = defaultProvider
			p.logger.Printf("✅ Provider padrão inicializado: %s", p.config.EnvConfig.DBDriver)
		}
	}

	// Inicializa providers específicos de tenants
	for tenantID, tenantConfig := range p.config.Tenants {
		provider, err := p.createTenantProvider(tenantConfig)
		if err != nil {
			p.logger.Printf("❌ Erro ao inicializar provider para tenant %s: %v", tenantID, err)
			continue
		}

		p.providers[tenantID] = provider
		p.logger.Printf("✅ Provider inicializado para tenant %s: %s", tenantID, tenantConfig.DBDriver)
	}

	return nil
}

// GetProvider retorna o provider para um tenant específico
func (p *MultiTenantProviderPool) GetProvider(tenantID string) DatabaseProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if provider, exists := p.providers[tenantID]; exists {
		return provider
	}

	// Se não encontrar o tenant, retorna o provider padrão
	if tenantID != p.config.DefaultTenant {
		p.logger.Printf("⚠️ Tenant %s não encontrado, usando provider padrão", tenantID)
	}
	return p.defaultProvider
}

// createTenantProvider cria um provider específico para um tenant (singleton por tenant)
func (p *MultiTenantProviderPool) createTenantProvider(config *TenantConfig) (DatabaseProvider, error) {
	// Cria chave única para este tenant
	cacheKey := fmt.Sprintf("%s:%s@%s:%s/%s", config.DBDriver, config.DBUser, config.DBHost, config.DBPort, config.DBName)
	
	// Verifica se já existe no cache global
	providerCacheMu.RLock()
	if cached, exists := providerCache[cacheKey]; exists {
		providerCacheMu.RUnlock()
		p.logger.Printf("📦 Reusando provider cacheado para tenant %s", config.TenantID)
		return cached, nil
	}
	providerCacheMu.RUnlock()
	
	// Cria novo provider
	providerCacheMu.Lock()
	defer providerCacheMu.Unlock()
	
	// Double-check
	if cached, exists := providerCache[cacheKey]; exists {
		return cached, nil
	}
	
	var provider DatabaseProvider
	
	// Cria provider baseado no driver
	switch config.DBDriver {
	case "postgresql", "postgres", "pgx":
		provider = NewPostgreSQLProvider()
	case "mysql":
		provider = NewMySQLProvider()
	case "oracle":
		provider = NewOracleProvider()
	default:
		return nil, fmt.Errorf("tipo de provider não suportado: %s", config.DBDriver)
	}

	if provider == nil {
		return nil, fmt.Errorf("falha ao criar provider para tipo: %s", config.DBDriver)
	}

	// Configura a conexão específica do tenant
	connectionString := config.BuildConnectionString()

	// Tenta configurar a conexão usando interface específica
	if configurableProvider, ok := provider.(interface {
		ConfigureConnection(connectionString string, maxOpen, maxIdle int, maxLifetime interface{}) error
	}); ok {
		err := configurableProvider.ConfigureConnection(
			connectionString,
			config.DBMaxOpenConns,
			config.DBMaxIdleConns,
			config.DBConnMaxLifetime,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao configurar conexão para tenant %s: %w", config.TenantID, err)
		}
	} else {
		// Se não suporta configuração dinâmica, tenta métodos alternativos
		p.logger.Printf("⚠️ Provider %s não suporta configuração dinâmica para tenant %s", config.DBDriver, config.TenantID)
	}

	// Adiciona ao cache global
	providerCache[cacheKey] = provider
	
	return provider, nil
}

// AddTenant adiciona um novo tenant dinamicamente
func (p *MultiTenantProviderPool) AddTenant(tenantID string, config *TenantConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	provider, err := p.createTenantProvider(config)
	if err != nil {
		return fmt.Errorf("erro ao criar provider para tenant %s: %w", tenantID, err)
	}

	p.providers[tenantID] = provider
	p.config.Tenants[tenantID] = config
	p.logger.Printf("✅ Tenant %s adicionado dinamicamente: %s", tenantID, config.DBDriver)

	return nil
}

// RemoveTenant remove um tenant e fecha sua conexão
func (p *MultiTenantProviderPool) RemoveTenant(tenantID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if provider, exists := p.providers[tenantID]; exists {
		// Fecha a conexão do provider
		if err := provider.Close(); err != nil {
			p.logger.Printf("❌ Erro ao fechar conexão do tenant %s: %v", tenantID, err)
		}

		// Remove das estruturas
		delete(p.providers, tenantID)
		delete(p.config.Tenants, tenantID)
		p.logger.Printf("✅ Tenant %s removido", tenantID)
	}

	return nil
}

// GetTenantStats retorna estatísticas de um tenant específico
func (p *MultiTenantProviderPool) GetTenantStats(tenantID string) map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["tenant_id"] = tenantID
	stats["exists"] = false

	if provider, exists := p.providers[tenantID]; exists {
		stats["exists"] = true
		stats["provider_type"] = fmt.Sprintf("%T", provider)

		// Tenta obter estatísticas da conexão
		if db := provider.GetConnection(); db != nil {
			dbStats := db.Stats()
			stats["open_connections"] = dbStats.OpenConnections
			stats["in_use"] = dbStats.InUse
			stats["idle"] = dbStats.Idle
			stats["wait_count"] = dbStats.WaitCount
			stats["wait_duration"] = dbStats.WaitDuration.String()
			stats["max_idle_closed"] = dbStats.MaxIdleClosed
			stats["max_lifetime_closed"] = dbStats.MaxLifetimeClosed
		}
	}

	return stats
}

// GetAllStats retorna estatísticas de todos os tenants
func (p *MultiTenantProviderPool) GetAllStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allStats := make(map[string]interface{})
	allStats["total_tenants"] = len(p.providers)
	allStats["tenants"] = make(map[string]interface{})

	for tenantID := range p.providers {
		allStats["tenants"].(map[string]interface{})[tenantID] = p.GetTenantStats(tenantID)
	}

	return allStats
}

// HealthCheck verifica a saúde de todas as conexões
func (p *MultiTenantProviderPool) HealthCheck() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	health := make(map[string]interface{})
	health["status"] = "healthy"
	health["tenants"] = make(map[string]interface{})

	unhealthyCount := 0

	// Verifica provider padrão
	if p.defaultProvider != nil {
		if db := p.defaultProvider.GetConnection(); db != nil {
			if err := db.Ping(); err != nil {
				health["default_provider"] = "unhealthy"
				health["default_provider_error"] = err.Error()
				unhealthyCount++
			} else {
				health["default_provider"] = "healthy"
			}
		}
	}

	// Verifica todos os tenants
	for tenantID, provider := range p.providers {
		tenantHealth := make(map[string]interface{})
		tenantHealth["status"] = "healthy"

		if db := provider.GetConnection(); db != nil {
			if err := db.Ping(); err != nil {
				tenantHealth["status"] = "unhealthy"
				tenantHealth["error"] = err.Error()
				unhealthyCount++
			}
		} else {
			tenantHealth["status"] = "no_connection"
			unhealthyCount++
		}

		health["tenants"].(map[string]interface{})[tenantID] = tenantHealth
	}

	if unhealthyCount > 0 {
		health["status"] = "degraded"
		health["unhealthy_count"] = unhealthyCount
	}

	return health
}

// Close fecha todas as conexões do pool
func (p *MultiTenantProviderPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errors []error

	// Fecha provider padrão
	if p.defaultProvider != nil {
		if err := p.defaultProvider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("erro ao fechar provider padrão: %w", err))
		}
	}

	// Fecha providers de tenants
	for tenantID, provider := range p.providers {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("erro ao fechar provider do tenant %s: %w", tenantID, err))
		}
	}

	// Limpa os maps
	p.providers = make(map[string]DatabaseProvider)

	if len(errors) > 0 {
		return fmt.Errorf("erros ao fechar pool: %v", errors)
	}

	p.logger.Printf("✅ Pool multi-tenant fechado com sucesso")
	return nil
}

// IsEnabled retorna se o modo multi-tenant está habilitado
func (p *MultiTenantProviderPool) IsEnabled() bool {
	return p.config.Enabled
}

// GetTenantList retorna lista de tenants disponíveis
func (p *MultiTenantProviderPool) GetTenantList() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var tenants []string
	for tenantID := range p.providers {
		tenants = append(tenants, tenantID)
	}

	// Adiciona o tenant padrão se não estiver na lista
	defaultTenant := p.config.DefaultTenant
	found := false
	for _, tenant := range tenants {
		if tenant == defaultTenant {
			found = true
			break
		}
	}
	if !found {
		tenants = append(tenants, defaultTenant)
	}

	return tenants
}
