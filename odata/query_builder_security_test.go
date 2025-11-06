package odata

import (
	"context"
	"strings"
	"testing"
)

// TestSQLInjectionProtection testa proteção contra SQL injection
func TestSQLInjectionProtection(t *testing.T) {
	qb := NewQueryBuilder("mysql")
	ctx := context.Background()

	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "id", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "username", ColumnName: "username", Type: "string"},
			{Name: "email", ColumnName: "email", Type: "string"},
		},
	}

	// Testes de tentativas de SQL injection que DEVEM ser bloqueadas pelo uso de prepared statements
	injectionAttempts := []struct {
		name        string
		filter      string
		description string
	}{
		{
			name:        "union select",
			filter:      "username eq 'admin' UNION SELECT * FROM passwords--",
			description: "Tentativa de UNION para extrair dados de outra tabela",
		},
		{
			name:        "or 1=1 comment",
			filter:      "username eq 'admin'--' OR 1=1--",
			description: "Tentativa clássica de bypass com OR 1=1",
		},
		{
			name:        "drop table",
			filter:      "username eq 'admin'; DROP TABLE users;--",
			description: "Tentativa de drop table",
		},
		{
			name:        "nested quotes",
			filter:      "username eq 'admin'' OR ''1''=''1",
			description: "Tentativa com aspas aninhadas",
		},
		{
			name:        "hex injection",
			filter:      "username eq 0x61646d696e",
			description: "Tentativa com valores hexadecimais",
		},
	}

	for _, tt := range injectionAttempts {
		t.Run(tt.name, func(t *testing.T) {
			// Parse o filter (isso pode falhar na validação, o que é esperado)
			filterQuery, err := ParseFilterString(ctx, tt.filter)

			// Se o parser detectou o problema, ok
			if err != nil {
				t.Logf("✅ Parser detectou e rejeitou: %v", err)
				return
			}

			// Se passou no parser, verifica se a query gerada usa prepared statements
			if filterQuery != nil && filterQuery.Tree != nil {
				whereClause, args, err := qb.BuildWhereClause(ctx, filterQuery.Tree, metadata)

				// Mesmo que a query seja gerada, ela DEVE usar placeholders (?)
				if err == nil && whereClause != "" {
					// Verifica se NÃO há valores literais maliciosos na query
					dangerousPatterns := []string{
						"UNION", "DROP", "DELETE FROM", "INSERT INTO",
						"UPDATE ", "--", "/*", "*/", "';",
					}

					whereUpper := strings.ToUpper(whereClause)
					for _, pattern := range dangerousPatterns {
						if strings.Contains(whereUpper, pattern) {
							t.Errorf("❌ VULNERABILIDADE: Query contém padrão perigoso '%s': %s", pattern, whereClause)
						}
					}

					// Verifica se há argumentos parametrizados
					if len(args) > 0 {
						t.Logf("✅ Query usa %d argumentos parametrizados (seguro)", len(args))
						t.Logf("   Query: %s", whereClause)
						t.Logf("   Args: %v", args)
					}
				}
			}
		})
	}
}

// TestPreparedStatementsUsage verifica que prepared statements são usados consistentemente
func TestPreparedStatementsUsage(t *testing.T) {
	qb := NewQueryBuilder("mysql")
	ctx := context.Background()

	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "username", ColumnName: "username", Type: "string"},
			{Name: "age", ColumnName: "age", Type: "int64"},
		},
	}

	testCases := []struct {
		name       string
		filter     string
		expectArgs bool
		minArgs    int
	}{
		{
			name:       "simple equality",
			filter:     "username eq 'john'",
			expectArgs: true,
			minArgs:    1,
		},
		{
			name:       "numeric comparison",
			filter:     "age gt 18",
			expectArgs: true,
			minArgs:    1,
		},
		{
			name:       "complex and",
			filter:     "username eq 'john' and age gt 18",
			expectArgs: true,
			minArgs:    2,
		},
		{
			name:       "contains function",
			filter:     "contains(username, 'admin')",
			expectArgs: true,
			minArgs:    1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterQuery, err := ParseFilterString(ctx, tc.filter)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			whereClause, args, err := qb.BuildWhereClause(ctx, filterQuery.Tree, metadata)
			if err != nil {
				t.Fatalf("BuildWhereClause error: %v", err)
			}

			if tc.expectArgs {
				if len(args) < tc.minArgs {
					t.Errorf("Expected at least %d prepared statement args, got %d", tc.minArgs, len(args))
				}

				// Verifica se a query contém placeholders
				if !strings.Contains(whereClause, ":param") {
					t.Errorf("Query doesn't contain placeholders: %s", whereClause)
				}

				t.Logf("✅ Query uses %d prepared statement args", len(args))
				t.Logf("   Query: %s", whereClause)
			}
		})
	}
}

// TestNoRawValueInjection verifica que valores não são injetados diretamente
func TestNoRawValueInjection(t *testing.T) {
	qb := NewQueryBuilder("postgresql")
	ctx := context.Background()

	metadata := EntityMetadata{
		Name:      "Products",
		TableName: "products",
		Properties: []PropertyMetadata{
			{Name: "name", ColumnName: "name", Type: "string"},
			{Name: "price", ColumnName: "price", Type: "float64"},
		},
	}

	// Valores potencialmente perigosos
	dangerousValues := []string{
		"'; DROP TABLE products; --",
		"' OR '1'='1",
		"admin'--",
		"1' UNION SELECT NULL--",
	}

	for _, dangerousValue := range dangerousValues {
		t.Run("dangerous_value_"+dangerousValue, func(t *testing.T) {
			filter := "name eq '" + dangerousValue + "'"

			filterQuery, err := ParseFilterString(ctx, filter)

			if err != nil {
				t.Logf("✅ Parser rejected dangerous value")
				return
			}

			if filterQuery != nil && filterQuery.Tree != nil {
				whereClause, args, err := qb.BuildWhereClause(ctx, filterQuery.Tree, metadata)

				if err == nil {
					// O valor perigoso NÃO deve aparecer literalmente na query
					if strings.Contains(whereClause, dangerousValue) {
						t.Errorf("❌ VULNERABILIDADE: Valor perigoso aparece literalmente na query")
					}

					// Deve estar nos args parametrizados
					found := false
					for _, arg := range args {
						if argStr, ok := arg.(string); ok {
							if strings.Contains(argStr, dangerousValue) {
								found = true
								t.Logf("✅ Valor perigoso está seguramente nos args parametrizados")
								break
							}
						}
					}

					if !found {
						t.Logf("⚠️  Valor não encontrado, pode ter sido rejeitado")
					}
				}
			}
		})
	}
}

// TestPropertyNameValidation verifica validação de nomes de propriedades
func TestPropertyNameValidation(t *testing.T) {
	qb := NewQueryBuilder("mysql")

	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "id", ColumnName: "id", Type: "int64"},
			{Name: "username", ColumnName: "username", Type: "string"},
		},
	}

	tests := []struct {
		name         string
		propertyName string
		shouldError  bool
	}{
		{"valid property", "username", false},
		{"invalid property with sql", "username; DROP TABLE", true},
		{"non-existent property", "nonexistent", true},
		{"property with spaces", "user name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := qb.buildPropertyExpression(&ParseNode{
				Token: &Token{
					Type:  int(FilterTokenProperty),
					Value: tt.propertyName,
				},
			}, metadata)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error for property '%s', got nil", tt.propertyName)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for property '%s': %v", tt.propertyName, err)
			}
		})
	}
}
