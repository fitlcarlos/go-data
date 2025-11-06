package odata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // SQLite driver
)

// Test_BatchProcessor_executeChangeset_WithTransaction tests that changesets use real transactions
func Test_BatchProcessor_executeChangeset_WithTransaction(t *testing.T) {
	// Create real SQLite in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create table
	_, err = db.Exec("CREATE TABLE test_entity (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)")
	require.NoError(t, err)

	// Setup mock provider with transaction support
	mockProvider := &BatchMockDatabaseProvider{
		beginTxFunc: func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
			return db.BeginTx(ctx, opts)
		},
	}

	server := &Server{
		provider:   mockProvider,
		entities:   make(map[string]EntityService),
		logger:     log.New(os.Stdout, "[TEST] ", log.LstdFlags),
		config:     DefaultServerConfig(),
		mu:         sync.RWMutex{},
		entityAuth: make(map[string]EntityAuthConfig),
	}

	// Register a test entity
	metadata := EntityMetadata{
		Name:      "TestEntity",
		TableName: "test_entity",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", IsKey: true, Type: "int"},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}
	service := NewBaseEntityService(mockProvider, metadata, server)
	server.entities["TestEntity"] = service

	processor := NewBatchProcessor(server)

	t.Run("Success - All operations commit", func(t *testing.T) {
		operations := []*BatchHTTPOperation{
			{
				Method:    "POST",
				URL:       "/odata/TestEntity",
				Body:      []byte(`{"Name":"Test1"}`),
				ContentID: "1",
			},
			{
				Method:    "POST",
				URL:       "/odata/TestEntity",
				Body:      []byte(`{"Name":"Test2"}`),
				ContentID: "2",
			},
		}

		contentIDMap := make(map[string]interface{})
		responses, err := processor.executeChangeset(context.Background(), operations, contentIDMap)

		assert.NoError(t, err)
		assert.Len(t, responses, 2)
		assert.Equal(t, http.StatusCreated, responses[0].StatusCode)
		assert.Equal(t, http.StatusCreated, responses[1].StatusCode)
	})

	t.Run("Failure - Rollback on error", func(t *testing.T) {
		// Create operations where second one will fail
		operations := []*BatchHTTPOperation{
			{
				Method:    "POST",
				URL:       "/odata/TestEntity",
				Body:      []byte(`{"Name":"Test1"}`),
				ContentID: "1",
			},
			{
				Method:    "POST",
				URL:       "/odata/NonExistent", // This will fail
				Body:      []byte(`{"Name":"Test2"}`),
				ContentID: "2",
			},
		}

		contentIDMap := make(map[string]interface{})
		_, err := processor.executeChangeset(context.Background(), operations, contentIDMap)

		// Should fail
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rolled back")
	})
}

// Test_BatchProcessor_executeOperationInTx tests executing operations within a transaction
func Test_BatchProcessor_executeOperationInTx(t *testing.T) {
	mockProvider := &BatchMockDatabaseProvider{}
	server := &Server{
		provider:   mockProvider,
		entities:   make(map[string]EntityService),
		logger:     log.New(os.Stdout, "[TEST] ", log.LstdFlags),
		config:     DefaultServerConfig(),
		mu:         sync.RWMutex{},
		entityAuth: make(map[string]EntityAuthConfig),
	}

	// Register test entity
	metadata := EntityMetadata{
		Name:      "TestEntity",
		TableName: "test_entity",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", IsKey: true, Type: "int"},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}
	service := NewBaseEntityService(mockProvider, metadata, server)
	server.entities["TestEntity"] = service

	processor := NewBatchProcessor(server)

	// Create a mock transaction
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	t.Run("POST operation", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "POST",
			URL:       "/odata/TestEntity",
			Body:      []byte(`{"Name":"Test"}`),
			ContentID: "1",
		}

		contentIDMap := make(map[string]interface{})
		resp, err := processor.executeOperationInTx(context.Background(), tx, op, contentIDMap)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "1", resp.ContentID)
	})

	t.Run("PUT operation", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "PUT",
			URL:       "/odata/TestEntity(1)",
			Body:      []byte(`{"Name":"UpdatedTest"}`),
			ContentID: "2",
		}

		contentIDMap := make(map[string]interface{})
		resp, err := processor.executeOperationInTx(context.Background(), tx, op, contentIDMap)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "2", resp.ContentID)
	})

	t.Run("DELETE operation", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "DELETE",
			URL:       "/odata/TestEntity(1)",
			ContentID: "3",
		}

		contentIDMap := make(map[string]interface{})
		resp, err := processor.executeOperationInTx(context.Background(), tx, op, contentIDMap)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "3", resp.ContentID)
	})

	t.Run("Entity not found", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "POST",
			URL:       "/odata/NonExistent",
			Body:      []byte(`{"Name":"Test"}`),
			ContentID: "4",
		}

		contentIDMap := make(map[string]interface{})
		resp, err := processor.executeOperationInTx(context.Background(), tx, op, contentIDMap)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Unsupported method", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "HEAD",
			URL:       "/odata/TestEntity",
			ContentID: "5",
		}

		contentIDMap := make(map[string]interface{})
		resp, err := processor.executeOperationInTx(context.Background(), tx, op, contentIDMap)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

