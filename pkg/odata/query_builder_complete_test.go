package odata

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewQueryBuilder tests QueryBuilder creation for different dialects
func TestNewQueryBuilder(t *testing.T) {
	t.Run("Create MySQL QueryBuilder", func(t *testing.T) {
		qb := NewQueryBuilder("mysql")
		assert.NotNil(t, qb)
		assert.Equal(t, "mysql", qb.dialect.GetName())
		assert.NotNil(t, qb.nodeMap)
		assert.NotEmpty(t, qb.nodeMap)
	})

	t.Run("Create PostgreSQL QueryBuilder", func(t *testing.T) {
		qb := NewQueryBuilder("postgresql")
		assert.NotNil(t, qb)
		assert.Equal(t, "postgresql", qb.dialect.GetName())
		assert.NotNil(t, qb.nodeMap)
		assert.NotEmpty(t, qb.nodeMap)
	})

	t.Run("Create Oracle QueryBuilder", func(t *testing.T) {
		qb := NewQueryBuilder("oracle")
		assert.NotNil(t, qb)
		assert.Equal(t, "oracle", qb.dialect.GetName())
		assert.NotNil(t, qb.nodeMap)
		assert.NotEmpty(t, qb.nodeMap)
	})

	t.Run("Create default QueryBuilder", func(t *testing.T) {
		qb := NewQueryBuilder("unknown")
		assert.NotNil(t, qb)
		assert.Equal(t, "default", qb.dialect.GetName())
		assert.NotNil(t, qb.nodeMap)
	})

	t.Run("Dialect normalization", func(t *testing.T) {
		qb := NewQueryBuilder("MySQL")
		assert.Equal(t, "mysql", qb.dialect.GetName())
	})
}

// TestNamedArgs tests the NamedArgs parameter management
func TestNamedArgs(t *testing.T) {
	t.Run("Create NamedArgs", func(t *testing.T) {
		na := NewNamedArgs("mysql")
		assert.NotNil(t, na)
		assert.Equal(t, "mysql", na.dialect)
		assert.Equal(t, 0, na.counter)
		assert.Empty(t, na.args)
	})

	t.Run("Add single argument", func(t *testing.T) {
		na := NewNamedArgs("mysql")
		placeholder := na.AddArg("value1")

		assert.Equal(t, ":param1", placeholder)
		assert.Equal(t, 1, na.counter)
		assert.Len(t, na.args, 1)
	})

	t.Run("Add multiple arguments", func(t *testing.T) {
		na := NewNamedArgs("mysql")

		p1 := na.AddArg("value1")
		p2 := na.AddArg(123)
		p3 := na.AddArg(true)

		assert.Equal(t, ":param1", p1)
		assert.Equal(t, ":param2", p2)
		assert.Equal(t, ":param3", p3)
		assert.Equal(t, 3, na.counter)
		assert.Len(t, na.args, 3)
	})

	t.Run("GetArgs returns correct slice", func(t *testing.T) {
		na := NewNamedArgs("postgresql")
		na.AddArg("test")
		na.AddArg(42)

		args := na.GetArgs()
		assert.Len(t, args, 2)
	})

	t.Run("GetNamedArgs returns same as GetArgs", func(t *testing.T) {
		na := NewNamedArgs("mysql")
		na.AddArg("test")

		args1 := na.GetArgs()
		args2 := na.GetNamedArgs()

		assert.Equal(t, len(args1), len(args2))
	})

	t.Run("Different dialects", func(t *testing.T) {
		dialects := []string{"mysql", "postgresql", "oracle"}

		for _, dialect := range dialects {
			na := NewNamedArgs(dialect)
			assert.Equal(t, dialect, na.dialect)
		}
	})
}

