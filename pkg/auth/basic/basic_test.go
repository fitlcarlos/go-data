package basic

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/fitlcarlos/go-data/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func TestBasicAuth_Validation(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	// Validador simples para testes
	validator := func(username, password string) (*auth.UserIdentity, error) {
		if username == "admin" && password == "secret" {
			return &auth.UserIdentity{
				Username: username,
				Roles:    []string{"admin"},
				Admin:    true,
			}, nil
		}
		if username == "user" && password == "pass" {
			return &auth.UserIdentity{
				Username: username,
				Roles:    []string{"user"},
				Admin:    false,
			}, nil
		}
		return nil, ErrInvalidCredentials
	}

	basicAuth := NewBasicAuth(config, validator)

	t.Run("ValidCredentials_Admin", func(t *testing.T) {
		// Criar token Base64: admin:secret
		token := base64.StdEncoding.EncodeToString([]byte("admin:secret"))

		user, err := basicAuth.ValidateToken(token)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "admin", user.Username)
		assert.True(t, user.HasRole("admin"))
		assert.True(t, user.IsAdmin())
	})

	t.Run("ValidCredentials_User", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte("user:pass"))

		user, err := basicAuth.ValidateToken(token)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user", user.Username)
		assert.True(t, user.HasRole("user"))
		assert.False(t, user.IsAdmin())
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte("invalid:wrongpass"))

		_, err := basicAuth.ValidateToken(token)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
	})

	t.Run("WrongPassword", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte("admin:wrongpass"))

		_, err := basicAuth.ValidateToken(token)
		assert.Error(t, err)
	})

	t.Run("EmptyCredentials", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte(""))

		_, err := basicAuth.ValidateToken(token)
		assert.Error(t, err)
	})
}

func TestBasicAuth_Base64Decoding(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	validator := func(username, password string) (*auth.UserIdentity, error) {
		return &auth.UserIdentity{Username: username}, nil
	}

	basicAuth := NewBasicAuth(config, validator)

	t.Run("ValidBase64", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte("user:pass"))

		user, err := basicAuth.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user", user.Username)
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		token := "not-valid-base64!@#$"

		_, err := basicAuth.ValidateToken(token)
		assert.Error(t, err)
	})

	t.Run("NoColon", func(t *testing.T) {
		// Credencial sem ':' separador
		token := base64.StdEncoding.EncodeToString([]byte("userpassword"))

		_, err := basicAuth.ValidateToken(token)
		assert.Error(t, err)
	})

	t.Run("MultipleColons", func(t *testing.T) {
		// Múltiplos ':' - deve usar apenas o primeiro
		token := base64.StdEncoding.EncodeToString([]byte("user:pass:word"))

		user, err := basicAuth.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "user", user.Username)
	})
}

func TestBasicAuth_CustomValidator(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	// Validador que adiciona custom claims
	validator := func(username, password string) (*auth.UserIdentity, error) {
		if username == "admin" && password == "secret" {
			return &auth.UserIdentity{
				Username: username,
				Roles:    []string{"admin"},
				Scopes:   []string{"read", "write", "delete"},
				Admin:    true,
				Custom: map[string]interface{}{
					"department": "IT",
					"level":      5,
				},
			}, nil
		}
		return nil, ErrInvalidCredentials
	}

	basicAuth := NewBasicAuth(config, validator)

	token := base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	user, err := basicAuth.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, "admin", user.Username)
	assert.True(t, user.HasScope("read"))
	assert.True(t, user.HasScope("write"))
	assert.True(t, user.HasScope("delete"))
	assert.Equal(t, "IT", user.Custom["department"])
	assert.Equal(t, 5, user.Custom["level"])
}

func TestBasicAuth_EmptyPassword(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	validator := func(username, password string) (*auth.UserIdentity, error) {
		// Permitir senha vazia para usuário específico
		if username == "guest" && password == "" {
			return &auth.UserIdentity{
				Username: username,
				Roles:    []string{"guest"},
			}, nil
		}
		return nil, ErrInvalidCredentials
	}

	basicAuth := NewBasicAuth(config, validator)

	t.Run("GuestWithEmptyPassword", func(t *testing.T) {
		token := base64.StdEncoding.EncodeToString([]byte("guest:"))

		user, err := basicAuth.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "guest", user.Username)
	})
}

func TestBasicAuth_GenerateToken(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	validator := func(username, password string) (*auth.UserIdentity, error) {
		return &auth.UserIdentity{Username: username}, nil
	}

	basicAuth := NewBasicAuth(config, validator)

	user := &auth.UserIdentity{
		Username: "testuser",
	}

	// Basic Auth normalmente não gera tokens, mas implementa a interface
	token, err := basicAuth.GenerateToken(user)

	// Pode retornar vazio ou erro, dependendo da implementação
	// Aqui apenas verificamos que não quebra
	_ = token
	_ = err
}

func TestBasicAuth_Realm(t *testing.T) {
	t.Run("CustomRealm", func(t *testing.T) {
		config := &BasicAuthConfig{
			Realm: "My Custom Realm",
		}

		validator := func(username, password string) (*auth.UserIdentity, error) {
			return &auth.UserIdentity{Username: username}, nil
		}

		basicAuth := NewBasicAuth(config, validator)
		assert.Equal(t, "My Custom Realm", basicAuth.config.Realm)
	})

	t.Run("EmptyRealm", func(t *testing.T) {
		config := &BasicAuthConfig{
			Realm: "",
		}

		validator := func(username, password string) (*auth.UserIdentity, error) {
			return &auth.UserIdentity{Username: username}, nil
		}

		basicAuth := NewBasicAuth(config, validator)
		assert.Equal(t, "", basicAuth.config.Realm)
	})
}

func TestBasicAuth_NilValidator(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	// Criar Basic Auth sem validator deve causar panic
	assert.Panics(t, func() {
		NewBasicAuth(config, nil)
	}, "NewBasicAuth should panic with nil validator")
}

func TestBasicAuth_SpecialCharactersInPassword(t *testing.T) {
	config := &BasicAuthConfig{
		Realm: "Test Realm",
	}

	validator := func(username, password string) (*auth.UserIdentity, error) {
		if username == "user" && password == "p@ss:w0rd!#$%^&*()" {
			return &auth.UserIdentity{Username: username}, nil
		}
		return nil, ErrInvalidCredentials
	}

	basicAuth := NewBasicAuth(config, validator)

	// Senha com caracteres especiais incluindo ':'
	token := base64.StdEncoding.EncodeToString([]byte("user:p@ss:w0rd!#$%^&*()"))

	user, err := basicAuth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user", user.Username)
}