// Test_BatchProcessor_parseOperationURL tests URL parsing for operations
func Test_BatchProcessor_parseOperationURL(t *testing.T) {
	processor := &BatchProcessor{}

	tests := []struct {
		name           string
		url            string
		expectedEntity string
		expectedID     string
	}{
		{
			name:           "Simple entity",
			url:            "/odata/Products",
			expectedEntity: "Products",
			expectedID:     "",
		},
		{
			name:           "Entity with numeric ID",
			url:            "/odata/Products(123)",
			expectedEntity: "Products",
			expectedID:     "123",
		},
		{
			name:           "Entity with string ID",
			url:            "/odata/Products('abc')",
			expectedEntity: "Products",
			expectedID:     "abc",
		},
		{
			name:           "Entity with double quotes",
			url:            `/odata/Products("abc")`,
			expectedEntity: "Products",
			expectedID:     "abc",
		},
		{
			name:           "Without /odata/ prefix",
			url:            "Products(123)",
			expectedEntity: "Products",
			expectedID:     "123",
		},
		{
			name:           "With /api/v1/ prefix",
			url:            "/api/v1/Products(123)",
			expectedEntity: "Products",
			expectedID:     "123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entity, id, err := processor.parseOperationURL(tt.url)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEntity, entity)
			assert.Equal(t, tt.expectedID, id)
		})
	}
}

// Test_BatchProcessor_CRUD_Operations tests individual CRUD operations
func Test_BatchProcessor_CRUD_Operations(t *testing.T) {
	mockProvider := &BatchMockDatabaseProvider{}
	server := &Server{
		provider:   mockProvider,
		entities:   make(map[string]EntityService),
		logger:     log.New(os.Stdout, "[TEST] ", log.LstdFlags),
		config:     DefaultServerConfig(),
		mu:         sync.RWMutex{},
		entityAuth: make(map[string]EntityAuthConfig),
	}

	// Register test entity
	metadata := EntityMetadata{
		Name:      "TestEntity",
		TableName: "test_entity",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", IsKey: true, Type: "int"},
			{Name: "Name", ColumnName: "name", Type: "string"},
		},
	}
	service := NewBaseEntityService(mockProvider, metadata, server)
	server.entities["TestEntity"] = service

	processor := NewBatchProcessor(server)

	// Create a mock transaction
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer tx.Rollback()

	t.Run("CREATE operation", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "POST",
			URL:       "/odata/TestEntity",
			Body:      []byte(`{"Name":"New Entity"}`),
			ContentID: "create-1",
		}

		resp, err := processor.executeCreate(context.Background(), tx, service, op)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Contains(t, resp.Headers, "Content-Type")
		assert.Contains(t, resp.Headers, "Location")
		assert.Equal(t, "create-1", resp.ContentID)
	})

	t.Run("UPDATE operation", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "PUT",
			URL:       "/odata/TestEntity(1)",
			Body:      []byte(`{"Name":"Updated Entity"}`),
			ContentID: "update-1",
		}

		resp, err := processor.executeUpdate(context.Background(), tx, service, "1", op)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Headers, "Content-Type")
		assert.Equal(t, "update-1", resp.ContentID)
	})

	t.Run("DELETE operation", func(t *testing.T) {
		op := &BatchHTTPOperation{
			Method:    "DELETE",
			URL:       "/odata/TestEntity(1)",
			ContentID: "delete-1",
		}

		resp, err := processor.executeDelete(context.Background(), tx, service, "1", op)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Empty(t, resp.Body)
		assert.Equal(t, "delete-1", resp.ContentID)
	})
}

