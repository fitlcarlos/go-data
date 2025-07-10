package odata

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// UserIdentity representa a identidade do usuário autenticado
type UserIdentity struct {
	Username string                 `json:"username"`
	Roles    []string               `json:"roles"`
	Scopes   []string               `json:"scopes"`
	Admin    bool                   `json:"admin"`
	Custom   map[string]interface{} `json:"custom"`
}

// HasRole verifica se o usuário possui uma role específica
func (u *UserIdentity) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasScope verifica se o usuário possui um scope específico
func (u *UserIdentity) HasScope(scope string) bool {
	for _, s := range u.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyRole verifica se o usuário possui pelo menos uma das roles
func (u *UserIdentity) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAnyScope verifica se o usuário possui pelo menos um dos scopes
func (u *UserIdentity) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if u.HasScope(scope) {
			return true
		}
	}
	return false
}

// IsAdmin verifica se o usuário é administrador
func (u *UserIdentity) IsAdmin() bool {
	return u.Admin
}

// GetCustomClaim obtém um claim customizado
func (u *UserIdentity) GetCustomClaim(key string) (interface{}, bool) {
	if u.Custom == nil {
		return nil, false
	}
	value, exists := u.Custom[key]
	return value, exists
}

// DefaultJWTConfig retorna configuração padrão para JWT
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey: "your-secret-key-change-this-in-production",
		Issuer:    "godata-server",
		ExpiresIn: 24 * time.Hour,
		RefreshIn: 7 * 24 * time.Hour,
		Algorithm: "HS256",
	}
}
