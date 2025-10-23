package jwt

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

// JWTConfig configurações para JWT
type JWTConfig struct {
	SecretKey string        `json:"secret_key"`
	Issuer    string        `json:"issuer"`
	ExpiresIn time.Duration `json:"expires_in"`
	RefreshIn time.Duration `json:"refresh_in"`
	Algorithm string        `json:"algorithm"` // default: HS256
}

// JWTClaims representa os claims do token JWT
type JWTClaims struct {
	Username string                 `json:"username"`
	Roles    []string               `json:"roles,omitempty"`
	Scopes   []string               `json:"scopes,omitempty"`
	Admin    bool                   `json:"admin,omitempty"`
	Custom   map[string]interface{} `json:"custom,omitempty"`
	jwt.RegisteredClaims
}

// DefaultJWTConfig retorna configuração padrão para JWT
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey: "your-secret-key-change-this-in-production",
		Issuer:    "go-data-server",
		ExpiresIn: 24 * time.Hour,
		RefreshIn: 7 * 24 * time.Hour,
		Algorithm: "HS256",
	}
}

// LoadConfigFromEnv carrega configurações JWT do arquivo .env
// Ordem de prioridade: .env > valores padrão
func LoadConfigFromEnv() *JWTConfig {
	// Tenta carregar .env (ignora erro se não existir)
	_ = godotenv.Load()

	config := &JWTConfig{
		SecretKey: getEnv("JWT_SECRET", ""),
		Issuer:    getEnv("JWT_ISSUER", "go-data-server"),
		ExpiresIn: getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
		RefreshIn: getEnvDuration("JWT_REFRESH_EXPIRATION", 7*24*time.Hour),
		Algorithm: getEnv("JWT_ALGORITHM", "HS256"),
	}

	// Validação do secret
	if config.SecretKey == "" {
		log.Println("⚠️  JWT_SECRET não definido no .env, usando valor padrão (NÃO USE EM PRODUÇÃO!)")
		config.SecretKey = "your-secret-key-change-this-in-production"
	} else if len(config.SecretKey) < 32 {
		log.Printf("⚠️  JWT_SECRET tem apenas %d caracteres, recomendado mínimo 32 caracteres", len(config.SecretKey))
	}

	return config
}

// MergeConfig faz merge de uma configuração customizada sobre a base
// Valores não-zero da configuração customizada sobrescrevem os da base
func MergeConfig(base *JWTConfig, custom *JWTConfig) *JWTConfig {
	if custom == nil {
		return base
	}

	result := *base // cópia

	// Override apenas valores não-vazios/não-zero
	if custom.SecretKey != "" {
		result.SecretKey = custom.SecretKey
	}
	if custom.Issuer != "" {
		result.Issuer = custom.Issuer
	}
	if custom.ExpiresIn > 0 {
		result.ExpiresIn = custom.ExpiresIn
	}
	if custom.RefreshIn > 0 {
		result.RefreshIn = custom.RefreshIn
	}
	if custom.Algorithm != "" {
		result.Algorithm = custom.Algorithm
	}

	return &result
}

// Validate valida a configuração JWT
func (c *JWTConfig) Validate() error {
	if c.SecretKey == "" {
		return fmt.Errorf("JWT secret key is required")
	}
	if len(c.SecretKey) < 16 {
		return fmt.Errorf("JWT secret key must be at least 16 characters, got %d", len(c.SecretKey))
	}
	if c.ExpiresIn <= 0 {
		return fmt.Errorf("JWT expiration must be positive, got %v", c.ExpiresIn)
	}
	if c.RefreshIn <= 0 {
		return fmt.Errorf("JWT refresh expiration must be positive, got %v", c.RefreshIn)
	}
	if c.RefreshIn <= c.ExpiresIn {
		return fmt.Errorf("JWT refresh expiration (%v) must be greater than expiration (%v)", c.RefreshIn, c.ExpiresIn)
	}
	return nil
}

// Helper functions

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		// Tenta parsear como segundos
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Tenta parsear como duration (ex: "24h", "30m")
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}