// BatchMockDatabaseProvider with transaction support
type BatchMockDatabaseProvider struct {
	beginTxFunc func(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func (m *BatchMockDatabaseProvider) Connect(connectionString string) error {
	return nil
}

func (m *BatchMockDatabaseProvider) Close() error {
	return nil
}

func (m *BatchMockDatabaseProvider) GetConnection() *sql.DB {
	return nil
}

func (m *BatchMockDatabaseProvider) GetDriverName() string {
	return "mock"
}

func (m *BatchMockDatabaseProvider) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if m.beginTxFunc != nil {
		return m.beginTxFunc(ctx, opts)
	}
	// Default: create in-memory SQLite transaction
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	return db.BeginTx(ctx, opts)
}

func (m *BatchMockDatabaseProvider) BuildSelectQuery(entity EntityMetadata, options QueryOptions) (string, []interface{}, error) {
	return "SELECT * FROM test", nil, nil
}

func (m *BatchMockDatabaseProvider) BuildInsertQuery(entity EntityMetadata, data map[string]interface{}) (string, []interface{}, error) {
	// Construir query INSERT real para SQLite
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	// Coletar campos e valores
	var fields []string
	var placeholders []string
	var values []interface{}

	for key, value := range data {
		// Encontrar a coluna correspondente
		columnName := key
		for _, prop := range entity.Properties {
			if prop.Name == key {
				if prop.ColumnName != "" {
					columnName = prop.ColumnName
				}
				break
			}
		}
		fields = append(fields, columnName)
		placeholders = append(placeholders, "?")
		values = append(values, value)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	return query, values, nil
}

func (m *BatchMockDatabaseProvider) BuildUpdateQuery(entity EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error) {
	// Construir query UPDATE real para SQLite
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	// Coletar campos a serem atualizados
	var sets []string
	var values []interface{}

	for key, value := range data {
		columnName := key
		for _, prop := range entity.Properties {
			if prop.Name == key {
				if prop.ColumnName != "" {
					columnName = prop.ColumnName
				}
				break
			}
		}
		sets = append(sets, fmt.Sprintf("%s = ?", columnName))
		values = append(values, value)
	}

	// Adicionar condição WHERE com as chaves
	var whereConditions []string
	for key, value := range keyValues {
		columnName := key
		for _, prop := range entity.Properties {
			if prop.Name == key {
				if prop.ColumnName != "" {
					columnName = prop.ColumnName
				}
				break
			}
		}
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", columnName))
		values = append(values, value)
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(sets, ", "),
		strings.Join(whereConditions, " AND "))

	return query, values, nil
}

func (m *BatchMockDatabaseProvider) BuildDeleteQuery(entity EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error) {
	// Construir query DELETE real para SQLite
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	// Adicionar condição WHERE com as chaves
	var whereConditions []string
	var values []interface{}

	for key, value := range keyValues {
		columnName := key
		for _, prop := range entity.Properties {
			if prop.Name == key {
				if prop.ColumnName != "" {
					columnName = prop.ColumnName
				}
				break
			}
		}
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", columnName))
		values = append(values, value)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereConditions, " AND "))

	return query, values, nil
}

func (m *BatchMockDatabaseProvider) BuildWhereClause(filter string, metadata EntityMetadata) (string, []interface{}, error) {
	return "id = ?", []interface{}{1}, nil
}

func (m *BatchMockDatabaseProvider) BuildOrderByClause(orderBy string, metadata EntityMetadata) (string, error) {
	return "name ASC", nil
}

func (m *BatchMockDatabaseProvider) MapGoTypeToSQL(goType string) string {
	return "TEXT"
}

func (m *BatchMockDatabaseProvider) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
