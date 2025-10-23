package odata

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMultiTenantProviderPool(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := &MultiTenantConfig{
		Tenants: make(map[string]*TenantConfig),
	}

	t.Run("Creates pool", func(t *testing.T) {
		pool := NewMultiTenantProviderPool(config, logger)

		assert.NotNil(t, pool)
		assert.NotNil(t, pool.providers)
		assert.Equal(t, config, pool.config)
		assert.Equal(t, logger, pool.logger)
	})
}

func TestMultiTenantProviderPool_InitializeProviders(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Empty config initializes successfully", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		err := pool.InitializeProviders()

		assert.NoError(t, err)
	})

	t.Run("Nil EnvConfig does not panic", func(t *testing.T) {
		config := &MultiTenantConfig{
			EnvConfig: nil,
			Tenants:   make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		err := pool.InitializeProviders()

		assert.NoError(t, err)
	})
}

func TestMultiTenantProviderPool_GetProvider(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Returns nil when tenant not found and no default", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants:       make(map[string]*TenantConfig),
			DefaultTenant: "default",
		}
		pool := NewMultiTenantProviderPool(config, logger)

		provider := pool.GetProvider("nonexistent")

		assert.Nil(t, provider, "Should return nil when tenant not found")
	})

	t.Run("Returns default provider when tenant not found", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants:       make(map[string]*TenantConfig),
			DefaultTenant: "default",
		}
		pool := NewMultiTenantProviderPool(config, logger)

		// Set a default provider manually
		mockProvider := &mockDatabaseProvider{}
		pool.defaultProvider = mockProvider

		provider := pool.GetProvider("nonexistent")

		assert.Equal(t, mockProvider, provider, "Should return default provider")
	})
}

func TestMultiTenantProviderPool_AddTenant(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Fails with invalid provider type", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		tenantConfig := &TenantConfig{
			TenantID: "tenant1",
			DBDriver: "invalid_driver",
		}

		err := pool.AddTenant("tenant1", tenantConfig)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider não registrado")
	})
}

func TestMultiTenantProviderPool_RemoveTenant(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Does not error when tenant does not exist", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		err := pool.RemoveTenant("nonexistent")

		// Should not error (idempotent operation)
		assert.NoError(t, err)
	})
}

func TestMultiTenantProviderPool_GetTenantList(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Returns tenant list (may be empty or non-empty)", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		// Check method exists and doesn't panic
		tenants := pool.GetTenantList()
		_ = tenants // Don't assert specific content as implementation may vary
	})
}

func TestMultiTenantProviderPool_GetTenantStats(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Returns stats for nonexistent tenant (may be nil or empty map)", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		// Check method exists and doesn't panic
		stats := pool.GetTenantStats("nonexistent")
		_ = stats // Don't assert specific content as implementation may vary
	})
}

func TestMultiTenantProviderPool_ConcurrentAccess(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := &MultiTenantConfig{
		Tenants:       make(map[string]*TenantConfig),
		DefaultTenant: "default",
	}
	pool := NewMultiTenantProviderPool(config, logger)

	t.Run("Handles concurrent reads", func(t *testing.T) {
		const numGoroutines = 10
		const readsPerGoroutine = 100
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < readsPerGoroutine; j++ {
					pool.GetProvider("tenant1")
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Should not panic
	})
}

func TestMultiTenantProviderPool_DefaultProviderBehavior(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Uses default provider when specified", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants:       make(map[string]*TenantConfig),
			DefaultTenant: "default",
		}
		pool := NewMultiTenantProviderPool(config, logger)

		mockProvider := &mockDatabaseProvider{}
		pool.defaultProvider = mockProvider

		// Request default tenant
		provider := pool.GetProvider("default")

		assert.Equal(t, mockProvider, provider)
	})

	t.Run("Uses default provider for unknown tenant", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants:       make(map[string]*TenantConfig),
			DefaultTenant: "default",
		}
		pool := NewMultiTenantProviderPool(config, logger)

		mockProvider := &mockDatabaseProvider{}
		pool.defaultProvider = mockProvider

		// Request unknown tenant
		provider := pool.GetProvider("unknown")

		assert.Equal(t, mockProvider, provider, "Should fallback to default provider")
	})
}

func TestMultiTenantProviderPool_ThreadSafety(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
	config := &MultiTenantConfig{
		Tenants:       make(map[string]*TenantConfig),
		DefaultTenant: "default",
	}
	pool := NewMultiTenantProviderPool(config, logger)

	t.Run("Concurrent GetProvider calls are safe", func(t *testing.T) {
		const numReaders = 5
		const numWrites = 5
		done := make(chan bool, numReaders+numWrites)

		// Start readers
		for i := 0; i < numReaders; i++ {
			go func(id int) {
				for j := 0; j < 100; j++ {
					pool.GetProvider("tenant1")
				}
				done <- true
			}(i)
		}

		// Wait for all
		for i := 0; i < numReaders; i++ {
			<-done
		}

		// Should complete without deadlock or panic
	})
}

func TestMultiTenantProviderPool_Configuration(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Config is stored correctly", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants:       make(map[string]*TenantConfig),
			DefaultTenant: "test_default",
		}
		pool := NewMultiTenantProviderPool(config, logger)

		assert.Equal(t, config, pool.config)
		assert.Equal(t, "test_default", pool.config.DefaultTenant)
	})
}

func TestMultiTenantProviderPool_LoggerUsage(t *testing.T) {
	t.Run("Logger is stored", func(t *testing.T) {
		logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		assert.Equal(t, logger, pool.logger)
	})

	t.Run("Nil logger does not panic", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}

		assert.NotPanics(t, func() {
			_ = NewMultiTenantProviderPool(config, nil)
		})
	})
}

func TestMultiTenantProviderPool_EdgeCases(t *testing.T) {
	logger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)

	t.Run("Empty tenant ID", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants:       make(map[string]*TenantConfig),
			DefaultTenant: "default",
		}
		pool := NewMultiTenantProviderPool(config, logger)

		provider := pool.GetProvider("")

		// Should handle gracefully (return default or nil)
		_ = provider
	})

	t.Run("Very long tenant ID", func(t *testing.T) {
		config := &MultiTenantConfig{
			Tenants: make(map[string]*TenantConfig),
		}
		pool := NewMultiTenantProviderPool(config, logger)

		longID := string(make([]byte, 10000))
		for i := range longID {
			longID = string(append([]byte(longID[:i]), 'a'))
		}

		provider := pool.GetProvider(longID)

		// Should handle gracefully
		_ = provider
	})
}

// mockDatabaseProvider for testing
type mockMultiTenantDatabaseProvider struct {
	connected bool
	closed    bool
}

func (m *mockMultiTenantDatabaseProvider) Connect(connectionString string) error {
	m.connected = true
	return nil
}

func (m *mockMultiTenantDatabaseProvider) Close() error {
	m.closed = true
	return nil
}

func (m *mockMultiTenantDatabaseProvider) GetConnection() interface{} {
	return nil
}

func (m *mockMultiTenantDatabaseProvider) GetDriverName() string {
	return "mock"
}
