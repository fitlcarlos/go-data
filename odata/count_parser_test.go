package odata

import (
	"encoding/json"
	"testing"
)

func TestParseCountString(t *testing.T) {
	tests := []struct {
		name        string
		count       string
		expected    bool
		expectError bool
		description string
	}{
		{
			name:        "Empty string",
			count:       "",
			expected:    false,
			expectError: false,
			description: "Empty string should return nil",
		},
		{
			name:        "True lowercase",
			count:       "true",
			expected:    true,
			expectError: false,
			description: "Lowercase true should be parsed correctly",
		},
		{
			name:        "True uppercase",
			count:       "TRUE",
			expected:    true,
			expectError: false,
			description: "Uppercase TRUE should be parsed correctly",
		},
		{
			name:        "True mixed case",
			count:       "True",
			expected:    true,
			expectError: false,
			description: "Mixed case True should be parsed correctly",
		},
		{
			name:        "False lowercase",
			count:       "false",
			expected:    false,
			expectError: false,
			description: "Lowercase false should be parsed correctly",
		},
		{
			name:        "False uppercase",
			count:       "FALSE",
			expected:    false,
			expectError: false,
			description: "Uppercase FALSE should be parsed correctly",
		},
		{
			name:        "False mixed case",
			count:       "False",
			expected:    false,
			expectError: false,
			description: "Mixed case False should be parsed correctly",
		},
		{
			name:        "Numeric true",
			count:       "1",
			expected:    true,
			expectError: false,
			description: "Numeric 1 should be parsed as true",
		},
		{
			name:        "Numeric false",
			count:       "0",
			expected:    false,
			expectError: false,
			description: "Numeric 0 should be parsed as false",
		},
		{
			name:        "Single letter true",
			count:       "t",
			expected:    true,
			expectError: false,
			description: "Single letter t should be parsed as true",
		},
		{
			name:        "Single letter false",
			count:       "f",
			expected:    false,
			expectError: false,
			description: "Single letter f should be parsed as false",
		},
		{
			name:        "With whitespace",
			count:       "  true  ",
			expected:    true,
			expectError: false,
			description: "Value with whitespace should be parsed correctly",
		},
		{
			name:        "Invalid value",
			count:       "invalid",
			expected:    false,
			expectError: true,
			description: "Invalid value should return error",
		},
		{
			name:        "Empty after trim",
			count:       "   ",
			expected:    false,
			expectError: true,
			description: "Empty after trim should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCountString(tt.count)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for count '%s', but got none", tt.count)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for count '%s': %v", tt.count, err)
				return
			}

			if tt.count == "" {
				if result != nil {
					t.Errorf("Expected nil result for empty count, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected non-nil result for count '%s'", tt.count)
				return
			}

			if bool(*result) != tt.expected {
				t.Errorf("Expected %v for count '%s', got %v", tt.expected, tt.count, bool(*result))
			}
		})
	}
}

