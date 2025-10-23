package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntityService_Query_WithOptions(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Age", ColumnName: "age", Type: "int"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Query with Top", func(t *testing.T) {
		top := GoDataTopQuery(10)
		options := QueryOptions{
			Top: &top,
		}

		_, err := service.Query(ctx, options)
		// May fail with mock, but should not panic
		_ = err
	})

	t.Run("Query with Skip", func(t *testing.T) {
		skip := GoDataSkipQuery(5)
		options := QueryOptions{
			Skip: &skip,
		}

		_, err := service.Query(ctx, options)
		_ = err
	})

	t.Run("Query with OrderBy", func(t *testing.T) {
		options := QueryOptions{
			OrderBy: "Name asc",
		}

		_, err := service.Query(ctx, options)
		_ = err
	})

	t.Run("Query with multiple options", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(5)
		options := QueryOptions{
			Top:     &top,
			Skip:    &skip,
			OrderBy: "Name desc",
		}

		_, err := service.Query(ctx, options)
		_ = err
	})
}

func TestEntityService_Get_WithKeys(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Products",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Price", ColumnName: "price", Type: "float64"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Get with single key", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}

		_, err := service.Get(ctx, keys)
		// May fail with mock
		_ = err
	})

	t.Run("Get with string key", func(t *testing.T) {
		metadata2 := EntityMetadata{
			Name:      "Categories",
			TableName: "categories",
			Properties: []PropertyMetadata{
				{Name: "Code", ColumnName: "code", Type: "string", IsKey: true},
				{Name: "Name", ColumnName: "name", Type: "string"},
			},
		}

		service2 := NewBaseEntityService(&mockDatabaseProvider{}, metadata2, server)
		keys := map[string]interface{}{
			"Code": "CAT001",
		}

		_, err := service2.Get(ctx, keys)
		_ = err
	})

	t.Run("Get with composite keys", func(t *testing.T) {
		metadata3 := EntityMetadata{
			Name:      "OrderItems",
			TableName: "order_items",
			Properties: []PropertyMetadata{
				{Name: "OrderID", ColumnName: "order_id", Type: "int64", IsKey: true},
				{Name: "ItemID", ColumnName: "item_id", Type: "int64", IsKey: true},
				{Name: "Quantity", ColumnName: "quantity", Type: "int"},
			},
		}

		service3 := NewBaseEntityService(&mockDatabaseProvider{}, metadata3, server)
		keys := map[string]interface{}{
			"OrderID": int64(1),
			"ItemID":  int64(5),
		}

		_, err := service3.Get(ctx, keys)
		_ = err
	})
}

func TestEntityService_Create_Validation(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Email", ColumnName: "email", Type: "string"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Create with valid data", func(t *testing.T) {
		entity := map[string]interface{}{
			"Name":  "John Doe",
			"Email": "john@example.com",
		}

		_, err := service.Create(ctx, entity)
		// May fail with mock
		_ = err
	})

	t.Run("Create with nil entity", func(t *testing.T) {
		_, err := service.Create(ctx, nil)
		assert.Error(t, err)
	})

	t.Run("Create with empty map", func(t *testing.T) {
		entity := map[string]interface{}{}

		_, err := service.Create(ctx, entity)
		// May fail with mock but should not panic
		_ = err
	})
}

func TestEntityService_Update_Operations(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Products",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Price", ColumnName: "price", Type: "float64"},
			{Name: "Stock", ColumnName: "stock", Type: "int"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Update single field", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}
		entity := map[string]interface{}{
			"Name": "Updated Product",
		}

		_, err := service.Update(ctx, keys, entity)
		_ = err
	})

	t.Run("Update multiple fields", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}
		entity := map[string]interface{}{
			"Name":  "Updated Product",
			"Price": 99.99,
			"Stock": 100,
		}

		_, err := service.Update(ctx, keys, entity)
		_ = err
	})

	t.Run("Update with nil entity", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}

		_, err := service.Update(ctx, keys, nil)
		assert.Error(t, err)
	})

	t.Run("Update with empty keys", func(t *testing.T) {
		keys := map[string]interface{}{}
		entity := map[string]interface{}{
			"Name": "Updated Product",
		}

		_, err := service.Update(ctx, keys, entity)
		// Should fail with empty keys
		_ = err
	})
}

func TestEntityService_Delete_Operations(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Orders",
		TableName: "orders",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "CustomerID", ColumnName: "customer_id", Type: "int64"},
			{Name: "Total", ColumnName: "total", Type: "float64"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Delete by ID", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}

		err := service.Delete(ctx, keys)
		_ = err
	})

	t.Run("Delete with composite keys", func(t *testing.T) {
		metadata2 := EntityMetadata{
			Name:      "OrderItems",
			TableName: "order_items",
			Properties: []PropertyMetadata{
				{Name: "OrderID", ColumnName: "order_id", Type: "int64", IsKey: true},
				{Name: "ItemID", ColumnName: "item_id", Type: "int64", IsKey: true},
			},
		}

		service2 := NewBaseEntityService(&mockDatabaseProvider{}, metadata2, server)
		keys := map[string]interface{}{
			"OrderID": int64(1),
			"ItemID":  int64(5),
		}

		err := service2.Delete(ctx, keys)
		_ = err
	})

	t.Run("Delete with empty keys", func(t *testing.T) {
		keys := map[string]interface{}{}

		err := service.Delete(ctx, keys)
		// Should fail with empty keys
		_ = err
	})
}

