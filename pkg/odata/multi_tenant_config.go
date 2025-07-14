package odata

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// TenantConfig representa configurações específicas de um tenant
type TenantConfig struct {
	TenantID           string
	DBType             string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	DBSchema           string
	DBConnectionString string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration

	// Configurações específicas do tenant
	CustomSettings map[string]string
}

// MultiTenantConfig representa configurações multi-tenant
type MultiTenantConfig struct {
	Enabled            bool
	IdentificationMode string // header, subdomain, path, jwt
	HeaderName         string
	DefaultTenant      string
	Tenants            map[string]*TenantConfig

	// Configurações globais herdadas
	*EnvConfig
}

// BuildConnectionString constrói a string de conexão para um tenant
func (tc *TenantConfig) BuildConnectionString() string {
	if tc.DBConnectionString != "" {
		return tc.DBConnectionString
	}

	switch tc.DBType {
	case "oracle":
		return fmt.Sprintf("oracle://%s:%s@%s:%s/%s",
			tc.DBUser, tc.DBPassword, tc.DBHost, tc.DBPort, tc.DBName)
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			tc.DBHost, tc.DBPort, tc.DBUser, tc.DBPassword, tc.DBName)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			tc.DBUser, tc.DBPassword, tc.DBHost, tc.DBPort, tc.DBName)
	default:
		return tc.DBConnectionString
	}
}

// LoadMultiTenantConfig carrega configurações multi-tenant automaticamente
func LoadMultiTenantConfig() *MultiTenantConfig {
	// Tenta carregar configurações do .env
	envConfig, err := LoadEnvOrDefault()
	if err != nil {
		log.Printf("Aviso: Não foi possível carregar configurações do .env: %v", err)
		// Retorna configuração padrão (single-tenant)
		return &MultiTenantConfig{
			Enabled:       false,
			DefaultTenant: "default",
			Tenants:       make(map[string]*TenantConfig),
			EnvConfig:     envConfig,
		}
	}

	return envConfig.parseMultiTenantVariables()
}

// parseMultiTenantVariables parseia as variáveis multi-tenant do .env
func (c *EnvConfig) parseMultiTenantVariables() *MultiTenantConfig {
	multiTenant := &MultiTenantConfig{
		Enabled:            c.getEnvBool("MULTI_TENANT_ENABLED", false),
		IdentificationMode: c.getEnvString("TENANT_IDENTIFICATION_MODE", "header"),
		HeaderName:         c.getEnvString("TENANT_HEADER_NAME", "X-Tenant-ID"),
		DefaultTenant:      c.getEnvString("DEFAULT_TENANT", "default"),
		Tenants:            make(map[string]*TenantConfig),
		EnvConfig:          c,
	}

	if !multiTenant.Enabled {
		return multiTenant
	}

	// Parse configurações específicas de tenants
	for key, value := range c.Variables {
		if strings.HasPrefix(key, "TENANT_") && strings.Contains(key, "_DB_") {
			parts := strings.Split(key, "_")
			if len(parts) >= 4 {
				tenantID := parts[1]
				dbConfigKey := strings.Join(parts[2:], "_")

				if _, exists := multiTenant.Tenants[tenantID]; !exists {
					multiTenant.Tenants[tenantID] = &TenantConfig{
						TenantID:          tenantID,
						CustomSettings:    make(map[string]string),
						DBMaxOpenConns:    25,
						DBMaxIdleConns:    5,
						DBConnMaxLifetime: 10 * time.Minute,
					}
				}

				tenant := multiTenant.Tenants[tenantID]
				switch dbConfigKey {
				case "DB_TYPE":
					tenant.DBType = value
				case "DB_HOST":
					tenant.DBHost = value
				case "DB_PORT":
					tenant.DBPort = value
				case "DB_NAME":
					tenant.DBName = value
				case "DB_USER":
					tenant.DBUser = value
				case "DB_PASSWORD":
					tenant.DBPassword = value
				case "DB_SCHEMA":
					tenant.DBSchema = value
				case "DB_CONNECTION_STRING":
					tenant.DBConnectionString = value
				case "DB_MAX_OPEN_CONNS":
					if intVal := c.getEnvInt("TENANT_"+tenantID+"_"+dbConfigKey, 25); intVal > 0 {
						tenant.DBMaxOpenConns = intVal
					}
				case "DB_MAX_IDLE_CONNS":
					if intVal := c.getEnvInt("TENANT_"+tenantID+"_"+dbConfigKey, 5); intVal > 0 {
						tenant.DBMaxIdleConns = intVal
					}
				case "DB_CONN_MAX_LIFETIME":
					if durVal := c.getEnvDuration("TENANT_"+tenantID+"_"+dbConfigKey, 10*time.Minute); durVal > 0 {
						tenant.DBConnMaxLifetime = durVal
					}
				}
			}
		}
	}

	return multiTenant
}

// GetTenantConfig retorna configuração de um tenant específico
func (mtc *MultiTenantConfig) GetTenantConfig(tenantID string) *TenantConfig {
	if config, exists := mtc.Tenants[tenantID]; exists {
		return config
	}
	return nil
}

// TenantExists verifica se um tenant existe
func (mtc *MultiTenantConfig) TenantExists(tenantID string) bool {
	if tenantID == mtc.DefaultTenant {
		return true
	}
	_, exists := mtc.Tenants[tenantID]
	return exists
}

// GetAllTenantIDs retorna lista de todos os tenant IDs
func (mtc *MultiTenantConfig) GetAllTenantIDs() []string {
	var tenantIDs []string

	// Adiciona o tenant padrão
	tenantIDs = append(tenantIDs, mtc.DefaultTenant)

	// Adiciona tenants configurados
	for tenantID := range mtc.Tenants {
		if tenantID != mtc.DefaultTenant {
			tenantIDs = append(tenantIDs, tenantID)
		}
	}

	return tenantIDs
}

// PrintMultiTenantConfig imprime as configurações multi-tenant para debug
func (mtc *MultiTenantConfig) PrintMultiTenantConfig() {
	fmt.Println("🏢 Configurações Multi-Tenant:")
	fmt.Printf("   Enabled: %v\n", mtc.Enabled)

	if mtc.Enabled {
		fmt.Printf("   Identification Mode: %s\n", mtc.IdentificationMode)
		fmt.Printf("   Header Name: %s\n", mtc.HeaderName)
		fmt.Printf("   Default Tenant: %s\n", mtc.DefaultTenant)
		fmt.Printf("   Configured Tenants: %d\n", len(mtc.Tenants))

		for tenantID, config := range mtc.Tenants {
			fmt.Printf("     - %s: %s://%s:%s/%s\n",
				tenantID, config.DBType, config.DBHost, config.DBPort, config.DBName)
		}
	} else {
		fmt.Println("   Single-tenant mode")
	}
	fmt.Println()
}
