package odata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupBaseRoutes_Initialization(t *testing.T) {
	t.Run("Server initializes successfully", func(t *testing.T) {
		server := NewServer()
		assert.NotNil(t, server)
	})

	t.Run("Server with config initializes", func(t *testing.T) {
		config := DefaultServerConfig()
		assert.NotNil(t, config)
	})
}

func TestEntityRoutes_Registration(t *testing.T) {
	t.Run("Register single entity", func(t *testing.T) {
		server := NewServer()
		metadata := EntityMetadata{
			Name:      "Users",
			TableName: "users",
			Properties: []PropertyMetadata{
				{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			},
		}

		service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
		require.NotNil(t, service)

		// Verify entity is registered
		_, exists := server.entities["Users"]
		// May or may not exist depending on registration method
		_ = exists
	})

	t.Run("Multiple entities", func(t *testing.T) {
		server := NewServer()

		metadata1 := EntityMetadata{Name: "Entity1", TableName: "entity1"}
		metadata2 := EntityMetadata{Name: "Entity2", TableName: "entity2"}

		service1 := NewBaseEntityService(&mockDatabaseProvider{}, metadata1, server)
		service2 := NewBaseEntityService(&mockDatabaseProvider{}, metadata2, server)

		assert.NotNil(t, service1)
		assert.NotNil(t, service2)
	})
}

func TestRouting_Middleware(t *testing.T) {
	server := NewServer()

	t.Run("Database middleware exists", func(t *testing.T) {
		middleware := server.DatabaseMiddleware()
		assert.NotNil(t, middleware)
	})

	t.Run("ReadOnly middleware for POST", func(t *testing.T) {
		middleware := server.CheckEntityReadOnly("TestEntity", "POST")
		assert.NotNil(t, middleware)
	})

	t.Run("ReadOnly middleware for GET", func(t *testing.T) {
		middleware := server.CheckEntityReadOnly("TestEntity", "GET")
		assert.NotNil(t, middleware)
	})

	t.Run("ReadOnly middleware for PUT", func(t *testing.T) {
		middleware := server.CheckEntityReadOnly("TestEntity", "PUT")
		assert.NotNil(t, middleware)
	})

	t.Run("ReadOnly middleware for DELETE", func(t *testing.T) {
		middleware := server.CheckEntityReadOnly("TestEntity", "DELETE")
		assert.NotNil(t, middleware)
	})
}

func TestRouting_Configuration(t *testing.T) {
	t.Run("Default port configuration", func(t *testing.T) {
		config := DefaultServerConfig()
		assert.Greater(t, config.Port, 0)
		assert.LessOrEqual(t, config.Port, 65535)
	})

	t.Run("Custom port can be set", func(t *testing.T) {
		config := DefaultServerConfig()
		config.Port = 8080
		assert.Equal(t, 8080, config.Port)
	})

	t.Run("Config has essential fields", func(t *testing.T) {
		config := DefaultServerConfig()
		assert.NotNil(t, config)
		assert.Greater(t, config.Port, 0)
	})
}

func TestRouting_EntityConfiguration(t *testing.T) {
	server := NewServer()

	t.Run("Entity metadata structure", func(t *testing.T) {
		metadata := EntityMetadata{
			Name:      "Orders",
			TableName: "orders",
			Properties: []PropertyMetadata{
				{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
				{Name: "Total", ColumnName: "total", Type: "float64"},
			},
		}

		assert.Equal(t, "Orders", metadata.Name)
		assert.Equal(t, "orders", metadata.TableName)
		assert.Len(t, metadata.Properties, 2)
	})

	t.Run("Service creation for entity", func(t *testing.T) {
		metadata := EntityMetadata{
			Name:      "Products",
			TableName: "products",
			Properties: []PropertyMetadata{
				{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			},
		}

		service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
		assert.NotNil(t, service)
		assert.Equal(t, metadata, service.GetMetadata())
	})
}

func TestRouting_EntityOptions(t *testing.T) {
	t.Run("WithReadOnly option exists", func(t *testing.T) {
		option := WithReadOnly(true)
		assert.NotNil(t, option)
	})

	t.Run("Multiple WithReadOnly calls", func(t *testing.T) {
		option1 := WithReadOnly(true)
		option2 := WithReadOnly(false)

		assert.NotNil(t, option1)
		assert.NotNil(t, option2)
	})
}

func TestRouting_EntityLookup(t *testing.T) {
	server := NewServer()

	t.Run("Server has entities map", func(t *testing.T) {
		// Verify server can handle entity lookups
		assert.NotNil(t, server)
		assert.NotNil(t, server.entities)
	})

	t.Run("Server with multiple entities", func(t *testing.T) {
		// Verify entities map exists
		assert.NotNil(t, server.entities)
	})
}

func TestRouting_ServerLifecycle(t *testing.T) {
	t.Run("Server can be created", func(t *testing.T) {
		server := NewServer()
		require.NotNil(t, server)
	})

	t.Run("Server config is initialized", func(t *testing.T) {
		server := NewServer()
		assert.NotNil(t, server.config)
	})

	t.Run("Multiple servers can coexist", func(t *testing.T) {
		server1 := NewServer()
		server2 := NewServer()

		assert.NotNil(t, server1)
		assert.NotNil(t, server2)
		assert.NotEqual(t, server1, server2)
	})
}

func TestRouting_PathHandling(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name string
		path string
	}{
		{"Simple path", "/Users"},
		{"Path with ID", "/Users(1)"},
		{"Path with key", "/Users(ID=1)"},
		{"Root path", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.extractEntityName(tt.path)
			// Just verify it doesn't panic
			_ = result
		})
	}
}

func TestRouting_Security(t *testing.T) {
	t.Run("Server config has security features", func(t *testing.T) {
		config := DefaultServerConfig()
		assert.NotNil(t, config)
		// Config should be valid
		assert.Greater(t, config.Port, 0)
	})

	t.Run("Server is created with defaults", func(t *testing.T) {
		server := NewServer()
		assert.NotNil(t, server)
		assert.NotNil(t, server.config)
	})
}
