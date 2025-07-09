package odata

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalParserSingleton(t *testing.T) {
	// Testa se o singleton funciona corretamente
	parser1 := GetGlobalParser()
	parser2 := GetGlobalParser()
	parser3 := NewODataParser()

	assert.Same(t, parser1, parser2, "GetGlobalParser should return the same instance")
	assert.Same(t, parser1, parser3, "NewODataParser should return the global singleton")
}

func TestODataParser_ParseQueryOptions(t *testing.T) {
	parser := NewODataParser()

	t.Run("Parse query options case sensitive", func(t *testing.T) {
		values := url.Values{
			"$filter":  []string{"Name eq 'John'"},
			"$orderby": []string{"Name desc"},
			"$select":  []string{"Name,Age"},
			"$expand":  []string{"Orders,Products"},
			"$skip":    []string{"10"},
			"$top":     []string{"5"},
			"$count":   []string{"true"},
		}

		options, err := parser.ParseQueryOptions(values)
		assert.NoError(t, err)

		assert.NotNil(t, options.Filter)
		assert.Equal(t, "Name eq 'John'", options.Filter.RawValue)
		assert.Equal(t, "Name desc", options.OrderBy)
		assert.Equal(t, []string{"Name", "Age"}, options.Select)
		assert.Len(t, options.Expand.ExpandItems, 2)
		assert.Equal(t, "Orders", options.Expand.ExpandItems[0].Path[0].Value)
		assert.Equal(t, "Products", options.Expand.ExpandItems[1].Path[0].Value)
		assert.Equal(t, 10, options.Skip)
		assert.Equal(t, 5, options.Top)
		assert.NotNil(t, options.Count)
		assert.True(t, GetCountValue(options.Count))
	})

	t.Run("Parse query options case insensitive", func(t *testing.T) {
		values := url.Values{
			"$FILTER":  []string{"Name eq 'John'"},
			"$ORDERBY": []string{"Name desc"},
			"$SELECT":  []string{"Name,Age"},
			"$EXPAND":  []string{"Orders,Products"},
			"$SKIP":    []string{"10"},
			"$TOP":     []string{"5"},
			"$COUNT":   []string{"true"},
		}

		options, err := parser.ParseQueryOptions(values)
		assert.NoError(t, err)

		assert.NotNil(t, options.Filter)
		assert.Equal(t, "Name eq 'John'", options.Filter.RawValue)
		assert.Equal(t, "Name desc", options.OrderBy)
		assert.Equal(t, []string{"Name", "Age"}, options.Select)
		assert.Len(t, options.Expand.ExpandItems, 2)
		assert.Equal(t, "Orders", options.Expand.ExpandItems[0].Path[0].Value)
		assert.Equal(t, "Products", options.Expand.ExpandItems[1].Path[0].Value)
		assert.Equal(t, 10, options.Skip)
		assert.Equal(t, 5, options.Top)
		assert.NotNil(t, options.Count)
		assert.True(t, GetCountValue(options.Count))
	})

	t.Run("Parse query options mixed case", func(t *testing.T) {
		values := url.Values{
			"$Filter":  []string{"Name eq 'John'"},
			"$OrderBy": []string{"Name desc"},
			"$Select":  []string{"Name,Age"},
			"$Expand":  []string{"Orders,Products"},
			"$Skip":    []string{"10"},
			"$Top":     []string{"5"},
			"$Count":   []string{"true"},
		}

		options, err := parser.ParseQueryOptions(values)
		assert.NoError(t, err)

		assert.NotNil(t, options.Filter)
		assert.Equal(t, "Name eq 'John'", options.Filter.RawValue)
		assert.Equal(t, "Name desc", options.OrderBy)
		assert.Equal(t, []string{"Name", "Age"}, options.Select)
		assert.Len(t, options.Expand.ExpandItems, 2)
		assert.Equal(t, "Orders", options.Expand.ExpandItems[0].Path[0].Value)
		assert.Equal(t, "Products", options.Expand.ExpandItems[1].Path[0].Value)
		assert.Equal(t, 10, options.Skip)
		assert.Equal(t, 5, options.Top)
		assert.NotNil(t, options.Count)
		assert.True(t, GetCountValue(options.Count))
	})

	t.Run("Parse empty query options", func(t *testing.T) {
		values := url.Values{}

		options, err := parser.ParseQueryOptions(values)
		assert.NoError(t, err)

		assert.Nil(t, options.Filter)
		assert.Equal(t, "", options.OrderBy)
		assert.Nil(t, options.Select)
		assert.Len(t, options.Expand, 0)
		assert.Equal(t, 0, options.Skip)
		assert.Equal(t, 100, options.Top)
		assert.Nil(t, options.Count)
	})

	t.Run("Validate unsupported parameters - strict mode", func(t *testing.T) {
		values := url.Values{
			"$filter":      []string{"Name eq 'John'"},
			"$unsupported": []string{"value"},
		}

		_, err := parser.ParseQueryOptionsWithConfig(values, ComplianceStrict)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query parameter '$unsupported' is not supported")
	})

	t.Run("Ignore unsupported parameters - lenient mode", func(t *testing.T) {
		values := url.Values{
			"$filter":      []string{"Name eq 'John'"},
			"$unsupported": []string{"value"},
		}

		options, err := parser.ParseQueryOptionsWithConfig(values, ComplianceIgnoreUnknownKeywords)
		assert.NoError(t, err)
		assert.NotNil(t, options.Filter)
		assert.Equal(t, "Name eq 'John'", options.Filter.RawValue)
	})

	t.Run("Validate duplicate parameters - strict mode", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{"Name eq 'John'", "Age gt 18"},
		}

		_, err := parser.ParseQueryOptionsWithConfig(values, ComplianceStrict)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query parameter '$filter' cannot be specified more than once")
	})

	t.Run("Validate $skip negative value", func(t *testing.T) {
		values := url.Values{
			"$skip": []string{"-5"},
		}

		_, err := parser.ParseQueryOptions(values)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "$skip must be non-negative")
	})

	t.Run("Validate $top negative value", func(t *testing.T) {
		values := url.Values{
			"$top": []string{"-10"},
		}

		_, err := parser.ParseQueryOptions(values)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "$top must be non-negative")
	})

	t.Run("Validate $top exceeds limit", func(t *testing.T) {
		values := url.Values{
			"$top": []string{"15000"},
		}

		_, err := parser.ParseQueryOptions(values)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "$top cannot exceed 10000")
	})
}