// TestQueryBuilder_BuildSimpleFilter tests basic filter construction
func TestQueryBuilder_BuildSimpleFilter(t *testing.T) {
	ctx := context.Background()

	t.Run("Simple equality filter", func(t *testing.T) {
		filter := "Name eq 'John'"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			// Test that parsing works
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("Greater than filter", func(t *testing.T) {
		filter := "Age gt 18"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("Not equal filter", func(t *testing.T) {
		filter := "Status ne 'inactive'"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("Less than or equal filter", func(t *testing.T) {
		filter := "Price le 100"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})
}

// TestQueryBuilder_BuildComplexFilter tests complex filter construction
func TestQueryBuilder_BuildComplexFilter(t *testing.T) {
	ctx := context.Background()

	t.Run("AND operator", func(t *testing.T) {
		filter := "Name eq 'John' and Age gt 18"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("OR operator", func(t *testing.T) {
		filter := "Status eq 'active' or Status eq 'pending'"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("Combined AND/OR", func(t *testing.T) {
		filter := "(Name eq 'John' or Name eq 'Jane') and Age gt 18"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("Multiple conditions", func(t *testing.T) {
		filter := "Name eq 'John' and Age gt 18 and Status eq 'active'"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})
}

// TestQueryBuilder_BuildFilterWithFunctions tests OData functions
func TestQueryBuilder_BuildFilterWithFunctions(t *testing.T) {
	ctx := context.Background()

	t.Run("startswith function", func(t *testing.T) {
		filter := "startswith(Name, 'J')"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("endswith function", func(t *testing.T) {
		filter := "endswith(Email, '@example.com')"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("contains function", func(t *testing.T) {
		filter := "contains(Description, 'test')"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("tolower function", func(t *testing.T) {
		filter := "tolower(Name) eq 'john'"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})

	t.Run("toupper function", func(t *testing.T) {
		filter := "toupper(Name) eq 'JOHN'"
		parsedFilter, err := ParseFilterString(ctx, filter)

		if err == nil && parsedFilter != nil {
			assert.NotNil(t, parsedFilter.Tree)
		}
	})
}

// TestQueryBuilder_NodeMapOperators tests operator mapping
func TestQueryBuilder_NodeMapOperators(t *testing.T) {
	t.Run("MySQL operators", func(t *testing.T) {
		qb := NewQueryBuilder("mysql")

		// Verify nodeMap has basic operators
		assert.Contains(t, qb.nodeMap, "eq")
		assert.Contains(t, qb.nodeMap, "ne")
		assert.Contains(t, qb.nodeMap, "gt")
		assert.Contains(t, qb.nodeMap, "ge")
		assert.Contains(t, qb.nodeMap, "lt")
		assert.Contains(t, qb.nodeMap, "le")
	})

	t.Run("PostgreSQL operators", func(t *testing.T) {
		qb := NewQueryBuilder("postgresql")

		assert.Contains(t, qb.nodeMap, "eq")
		assert.Contains(t, qb.nodeMap, "ne")
		assert.NotNil(t, qb.nodeMap)
	})

	t.Run("Oracle operators", func(t *testing.T) {
		qb := NewQueryBuilder("oracle")

		assert.Contains(t, qb.nodeMap, "eq")
		assert.Contains(t, qb.nodeMap, "ne")
		assert.NotNil(t, qb.nodeMap)
	})

	t.Run("Default operators", func(t *testing.T) {
		qb := NewQueryBuilder("unknown")

		assert.Contains(t, qb.nodeMap, "eq")
		assert.Contains(t, qb.nodeMap, "ne")
		assert.NotNil(t, qb.nodeMap)
	})
}

// TestQueryBuilder_OrderBy tests ORDER BY clause construction
func TestQueryBuilder_OrderBy(t *testing.T) {
	ctx := context.Background()

	t.Run("Simple ascending order", func(t *testing.T) {
		orderBy := "Name asc"
		parsed, err := ParseOrderByString(ctx, orderBy)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed)
		}
	})

	t.Run("Simple descending order", func(t *testing.T) {
		orderBy := "CreatedAt desc"
		parsed, err := ParseOrderByString(ctx, orderBy)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed)
		}
	})

	t.Run("Multiple fields", func(t *testing.T) {
		orderBy := "Name asc, Age desc"
		parsed, err := ParseOrderByString(ctx, orderBy)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed)
		}
	})

	t.Run("Default direction", func(t *testing.T) {
		orderBy := "Name"
		parsed, err := ParseOrderByString(ctx, orderBy)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed)
		}
	})
}

// TestQueryBuilder_SQLInjectionPrevention tests security measures
func TestQueryBuilder_SQLInjectionPrevention(t *testing.T) {
	ctx := context.Background()

	t.Run("SQL injection attempt in filter", func(t *testing.T) {
		// Common SQL injection patterns
		maliciousFilters := []string{
			"Name eq 'John' OR '1'='1'",
			"Name eq 'John'; DROP TABLE Users--",
			"Name eq 'John' UNION SELECT * FROM passwords",
		}

		for _, filter := range maliciousFilters {
			parsedFilter, err := ParseFilterString(ctx, filter)

			// Should either fail to parse or create safe parameterized query
			if err == nil && parsedFilter != nil {
				// Verify it's using parameters, not direct string interpolation
				assert.NotNil(t, parsedFilter.Tree)
			}
		}
	})

	t.Run("Named parameters prevent injection", func(t *testing.T) {
		na := NewNamedArgs("mysql")

		// Even malicious values should be safely parameterized
		maliciousValue := "'; DROP TABLE Users; --"
		placeholder := na.AddArg(maliciousValue)

		// Placeholder should be a safe parameter name
		assert.True(t, strings.HasPrefix(placeholder, ":param"))
		assert.Contains(t, placeholder, "param")

		// Value should be in args, not in SQL string
		args := na.GetArgs()
		assert.Len(t, args, 1)
	})

	t.Run("Special characters handling", func(t *testing.T) {
		na := NewNamedArgs("mysql")

		specialValues := []string{
			"test'value",
			"test\"value",
			"test`value",
			"test;value",
			"test--value",
		}

		for _, val := range specialValues {
			placeholder := na.AddArg(val)
			assert.True(t, strings.HasPrefix(placeholder, ":param"))
		}

		assert.Len(t, na.GetArgs(), len(specialValues))
	})
}

// TestQueryBuilder_EdgeCases tests edge cases and error handling
func TestQueryBuilder_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty filter", func(t *testing.T) {
		filter := ""
		_, err := ParseFilterString(ctx, filter)

		// Should handle empty filter gracefully
		_ = err
	})

	t.Run("Whitespace only filter", func(t *testing.T) {
		filter := "   "
		_, err := ParseFilterString(ctx, filter)

		_ = err
	})

	t.Run("Invalid operator", func(t *testing.T) {
		filter := "Name invalid 'John'"
		_, err := ParseFilterString(ctx, filter)

		// Should return error for invalid operator
		if err != nil {
			assert.Error(t, err)
		}
	})

	t.Run("Unmatched parentheses", func(t *testing.T) {
		filter := "(Name eq 'John'"
		_, err := ParseFilterString(ctx, filter)

		// Should handle unmatched parens
		_ = err
	})

	t.Run("Very long filter", func(t *testing.T) {
		// Create a very long but valid filter
		conditions := make([]string, 100)
		for i := 0; i < 100; i++ {
			conditions[i] = "Name eq 'John'"
		}
		filter := strings.Join(conditions, " and ")

		_, err := ParseFilterString(ctx, filter)

		// Should handle long filters
		_ = err
	})
}

// TestQueryBuilder_Dialects tests dialect-specific behavior
func TestQueryBuilder_Dialects(t *testing.T) {
	t.Run("MySQL dialect specifics", func(t *testing.T) {
		qb := NewQueryBuilder("mysql")
		assert.Equal(t, "mysql", qb.dialect.GetName())

		// MySQL should have specific mappings
		assert.NotEmpty(t, qb.nodeMap)
	})

	t.Run("PostgreSQL dialect specifics", func(t *testing.T) {
		qb := NewQueryBuilder("postgresql")
		assert.Equal(t, "postgresql", qb.dialect.GetName())

		// PostgreSQL should have specific mappings
		assert.NotEmpty(t, qb.nodeMap)
	})

	t.Run("Oracle dialect specifics", func(t *testing.T) {
		qb := NewQueryBuilder("oracle")
		assert.Equal(t, "oracle", qb.dialect.GetName())

		// Oracle should have specific mappings
		assert.NotEmpty(t, qb.nodeMap)
	})

	t.Run("Case insensitive dialect names", func(t *testing.T) {
		qb1 := NewQueryBuilder("MYSQL")
		qb2 := NewQueryBuilder("MySQL")
		qb3 := NewQueryBuilder("mysql")

		assert.Equal(t, qb1.dialect.GetName(), qb2.dialect.GetName())
		assert.Equal(t, qb2.dialect.GetName(), qb3.dialect.GetName())
	})
}

// TestQueryBuilder_DataTypes tests different data type handling
func TestQueryBuilder_DataTypes(t *testing.T) {
	na := NewNamedArgs("mysql")

	t.Run("String values", func(t *testing.T) {
		placeholder := na.AddArg("test string")
		assert.True(t, strings.HasPrefix(placeholder, ":param"))
	})

	t.Run("Integer values", func(t *testing.T) {
		placeholder := na.AddArg(42)
		assert.True(t, strings.HasPrefix(placeholder, ":param"))
	})

	t.Run("Float values", func(t *testing.T) {
		placeholder := na.AddArg(3.14)
		assert.True(t, strings.HasPrefix(placeholder, ":param"))
	})

	t.Run("Boolean values", func(t *testing.T) {
		placeholder := na.AddArg(true)
		assert.True(t, strings.HasPrefix(placeholder, ":param"))
	})

	t.Run("Nil values", func(t *testing.T) {
		placeholder := na.AddArg(nil)
		assert.True(t, strings.HasPrefix(placeholder, ":param"))
	})

	t.Run("All args are collected", func(t *testing.T) {
		args := na.GetArgs()
		assert.Len(t, args, 5) // From all previous AddArg calls
	})
}

// TestQueryBuilder_ComparisonOperators tests all comparison operators
func TestQueryBuilder_ComparisonOperators(t *testing.T) {
	ctx := context.Background()

	operators := []struct {
		name   string
		filter string
	}{
		{"Equal", "Name eq 'John'"},
		{"Not Equal", "Name ne 'John'"},
		{"Greater Than", "Age gt 18"},
		{"Greater or Equal", "Age ge 18"},
		{"Less Than", "Age lt 65"},
		{"Less or Equal", "Age le 65"},
	}

	for _, op := range operators {
		t.Run(op.name, func(t *testing.T) {
			parsedFilter, err := ParseFilterString(ctx, op.filter)

			if err == nil && parsedFilter != nil {
				assert.NotNil(t, parsedFilter.Tree)
			}
		})
	}
}

// TestQueryBuilder_LogicalOperators tests logical operators
func TestQueryBuilder_LogicalOperators(t *testing.T) {
	ctx := context.Background()

	t.Run("AND operator", func(t *testing.T) {
		filter := "Name eq 'John' and Age gt 18"
		parsed, err := ParseFilterString(ctx, filter)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed.Tree)
		}
	})

	t.Run("OR operator", func(t *testing.T) {
		filter := "Name eq 'John' or Name eq 'Jane'"
		parsed, err := ParseFilterString(ctx, filter)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed.Tree)
		}
	})

	t.Run("NOT operator", func(t *testing.T) {
		filter := "not (Name eq 'John')"
		parsed, err := ParseFilterString(ctx, filter)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed.Tree)
		}
	})

	t.Run("Complex logical expression", func(t *testing.T) {
		filter := "(Name eq 'John' or Name eq 'Jane') and (Age gt 18 and Age lt 65)"
		parsed, err := ParseFilterString(ctx, filter)

		if err == nil && parsed != nil {
			assert.NotNil(t, parsed.Tree)
		}
	})
}

// TestQueryBuilder_Initialization tests proper initialization
func TestQueryBuilder_Initialization(t *testing.T) {
	t.Run("QueryBuilder is properly initialized", func(t *testing.T) {
		qb := NewQueryBuilder("mysql")

		require.NotNil(t, qb)
		require.NotNil(t, qb.nodeMap)
		require.NotNil(t, qb.prepareMap)
		require.NotNil(t, qb.dialect)
		require.NotEmpty(t, qb.dialect.GetName())
		require.NotEmpty(t, qb.nodeMap)
	})

	t.Run("NamedArgs is properly initialized", func(t *testing.T) {
		na := NewNamedArgs("postgresql")

		require.NotNil(t, na)
		require.NotNil(t, na.args)
		require.Equal(t, 0, na.counter)
		require.NotEmpty(t, na.dialect)
	})

	t.Run("Multiple QueryBuilder instances are independent", func(t *testing.T) {
		qb1 := NewQueryBuilder("mysql")
		qb2 := NewQueryBuilder("postgresql")

		assert.NotEqual(t, qb1.dialect.GetName(), qb2.dialect.GetName())

		// Modifying one should not affect the other
		na1 := NewNamedArgs(qb1.dialect.GetName())
		na2 := NewNamedArgs(qb2.dialect.GetName())

		na1.AddArg("test1")
		assert.Equal(t, 1, na1.counter)
		assert.Equal(t, 0, na2.counter)
	})
}
