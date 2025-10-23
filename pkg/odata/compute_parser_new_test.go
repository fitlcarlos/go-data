package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseComputeString_Empty(t *testing.T) {
	ctx := context.Background()
	parser := NewComputeParser()
	result, err := parser.ParseCompute(ctx, "")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Expressions)
}

func TestParseComputeString_Simple(t *testing.T) {
	ctx := context.Background()
	parser := NewComputeParser()

	t.Run("Simple expression", func(t *testing.T) {
		result, err := parser.ParseCompute(ctx, "Price mul Quantity as Total")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Expressions, 1)
	})

	t.Run("Multiple expressions", func(t *testing.T) {
		result, err := parser.ParseCompute(ctx, "Price mul Quantity as Total, Price div 100 as Discount")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Expressions), 1)
	})
}

func TestParseComputeString_Operators(t *testing.T) {
	ctx := context.Background()
	parser := NewComputeParser()

	tests := []struct {
		name  string
		input string
	}{
		{"Multiplication", "Price mul 2 as Double"},
		{"Division", "Total div 2 as Half"},
		{"Addition", "Price add Tax as TotalPrice"},
		{"Subtraction", "Price sub Discount as FinalPrice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseCompute(ctx, tt.input)

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestComputeOption_FindExpression(t *testing.T) {
	ctx := context.Background()
	parser := NewComputeParser()

	t.Run("Find computed expression by alias", func(t *testing.T) {
		result, _ := parser.ParseCompute(ctx, "Price mul Quantity as Total")

		assert.NotNil(t, result)
		assert.Len(t, result.Expressions, 1)
		assert.Equal(t, "Total", result.Expressions[0].Alias)
	})

	t.Run("Expression with different alias", func(t *testing.T) {
		result, _ := parser.ParseCompute(ctx, "Price mul Quantity as NonExistent")

		assert.NotNil(t, result)
		assert.Len(t, result.Expressions, 1)
	})
}

func TestComputeOption_MultipleExpressions(t *testing.T) {
	ctx := context.Background()
	parser := NewComputeParser()
	result, _ := parser.ParseCompute(ctx, "Price mul Quantity as Total, Price div 2 as Half")

	t.Run("Has multiple expressions", func(t *testing.T) {
		assert.Len(t, result.Expressions, 2)
		assert.Equal(t, "Total", result.Expressions[0].Alias)
		assert.Equal(t, "Half", result.Expressions[1].Alias)
	})
}