func TestIsCountRequested(t *testing.T) {
	tests := []struct {
		name        string
		count       *GoDataCountQuery
		expected    bool
		description string
	}{
		{
			name:        "Nil count",
			count:       nil,
			expected:    false,
			description: "Nil count should return false",
		},
		{
			name:        "True count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(true); return &c }(),
			expected:    true,
			description: "True count should return true",
		},
		{
			name:        "False count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(false); return &c }(),
			expected:    false,
			description: "False count should return false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCountRequested(tt.count)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetCountValue(t *testing.T) {
	tests := []struct {
		name        string
		count       *GoDataCountQuery
		expected    bool
		description string
	}{
		{
			name:        "Nil count",
			count:       nil,
			expected:    false,
			description: "Nil count should return false",
		},
		{
			name:        "True count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(true); return &c }(),
			expected:    true,
			description: "True count should return true",
		},
		{
			name:        "False count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(false); return &c }(),
			expected:    false,
			description: "False count should return false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCountValue(tt.count)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetCountValue(t *testing.T) {
	tests := []struct {
		name        string
		value       bool
		description string
	}{
		{
			name:        "Set true",
			value:       true,
			description: "Setting true should work correctly",
		},
		{
			name:        "Set false",
			value:       false,
			description: "Setting false should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetCountValue(tt.value)

			if result == nil {
				t.Errorf("Expected non-nil result")
				return
			}

			if bool(*result) != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, bool(*result))
			}
		})
	}
}

func TestGoDataCountQuery_String(t *testing.T) {
	tests := []struct {
		name        string
		count       *GoDataCountQuery
		expected    string
		description string
	}{
		{
			name:        "Nil count",
			count:       nil,
			expected:    "false",
			description: "Nil count should return 'false'",
		},
		{
			name:        "True count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(true); return &c }(),
			expected:    "true",
			description: "True count should return 'true'",
		},
		{
			name:        "False count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(false); return &c }(),
			expected:    "false",
			description: "False count should return 'false'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.count.String()

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGoDataCountQuery_MarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		count       *GoDataCountQuery
		expected    string
		description string
	}{
		{
			name:        "Nil count",
			count:       nil,
			expected:    "false",
			description: "Nil count should marshal to 'false'",
		},
		{
			name:        "True count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(true); return &c }(),
			expected:    "true",
			description: "True count should marshal to 'true'",
		},
		{
			name:        "False count",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(false); return &c }(),
			expected:    "false",
			description: "False count should marshal to 'false'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.count.MarshalJSON()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, string(result))
			}
		})
	}
}

func TestGoDataCountQuery_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		expected    bool
		expectError bool
		description string
	}{
		{
			name:        "True value",
			data:        "true",
			expected:    true,
			expectError: false,
			description: "True value should unmarshal correctly",
		},
		{
			name:        "False value",
			data:        "false",
			expected:    false,
			expectError: false,
			description: "False value should unmarshal correctly",
		},
		{
			name:        "Quoted true",
			data:        `"true"`,
			expected:    true,
			expectError: false,
			description: "Quoted true should unmarshal correctly",
		},
		{
			name:        "Quoted false",
			data:        `"false"`,
			expected:    false,
			expectError: false,
			description: "Quoted false should unmarshal correctly",
		},
		{
			name:        "Invalid value",
			data:        `"invalid"`,
			expected:    false,
			expectError: true,
			description: "Invalid value should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count GoDataCountQuery
			err := count.UnmarshalJSON([]byte(tt.data))

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for data '%s', but got none", tt.data)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for data '%s': %v", tt.data, err)
				return
			}

			if bool(count) != tt.expected {
				t.Errorf("Expected %v for data '%s', got %v", tt.expected, tt.data, bool(count))
			}
		})
	}
}

func TestParseCountParameter(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]string
		expected    bool
		expectError bool
		expectNil   bool
		description string
	}{
		{
			name:        "No count parameter",
			params:      map[string]string{},
			expected:    false,
			expectError: false,
			expectNil:   true,
			description: "No count parameter should return nil",
		},
		{
			name:        "Valid true count",
			params:      map[string]string{"$count": "true"},
			expected:    true,
			expectError: false,
			expectNil:   false,
			description: "Valid true count should be parsed correctly",
		},
		{
			name:        "Valid false count",
			params:      map[string]string{"$count": "false"},
			expected:    false,
			expectError: false,
			expectNil:   false,
			description: "Valid false count should be parsed correctly",
		},
		{
			name:        "Invalid count",
			params:      map[string]string{"$count": "invalid"},
			expected:    false,
			expectError: true,
			expectNil:   false,
			description: "Invalid count should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCountParameter(tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for params %v, but got none", tt.params)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for params %v: %v", tt.params, err)
				return
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil result for params %v, got %v", tt.params, result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected non-nil result for params %v", tt.params)
				return
			}

			if bool(*result) != tt.expected {
				t.Errorf("Expected %v for params %v, got %v", tt.expected, tt.params, bool(*result))
			}
		})
	}
}

