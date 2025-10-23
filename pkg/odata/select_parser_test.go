package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSelectString_Empty(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.SelectItems, 0)
	assert.Equal(t, "", result.RawValue)
}

func TestParseSelectString_SingleProperty(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.SelectItems, 1)
	assert.Equal(t, "Name", result.SelectItems[0].Segments[0].Value)
	assert.Equal(t, "Name", result.RawValue)
}

func TestParseSelectString_MultipleProperties(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Description,Price")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.SelectItems, 3)
	assert.Equal(t, "Name", result.SelectItems[0].Segments[0].Value)
	assert.Equal(t, "Description", result.SelectItems[1].Segments[0].Value)
	assert.Equal(t, "Price", result.SelectItems[2].Segments[0].Value)
}

func TestParseSelectString_WithSpaces(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, " Name , Description , Price ")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.SelectItems, 3)
}

func TestParseSelectString_EmptyItem(t *testing.T) {
	ctx := context.Background()

	_, err := ParseSelectString(ctx, "Name,,Description")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty select item")
}

func TestGetSelectedProperties(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price,Stock")
	require.NoError(t, err)

	props := GetSelectedProperties(result)
	assert.Len(t, props, 3)
	assert.Contains(t, props, "Name")
	assert.Contains(t, props, "Price")
	assert.Contains(t, props, "Stock")
}

func TestGetSelectedProperties_Nil(t *testing.T) {
	props := GetSelectedProperties(nil)
	assert.Len(t, props, 0)
}

func TestIsSelectAll_Nil(t *testing.T) {
	assert.True(t, IsSelectAll(nil))
}

func TestIsSelectAll_False(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	assert.False(t, IsSelectAll(result))
}

func TestGetSelectComplexity_Nil(t *testing.T) {
	complexity := GetSelectComplexity(nil)
	assert.Equal(t, 0, complexity)
}

func TestGetSelectComplexity_Simple(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	complexity := GetSelectComplexity(result)
	assert.Equal(t, 2, complexity) // 2 propriedades simples = 2 segmentos
}

func TestGoDataSelectQuery_String(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	assert.Equal(t, "Name,Price", result.String())
}

func TestGoDataSelectQuery_String_Nil(t *testing.T) {
	var query *GoDataSelectQuery
	assert.Equal(t, "", query.String())
}

func TestGoDataSelectQuery_HasProperty(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	assert.True(t, result.HasProperty("Name"))
	assert.True(t, result.HasProperty("Price"))
	assert.False(t, result.HasProperty("Description"))
}

func TestGoDataSelectQuery_HasProperty_Nil(t *testing.T) {
	var query *GoDataSelectQuery
	// Nil retorna true (todas as propriedades são incluídas)
	assert.True(t, query.HasProperty("AnyProperty"))
}

func TestGoDataSelectQuery_HasProperty_CaseInsensitive(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	assert.True(t, result.HasProperty("name"))
	assert.True(t, result.HasProperty("NAME"))
	assert.True(t, result.HasProperty("price"))
}

func TestGoDataSelectQuery_GetSelectItemByProperty(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	item := result.GetSelectItemByProperty("Name")
	assert.NotNil(t, item)
	assert.Equal(t, "Name", item.Segments[0].Value)

	item2 := result.GetSelectItemByProperty("Description")
	assert.Nil(t, item2)
}

func TestGoDataSelectQuery_GetSelectItemByProperty_Nil(t *testing.T) {
	var query *GoDataSelectQuery
	item := query.GetSelectItemByProperty("Name")
	assert.Nil(t, item)
}

func TestGoDataSelectQuery_AddSelectItem(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name")
	require.NoError(t, err)

	assert.Len(t, result.SelectItems, 1)

	result.AddSelectItem("Price")
	assert.Len(t, result.SelectItems, 2)
	assert.True(t, result.HasProperty("Price"))
}

func TestGoDataSelectQuery_AddSelectItem_Duplicate(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name")
	require.NoError(t, err)

	result.AddSelectItem("Name")
	assert.Len(t, result.SelectItems, 1) // Não duplica
}

func TestGoDataSelectQuery_AddSelectItem_Nil(t *testing.T) {
	var query *GoDataSelectQuery
	// Não deve causar panic
	query.AddSelectItem("Name")
}

