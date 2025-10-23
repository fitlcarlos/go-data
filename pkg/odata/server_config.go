package odata

import (
	"crypto/tls"
	"time"
)

// EntityAuthConfig configurações de autenticação por entidade
type EntityAuthConfig struct {
	RequireAuth    bool     // Se true, todas as operações requerem autenticação
	RequiredRoles  []string // Roles necessárias para acessar a entidade
	RequiredScopes []string // Scopes necessários para acessar a entidade
	RequireAdmin   bool     // Se true, apenas administradores podem acessar
	ReadOnly       bool     // Se true, apenas operações de leitura são permitidas
}

// ServerConfig representa as configurações do servidor
type ServerConfig struct {
	// Configurações básicas
	Name        string
	DisplayName string
	Description string

	// Configurações de host e porta
	Host string
	Port int

	// Configurações de TLS
	TLSConfig   *tls.Config
	CertFile    string
	CertKeyFile string

	// Configurações de CORS
	EnableCORS       bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool

	// Configurações de log
	EnableLogging bool
	LogLevel      string
	LogFile       string

	// Configurações de middleware
	EnableCompression bool
	MaxRequestSize    int64

	// Configurações de graceful shutdown
	ShutdownTimeout time.Duration

	// Configurações de prefixo
	RoutePrefix string

	// Configurações JWT
	EnableJWT   bool
	JWTConfig   *JWTConfig
	RequireAuth bool // Se true, todas as rotas requerem autenticação por padrão

	// Configurações de Rate Limit
	RateLimitConfig *RateLimitConfig

	// Configurações de Validação
	ValidationConfig *ValidationConfig

	// Configurações de Security Headers
	SecurityHeadersConfig *SecurityHeadersConfig

	// Configurações de Audit Logging
	AuditLogConfig *AuditLogConfig

	// Performance: Desabilita JOIN automático para expand (força batching)
	// Default: false (usa detecção automática baseada em relacionamento)
	DisableJoinForExpand bool
}

// DefaultServerConfig retorna uma configuração padrão do servidor
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Name:                  "godata-service",
		DisplayName:           "GoData OData Service",
		Description:           "Serviço GoData OData v4 para APIs RESTful",
		Host:                  "localhost",
		Port:                  8080,
		EnableCORS:            true,
		AllowedOrigins:        []string{"*"},
		AllowedMethods:        []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:        []string{"*"},
		ExposedHeaders:        []string{"OData-Version", "Content-Type"},
		AllowCredentials:      false,
		EnableLogging:         true,
		LogLevel:              "INFO",
		EnableCompression:     false,            // Desabilitado por padrão para evitar problemas
		MaxRequestSize:        10 * 1024 * 1024, // 10MB
		ShutdownTimeout:       30 * time.Second,
		RoutePrefix:           "/odata",
		ValidationConfig:      DefaultValidationConfig(),
		SecurityHeadersConfig: DefaultSecurityHeadersConfig(),
		RateLimitConfig:       DefaultRateLimitConfig(),
		AuditLogConfig:        DefaultAuditLogConfig(),
		DisableJoinForExpand:  false, // JOIN automático habilitado por padrão
	}
}
