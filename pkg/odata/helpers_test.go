package odata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractEntityName(t *testing.T) {
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
		{
			name:     "Entity with prefix",
			path:     "/odata/Users",
			expected: "Users",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "Root path",
			path:     "/",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer()
			result := server.extractEntityName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseKeyValue(t *testing.T) {
	server := NewServer()

	tests := []struct {
		name     string
		value    string
		dataType string
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "Integer",
			value:    "42",
			dataType: "int",
			want:     int64(42),
			wantErr:  false,
		},
		{
			name:     "String",
			value:    "hello",
			dataType: "string",
			want:     "hello",
			wantErr:  false,
		},
		{
			name:     "Boolean true",
			value:    "true",
			dataType: "bool",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "Boolean false",
			value:    "false",
			dataType: "bool",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "Int64",
			value:    "9223372036854775807",
			dataType: "int64",
			want:     int64(9223372036854775807),
			wantErr:  false,
		},
		{
			name:     "Invalid integer",
			value:    "not-a-number",
			dataType: "int",
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := server.parseKeyValue(tt.value, tt.dataType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMapODataType(t *testing.T) {
	server := NewServer()

	tests := []struct {
		internalType string
		expected     string
	}{
		{"string", "Edm.String"},
		{"int", "Edm.Int32"},
		{"int32", "Edm.Int32"},
		{"int64", "Edm.Int64"},
		{"bool", "Edm.Boolean"},
		{"float32", "Edm.Single"},
		{"float64", "Edm.Double"},
		{"time.Time", "Edm.DateTimeOffset"},
		{"[]byte", "Edm.Binary"},
		{"unknown", "Edm.String"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.internalType, func(t *testing.T) {
			result := server.mapODataType(tt.internalType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEntityKeys(t *testing.T) {
	server := NewServer()

	metadata := EntityMetadata{
		Name: "TestEntity",
		Properties: []PropertyMetadata{
			{Name: "ID", IsKey: true},
			{Name: "Name", IsKey: false},
			{Name: "SecondaryID", IsKey: true},
		},
	}

	keys := server.getEntityKeys(metadata)

	assert.Len(t, keys, 2)
	assert.Contains(t, keys, "ID")
	assert.Contains(t, keys, "SecondaryID")
	assert.NotContains(t, keys, "Name")
}

func TestBuildEntitySets(t *testing.T) {
	server := NewServer()

	// Registrar algumas entidades
	type TestEntity struct {
		ID   int    `odata:"id,key"`
		Name string `odata:"name"`
	}

	server.RegisterEntity("Users", TestEntity{})
	server.RegisterEntity("Products", TestEntity{})

	entitySets := server.buildEntitySets()

	assert.Len(t, entitySets, 2)

	// Verificar que contém as entidades
	names := make([]string, len(entitySets))
	for i, es := range entitySets {
		names[i] = es["name"].(string)
	}

	assert.Contains(t, names, "Users")
	assert.Contains(t, names, "Products")
}

func TestWriteError(t *testing.T) {
	// Este teste seria melhor com um mock do fiber.Ctx
	// Por enquanto, apenas verificamos que a função existe
	server := NewServer()
	assert.NotNil(t, server.writeError)
}
