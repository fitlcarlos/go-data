package odata

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

func TestURLParser_ParseQuery(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		rawQuery string
		expected map[string]string
	}{
		{
			name:     "Simple filter",
			rawQuery: "$filter=nome eq 'João'",
			expected: map[string]string{
				"$filter": "nome eq 'João'",
			},
		},
		{
			name:     "Complex expand with nested options",
			rawQuery: "$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc;$expand=FabTarefa($orderby=id desc))",
			expected: map[string]string{
				"$expand": "FabOperacao($filter=codigo eq 13;$orderby=id desc;$expand=FabTarefa($orderby=id desc))",
			},
		},
		{
			name:     "Multiple parameters",
			rawQuery: "$filter=ativo eq 'S'&$orderby=nome asc&$top=10&$skip=5",
			expected: map[string]string{
				"$filter":  "ativo eq 'S'",
				"$orderby": "nome asc",
				"$top":     "10",
				"$skip":    "5",
			},
		},
		{
			name:     "URL encoded parameters",
			rawQuery: "$filter=nome%20eq%20'João'&$orderby=data%20desc",
			expected: map[string]string{
				"$filter":  "nome eq 'João'",
				"$orderby": "data desc",
			},
		},
		{
			name:     "Complex filter with parentheses",
			rawQuery: "$filter=(nome eq 'João' and idade gt 18) or (nome eq 'Maria' and idade lt 25)",
			expected: map[string]string{
				"$filter": "(nome eq 'João' and idade gt 18) or (nome eq 'Maria' and idade lt 25)",
			},
		},
		{
			name:     "Contains function",
			rawQuery: "$filter=contains(nome,'João')&$select=nome,idade",
			expected: map[string]string{
				"$filter": "contains(nome,'João')",
				"$select": "nome,idade",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := parser.ParseQuery(tt.rawQuery)
			if err != nil {
				t.Errorf("ParseQuery() error = %v", err)
				return
			}

			for key, expectedValue := range tt.expected {
				actualValue := values.Get(key)
				if actualValue != expectedValue {
					t.Errorf("Parameter %s: expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestURLParser_ValidateODataQuery(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name      string
		query     string
		shouldErr bool
	}{
		{
			name:      "Valid query",
			query:     "$filter=nome eq 'João'&$orderby=data desc",
			shouldErr: false,
		},
		{
			name:      "Unbalanced parentheses",
			query:     "$filter=(nome eq 'João' and idade gt 18",
			shouldErr: true,
		},
		{
			name:      "Unbalanced quotes",
			query:     "$filter=nome eq 'João and idade gt 18",
			shouldErr: true,
		},
		{
			name:      "Complex valid query",
			query:     "$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc)",
			shouldErr: false,
		},
		{
			name:      "Empty query",
			query:     "",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateODataQuery(tt.query)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateODataQuery() error = %v, shouldErr = %v", err, tt.shouldErr)
			}
		})
	}
}

func TestURLParser_SplitQueryParams(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "Simple parameters",
			query:    "$filter=nome eq 'João'&$orderby=data desc",
			expected: []string{"$filter=nome eq 'João'", "$orderby=data desc"},
		},
		{
			name:     "Parameter with parentheses",
			query:    "$expand=FabOperacao($filter=codigo eq 13)&$top=10",
			expected: []string{"$expand=FabOperacao($filter=codigo eq 13)", "$top=10"},
		},
		{
			name:     "Parameter with nested parentheses",
			query:    "$expand=FabOperacao($filter=(codigo eq 13 and ativo eq 'S'))&$skip=5",
			expected: []string{"$expand=FabOperacao($filter=(codigo eq 13 and ativo eq 'S'))", "$skip=5"},
		},
		{
			name:     "Single parameter",
			query:    "$filter=nome eq 'João'",
			expected: []string{"$filter=nome eq 'João'"},
		},
		{
			name:     "Empty query",
			query:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.splitQueryParams(tt.query)
			if len(result) != len(tt.expected) {
				t.Errorf("splitQueryParams() length = %d, expected = %d", len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("splitQueryParams()[%d] = %q, expected = %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestURLParser_ParseParam(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name          string
		param         string
		expectedKey   string
		expectedValue string
	}{
		{
			name:          "Simple parameter",
			param:         "$filter=nome eq 'João'",
			expectedKey:   "$filter",
			expectedValue: "nome eq 'João'",
		},
		{
			name:          "Parameter with parentheses",
			param:         "$expand=FabOperacao($filter=codigo eq 13)",
			expectedKey:   "$expand",
			expectedValue: "FabOperacao($filter=codigo eq 13)",
		},
		{
			name:          "Parameter with equals in value",
			param:         "$filter=nome eq 'João = Silva'",
			expectedKey:   "$filter",
			expectedValue: "nome eq 'João = Silva'",
		},
		{
			name:          "Parameter without value",
			param:         "$count",
			expectedKey:   "$count",
			expectedValue: "",
		},
		{
			name:          "Empty parameter",
			param:         "",
			expectedKey:   "",
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := parser.parseParam(tt.param)
			if key != tt.expectedKey {
				t.Errorf("parseParam() key = %q, expected = %q", key, tt.expectedKey)
			}
			if value != tt.expectedValue {
				t.Errorf("parseParam() value = %q, expected = %q", value, tt.expectedValue)
			}
		})
	}
}

func TestURLParser_NormalizeODataQuery(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "Multiple spaces",
			query:    "$filter=nome    eq     'João'",
			expected: "$filter=nome eq 'João'",
		},
		{
			name:     "Tabs and newlines",
			query:    "$filter=nome\teq\n'João'",
			expected: "$filter=nome eq 'João'",
		},
		{
			name:     "Spaces in quotes preserved",
			query:    "$filter=nome eq 'João    Silva'",
			expected: "$filter=nome eq 'João    Silva'",
		},
		{
			name:     "Leading and trailing spaces",
			query:    "  $filter=nome eq 'João'  ",
			expected: "$filter=nome eq 'João'",
		},
		{
			name:     "Empty query",
			query:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.NormalizeODataQuery(tt.query)
			if result != tt.expected {
				t.Errorf("NormalizeODataQuery() = %q, expected = %q", result, tt.expected)
			}
		})
	}
}

