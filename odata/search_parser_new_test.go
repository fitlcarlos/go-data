package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSearchString_Empty(t *testing.T) {
	ctx := context.Background()
	result, err := ParseSearchString(ctx, "")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseSearchString_Simple(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{"Single word", "john"},
		{"Multiple words", "john doe"},
		{"With quotes", "\"john doe\""},
		{"With AND", "john AND doe"},
		{"With OR", "john OR jane"},
		{"With NOT", "NOT spam"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSearchString(ctx, tt.input)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.Query)
		})
	}
}

func TestParseSearchString_Complex(t *testing.T) {
	ctx := context.Background()

	t.Run("Complex expression", func(t *testing.T) {
		result, err := ParseSearchString(ctx, "(john OR jane) AND NOT spam")

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestSearchQuery_String(t *testing.T) {
	ctx := context.Background()
	result, _ := ParseSearchString(ctx, "john doe")

	t.Run("Returns raw query", func(t *testing.T) {
		str := result.String()
		assert.NotEmpty(t, str)
	})
}

func TestSearchQuery_GetTerms(t *testing.T) {
	ctx := context.Background()
	result, _ := ParseSearchString(ctx, "john doe")

	t.Run("Returns search terms", func(t *testing.T) {
		terms := result.GetTerms()
		assert.NotEmpty(t, terms)
	})
}