func TestODataParser_ParseFilter(t *testing.T) {
	parser := NewODataParser()

	t.Run("Parse simple filter", func(t *testing.T) {
		expressions, err := parser.ParseFilter("Name eq 'John'")
		assert.NoError(t, err)
		assert.Len(t, expressions, 1)

		expr := expressions[0]
		assert.Equal(t, "Name", expr.Property)
		assert.Equal(t, FilterEq, expr.Operator)
		assert.Equal(t, "John", expr.Value)
	})

	t.Run("Parse filter with case insensitive operator", func(t *testing.T) {
		expressions, err := parser.ParseFilter("Name EQ 'John'")
		assert.NoError(t, err)
		assert.Len(t, expressions, 1)

		expr := expressions[0]
		assert.Equal(t, "Name", expr.Property)
		assert.Equal(t, FilterEq, expr.Operator)
		assert.Equal(t, "John", expr.Value)
	})

	t.Run("Parse filter with case insensitive AND", func(t *testing.T) {
		expressions, err := parser.ParseFilter("Name eq 'John' AND Age gt 25")
		assert.NoError(t, err)
		assert.Len(t, expressions, 2)

		assert.Equal(t, "Name", expressions[0].Property)
		assert.Equal(t, FilterEq, expressions[0].Operator)
		assert.Equal(t, "John", expressions[0].Value)

		assert.Equal(t, "Age", expressions[1].Property)
		assert.Equal(t, FilterGt, expressions[1].Operator)
		assert.Equal(t, "25", expressions[1].Value)
	})

	t.Run("Parse filter with case insensitive function", func(t *testing.T) {
		expressions, err := parser.ParseFilter("CONTAINS(Name, 'John')")
		assert.NoError(t, err)
		assert.Len(t, expressions, 1)

		expr := expressions[0]
		assert.Equal(t, "Name", expr.Property)
		assert.Equal(t, FilterContains, expr.Operator)
		assert.Equal(t, "John", expr.Value)
	})

	t.Run("Parse empty filter", func(t *testing.T) {
		expressions, err := parser.ParseFilter("")
		assert.NoError(t, err)
		assert.Nil(t, expressions)
	})
}

