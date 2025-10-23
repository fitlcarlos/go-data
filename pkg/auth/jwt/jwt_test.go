package jwt

import (
	"testing"
	"time"

	"github.com/fitlcarlos/go-data/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWT_TokenGeneration(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "testuser",
		Roles:    []string{"admin", "user"},
		Scopes:   []string{"read", "write"},
		Admin:    true,
		Custom: map[string]interface{}{
			"email": "test@example.com",
			"id":    123,
		},
	}

	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err, "Should generate token successfully")
	assert.NotEmpty(t, token, "Token should not be empty")
	assert.Contains(t, token, ".", "Token should contain dots (JWT format)")
}

func TestJWT_TokenValidation(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "testuser",
		Roles:    []string{"admin"},
		Scopes:   []string{"read"},
		Admin:    false,
	}

	// Gerar token válido
	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err)

	// Validar token
	validated, err := jwtAuth.ValidateToken(token)
	require.NoError(t, err, "Should validate token successfully")
	assert.NotNil(t, validated, "Validated user should not be nil")
	assert.Equal(t, "testuser", validated.Username)
	assert.Equal(t, []string{"admin"}, validated.Roles)
	assert.Equal(t, []string{"read"}, validated.Scopes)
	assert.False(t, validated.Admin)
}

func TestJWT_TokenValidation_InvalidToken(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	// Token inválido
	_, err := jwtAuth.ValidateToken("invalid.token.here")
	assert.Error(t, err, "Should fail to validate invalid token")
}

func TestJWT_TokenValidation_ExpiredToken(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 1 * time.Millisecond, // Expira rapidamente
		RefreshIn: 1 * time.Hour,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "testuser",
		Roles:    []string{"user"},
	}

	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err)

	// Aguarda token expirar
	time.Sleep(2 * time.Millisecond)

	// Tentar validar token expirado
	_, err = jwtAuth.ValidateToken(token)
	assert.Error(t, err, "Should fail to validate expired token")
}

func TestJWT_TokenValidation_WrongSecret(t *testing.T) {
	config1 := &JWTConfig{
		SecretKey: "secret-key-1-with-minimum-length-required",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
		RefreshIn: 1 * time.Hour,
	}

	config2 := &JWTConfig{
		SecretKey: "secret-key-2-different-with-minimum-length",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
		RefreshIn: 1 * time.Hour,
	}

	jwtAuth1 := NewJwtAuth(config1)
	jwtAuth2 := NewJwtAuth(config2)

	user := &auth.UserIdentity{
		Username: "testuser",
	}

	// Gerar token com secret1
	token, err := jwtAuth1.GenerateToken(user)
	require.NoError(t, err)

	// Tentar validar com secret2
	_, err = jwtAuth2.ValidateToken(token)
	assert.Error(t, err, "Should fail to validate token with wrong secret")
}

func TestJWT_CustomClaims(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "testuser",
		Custom: map[string]interface{}{
			"user_id":    42,
			"department": "Engineering",
			"level":      5,
		},
	}

	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err)

	validated, err := jwtAuth.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, 42, int(validated.Custom["user_id"].(float64)))
	assert.Equal(t, "Engineering", validated.Custom["department"])
	assert.Equal(t, 5, int(validated.Custom["level"].(float64)))
}

func TestJWT_RolesAndScopes(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "testuser",
		Roles:    []string{"admin", "moderator", "user"},
		Scopes:   []string{"read", "write", "delete"},
		Admin:    true,
	}

	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err)

	validated, err := jwtAuth.ValidateToken(token)
	require.NoError(t, err)

	// Verificar roles
	assert.True(t, validated.HasRole("admin"))
	assert.True(t, validated.HasRole("moderator"))
	assert.True(t, validated.HasRole("user"))
	assert.False(t, validated.HasRole("superadmin"))

	// Verificar scopes
	assert.True(t, validated.HasScope("read"))
	assert.True(t, validated.HasScope("write"))
	assert.True(t, validated.HasScope("delete"))
	assert.False(t, validated.HasScope("execute"))

	// Verificar admin
	assert.True(t, validated.IsAdmin())
}

func TestJWT_CustomTokenGenerator(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	// Salvar referência ao gerador padrão
	defaultGenerator := jwtAuth.DefaultGenerateToken

	// Definir gerador customizado que adiciona claim extra
	jwtAuth.TokenGenerator = func(user *auth.UserIdentity) (string, error) {
		// Adicionar claim customizado
		if user.Custom == nil {
			user.Custom = make(map[string]interface{})
		}
		user.Custom["generated_by"] = "custom_generator"

		// Usar gerador padrão
		return defaultGenerator(user)
	}

	user := &auth.UserIdentity{
		Username: "testuser",
	}

	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err)

	validated, err := jwtAuth.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, "custom_generator", validated.Custom["generated_by"])
}

func TestJWT_EmptyUsername(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "", // Username vazio
		Roles:    []string{"user"},
	}

	token, err := jwtAuth.GenerateToken(user)
	assert.NoError(t, err, "Should allow empty username")
	assert.NotEmpty(t, token)
}

