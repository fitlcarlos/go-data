package odata

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/fitlcarlos/go-data/pkg/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerCreation(t *testing.T) {
	t.Run("NewServer_Default", func(t *testing.T) {
		server := NewServer()

		assert.NotNil(t, server)
		assert.NotNil(t, server.entities)
		assert.NotNil(t, server.config)
	})

	t.Run("NewServerWithProvider", func(t *testing.T) {
		mockProvider := &mockDatabaseProvider{}

		server := NewServerWithProvider(mockProvider, "localhost", 8080, "/odata")

		assert.NotNil(t, server)
		assert.NotNil(t, server.entities)
	})

	t.Run("DefaultServerConfig", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.NotNil(t, config)
		assert.NotNil(t, config.RateLimitConfig)
		assert.NotNil(t, config.ValidationConfig)
		assert.NotNil(t, config.SecurityHeadersConfig)
		assert.NotNil(t, config.AuditLogConfig)

		// Verificar valores padrão de segurança
		assert.True(t, config.RateLimitConfig.Enabled, "Rate limit should be enabled by default")
		assert.False(t, config.DisableJoinForExpand, "JOIN for expand should be enabled by default")
	})
}

func TestServerRegistration(t *testing.T) {
	// Definir uma entidade de teste simples
	type TestEntity struct {
		ID   int    `odata:"id,key"`
		Name string `odata:"name"`
	}

	t.Run("RegisterEntity_Simple", func(t *testing.T) {
		server := NewServer()

		err := server.RegisterEntity("TestEntity", TestEntity{})
		require.NoError(t, err)

		// Verificar se entidade foi registrada
		assert.Contains(t, server.entities, "TestEntity")

		// Verificar se metadata foi criado
		service := server.entities["TestEntity"]
		assert.NotNil(t, service)

		metadata := service.GetMetadata()
		assert.Equal(t, "TestEntity", metadata.Name)
		assert.NotEmpty(t, metadata.Properties)
	})

	t.Run("RegisterEntity_WithAuth", func(t *testing.T) {
		server := NewServer()

		// Mock auth provider
		mockAuth := &mockAuthProvider{}

		err := server.RegisterEntity("SecureEntity", TestEntity{}, WithAuth(mockAuth))
		require.NoError(t, err)

		assert.Contains(t, server.entities, "SecureEntity")
		assert.Contains(t, server.entityAuth, "SecureEntity")
	})

	t.Run("RegisterEntity_WithReadOnly", func(t *testing.T) {
		server := NewServer()

		err := server.RegisterEntity("ReadOnlyEntity", TestEntity{}, WithReadOnly(true))
		require.NoError(t, err)

		assert.Contains(t, server.entities, "ReadOnlyEntity")

		// Verificar configuração de read-only na entityAuth
		authConfig, exists := server.entityAuth["ReadOnlyEntity"]
		if exists {
			assert.True(t, authConfig.ReadOnly)
		}
	})

	t.Run("RegisterEntity_MultipleOptions", func(t *testing.T) {
		server := NewServer()
		mockAuth := &mockAuthProvider{}

		err := server.RegisterEntity("ComplexEntity", TestEntity{},
			WithAuth(mockAuth),
			WithReadOnly(true),
		)
		require.NoError(t, err)

		assert.Contains(t, server.entities, "ComplexEntity")
		assert.Contains(t, server.entityAuth, "ComplexEntity")

		authConfig := server.entityAuth["ComplexEntity"]
		assert.True(t, authConfig.ReadOnly)
	})

	t.Run("RegisterEntity_DuplicateName", func(t *testing.T) {
		server := NewServer()

		err := server.RegisterEntity("TestEntity", TestEntity{})
		require.NoError(t, err)

		// Tentar registrar novamente com mesmo nome
		err = server.RegisterEntity("TestEntity", TestEntity{})
		assert.NoError(t, err, "Should allow re-registering entity")
	})

	t.Run("RegisterEntity_InvalidStruct", func(t *testing.T) {
		server := NewServer()

		// Tentar registrar algo que não é struct
		err := server.RegisterEntity("Invalid", "not a struct")
		assert.Error(t, err, "Should fail to register non-struct")
	})
}