func TestURLParser_CompareWithStandardParser(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		rawQuery string
		testKey  string
	}{
		{
			name:     "Simple filter",
			rawQuery: "$filter=nome eq 'João'",
			testKey:  "$filter",
		},
		{
			name:     "Multiple parameters",
			rawQuery: "$filter=ativo eq 'S'&$orderby=nome asc&$top=10",
			testKey:  "$filter",
		},
		{
			name:     "URL encoded",
			rawQuery: "$filter=nome%20eq%20'João'",
			testKey:  "$filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parser customizado
			customValues, err := parser.ParseQuery(tt.rawQuery)
			if err != nil {
				t.Errorf("Custom parser error: %v", err)
				return
			}

			// Parser padrão
			standardValues, err := url.ParseQuery(tt.rawQuery)
			if err != nil {
				t.Errorf("Standard parser error: %v", err)
				return
			}

			// Compara os valores para a chave de teste
			customValue := customValues.Get(tt.testKey)
			standardValue := standardValues.Get(tt.testKey)

			t.Logf("Custom: %q, Standard: %q", customValue, standardValue)

			// Para casos simples, os valores devem ser iguais ou o customizado deve ser melhor
			if customValue == "" && standardValue != "" {
				t.Errorf("Custom parser failed to parse %s", tt.testKey)
			}
		})
	}
}

func TestURLParser_ExtractODataSystemParams(t *testing.T) {
	parser := NewURLParser()

	// Cria url.Values de teste
	values := url.Values{}
	values.Add("$filter", "nome eq 'João'")
	values.Add("$orderby", "data desc")
	values.Add("$top", "10")
	values.Add("custom", "value")

	result := parser.ExtractODataSystemParams(values)

	expected := map[string]string{
		"$filter":  "nome eq 'João'",
		"$orderby": "data desc",
		"$top":     "10",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists || actualValue != expectedValue {
			t.Errorf("ExtractODataSystemParams() %s = %q, expected = %q", key, actualValue, expectedValue)
		}
	}

	// Verifica que parâmetros customizados não foram incluídos
	if _, exists := result["custom"]; exists {
		t.Error("ExtractODataSystemParams() included non-system parameter")
	}
}

func TestOptimizedParserSemicolon(t *testing.T) {
	query := "$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc)"

	parser := NewURLParser()

	fmt.Printf("Testing query: %s\n", query)

	result, err := parser.ParseQueryFast(query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		t.Errorf("ParseQueryFast failed: %v", err)
		return
	}

	fmt.Printf("Result: %v\n", result)

	// Test validation
	err = parser.ValidateODataQueryFast(query)
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
		t.Errorf("ValidateODataQueryFast failed: %v", err)
		return
	}

	fmt.Println("Test passed!")
}

