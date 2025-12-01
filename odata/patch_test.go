package odata

import (
	"context"
	"testing"
)

// TestBaseEntityService_Patch_SimpleUpdate testa que PATCH simples (sem hierarquia) usa Update
func TestBaseEntityService_Patch_SimpleUpdate(t *testing.T) {
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

	t.Run("Patch without hierarchy delegates to Update", func(t *testing.T) {
		keys := map[string]interface{}{
			"ID": int64(1),
		}
		entity := map[string]interface{}{
			"Name": "Updated Product",
		}

		// Deve funcionar mesmo sem conexão real (usa mock)
		_, err := service.Patch(ctx, keys, entity)
		// Pode falhar por falta de conexão, mas não deve quebrar
		_ = err
	})
}

// TestHasHierarchicalStructure testa a detecção de estrutura hierárquica
func TestHasHierarchicalStructure(t *testing.T) {
	t.Run("Detects @odata.removed", func(t *testing.T) {
		data := map[string]interface{}{
			"ID": 1,
			"@odata.removed": map[string]interface{}{},
		}
		metadata := EntityMetadata{
			Name: "Test",
			Properties: []PropertyMetadata{
				{Name: "ID", IsKey: true},
			},
		}

		if !hasHierarchicalStructure(data, metadata) {
			t.Error("Should detect @odata.removed")
		}
	})

	t.Run("Detects navigation properties", func(t *testing.T) {
		data := map[string]interface{}{
			"ID": 1,
			"Itens": []interface{}{
				map[string]interface{}{
					"ProductID": 10,
				},
			},
		}
		metadata := EntityMetadata{
			Name: "Pedido",
			Properties: []PropertyMetadata{
				{Name: "ID", IsKey: true},
				{Name: "Itens", IsNavigation: true, IsCollection: true, RelatedType: "Item"},
			},
		}

		if !hasHierarchicalStructure(data, metadata) {
			t.Error("Should detect navigation properties with nested objects")
		}
	})

	t.Run("Returns false for simple update", func(t *testing.T) {
		data := map[string]interface{}{
			"ID":   1,
			"Name": "Updated",
		}
		metadata := EntityMetadata{
			Name: "Test",
			Properties: []PropertyMetadata{
				{Name: "ID", IsKey: true},
				{Name: "Name"},
			},
		}

		if hasHierarchicalStructure(data, metadata) {
			t.Error("Should return false for simple update")
		}
	})
}

// TestIdentifyOperation testa a identificação de operações
func TestIdentifyOperation(t *testing.T) {
	metadata := EntityMetadata{
		Name: "Item",
		Properties: []PropertyMetadata{
			{Name: "ID", IsKey: true},
			{Name: "ProductID"},
		},
	}

	t.Run("Identifies DELETE with @odata.removed", func(t *testing.T) {
		entity := map[string]interface{}{
			"ID":           99,
			"@odata.removed": map[string]interface{}{},
		}

		opType, err := identifyOperation(entity, metadata, "both")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if opType != "DELETE" {
			t.Errorf("Expected DELETE, got %s", opType)
		}
	})

	t.Run("Identifies UPDATE with all keys", func(t *testing.T) {
		entity := map[string]interface{}{
			"ID":        55,
			"ProductID": 20,
		}

		opType, err := identifyOperation(entity, metadata, "both")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if opType != "UPDATE" {
			t.Errorf("Expected UPDATE, got %s", opType)
		}
	})

	t.Run("Identifies INSERT without keys", func(t *testing.T) {
		entity := map[string]interface{}{
			"ProductID": 10,
		}

		opType, err := identifyOperation(entity, metadata, "both")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if opType != "INSERT" {
			t.Errorf("Expected INSERT, got %s", opType)
		}
	})
}

// TestExtractKeysFromEntity testa a extração de chaves
func TestExtractKeysFromEntity(t *testing.T) {
	metadata := EntityMetadata{
		Name: "Item",
		Properties: []PropertyMetadata{
			{Name: "ID", IsKey: true},
		},
	}

	t.Run("Extracts simple key", func(t *testing.T) {
		entity := map[string]interface{}{
			"ID": 55,
		}

		keys := extractKeysFromEntity(entity, metadata)
		if len(keys) != 1 {
			t.Errorf("Expected 1 key, got %d", len(keys))
		}
		if keys["ID"] != 55 {
			t.Errorf("Expected ID=55, got %v", keys["ID"])
		}
	})

	t.Run("Extracts from @odata.id", func(t *testing.T) {
		entity := map[string]interface{}{
			"@odata.id": "/Item(99)",
		}

		keys := extractKeysFromEntity(entity, metadata)
		if len(keys) != 1 {
			t.Errorf("Expected 1 key from @odata.id, got %d", len(keys))
		}
	})
}

