package odata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_resolveContentID tests Content-ID resolution in URLs
func Test_resolveContentID(t *testing.T) {
	processor := &BatchProcessor{
		server: &Server{},
	}

	t.Run("No Content-ID reference", func(t *testing.T) {
		url := "/Products(123)"
		contentIDMap := make(map[string]interface{})

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products(123)", resolved)
	})

	t.Run("Single Content-ID reference with $", func(t *testing.T) {
		url := "/Products($1)/Items"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"ID":123,"Name":"Product A"}`),
			},
		}

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products(123)/Items", resolved)
	})

	t.Run("Single Content-ID reference with ${}", func(t *testing.T) {
		url := "/Products(${1})/Items"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"ID":456,"Name":"Product B"}`),
			},
		}

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products(456)/Items", resolved)
	})

	t.Run("Multiple Content-ID references", func(t *testing.T) {
		url := "/Categories($1)/Products($2)"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"ID":10}`),
			},
			"2": &BatchOperationResponse{
				Body: []byte(`{"ID":20}`),
			},
		}

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Categories(10)/Products(20)", resolved)
	})

	t.Run("Mixed $ and ${} references", func(t *testing.T) {
		url := "/Categories($1)/Products(${2})"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"ID":100}`),
			},
			"2": &BatchOperationResponse{
				Body: []byte(`{"ID":200}`),
			},
		}

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Categories(100)/Products(200)", resolved)
	})

	t.Run("Content-ID not found in map", func(t *testing.T) {
		url := "/Products($1)/Items"
		contentIDMap := make(map[string]interface{})

		// Should return unchanged URL when Content-ID not found
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products($1)/Items", resolved)
	})

	t.Run("Invalid JSON response", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`invalid json`),
			},
		}

		// Should return unchanged URL when JSON is invalid
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products($1)", resolved)
	})

	t.Run("ID field with lowercase 'id'", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"id":789}`),
			},
		}

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products(789)", resolved)
	})

	t.Run("String ID", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"ID":"abc-123"}`),
			},
		}

		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products(abc-123)", resolved)
	})
}

// Test_extractIDFromResponse tests ID extraction from JSON responses
func Test_extractIDFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "ID in uppercase",
			body:     `{"ID":123,"Name":"Test"}`,
			expected: "123",
		},
		{
			name:     "ID in lowercase",
			body:     `{"id":456,"name":"test"}`,
			expected: "456",
		},
		{
			name:     "ID capitalized",
			body:     `{"Id":789}`,
			expected: "789",
		},
		{
			name:     "String ID",
			body:     `{"ID":"abc-123"}`,
			expected: "abc-123",
		},
		{
			name:     "@odata.id with full URL",
			body:     `{"@odata.id":"/Products(999)"}`,
			expected: "999",
		},
		{
			name:     "@odata.id with string ID",
			body:     `{"@odata.id":"/Products('xyz')"}`,
			expected: "xyz",
		},
		{
			name:     "No ID field",
			body:     `{"Name":"Test"}`,
			expected: "",
		},
		{
			name:     "Invalid JSON",
			body:     `invalid`,
			expected: "",
		},
		{
			name:     "Empty JSON",
			body:     `{}`,
			expected: "",
		},
		{
			name:     "Numeric ID as number",
			body:     `{"ID":42}`,
			expected: "42",
		},
		{
			name:     "Float ID",
			body:     `{"ID":3.14}`,
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIDFromResponse([]byte(tt.body))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_BatchProcessor_ContentID_Integration tests end-to-end Content-ID resolution
func Test_BatchProcessor_ContentID_Integration(t *testing.T) {
	// This test simulates a complete batch with Content-ID references

	processor := &BatchProcessor{
		server: &Server{},
	}

	// Simulate content ID map from previous operations
	contentIDMap := map[string]interface{}{
		"product1": &BatchOperationResponse{
			StatusCode: 201,
			Body:       []byte(`{"ID":100,"Name":"New Product"}`),
			ContentID:  "product1",
		},
		"category1": &BatchOperationResponse{
			StatusCode: 201,
			Body:       []byte(`{"ID":50,"Name":"Electronics"}`),
			ContentID:  "category1",
		},
	}

	t.Run("Reference to product1", func(t *testing.T) {
		url := "/Products($product1)/Reviews"
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products(100)/Reviews", resolved)
	})

	t.Run("Reference to category1", func(t *testing.T) {
		url := "/Categories(${category1})/Products"
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Categories(50)/Products", resolved)
	})

	t.Run("Multiple references in one URL", func(t *testing.T) {
		url := "/Categories($category1)/Products($product1)/Details"
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Categories(50)/Products(100)/Details", resolved)
	})

	t.Run("No $ in URL", func(t *testing.T) {
		url := "/Categories(10)/Products"
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Categories(10)/Products", resolved)
	})
}

// Test_ContentID_EdgeCases tests edge cases and error scenarios
func Test_ContentID_EdgeCases(t *testing.T) {
	processor := &BatchProcessor{
		server: &Server{},
	}

	t.Run("Empty URL", func(t *testing.T) {
		url := ""
		contentIDMap := make(map[string]interface{})
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "", resolved)
	})

	t.Run("Empty content ID map", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := make(map[string]interface{})
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products($1)", resolved)
	})

	t.Run("Nil response in map", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := map[string]interface{}{
			"1": nil,
		}
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products($1)", resolved)
	})

	t.Run("Wrong type in map", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := map[string]interface{}{
			"1": "not a batch response",
		}
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products($1)", resolved)
	})

	t.Run("Response with no body", func(t *testing.T) {
		url := "/Products($1)"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				StatusCode: 204,
				Body:       []byte{},
			},
		}
		resolved := processor.resolveContentID(url, contentIDMap)
		assert.Equal(t, "/Products($1)", resolved)
	})

	t.Run("$ at end of URL", func(t *testing.T) {
		url := "/Products$"
		contentIDMap := map[string]interface{}{
			"1": &BatchOperationResponse{
				Body: []byte(`{"ID":123}`),
			},
		}
		// Should not cause issues even though $ is at the end
		resolved := processor.resolveContentID(url, contentIDMap)
		require.NotPanics(t, func() {
			processor.resolveContentID(url, contentIDMap)
		})
		assert.Equal(t, "/Products$", resolved)
	})
}
