package odata

import (
	"context"
	"testing"
)

func TestParseFilterString(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		expectError bool
		description string
	}{
		{
			name:        "Empty filter",
			filter:      "",
			expectError: false,
			description: "Empty filter should return nil",
		},
		{
			name:        "Simple equality",
			filter:      "Name eq 'John'",
			expectError: false,
			description: "Simple equality comparison",
		},
		{
			name:        "Numeric comparison",
			filter:      "Age gt 18",
			expectError: false,
			description: "Numeric greater than comparison",
		},
		{
			name:        "Boolean comparison",
			filter:      "IsActive eq true",
			expectError: false,
			description: "Boolean equality comparison",
		},
		{
			name:        "DateTime comparison",
			filter:      "CreatedDate gt 2023-01-01T00:00:00Z",
			expectError: false,
			description: "DateTime comparison",
		},
		{
			name:        "GUID comparison",
			filter:      "Id eq 12345678-1234-1234-1234-123456789012",
			expectError: false,
			description: "GUID comparison",
		},
		{
			name:        "Complex AND expression",
			filter:      "Name eq 'John' and Age gt 18",
			expectError: false,
			description: "Complex AND expression",
		},
		{
			name:        "Complex OR expression",
			filter:      "Name eq 'John' or Name eq 'Jane'",
			expectError: false,
			description: "Complex OR expression",
		},
		{
			name:        "Function call",
			filter:      "contains(Name, 'John')",
			expectError: false,
			description: "Function call with contains",
		},
		{
			name:        "Invalid syntax",
			filter:      "Name eq",
			expectError: true,
			description: "Invalid syntax should return error",
		},
		{
			name:        "Invalid operator",
			filter:      "Name invalid 'John'",
			expectError: true,
			description: "Invalid operator should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := ParseFilterString(ctx, tt.filter)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for filter '%s', but got none", tt.filter)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for filter '%s': %v", tt.filter, err)
				return
			}

			if tt.filter == "" {
				if result != nil {
					t.Errorf("Expected nil result for empty filter, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected non-nil result for filter '%s'", tt.filter)
				return
			}

			if result.RawValue != tt.filter {
				t.Errorf("Expected RawValue '%s', got '%s'", tt.filter, result.RawValue)
			}

			if result.Tree == nil {
				t.Errorf("Expected non-nil Tree for filter '%s'", tt.filter)
			}
		})
	}
}

func TestSemanticizeFilterQuery(t *testing.T) {
	metadata := EntityMetadata{
		Name: "TestEntity",
		Properties: []PropertyMetadata{
			{Name: "Name", Type: "string"},
			{Name: "Age", Type: "int"},
			{Name: "IsActive", Type: "bool"},
		},
	}

	tests := []struct {
		name        string
		filter      string
		expectError bool
		description string
	}{
		{
			name:        "Valid property",
			filter:      "Name eq 'John'",
			expectError: false,
			description: "Valid property should pass validation",
		},
		{
			name:        "Invalid property",
			filter:      "InvalidProperty eq 'John'",
			expectError: true,
			description: "Invalid property should fail validation",
		},
		{
			name:        "Multiple valid properties",
			filter:      "Name eq 'John' and Age gt 18",
			expectError: false,
			description: "Multiple valid properties should pass validation",
		},
		{
			name:        "Mixed valid and invalid properties",
			filter:      "Name eq 'John' and InvalidProperty gt 18",
			expectError: true,
			description: "Mixed valid and invalid properties should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			filterQuery, err := ParseFilterString(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tt.filter, err)
			}

			err = SemanticizeFilterQuery(filterQuery, metadata)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for filter '%s', but got none", tt.filter)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for filter '%s': %v", tt.filter, err)
			}
		})
	}
}

func TestGetFilterProperties(t *testing.T) {
	tests := []struct {
		name               string
		filter             string
		expectedProperties []string
		description        string
	}{
		{
			name:               "Single property",
			filter:             "Name eq 'John'",
			expectedProperties: []string{"Name"},
			description:        "Single property should be extracted",
		},
		{
			name:               "Multiple properties",
			filter:             "Name eq 'John' and Age gt 18",
			expectedProperties: []string{"Name", "Age"},
			description:        "Multiple properties should be extracted",
		},
		{
			name:               "Duplicate properties",
			filter:             "Name eq 'John' and Name ne 'Jane'",
			expectedProperties: []string{"Name"},
			description:        "Duplicate properties should be deduplicated",
		},
		{
			name:               "Function with properties",
			filter:             "contains(Name, 'John') and Age gt 18",
			expectedProperties: []string{"Name", "Age"},
			description:        "Properties in functions should be extracted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			filterQuery, err := ParseFilterString(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tt.filter, err)
			}

			properties := GetFilterProperties(filterQuery)

			if len(properties) != len(tt.expectedProperties) {
				t.Errorf("Expected %d properties, got %d", len(tt.expectedProperties), len(properties))
				return
			}

			// Converte para map para comparação independente de ordem
			expectedMap := make(map[string]bool)
			for _, prop := range tt.expectedProperties {
				expectedMap[prop] = true
			}

			for _, prop := range properties {
				if !expectedMap[prop] {
					t.Errorf("Unexpected property '%s' in results", prop)
				}
			}
		})
	}
}