// Benchmarks migrados do arquivo url_parser_benchmark_test.go

// BenchmarkURLParsers_SimpleQuery compara performance em queries simples
func BenchmarkURLParsers_SimpleQuery(b *testing.B) {
	simpleQuery := "$filter=ativo eq 'S'&$top=10&$orderby=id desc"

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.ParseQuery(simpleQuery)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.ParseQueryFast(simpleQuery)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_ComplexQuery compara performance em queries complexas
func BenchmarkURLParsers_ComplexQuery(b *testing.B) {
	complexQuery := "$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc;$expand=FabTarefa($orderby=id desc))&$filter=ativo eq 'S'&$top=10&$skip=5"

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.ParseQuery(complexQuery)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.ParseQueryFast(complexQuery)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_VeryComplexQuery compara performance em queries muito complexas
func BenchmarkURLParsers_VeryComplexQuery(b *testing.B) {
	veryComplexQuery := "$expand=FabOperacao($filter=codigo eq 13 and descricao eq 'Test';$orderby=id desc,descricao asc;$expand=FabTarefa($filter=nome_classe eq 'TestClass';$orderby=id desc;$top=5))&$filter=ativo eq 'S' and codigo gt 10&$orderby=id desc,descricao asc&$top=10&$skip=5&$count=true"

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.ParseQuery(veryComplexQuery)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := parser.ParseQueryFast(veryComplexQuery)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_MixedQueries compara performance em queries mistas
func BenchmarkURLParsers_MixedQueries(b *testing.B) {
	queries := []string{
		"$filter=ativo eq 'S'",
		"$filter=codigo eq 13&$orderby=id desc",
		"$expand=FabOperacao($filter=codigo eq 13)",
		"$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc)",
		"$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc;$expand=FabTarefa($orderby=id desc))",
		"$filter=ativo eq 'S'&$orderby=id desc&$top=10&$skip=5",
		"$filter=contains(descricao,'Test')&$orderby=descricao asc",
		"$count=true&$top=10",
		"$search=test&$filter=ativo eq 'S'",
		"$compute=codigo mul 2 as codigoDouble&$filter=ativo eq 'S'",
	}

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, err := parser.ParseQuery(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, err := parser.ParseQueryFast(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_Validation compara performance de validação
func BenchmarkURLParsers_Validation(b *testing.B) {
	queries := []string{
		"$filter=ativo eq 'S'",
		"$filter=(codigo eq 13) and (ativo eq 'S')",
		"$expand=FabOperacao($filter=codigo eq 13)",
		"$filter=contains(descricao,'Test')",
		"$filter=ativo eq 'S'&$orderby=id desc",
	}

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			err := parser.ValidateODataQuery(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			err := parser.ValidateODataQueryFast(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_Normalization compara performance de normalização
func BenchmarkURLParsers_Normalization(b *testing.B) {
	queries := []string{
		"$filter=ativo   eq    'S'",
		"$filter=(codigo eq 13)  and  (ativo eq 'S')",
		"$expand=FabOperacao($filter=codigo  eq  13)",
		"$filter=contains(descricao,  'Test')",
		"$filter=ativo eq 'S'   &   $orderby=id desc",
	}

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_ = parser.NormalizeODataQuery(query)
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_ = parser.NormalizeODataQueryFast(query)
		}
	})
}

// BenchmarkURLParsers_URLEncodedQueries compara performance com queries URL-encoded
func BenchmarkURLParsers_URLEncodedQueries(b *testing.B) {
	encodedQueries := []string{
		"%24filter%3Dativo%20eq%20%27S%27",
		"%24filter%3D%28codigo%20eq%2013%29%20and%20%28ativo%20eq%20%27S%27%29",
		"%24expand%3DFabOperacao%28%24filter%3Dcodigo%20eq%2013%29",
		"%24filter%3Dcontains%28descricao%2C%27Test%27%29",
		"%24filter%3Dativo%20eq%20%27S%27%26%24orderby%3Did%20desc",
	}

	parser := NewURLParser()

	b.Run("URLParser", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := encodedQueries[i%len(encodedQueries)]
			_, err := parser.ParseQuery(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := encodedQueries[i%len(encodedQueries)]
			_, err := parser.ParseQueryFast(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_CachePerformance testa performance do cache
func BenchmarkURLParsers_CachePerformance(b *testing.B) {
	queries := []string{
		"$filter=ativo eq 'S'",
		"$filter=codigo eq 13&$orderby=id desc",
		"$expand=FabOperacao($filter=codigo eq 13)",
		"$filter=contains(descricao,'Test')",
		"$filter=ativo eq 'S'&$orderby=id desc&$top=10",
	}

	parser := NewURLParser()

	// Pré-aquece o cache
	for _, query := range queries {
		_, _ = parser.ParseQueryFast(query)
	}

	b.Run("URLParser_WithCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, err := parser.ParseQueryFast(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("URLParser_WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Cria um novo parser a cada iteração (sem cache)
			freshParser := NewURLParser()
			query := queries[i%len(queries)]
			_, err := freshParser.ParseQueryFast(query)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkURLParsers_PerformanceComparison executa comparação completa de performance
func BenchmarkURLParsers_PerformanceComparison(b *testing.B) {
	queries := []string{
		"$filter=ativo eq 'S'",
		"$filter=codigo eq 13&$orderby=id desc",
		"$expand=FabOperacao($filter=codigo eq 13)",
		"$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc;$expand=FabTarefa($orderby=id desc))",
		"$filter=ativo eq 'S'&$orderby=id desc&$top=10&$skip=5&$count=true",
	}

	parser := NewURLParser()

	b.Run("FullComparison", func(b *testing.B) {
		b.Log("Iniciando comparação completa de performance...")

		// Teste normal
		normalStart := time.Now()
		for i := 0; i < 10000; i++ {
			query := queries[i%len(queries)]
			_, err := parser.ParseQuery(query)
			if err != nil {
				b.Fatal(err)
			}
		}
		normalDuration := time.Since(normalStart)

		// Teste otimizado
		optimizedStart := time.Now()
		for i := 0; i < 10000; i++ {
			query := queries[i%len(queries)]
			_, err := parser.ParseQueryFast(query)
			if err != nil {
				b.Fatal(err)
			}
		}
		optimizedDuration := time.Since(optimizedStart)

		improvement := float64(normalDuration.Nanoseconds()) / float64(optimizedDuration.Nanoseconds())

		b.Logf("Parser Normal: %v", normalDuration)
		b.Logf("Parser Otimizado: %v", optimizedDuration)
		b.Logf("Melhoria: %.2fx mais rápido", improvement)

		// Estatísticas do cache
		normalizeEntries, validateEntries, simpleEntries := parser.GetCacheStats()
		b.Logf("Cache Stats - Normalize: %d, Validate: %d, Simple: %d", normalizeEntries, validateEntries, simpleEntries)
	})
}

// TestPerformanceComparison é um teste wrapper para a comparação de performance
func TestPerformanceComparison(t *testing.T) {
	queries := []string{
		"$filter=ativo eq 'S'",
		"$filter=codigo eq 13&$orderby=id desc",
		"$expand=FabOperacao($filter=codigo eq 13)",
		"$expand=FabOperacao($filter=codigo eq 13;$orderby=id desc;$expand=FabTarefa($orderby=id desc))",
		"$filter=ativo eq 'S'&$orderby=id desc&$top=10&$skip=5&$count=true",
	}

	parser := NewURLParser()

	t.Log("Iniciando teste de comparação de performance...")

	// Teste normal
	normalStart := time.Now()
	for i := 0; i < 1000; i++ {
		query := queries[i%len(queries)]
		_, err := parser.ParseQuery(query)
		if err != nil {
			t.Fatal(err)
		}
	}
	normalDuration := time.Since(normalStart)

	// Teste otimizado
	optimizedStart := time.Now()
	for i := 0; i < 1000; i++ {
		query := queries[i%len(queries)]
		_, err := parser.ParseQueryFast(query)
		if err != nil {
			t.Fatal(err)
		}
	}
	optimizedDuration := time.Since(optimizedStart)

	improvement := float64(normalDuration.Nanoseconds()) / float64(optimizedDuration.Nanoseconds())

	t.Logf("Parser Normal: %v", normalDuration)
	t.Logf("Parser Otimizado: %v", optimizedDuration)
	t.Logf("Melhoria: %.2fx mais rápido", improvement)

	// Estatísticas do cache
	normalizeEntries, validateEntries, simpleEntries := parser.GetCacheStats()
	t.Logf("Cache Stats - Normalize: %d, Validate: %d, Simple: %d", normalizeEntries, validateEntries, simpleEntries)
}
