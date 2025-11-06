package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGlobalFilterTokenizer(t *testing.T) {
	tokenizer := GetGlobalFilterTokenizer()
	assert.NotNil(t, tokenizer)

	// Deve retornar a mesma instância
	tokenizer2 := GetGlobalFilterTokenizer()
	assert.Equal(t, tokenizer, tokenizer2)
}

func TestTokenizer_Tokenize_SimpleProperty(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tokens, err := tokenizer.Tokenize(ctx, "Name")
	require.NoError(t, err)
	assert.Len(t, tokens, 1)
	assert.Equal(t, int(FilterTokenProperty), tokens[0].Type)
	assert.Equal(t, "Name", tokens[0].Value)
}

func TestTokenizer_Tokenize_MultipleProperties(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name  string
		input string
	}{
		{"With underscore", "product_name"},
		{"Simple lowercase", "productname"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(tokens), 1)
			// Tokenizer pode dividir em múltiplos tokens dependendo do padrão
		})
	}
}

func TestTokenizer_Tokenize_LogicalOperators(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"and lowercase", "and", "and"},
		{"or lowercase", "or", "or"},
		{"not lowercase", "not", "not"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, int(FilterTokenLogical), tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestTokenizer_Tokenize_ComparisonOperators(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Equal", "eq", "eq"},
		{"Not equal", "ne", "ne"},
		{"Greater than", "gt", "gt"},
		{"Greater or equal", "ge", "ge"},
		{"Less than", "lt", "lt"},
		{"Less or equal", "le", "le"},
		{"Has", "has", "has"},
		{"In", "in", "in"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, int(FilterTokenComparison), tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestTokenizer_Tokenize_ArithmeticOperators(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name  string
		input string
	}{
		{"Add", "add"},
		{"Sub", "sub"},
		{"Mul", "mul"},
		{"Div", "div"},
		{"DivBy", "divby"},
		{"Mod", "mod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, int(FilterTokenArithmetic), tokens[0].Type)
		})
	}
}

func TestTokenizer_Tokenize_Functions(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name  string
		input string
	}{
		{"Contains", "contains"},
		{"StartsWith", "startswith"},
		{"EndsWith", "endswith"},
		{"Length", "length"},
		{"ToLower", "tolower"},
		{"ToUpper", "toupper"},
		{"Trim", "trim"},
		{"Year", "year"},
		{"Month", "month"},
		{"Day", "day"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, int(FilterTokenFunction), tokens[0].Type)
		})
	}
}

func TestTokenizer_Tokenize_StringLiterals(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name  string
		input string
	}{
		{"Simple string", "'hello'"},
		{"With spaces", "'hello world'"},
		{"Empty string", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(tokens), 1)
			// Strings incluem as aspas no value
			assert.Equal(t, int(FilterTokenString), tokens[0].Type)
		})
	}
}

func TestTokenizer_Tokenize_Numbers(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name  string
		input string
	}{
		{"Integer", "123"},
		{"Negative", "-456"},
		{"Decimal", "123.45"},
		{"Negative decimal", "-123.45"},
		{"Scientific", "1.23e10"},
		{"Negative scientific", "-1.23e-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, int(FilterTokenNumber), tokens[0].Type)
		})
	}
}

func TestTokenizer_Tokenize_Boolean(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"true", "true"},
		{"false", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, int(FilterTokenBoolean), tokens[0].Type)
			assert.Equal(t, tt.expected, tokens[0].Value)
		})
	}
}

func TestTokenizer_Tokenize_Null(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tokens, err := tokenizer.Tokenize(ctx, "null")
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, int(FilterTokenNull), tokens[0].Type)
	assert.Equal(t, "null", tokens[0].Value)
}

func TestTokenizer_Tokenize_Parentheses(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tokens, err := tokenizer.Tokenize(ctx, "()")
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, int(FilterTokenOpenParen), tokens[0].Type)
	assert.Equal(t, int(FilterTokenCloseParen), tokens[1].Type)
}

func TestTokenizer_Tokenize_Comma(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tokens, err := tokenizer.Tokenize(ctx, ",")
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, int(FilterTokenComma), tokens[0].Type)
}

func TestTokenizer_Tokenize_ComplexExpression(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tests := []struct {
		name     string
		input    string
		minCount int
	}{
		{"Simple comparison", "Age eq 25", 3},
		{"With and", "Age gt 25 and Age lt 65", 7},
		{"With function", "contains(Description, 'test')", 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizer.Tokenize(ctx, tt.input)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(tokens), tt.minCount)
		})
	}
}

func TestTokenizer_Tokenize_EmptyString(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tokens, err := tokenizer.Tokenize(ctx, "")
	require.NoError(t, err)
	assert.Len(t, tokens, 0)
}

func TestTokenizer_Tokenize_WithWhitespace(t *testing.T) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()

	tokens, err := tokenizer.Tokenize(ctx, "  Age   eq   25  ")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tokens), 3)
	assert.Equal(t, int(FilterTokenProperty), tokens[0].Type)
	assert.Equal(t, int(FilterTokenComparison), tokens[1].Type)
	assert.Equal(t, int(FilterTokenNumber), tokens[2].Type)
}

func TestToken_Creation(t *testing.T) {
	token := &Token{
		Type:              int(FilterTokenProperty),
		Value:             "Name",
		SemanticType:      SemanticTypeProperty,
		SemanticReference: nil,
	}

	assert.Equal(t, int(FilterTokenProperty), token.Type)
	assert.Equal(t, "Name", token.Value)
	assert.Equal(t, SemanticTypeProperty, token.SemanticType)
}

func TestSemanticType_Values(t *testing.T) {
	assert.Equal(t, SemanticType(0), SemanticTypeUnknown)
	assert.Equal(t, SemanticType(1), SemanticTypeProperty)
	assert.Equal(t, SemanticType(2), SemanticTypeFunction)
	assert.Equal(t, SemanticType(3), SemanticTypeOperator)
	assert.Equal(t, SemanticType(4), SemanticTypeValue)
	assert.Equal(t, SemanticType(5), SemanticTypeKeyword)
}

func TestFilterTokenType_Values(t *testing.T) {
	assert.Equal(t, FilterTokenType(1), FilterTokenProperty)
	assert.Equal(t, FilterTokenType(2), FilterTokenFunction)
	assert.Equal(t, FilterTokenType(3), FilterTokenArithmetic)
	assert.Equal(t, FilterTokenType(4), FilterTokenString)
	assert.Equal(t, FilterTokenType(5), FilterTokenNumber)
}

// Benchmarks
func BenchmarkTokenizer_Tokenize_Simple(b *testing.B) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = tokenizer.Tokenize(ctx, "Name eq 'Test'")
	}
}

func BenchmarkTokenizer_Tokenize_Complex(b *testing.B) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = tokenizer.Tokenize(ctx, "Name eq 'Test' and Age gt 25 and contains(Description, 'important')")
	}
}

func BenchmarkTokenizer_Tokenize_WithFunctions(b *testing.B) {
	ctx := context.Background()
	tokenizer := GetGlobalFilterTokenizer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = tokenizer.Tokenize(ctx, "contains(tolower(Name), 'test') and startswith(Category, 'prod')")
	}
}

func BenchmarkGetGlobalFilterTokenizer(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = GetGlobalFilterTokenizer()
	}
}