func TestServerConfiguration(t *testing.T) {
	t.Run("RateLimitConfig", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.NotNil(t, config.RateLimitConfig)
		assert.True(t, config.RateLimitConfig.Enabled)
		assert.Greater(t, config.RateLimitConfig.RequestsPerMinute, 0)
		assert.Greater(t, config.RateLimitConfig.BurstSize, 0)
	})

	t.Run("ValidationConfig", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.NotNil(t, config.ValidationConfig)
		assert.Greater(t, config.ValidationConfig.MaxFilterLength, 0)
		assert.Greater(t, config.ValidationConfig.MaxExpandDepth, 0)
		assert.Greater(t, config.ValidationConfig.MaxTopValue, 0)
	})

	t.Run("SecurityHeadersConfig", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.NotNil(t, config.SecurityHeadersConfig)
		assert.True(t, config.SecurityHeadersConfig.Enabled)
		assert.NotEmpty(t, config.SecurityHeadersConfig.XFrameOptions)
		assert.NotEmpty(t, config.SecurityHeadersConfig.XContentTypeOptions)
	})

	t.Run("AuditLogConfig", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.NotNil(t, config.AuditLogConfig)
		// Audit log pode estar desabilitado por padrão
	})
}

func TestServerSecurityHeaders(t *testing.T) {
	t.Run("SecurityHeaders_Enabled", func(t *testing.T) {
		config := DefaultServerConfig()
		config.SecurityHeadersConfig.Enabled = true

		assert.True(t, config.SecurityHeadersConfig.Enabled)
	})

	t.Run("SecurityHeaders_Disabled", func(t *testing.T) {
		config := DefaultServerConfig()
		config.SecurityHeadersConfig.Enabled = false

		assert.False(t, config.SecurityHeadersConfig.Enabled)
	})

	t.Run("SecurityHeaders_CustomValues", func(t *testing.T) {
		config := DefaultServerConfig()
		config.SecurityHeadersConfig.Enabled = true
		config.SecurityHeadersConfig.XFrameOptions = "DENY"
		config.SecurityHeadersConfig.XContentTypeOptions = "nosniff"

		assert.Equal(t, "DENY", config.SecurityHeadersConfig.XFrameOptions)
		assert.Equal(t, "nosniff", config.SecurityHeadersConfig.XContentTypeOptions)
	})
}

func TestServerCORS(t *testing.T) {
	t.Run("CORS_EnabledByDefault", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.True(t, config.EnableCORS, "CORS should be enabled by default for APIs")
	})

	t.Run("CORS_Enabled", func(t *testing.T) {
		config := DefaultServerConfig()
		config.EnableCORS = true

		assert.True(t, config.EnableCORS)
	})
}

func TestServerLogging(t *testing.T) {
	t.Run("Logging_Enabled", func(t *testing.T) {
		config := DefaultServerConfig()
		config.EnableLogging = true

		assert.True(t, config.EnableLogging)
	})

	t.Run("Logging_Disabled", func(t *testing.T) {
		config := DefaultServerConfig()
		config.EnableLogging = false

		assert.False(t, config.EnableLogging)
	})
}

func TestEntityOptions(t *testing.T) {
	type TestEntity struct {
		ID   int    `odata:"id,key"`
		Name string `odata:"name"`
	}

	t.Run("WithAuth", func(t *testing.T) {
		server := NewServer()
		mockAuth := &mockAuthProvider{}

		err := server.RegisterEntity("Test", TestEntity{}, WithAuth(mockAuth))
		require.NoError(t, err)

		assert.Contains(t, server.entityAuth, "Test")
	})

	t.Run("WithReadOnly", func(t *testing.T) {
		server := NewServer()

		err := server.RegisterEntity("Test", TestEntity{}, WithReadOnly(true))
		require.NoError(t, err)

		authConfig, exists := server.entityAuth["Test"]
		if exists {
			assert.True(t, authConfig.ReadOnly)
		}
	})

	t.Run("WithReadOnly_False", func(t *testing.T) {
		server := NewServer()

		err := server.RegisterEntity("Test", TestEntity{}, WithReadOnly(false))
		require.NoError(t, err)

		authConfig, exists := server.entityAuth["Test"]
		if exists {
			assert.False(t, authConfig.ReadOnly)
		}
	})

	t.Run("MultipleOptions", func(t *testing.T) {
		server := NewServer()
		mockAuth := &mockAuthProvider{}

		err := server.RegisterEntity("Test", TestEntity{},
			WithAuth(mockAuth),
			WithReadOnly(true),
		)
		require.NoError(t, err)

		assert.Contains(t, server.entityAuth, "Test")
		authConfig := server.entityAuth["Test"]
		assert.True(t, authConfig.ReadOnly)
	})
}

