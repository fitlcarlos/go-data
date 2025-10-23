package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiTenantEntityService_Creation(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}

	t.Run("Create multi-tenant service", func(t *testing.T) {
		mtService := NewMultiTenantEntityService(metadata, server)

		assert.NotNil(t, mtService)
		assert.Equal(t, metadata, mtService.GetMetadata())
	})

	t.Run("Service wraps base service", func(t *testing.T) {
		mtService := NewMultiTenantEntityService(metadata, server)

		// Verify it has BaseEntityService
		assert.NotNil(t, mtService.BaseEntityService)
	})
}

func TestMultiTenantEntityService_Query(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Products",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)
	ctx := context.Background()

	t.Run("Query without tenant", func(t *testing.T) {
		options := QueryOptions{}

		_, err := mtService.Query(ctx, options)
		// May fail without tenant context, but should not panic
		_ = err
	})

	t.Run("Query with options", func(t *testing.T) {
		top := GoDataTopQuery(10)
		options := QueryOptions{
			Top: &top,
		}

		_, err := mtService.Query(ctx, options)
		_ = err
	})
}

func TestMultiTenantEntityService_Get(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Orders",
		TableName: "orders",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)
	ctx := context.Background()

	t.Run("Get entity by key", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}

		_, err := mtService.Get(ctx, keys)
		// May fail without proper setup
		_ = err
	})

	t.Run("Get with composite keys", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID":   int64(1),
			"Type": "standard",
		}

		_, err := mtService.Get(ctx, keys)
		_ = err
	})
}

func TestMultiTenantEntityService_Create(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Customers",
		TableName: "customers",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)
	ctx := context.Background()

	t.Run("Create entity", func(t *testing.T) {
		entity := map[string]interface{}{
			"Name": "Customer 1",
		}

		_, err := mtService.Create(ctx, entity)
		// May fail without proper setup
		_ = err
	})

	t.Run("Create with nil entity", func(t *testing.T) {
		_, err := mtService.Create(ctx, nil)
		assert.Error(t, err)
	})
}

func TestMultiTenantEntityService_Update(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Inventory",
		TableName: "inventory",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Quantity", ColumnName: "quantity", Type: "int"},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)
	ctx := context.Background()

	t.Run("Update entity", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}
		entity := map[string]interface{}{
			"Quantity": 100,
		}

		_, err := mtService.Update(ctx, keys, entity)
		// May fail without proper setup
		_ = err
	})

	t.Run("Update with nil entity", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}

		_, err := mtService.Update(ctx, keys, nil)
		assert.Error(t, err)
	})

	t.Run("Update with empty keys", func(t *testing.T) {
		keys := map[string]interface{}{}
		entity := map[string]interface{}{
			"Quantity": 50,
		}

		_, err := mtService.Update(ctx, keys, entity)
		_ = err
	})
}

func TestMultiTenantEntityService_Delete(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Logs",
		TableName: "logs",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)
	ctx := context.Background()

	t.Run("Delete entity", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}

		err := mtService.Delete(ctx, keys)
		// May fail without proper setup
		_ = err
	})

	t.Run("Delete with empty keys", func(t *testing.T) {
		keys := map[string]interface{}{}

		err := mtService.Delete(ctx, keys)
		_ = err
	})
}

func TestMultiTenantEntityService_Metadata(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Tenants",
		TableName: "tenants",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "string", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Active", ColumnName: "active", Type: "bool"},
		},
	}

	t.Run("GetMetadata returns correct info", func(t *testing.T) {
		mtService := NewMultiTenantEntityService(metadata, server)

		result := mtService.GetMetadata()

		assert.Equal(t, "Tenants", result.Name)
		assert.Equal(t, "tenants", result.TableName)
		assert.Len(t, result.Properties, 3)
	})

	t.Run("Metadata has correct property types", func(t *testing.T) {
		mtService := NewMultiTenantEntityService(metadata, server)

		result := mtService.GetMetadata()

		// Verify property types
		for _, prop := range result.Properties {
			assert.NotEmpty(t, prop.Type)
			if prop.IsKey {
				assert.Equal(t, "ID", prop.Name)
			}
		}
	})
}

func TestMultiTenantEntityService_ContextHandling(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Documents",
		TableName: "documents",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)

	t.Run("Operations with background context", func(t *testing.T) {
		ctx := context.Background()
		options := QueryOptions{}

		_, err := mtService.Query(ctx, options)
		// May fail, but should not panic
		_ = err
	})

	t.Run("Operations with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		options := QueryOptions{}
		_, err := mtService.Query(ctx, options)
		// Should handle canceled context gracefully
		_ = err
	})
}

func TestMultiTenantEntityService_EdgeCases(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "EdgeCase",
		TableName: "edge_case",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
		},
	}

	t.Run("Service initialization", func(t *testing.T) {
		mtService := NewMultiTenantEntityService(metadata, server)

		assert.NotNil(t, mtService)
		assert.NotNil(t, mtService.BaseEntityService)
	})

	t.Run("Multiple operations in sequence", func(t *testing.T) {
		mtService := NewMultiTenantEntityService(metadata, server)
		ctx := context.Background()

		// Query
		_, _ = mtService.Query(ctx, QueryOptions{})

		// Create
		_, _ = mtService.Create(ctx, map[string]interface{}{"Name": "Test"})

		// Update
		_, _ = mtService.Update(ctx, map[string]interface{}{"ID": int64(1)}, map[string]interface{}{"Name": "Updated"})

		// Delete
		_ = mtService.Delete(ctx, map[string]interface{}{"ID": int64(1)})

		// All should complete without panic
	})
}

func TestMultiTenantEntityService_Workflow(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Tasks",
		TableName: "tasks",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Title", ColumnName: "title", Type: "string"},
			{Name: "Status", ColumnName: "status", Type: "string"},
		},
	}

	mtService := NewMultiTenantEntityService(metadata, server)
	ctx := context.Background()

	t.Run("Full CRUD workflow", func(t *testing.T) {
		// Create
		newTask := map[string]interface{}{
			"Title":  "New Task",
			"Status": "pending",
		}
		_, createErr := mtService.Create(ctx, newTask)

		// Query
		top := GoDataTopQuery(10)
		queryOptions := QueryOptions{
			Top:     &top,
			OrderBy: "Title asc",
		}
		_, queryErr := mtService.Query(ctx, queryOptions)

		// Get
		keys := map[string]interface{}{"ID": int64(1)}
		_, getErr := mtService.Get(ctx, keys)

		// Update
		updateData := map[string]interface{}{"Status": "completed"}
		_, updateErr := mtService.Update(ctx, keys, updateData)

		// Delete
		deleteErr := mtService.Delete(ctx, keys)

		// All may fail with mock, but should not panic
		_ = createErr
		_ = queryErr
		_ = getErr
		_ = updateErr
		_ = deleteErr
	})
}