func TestJWT_NoRolesOrScopes(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key-very-long-for-security",
		Issuer:    "test-issuer",
		ExpiresIn: 15 * time.Minute,
	}

	jwtAuth := NewJwtAuth(config)

	user := &auth.UserIdentity{
		Username: "testuser",
		// Sem roles ou scopes
	}

	token, err := jwtAuth.GenerateToken(user)
	require.NoError(t, err)

	validated, err := jwtAuth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "testuser", validated.Username)
	assert.Empty(t, validated.Roles)
	assert.Empty(t, validated.Scopes)
}

// ============================================================================
// Tests for .env configuration loading
// ============================================================================

func TestLoadConfigFromEnv(t *testing.T) {
	// Testa sem variáveis de ambiente (usa defaults)
	config := LoadConfigFromEnv()

	require.NotNil(t, config)
	assert.NotEmpty(t, config.SecretKey)
	assert.Equal(t, "go-data-server", config.Issuer)
	assert.Equal(t, 24*time.Hour, config.ExpiresIn)
	assert.Equal(t, 7*24*time.Hour, config.RefreshIn)
	assert.Equal(t, "HS256", config.Algorithm)
}

func TestMergeConfig(t *testing.T) {
	base := &JWTConfig{
		SecretKey: "base-secret-key-with-32-chars-minimum",
		Issuer:    "base-issuer",
		ExpiresIn: 1 * time.Hour,
		RefreshIn: 24 * time.Hour,
		Algorithm: "HS256",
	}

	t.Run("merge with nil returns base", func(t *testing.T) {
		result := MergeConfig(base, nil)
		assert.Equal(t, base, result)
	})

	t.Run("merge overrides non-zero values", func(t *testing.T) {
		custom := &JWTConfig{
			ExpiresIn: 2 * time.Hour, // Override apenas expiration
		}

		result := MergeConfig(base, custom)

		assert.Equal(t, base.SecretKey, result.SecretKey)
		assert.Equal(t, base.Issuer, result.Issuer)
		assert.Equal(t, 2*time.Hour, result.ExpiresIn) // Overridden
		assert.Equal(t, base.RefreshIn, result.RefreshIn)
		assert.Equal(t, base.Algorithm, result.Algorithm)
	})

	t.Run("merge overrides all values", func(t *testing.T) {
		custom := &JWTConfig{
			SecretKey: "custom-secret-key-with-32-chars",
			Issuer:    "custom-issuer",
			ExpiresIn: 30 * time.Minute,
			RefreshIn: 48 * time.Hour,
			Algorithm: "HS512",
		}

		result := MergeConfig(base, custom)

		assert.Equal(t, custom.SecretKey, result.SecretKey)
		assert.Equal(t, custom.Issuer, result.Issuer)
		assert.Equal(t, custom.ExpiresIn, result.ExpiresIn)
		assert.Equal(t, custom.RefreshIn, result.RefreshIn)
		assert.Equal(t, custom.Algorithm, result.Algorithm)
	})
}

func TestJWTConfig_Validate(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		config := &JWTConfig{
			SecretKey: "valid-secret-key-with-32-chars",
			ExpiresIn: 1 * time.Hour,
			RefreshIn: 24 * time.Hour,
		}

		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty secret key fails", func(t *testing.T) {
		config := &JWTConfig{
			SecretKey: "",
			ExpiresIn: 1 * time.Hour,
			RefreshIn: 24 * time.Hour,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "secret key is required")
	})

	t.Run("short secret key fails", func(t *testing.T) {
		config := &JWTConfig{
			SecretKey: "short",
			ExpiresIn: 1 * time.Hour,
			RefreshIn: 24 * time.Hour,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 16 characters")
	})

	t.Run("zero expiration fails", func(t *testing.T) {
		config := &JWTConfig{
			SecretKey: "valid-secret-key-with-32-chars",
			ExpiresIn: 0,
			RefreshIn: 24 * time.Hour,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expiration must be positive")
	})

	t.Run("refresh less than expiration fails", func(t *testing.T) {
		config := &JWTConfig{
			SecretKey: "valid-secret-key-with-32-chars",
			ExpiresIn: 24 * time.Hour,
			RefreshIn: 1 * time.Hour,
		}

		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refresh expiration")
	})
}

func TestNewJwtAuth_WithConfigMerge(t *testing.T) {
	t.Run("nil config uses env defaults", func(t *testing.T) {
		jwtAuth := NewJwtAuth(nil)

		require.NotNil(t, jwtAuth)
		require.NotNil(t, jwtAuth.config)
		assert.NotEmpty(t, jwtAuth.config.SecretKey)
	})

	t.Run("custom config overrides env", func(t *testing.T) {
		customConfig := &JWTConfig{
			SecretKey: "custom-secret-key-with-32-chars-minimum",
			Issuer:    "custom-issuer",
			ExpiresIn: 30 * time.Minute,
			RefreshIn: 48 * time.Hour,
		}

		jwtAuth := NewJwtAuth(customConfig)

		require.NotNil(t, jwtAuth)
		assert.Equal(t, customConfig.SecretKey, jwtAuth.config.SecretKey)
		assert.Equal(t, customConfig.Issuer, jwtAuth.config.Issuer)
		assert.Equal(t, customConfig.ExpiresIn, jwtAuth.config.ExpiresIn)
	})
}