// Testes para novos tipos de tokens otimizados
func TestOptimizedTokenizer(t *testing.T) {
	tokenizer := GetGlobalFilterTokenizer()

	t.Run("Tokenize DateTime", func(t *testing.T) {
		tokens, err := tokenizer.Tokenize(context.Background(), "CreatedAt eq 2023-12-25T10:30:00Z")
		assert.NoError(t, err)
		assert.Len(t, tokens, 3)

		assert.Equal(t, FilterTokenProperty, tokens[0].Type)
		assert.Equal(t, "CreatedAt", tokens[0].Value)

		assert.Equal(t, FilterTokenLogical, tokens[1].Type)
		assert.Equal(t, "eq", tokens[1].Value)

		assert.Equal(t, FilterTokenDateTime, tokens[2].Type)
		assert.Equal(t, "2023-12-25T10:30:00Z", tokens[2].Value)
	})

	t.Run("Tokenize Date", func(t *testing.T) {
		tokens, err := tokenizer.Tokenize(context.Background(), "BirthDate eq 1990-05-15")
		assert.NoError(t, err)
		assert.Len(t, tokens, 3)

		assert.Equal(t, FilterTokenProperty, tokens[0].Type)
		assert.Equal(t, FilterTokenLogical, tokens[1].Type)
		assert.Equal(t, FilterTokenDate, tokens[2].Type)
		assert.Equal(t, "1990-05-15", tokens[2].Value)
	})

	t.Run("Tokenize Time", func(t *testing.T) {
		tokens, err := tokenizer.Tokenize(context.Background(), "StartTime eq 14:30:00")
		assert.NoError(t, err)
		assert.Len(t, tokens, 3)

		assert.Equal(t, FilterTokenProperty, tokens[0].Type)
		assert.Equal(t, FilterTokenLogical, tokens[1].Type)
		assert.Equal(t, FilterTokenTime, tokens[2].Type)
		assert.Equal(t, "14:30:00", tokens[2].Value)
	})

	t.Run("Tokenize GUID", func(t *testing.T) {
		tokens, err := tokenizer.Tokenize(context.Background(), "ID eq 12345678-1234-5678-9012-123456789012")
		assert.NoError(t, err)
		assert.Len(t, tokens, 3)

		assert.Equal(t, FilterTokenProperty, tokens[0].Type)
		assert.Equal(t, FilterTokenLogical, tokens[1].Type)
		assert.Equal(t, FilterTokenGuid, tokens[2].Type)
		assert.Equal(t, "12345678-1234-5678-9012-123456789012", tokens[2].Value)
	})

	t.Run("Tokenize Geography Point", func(t *testing.T) {
		tokens, err := tokenizer.Tokenize(context.Background(), "Location eq geography'POINT(-122.3 47.6)'")
		assert.NoError(t, err)
		assert.Len(t, tokens, 3)

		assert.Equal(t, FilterTokenProperty, tokens[0].Type)
		assert.Equal(t, FilterTokenLogical, tokens[1].Type)
		assert.Equal(t, FilterTokenGeographyPoint, tokens[2].Type)
		assert.Equal(t, "geography'POINT(-122.3 47.6)'", tokens[2].Value)
	})

	t.Run("Tokenize Function with Case Insensitive", func(t *testing.T) {
		tokens, err := tokenizer.Tokenize(context.Background(), "CONTAINS(Name, 'test')")
		assert.NoError(t, err)
		assert.Len(t, tokens, 6)

		assert.Equal(t, FilterTokenFunction, tokens[0].Type)
		assert.Equal(t, "contains", tokens[0].Value)

		assert.Equal(t, FilterTokenOpenParen, tokens[1].Type)
		assert.Equal(t, "(", tokens[1].Value)

		assert.Equal(t, FilterTokenProperty, tokens[2].Type)
		assert.Equal(t, "Name", tokens[2].Value)

		assert.Equal(t, FilterTokenComma, tokens[3].Type)

		assert.Equal(t, FilterTokenString, tokens[4].Type)
		assert.Equal(t, "'test'", tokens[4].Value)

		assert.Equal(t, FilterTokenCloseParen, tokens[5].Type)
		assert.Equal(t, ")", tokens[5].Value)
	})
}