func TestValidateCountParameter(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]string
		expectError bool
		description string
	}{
		{
			name:        "No count parameter",
			params:      map[string]string{},
			expectError: false,
			description: "No count parameter should pass validation",
		},
		{
			name:        "Valid count",
			params:      map[string]string{"$count": "true"},
			expectError: false,
			description: "Valid count should pass validation",
		},
		{
			name:        "Invalid count",
			params:      map[string]string{"$count": "invalid"},
			expectError: true,
			description: "Invalid count should fail validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCountParameter(tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for params %v, but got none", tt.params)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for params %v: %v", tt.params, err)
			}
		})
	}
}

func TestGetCountSQLFragment(t *testing.T) {
	tests := []struct {
		name        string
		count       *GoDataCountQuery
		tableName   string
		expected    string
		description string
	}{
		{
			name:        "Count not requested",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(false); return &c }(),
			tableName:   "users",
			expected:    "",
			description: "Count not requested should return empty string",
		},
		{
			name:        "Count requested",
			count:       func() *GoDataCountQuery { c := GoDataCountQuery(true); return &c }(),
			tableName:   "users",
			expected:    "SELECT COUNT(*) as count FROM users",
			description: "Count requested should return SQL fragment",
		},
		{
			name:        "Nil count",
			count:       nil,
			tableName:   "users",
			expected:    "",
			description: "Nil count should return empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCountSQLFragment(tt.count, tt.tableName)

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateCountLimit(t *testing.T) {
	tests := []struct {
		name           string
		estimatedCount int
		expectError    bool
		description    string
	}{
		{
			name:           "Within limit",
			estimatedCount: 1000,
			expectError:    false,
			description:    "Count within limit should pass validation",
		},
		{
			name:           "At limit",
			estimatedCount: 1000000,
			expectError:    false,
			description:    "Count at limit should pass validation",
		},
		{
			name:           "Over limit",
			estimatedCount: 1000001,
			expectError:    true,
			description:    "Count over limit should fail validation",
		},
		{
			name:           "Zero count",
			estimatedCount: 0,
			expectError:    false,
			description:    "Zero count should pass validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCountLimit(tt.estimatedCount)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for count %d, but got none", tt.estimatedCount)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for count %d: %v", tt.estimatedCount, err)
			}
		})
	}
}

func TestCountJSONIntegration(t *testing.T) {
	// Teste de integração com JSON
	type TestStruct struct {
		Count *GoDataCountQuery `json:"count"`
	}

	tests := []struct {
		name        string
		input       string
		expected    bool
		expectError bool
		description string
	}{
		{
			name:        "JSON with true",
			input:       `{"count": true}`,
			expected:    true,
			expectError: false,
			description: "JSON with true should unmarshal correctly",
		},
		{
			name:        "JSON with false",
			input:       `{"count": false}`,
			expected:    false,
			expectError: false,
			description: "JSON with false should unmarshal correctly",
		},
		{
			name:        "JSON with string true",
			input:       `{"count": "true"}`,
			expected:    true,
			expectError: false,
			description: "JSON with string true should unmarshal correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testStruct TestStruct
			err := json.Unmarshal([]byte(tt.input), &testStruct)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input '%s', but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				return
			}

			if testStruct.Count == nil {
				t.Errorf("Expected non-nil count for input '%s'", tt.input)
				return
			}

			if bool(*testStruct.Count) != tt.expected {
				t.Errorf("Expected %v for input '%s', got %v", tt.expected, tt.input, bool(*testStruct.Count))
			}

			// Teste de marshal de volta
			marshaled, err := json.Marshal(testStruct)
			if err != nil {
				t.Errorf("Unexpected error marshaling: %v", err)
				return
			}

			var testStruct2 TestStruct
			err = json.Unmarshal(marshaled, &testStruct2)
			if err != nil {
				t.Errorf("Unexpected error unmarshaling back: %v", err)
				return
			}

			if testStruct2.Count == nil || bool(*testStruct2.Count) != tt.expected {
				t.Errorf("Round-trip marshaling failed for input '%s'", tt.input)
			}
		})
	}
}