func TestIsSimpleFilter(t *testing.T) {
	tests := []struct {
		name         string
		filter       string
		expectSimple bool
		description  string
	}{
		{
			name:         "Simple equality",
			filter:       "Name eq 'John'",
			expectSimple: true,
			description:  "Simple equality should be considered simple",
		},
		{
			name:         "Simple numeric comparison",
			filter:       "Age gt 18",
			expectSimple: true,
			description:  "Simple numeric comparison should be considered simple",
		},
		{
			name:         "Complex AND expression",
			filter:       "Name eq 'John' and Age gt 18",
			expectSimple: false,
			description:  "Complex AND expression should not be considered simple",
		},
		{
			name:         "Function call",
			filter:       "contains(Name, 'John')",
			expectSimple: false,
			description:  "Function call should not be considered simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			filterQuery, err := ParseFilterString(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tt.filter, err)
			}

			isSimple := IsSimpleFilter(filterQuery)

			if isSimple != tt.expectSimple {
				t.Errorf("Expected IsSimpleFilter to return %v for '%s', got %v", tt.expectSimple, tt.filter, isSimple)
			}
		})
	}
}

func TestGetFilterComplexity(t *testing.T) {
	tests := []struct {
		name          string
		filter        string
		minComplexity int
		description   string
	}{
		{
			name:          "Simple equality",
			filter:        "Name eq 'John'",
			minComplexity: 1,
			description:   "Simple equality should have low complexity",
		},
		{
			name:          "Complex AND expression",
			filter:        "Name eq 'John' and Age gt 18",
			minComplexity: 3,
			description:   "Complex AND expression should have higher complexity",
		},
		{
			name:          "Function call",
			filter:        "contains(Name, 'John')",
			minComplexity: 2,
			description:   "Function call should have moderate complexity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			filterQuery, err := ParseFilterString(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tt.filter, err)
			}

			complexity := GetFilterComplexity(filterQuery)

			if complexity < tt.minComplexity {
				t.Errorf("Expected complexity >= %d for '%s', got %d", tt.minComplexity, tt.filter, complexity)
			}
		})
	}
}

func TestFormatFilterExpression(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		description string
	}{
		{
			name:        "Simple equality",
			filter:      "Name eq 'John'",
			description: "Simple equality should be formatted correctly",
		},
		{
			name:        "Complex expression",
			filter:      "Name eq 'John' and Age gt 18",
			description: "Complex expression should be formatted correctly",
		},
		{
			name:        "Function call",
			filter:      "contains(Name, 'John')",
			description: "Function call should be formatted correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			filterQuery, err := ParseFilterString(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Failed to parse filter '%s': %v", tt.filter, err)
			}

			formatted := FormatFilterExpression(filterQuery)

			if formatted == "" {
				t.Errorf("Expected non-empty formatted expression for '%s'", tt.filter)
			}

			// Verifica se a expressão original está preservada
			if filterQuery.RawValue != "" && formatted != filterQuery.RawValue {
				// Se RawValue existe, deve ser usado
				if formatted != filterQuery.RawValue {
					t.Errorf("Expected formatted expression to match RawValue '%s', got '%s'", filterQuery.RawValue, formatted)
				}
			}
		})
	}
}

func TestValidateFilterExpression(t *testing.T) {
	metadata := EntityMetadata{
		Name: "TestEntity",
		Properties: []PropertyMetadata{
			{Name: "Name", Type: "string"},
			{Name: "Age", Type: "int"},
		},
	}

	tests := []struct {
		name        string
		filter      string
		expectError bool
		description string
	}{
		{
			name:        "Valid filter",
			filter:      "Name eq 'John'",
			expectError: false,
			description: "Valid filter should pass validation",
		},
		{
			name:        "Invalid property",
			filter:      "InvalidProperty eq 'John'",
			expectError: true,
			description: "Invalid property should fail validation",
		},
		{
			name:        "Invalid syntax",
			filter:      "Name eq",
			expectError: true,
			description: "Invalid syntax should fail validation",
		},
		{
			name:        "Empty filter",
			filter:      "",
			expectError: false,
			description: "Empty filter should pass validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := ValidateFilterExpression(ctx, tt.filter, metadata)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for filter '%s', but got none", tt.filter)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for filter '%s': %v", tt.filter, err)
			}
		})
	}
}