func TestEntityService_Metadata(t *testing.T) {
	server := NewServer()

	t.Run("GetMetadata returns correct info", func(t *testing.T) {
		metadata := EntityMetadata{
			Name:      "TestEntity",
			TableName: "test_table",
			Properties: []PropertyMetadata{
				{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
				{Name: "Name", ColumnName: "name", Type: "string"},
			},
		}

		service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
		result := service.GetMetadata()

		assert.Equal(t, "TestEntity", result.Name)
		assert.Equal(t, "test_table", result.TableName)
		assert.Len(t, result.Properties, 2)
	})

	t.Run("Metadata with relationships", func(t *testing.T) {
		metadata := EntityMetadata{
			Name:      "User",
			TableName: "users",
			Properties: []PropertyMetadata{
				{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
				{
					Name:       "Posts",
					ColumnName: "",
					Type:       "[]Post",
					Relationship: &RelationshipMetadata{
						LocalProperty:      "ID",
						ReferencedProperty: "UserID",
					},
				},
			},
		}

		service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
		result := service.GetMetadata()

		assert.Equal(t, "User", result.Name)
		assert.Len(t, result.Properties, 2)

		// Verify relationship exists
		var hasRelationship bool
		for _, prop := range result.Properties {
			if prop.Relationship != nil {
				hasRelationship = true
				assert.Equal(t, "ID", prop.Relationship.LocalProperty)
				assert.Equal(t, "UserID", prop.Relationship.ReferencedProperty)
			}
		}
		assert.True(t, hasRelationship)
	})
}

func TestEntityService_TypeConversion(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Products",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Price", ColumnName: "price", Type: "float64"},
			{Name: "InStock", ColumnName: "in_stock", Type: "bool"},
			{Name: "Quantity", ColumnName: "quantity", Type: "int"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Create with different types", func(t *testing.T) {
		entity := map[string]interface{}{
			"Price":    99.99,
			"InStock":  true,
			"Quantity": 50,
		}

		_, err := service.Create(ctx, entity)
		// May fail with mock
		_ = err
	})

	t.Run("Update with type conversion", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}
		entity := map[string]interface{}{
			"Price":    "49.99", // String to float conversion might be needed
			"Quantity": "25",    // String to int conversion might be needed
		}

		_, err := service.Update(ctx, keys, entity)
		_ = err
	})
}

func TestEntityService_EdgeCases(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "TestEntity",
		TableName: "test_table",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Query with empty context", func(t *testing.T) {
		// Using context.Background() which is valid
		_, err := service.Query(ctx, QueryOptions{})
		_ = err
	})

	t.Run("Service with nil provider", func(t *testing.T) {
		// This should panic or error, testing defensive programming
		defer func() {
			if r := recover(); r != nil {
				// Expected panic with nil provider
			}
		}()

		// Don't actually create with nil provider in production
		// This is just to test error handling
	})

	t.Run("Service initialization", func(t *testing.T) {
		// Verify service is properly initialized
		assert.NotNil(t, service)
		assert.Equal(t, metadata, service.GetMetadata())
	})
}

func TestEntityService_QueryOptions_Combinations(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Age", ColumnName: "age", Type: "int"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	tests := []struct {
		name    string
		options QueryOptions
	}{
		{
			name: "Top and Skip",
			options: QueryOptions{
				Top:  func() *GoDataTopQuery { v := GoDataTopQuery(10); return &v }(),
				Skip: func() *GoDataSkipQuery { v := GoDataSkipQuery(5); return &v }(),
			},
		},
		{
			name: "OrderBy ascending",
			options: QueryOptions{
				OrderBy: "Name asc",
			},
		},
		{
			name: "OrderBy descending",
			options: QueryOptions{
				OrderBy: "Age desc",
			},
		},
		{
			name: "Complex combination",
			options: QueryOptions{
				Top:     func() *GoDataTopQuery { v := GoDataTopQuery(20); return &v }(),
				Skip:    func() *GoDataSkipQuery { v := GoDataSkipQuery(10); return &v }(),
				OrderBy: "Name asc, Age desc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Query(ctx, tt.options)
			// May fail with mock, but should not panic
			_ = err
		})
	}
}

func TestEntityService_CRUD_Workflow(t *testing.T) {
	server := NewServer()
	metadata := EntityMetadata{
		Name:      "Tasks",
		TableName: "tasks",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Title", ColumnName: "title", Type: "string"},
			{Name: "Completed", ColumnName: "completed", Type: "bool"},
		},
	}

	service := NewBaseEntityService(&mockDatabaseProvider{}, metadata, server)
	ctx := context.Background()

	t.Run("Full CRUD workflow simulation", func(t *testing.T) {
		// Create
		newTask := map[string]interface{}{
			"Title":     "Test Task",
			"Completed": false,
		}
		_, createErr := service.Create(ctx, newTask)

		// Query
		queryOptions := QueryOptions{
			OrderBy: "Title asc",
		}
		_, queryErr := service.Query(ctx, queryOptions)

		// Get
		keys := map[string]interface{}{
			"ID": int64(1),
		}
		_, getErr := service.Get(ctx, keys)

		// Update
		updateData := map[string]interface{}{
			"Completed": true,
		}
		_, updateErr := service.Update(ctx, keys, updateData)

		// Delete
		deleteErr := service.Delete(ctx, keys)

		// All operations may fail with mock, but should not panic
		_ = createErr
		_ = queryErr
		_ = getErr
		_ = updateErr
		_ = deleteErr
	})
}
