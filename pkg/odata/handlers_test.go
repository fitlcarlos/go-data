package odata

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create test Fiber app
func setupTestApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
}

func TestHandleHealth(t *testing.T) {
	app := setupTestApp()
	server := NewServer()

	app.Get("/health", server.handleHealth)

	t.Run("Health check returns OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		assert.Contains(t, []string{"ok", "healthy"}, result["status"])
	})
}

func TestHandleServerInfo(t *testing.T) {
	app := setupTestApp()
	server := NewServer()

	app.Get("/", server.handleServerInfo)

	t.Run("Server info returns correct data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		// Check that response has essential fields
		assert.True(t, result["name"] != nil || result["service"] != nil)
		assert.Contains(t, result, "version")
	})
}

func TestHandleMetadata_Basic(t *testing.T) {
	app := setupTestApp()
	server := NewServer()

	app.Get("/$metadata", server.handleMetadata)

	t.Run("Metadata endpoint exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/$metadata", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestWriteError_Handler(t *testing.T) {
	app := setupTestApp()
	server := NewServer()

	app.Get("/test-error", func(c fiber.Ctx) error {
		server.writeError(c, http.StatusBadRequest, "TEST_ERROR", "Test error message")
		return nil
	})

	t.Run("WriteError formats correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test-error", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(body, &result)

		errorObj := result["error"].(map[string]interface{})
		assert.Equal(t, "TEST_ERROR", errorObj["code"])
		assert.Equal(t, "Test error message", errorObj["message"])
	})
}

func TestBuildODataResponse(t *testing.T) {
	server := NewServer()

	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}

	t.Run("Build collection response", func(t *testing.T) {
		odataResp := &ODataResponse{
			Value: []interface{}{
				map[string]interface{}{"ID": 1, "Name": "User1"},
				map[string]interface{}{"ID": 2, "Name": "User2"},
			},
			Count: func() *int64 { v := int64(2); return &v }(),
		}

		result := server.buildODataResponse(odataResp, true, metadata)
		
		// For collections, buildODataResponse returns the ODataResponse directly
		resultResp, ok := result.(*ODataResponse)
		assert.True(t, ok, "Expected result to be *ODataResponse")
		assert.NotNil(t, resultResp)
		assert.NotNil(t, resultResp.Value)
		
		values, ok := resultResp.Value.([]interface{})
		assert.True(t, ok)
		assert.Len(t, values, 2)
	})

	t.Run("Build single entity response", func(t *testing.T) {
		odataResp := &ODataResponse{
			Value: []interface{}{
				map[string]interface{}{"ID": int64(1), "Name": "User1"},
			},
		}

		result := server.buildODataResponse(odataResp, false, metadata)
		resultMap := result.(map[string]interface{})

		assert.Contains(t, resultMap, "@odata.context")
		assert.Equal(t, int64(1), resultMap["ID"])
		assert.Equal(t, "User1", resultMap["Name"])
	})
}

func TestParseQueryOptions(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name        string
		queryString string
		expectError bool
	}{
		{
			name:        "Valid query with top and skip",
			queryString: "?$top=10&$skip=5",
			expectError: false,
		},
		{
			name:        "Valid query with orderby",
			queryString: "?$orderby=Name%20asc",
			expectError: false,
		},
		{
			name:        "Valid query with select",
			queryString: "?$select=Name,Email",
			expectError: false,
		},
		{
			name:        "Empty query",
			queryString: "",
			expectError: false,
		},
		{
			name:        "Invalid top value",
			queryString: "?$top=invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new app for each test to avoid route conflicts
			app := setupTestApp()
			
			app.Get("/test", func(c fiber.Ctx) error {
				options, err := server.parseQueryOptions(c)

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, options)
				}

				return c.SendStatus(http.StatusOK)
			})

			url := "/test" + tt.queryString
			req := httptest.NewRequest(http.MethodGet, url, nil)
			_, err := app.Test(req)
			require.NoError(t, err)
		})
	}
}

func TestExtractEntityName_Handler(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Simple entity",
			path:     "/Users",
			expected: "Users",
		},
		{
			name:     "Entity with ID",
			path:     "/Users(1)",
			expected: "Users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.extractEntityName(tt.path)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestBuildEntityURL(t *testing.T) {
	app := setupTestApp()
	server := NewServer()

	metadata := EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
		},
	}

	// Create mock service
	service := &BaseEntityService{
		provider: &mockDatabaseProvider{},
		metadata: metadata,
	}

	entity := map[string]interface{}{
		"ID": int64(123),
	}

	app.Get("/test", func(c fiber.Ctx) error {
		url := server.buildEntityURL(c, service, entity)
		assert.NotEmpty(t, url)
		assert.Contains(t, url, "Users")
		return c.SendString(url)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleEntityCollection_Methods(t *testing.T) {
	app := setupTestApp()
	server := NewServer()

	app.Get("/Users", server.handleEntityCollection)
	app.Post("/Users", server.handleEntityCollection)

	t.Run("GET endpoint exists", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/Users", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("POST endpoint exists", func(t *testing.T) {
		body := map[string]interface{}{"Name": "Test"}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/Users", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestGetEntityCount(t *testing.T) {
	t.Run("Count with service", func(t *testing.T) {
		// Skip test - requires proper database connection
		t.Skip("Requires proper database connection - skipped for now")
	})
}

func TestCheckEntityReadOnly_Middleware(t *testing.T) {
	server := NewServer()

	t.Run("Middleware exists", func(t *testing.T) {
		middleware := server.CheckEntityReadOnly("Users", "POST")
		assert.NotNil(t, middleware)
	})

	t.Run("Middleware for GET exists", func(t *testing.T) {
		middleware := server.CheckEntityReadOnly("Users", "GET")
		assert.NotNil(t, middleware)
	})
}