func TestServerEntityCount(t *testing.T) {
	type TestEntity struct {
		ID   int    `odata:"id,key"`
		Name string `odata:"name"`
	}

	t.Run("NoEntities", func(t *testing.T) {
		server := NewServer()

		assert.Len(t, server.entities, 0)
	})

	t.Run("MultipleEntities", func(t *testing.T) {
		server := NewServer()

		server.RegisterEntity("Entity1", TestEntity{})
		server.RegisterEntity("Entity2", TestEntity{})
		server.RegisterEntity("Entity3", TestEntity{})

		assert.Len(t, server.entities, 3)
		assert.Contains(t, server.entities, "Entity1")
		assert.Contains(t, server.entities, "Entity2")
		assert.Contains(t, server.entities, "Entity3")
	})
}

func TestServerConfigPort(t *testing.T) {
	t.Run("DefaultPort", func(t *testing.T) {
		config := DefaultServerConfig()

		assert.Equal(t, 8080, config.Port)
	})

	t.Run("CustomPort", func(t *testing.T) {
		config := DefaultServerConfig()
		config.Port = 3000

		assert.Equal(t, 3000, config.Port)
	})
}

// Mock AuthProvider para testes
type mockAuthProvider struct{}

func (m *mockAuthProvider) ValidateToken(token string) (*auth.UserIdentity, error) {
	if token == "valid-token" {
		return &auth.UserIdentity{Username: "testuser"}, nil
	}
	return nil, assert.AnError
}

func (m *mockAuthProvider) GenerateToken(user *auth.UserIdentity) (string, error) {
	return "generated-token", nil
}

func (m *mockAuthProvider) ExtractToken(c fiber.Ctx) string {
	return c.Get("Authorization")
}

// Mock DatabaseProvider para testes
type mockDatabaseProvider struct{}

func (m *mockDatabaseProvider) Connect(connectionString string) error {
	return nil
}

func (m *mockDatabaseProvider) Close() error {
	return nil
}

func (m *mockDatabaseProvider) GetConnection() *sql.DB {
	return nil
}

func (m *mockDatabaseProvider) GetDriverName() string {
	return "mock"
}

func (m *mockDatabaseProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	// Mock: retorna nil, testes não usam transações reais
	return nil, nil
}

func (m *mockDatabaseProvider) BuildSelectQuery(entity EntityMetadata, options QueryOptions) (string, []interface{}, error) {
	return "", nil, nil
}

func (m *mockDatabaseProvider) BuildInsertQuery(entity EntityMetadata, data map[string]interface{}) (string, []interface{}, error) {
	return "", nil, nil
}

func (m *mockDatabaseProvider) BuildUpdateQuery(entity EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error) {
	return "", nil, nil
}

func (m *mockDatabaseProvider) BuildDeleteQuery(entity EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error) {
	return "", nil, nil
}

func (m *mockDatabaseProvider) BuildWhereClause(filter string, metadata EntityMetadata) (string, []interface{}, error) {
	return "", nil, nil
}

func (m *mockDatabaseProvider) BuildOrderByClause(orderBy string, metadata EntityMetadata) (string, error) {
	return "", nil
}

func (m *mockDatabaseProvider) MapGoTypeToSQL(goType string) string {
	return "TEXT"
}

func (m *mockDatabaseProvider) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