func TestGoDataSelectQuery_RemoveSelectItem(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price,Stock")
	require.NoError(t, err)

	assert.Len(t, result.SelectItems, 3)

	result.RemoveSelectItem("Price")
	assert.Len(t, result.SelectItems, 2)
	assert.False(t, result.HasProperty("Price"))
	assert.True(t, result.HasProperty("Name"))
	assert.True(t, result.HasProperty("Stock"))
}

func TestGoDataSelectQuery_RemoveSelectItem_NotFound(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name")
	require.NoError(t, err)

	result.RemoveSelectItem("NonExistent")
	assert.Len(t, result.SelectItems, 1) // Não afeta
}

func TestGoDataSelectQuery_RemoveSelectItem_Nil(t *testing.T) {
	var query *GoDataSelectQuery
	// Não deve causar panic
	query.RemoveSelectItem("Name")
}

func TestFormatSelectExpression(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	formatted := FormatSelectExpression(result)
	assert.Contains(t, formatted, "Name")
	assert.Contains(t, formatted, "Price")
}

func TestFormatSelectExpression_Nil(t *testing.T) {
	formatted := FormatSelectExpression(nil)
	assert.Equal(t, "", formatted)
}

func TestValidateSelectProperty(t *testing.T) {
	entity := &EntityMetadata{
		Name: "Product",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id"},
			{Name: "Name", ColumnName: "name"},
			{Name: "Price", ColumnName: "price"},
		},
	}

	err := ValidateSelectProperty("Name", entity)
	assert.NoError(t, err)

	err = ValidateSelectProperty("NonExistent", entity)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConvertSelectToSQL_Nil(t *testing.T) {
	entity := &EntityMetadata{
		Name:      "Product",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", IsNavigation: false},
			{Name: "Name", ColumnName: "name", IsNavigation: false},
			{Name: "Price", ColumnName: "price", IsNavigation: false},
		},
	}

	columns, err := ConvertSelectToSQL(nil, entity)
	require.NoError(t, err)
	assert.Len(t, columns, 3)
	assert.Contains(t, columns, "products.id")
	assert.Contains(t, columns, "products.name")
	assert.Contains(t, columns, "products.price")
}

func TestConvertSelectToSQL_SpecificProperties(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Price")
	require.NoError(t, err)

	entity := &EntityMetadata{
		Name:      "Product",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", IsNavigation: false},
			{Name: "Name", ColumnName: "name", IsNavigation: false},
			{Name: "Price", ColumnName: "price", IsNavigation: false},
		},
	}

	columns, err := ConvertSelectToSQL(result, entity)
	require.NoError(t, err)
	assert.Len(t, columns, 2)
	assert.Contains(t, columns, "products.name")
	assert.Contains(t, columns, "products.price")
	assert.NotContains(t, columns, "products.id")
}

func TestConvertSelectToSQL_SkipsNavigation(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSelectString(ctx, "Name,Category")
	require.NoError(t, err)

	entity := &EntityMetadata{
		Name:      "Product",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "Name", ColumnName: "name", IsNavigation: false},
			{Name: "Category", ColumnName: "category_id", IsNavigation: true},
		},
	}

	columns, err := ConvertSelectToSQL(result, entity)
	require.NoError(t, err)
	assert.Len(t, columns, 1)
	assert.Contains(t, columns, "products.name")
	// Propriedades de navegação não são incluídas
}

// Benchmarks
func BenchmarkParseSelectString_Single(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseSelectString(ctx, "Name")
	}
}

func BenchmarkParseSelectString_Multiple(b *testing.B) {
	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseSelectString(ctx, "Name,Description,Price,Stock,Category")
	}
}

func BenchmarkGetSelectedProperties(b *testing.B) {
	ctx := context.Background()
	result, _ := ParseSelectString(ctx, "Name,Description,Price,Stock,Category")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GetSelectedProperties(result)
	}
}

func BenchmarkHasProperty(b *testing.B) {
	ctx := context.Background()
	result, _ := ParseSelectString(ctx, "Name,Description,Price,Stock,Category")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = result.HasProperty("Name")
	}
}