// Testes para Expression Parser otimizado
func TestOptimizedExpressionParser(t *testing.T) {
	parser := GetGlobalExpressionParser()

	t.Run("Parse simple boolean expression", func(t *testing.T) {
		tree, err := parser.ParseFilterExpression(context.Background(), "Name eq 'John'")
		assert.NoError(t, err)
		assert.NotNil(t, tree)

		assert.Equal(t, FilterTokenLogical, tree.Token.Type)
		assert.Equal(t, "eq", tree.Token.Value)
		assert.Len(t, tree.Children, 2)

		assert.Equal(t, FilterTokenProperty, tree.Children[0].Token.Type)
		assert.Equal(t, "Name", tree.Children[0].Token.Value)

		assert.Equal(t, FilterTokenString, tree.Children[1].Token.Type)
		assert.Equal(t, "'John'", tree.Children[1].Token.Value)
	})

	t.Run("Parse complex expression with precedence", func(t *testing.T) {
		tree, err := parser.ParseFilterExpression(context.Background(), "Name eq 'John' and Age gt 25")
		assert.NoError(t, err)
		assert.NotNil(t, tree)

		// Root deve ser 'and' (precedência mais baixa)
		assert.Equal(t, FilterTokenLogical, tree.Token.Type)
		assert.Equal(t, "and", tree.Token.Value)
		assert.Len(t, tree.Children, 2)

		// Lado esquerdo: Name eq 'John'
		leftChild := tree.Children[0]
		assert.Equal(t, FilterTokenLogical, leftChild.Token.Type)
		assert.Equal(t, "eq", leftChild.Token.Value)

		// Lado direito: Age gt 25
		rightChild := tree.Children[1]
		assert.Equal(t, FilterTokenLogical, rightChild.Token.Type)
		assert.Equal(t, "gt", rightChild.Token.Value)
	})

	t.Run("Parse function expression", func(t *testing.T) {
		tree, err := parser.ParseFilterExpression(context.Background(), "contains(Name, 'John')")
		assert.NoError(t, err)
		assert.NotNil(t, tree)

		assert.Equal(t, FilterTokenFunction, tree.Token.Type)
		assert.Equal(t, "contains", tree.Token.Value)
		assert.Len(t, tree.Children, 2)

		assert.Equal(t, FilterTokenProperty, tree.Children[0].Token.Type)
		assert.Equal(t, "Name", tree.Children[0].Token.Value)

		assert.Equal(t, FilterTokenString, tree.Children[1].Token.Type)
		assert.Equal(t, "'John'", tree.Children[1].Token.Value)
	})

	t.Run("Parse expression with parentheses", func(t *testing.T) {
		tree, err := parser.ParseFilterExpression(context.Background(), "(Name eq 'John' or Name eq 'Jane') and Age gt 25")
		assert.NoError(t, err)
		assert.NotNil(t, tree)

		// Root deve ser 'and'
		assert.Equal(t, FilterTokenLogical, tree.Token.Type)
		assert.Equal(t, "and", tree.Token.Value)
		assert.Len(t, tree.Children, 2)

		// Lado esquerdo deve ser 'or' (por causa dos parênteses)
		leftChild := tree.Children[0]
		assert.Equal(t, FilterTokenLogical, leftChild.Token.Type)
		assert.Equal(t, "or", leftChild.Token.Value)
	})
}

// Benchmark para comparar performance
func BenchmarkGlobalParser(b *testing.B) {
	values := url.Values{
		"$filter":  []string{"Name eq 'John' and Age gt 25"},
		"$orderby": []string{"Name desc"},
		"$select":  []string{"Name,Age,Email"},
		"$skip":    []string{"10"},
		"$top":     []string{"50"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := GetGlobalParser()
		_, err := parser.ParseQueryOptions(values)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTokenizer(b *testing.B) {
	input := "Name eq 'John' and Age gt 25 and contains(Email, 'test@example.com')"
	tokenizer := GetGlobalFilterTokenizer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tokenizer.Tokenize(context.Background(), input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExpressionParser(b *testing.B) {
	input := "Name eq 'John' and Age gt 25 and contains(Email, 'test@example.com')"
	parser := GetGlobalExpressionParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseFilterExpression(context.Background(), input)
		if err != nil {
			b.Fatal(err)
		}
	}
}
