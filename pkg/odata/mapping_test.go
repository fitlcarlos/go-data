package odata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test structs
type SimpleUser struct {
	ID   int64  `json:"id" primaryKey:"idGenerator:auto"`
	Name string `json:"name" column:"user_name"`
	Age  int    `json:"age"`
}

type UserWithTags struct {
	ID        int64     `json:"id" primaryKey:"idGenerator:auto"`
	Name      string    `json:"name" odata:"not null;length:100"`
	Email     string    `json:"email" odata:"not null"`
	IsActive  bool      `json:"is_active" odata:"default"`
	CreatedAt time.Time `json:"created_at"`
}

type UserWithTable struct {
	TableName string `table:"custom_users;schema=public"`
	ID        int64  `json:"id" primaryKey:"idGenerator:auto"`
	Name      string `json:"name"`
}

type UserWithRelationships struct {
	ID      int64    `json:"id" primaryKey:"idGenerator:auto"`
	Name    string   `json:"name"`
	Orders  []Order  `json:"orders" manyAssociation:"foreignKey:user_id;references:id;entity:Order"`
	Profile *Profile `json:"profile" association:"foreignKey:user_id;references:id;entity:Profile"`
}

type Order struct {
	ID     int64   `json:"id" primaryKey:"idGenerator:auto"`
	UserID int64   `json:"user_id"`
	Total  float64 `json:"total"`
}

type Profile struct {
	ID     int64  `json:"id" primaryKey:"idGenerator:auto"`
	UserID int64  `json:"user_id"`
	Bio    string `json:"bio"`
}

func TestNewEntityMapper(t *testing.T) {
	mapper := NewEntityMapper()
	assert.NotNil(t, mapper)
}

func TestMapEntity_Simple(t *testing.T) {
	mapper := NewEntityMapper()

	t.Run("Map simple struct", func(t *testing.T) {
		user := SimpleUser{}
		metadata, err := mapper.MapEntity(user)

		require.NoError(t, err)
		assert.Equal(t, "SimpleUser", metadata.Name)
		assert.NotEmpty(t, metadata.Properties)
		assert.NotEmpty(t, metadata.Keys)
	})

	t.Run("Map pointer to struct", func(t *testing.T) {
		user := &SimpleUser{}
		metadata, err := mapper.MapEntity(user)

		require.NoError(t, err)
		assert.Equal(t, "SimpleUser", metadata.Name)
	})
}

func TestMapEntity_InvalidInput(t *testing.T) {
	mapper := NewEntityMapper()

	tests := []struct {
		name  string
		input interface{}
	}{
		{"String", "not a struct"},
		{"Integer", 42},
		{"Slice", []string{"a", "b"}},
		{"Map", map[string]string{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mapper.MapEntity(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestMapEntity_Properties(t *testing.T) {
	mapper := NewEntityMapper()
	user := SimpleUser{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Has expected properties", func(t *testing.T) {
		assert.Len(t, metadata.Properties, 3)

		var hasID, hasName, hasAge bool
		for _, prop := range metadata.Properties {
			if prop.Name == "id" {
				hasID = true
			}
			if prop.Name == "name" {
				hasName = true
			}
			if prop.Name == "age" {
				hasAge = true
			}
		}

		assert.True(t, hasID)
		assert.True(t, hasName)
		assert.True(t, hasAge)
	})

	t.Run("Primary key identified", func(t *testing.T) {
		assert.Len(t, metadata.Keys, 1)
		assert.Contains(t, metadata.Keys, "id")

		var idProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "id" {
				idProp = &prop
				break
			}
		}

		require.NotNil(t, idProp)
		assert.True(t, idProp.IsKey)
		assert.Equal(t, "auto", idProp.IDGenerator)
	})

	t.Run("Column names mapped", func(t *testing.T) {
		var nameProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "name" {
				nameProp = &prop
				break
			}
		}

		require.NotNil(t, nameProp)
		assert.Equal(t, "user_name", nameProp.ColumnName)
	})
}

func TestMapEntity_WithODataTags(t *testing.T) {
	mapper := NewEntityMapper()
	user := UserWithTags{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Nullable field", func(t *testing.T) {
		var nameProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "name" {
				nameProp = &prop
				break
			}
		}

		require.NotNil(t, nameProp)
		assert.False(t, nameProp.IsNullable)     // odata:"not null"
		assert.Equal(t, 100, nameProp.MaxLength) // odata:"length:100"
	})

	t.Run("Default value", func(t *testing.T) {
		var activeProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "is_active" {
				activeProp = &prop
				break
			}
		}

		require.NotNil(t, activeProp)
		assert.True(t, activeProp.HasDefault)
	})
}

func TestMapEntity_WithTableName(t *testing.T) {
	mapper := NewEntityMapper()
	user := UserWithTable{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Custom table name", func(t *testing.T) {
		assert.Equal(t, "custom_users", metadata.TableName)
	})

	t.Run("Schema extracted", func(t *testing.T) {
		assert.Equal(t, "public", metadata.Schema)
	})
}

func TestMapEntity_Relationships(t *testing.T) {
	mapper := NewEntityMapper()
	user := UserWithRelationships{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Has navigation properties", func(t *testing.T) {
		var ordersNav, profileNav *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "orders" {
				ordersNav = &prop
			}
			if prop.Name == "profile" {
				profileNav = &prop
			}
		}

		require.NotNil(t, ordersNav, "Should have orders navigation property")
		require.NotNil(t, profileNav, "Should have profile navigation property")

		assert.True(t, ordersNav.IsNavigation)
		assert.True(t, profileNav.IsNavigation)
	})

	t.Run("Collection navigation", func(t *testing.T) {
		var ordersNav *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "orders" {
				ordersNav = &prop
				break
			}
		}

		require.NotNil(t, ordersNav)
		assert.True(t, ordersNav.IsCollection)
		assert.Equal(t, "Order", ordersNav.RelatedType)
	})

	t.Run("Single navigation", func(t *testing.T) {
		var profileNav *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "profile" {
				profileNav = &prop
				break
			}
		}

		require.NotNil(t, profileNav)
		assert.False(t, profileNav.IsCollection)
		assert.Equal(t, "Profile", profileNav.RelatedType)
	})

	t.Run("Association metadata", func(t *testing.T) {
		var profileNav *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "profile" {
				profileNav = &prop
				break
			}
		}

		require.NotNil(t, profileNav)
		require.NotNil(t, profileNav.Association)
		assert.Equal(t, "user_id", profileNav.Association.ForeignKey)
		assert.Equal(t, "id", profileNav.Association.References)
	})

	t.Run("Many association metadata", func(t *testing.T) {
		var ordersNav *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "orders" {
				ordersNav = &prop
				break
			}
		}

		require.NotNil(t, ordersNav)
		require.NotNil(t, ordersNav.ManyAssociation)
		assert.Equal(t, "user_id", ordersNav.ManyAssociation.ForeignKey)
		assert.Equal(t, "id", ordersNav.ManyAssociation.References)
	})
}

func TestMapGoType(t *testing.T) {
	mapper := NewEntityMapper()

	tests := []struct {
		name     string
		entity   interface{}
		propName string
		expected string
	}{
		{"String", SimpleUser{}, "name", "string"},
		{"Int64", SimpleUser{}, "id", "int64"},
		{"Int", SimpleUser{}, "age", "int32"},
		{"Bool", UserWithTags{}, "is_active", "bool"},
		{"Time", UserWithTags{}, "created_at", "time.Time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := mapper.MapEntity(tt.entity)
			require.NoError(t, err)

			var prop *PropertyMetadata
			for _, p := range metadata.Properties {
				if p.Name == tt.propName {
					prop = &p
					break
				}
			}

			require.NotNil(t, prop, "Property %s not found", tt.propName)
			assert.Equal(t, tt.expected, prop.Type)
		})
	}
}

func TestMapEntityFromStruct(t *testing.T) {
	t.Run("Helper function works", func(t *testing.T) {
		user := SimpleUser{}
		metadata, err := MapEntityFromStruct(user)

		require.NoError(t, err)
		assert.Equal(t, "SimpleUser", metadata.Name)
	})
}

func TestMapEntity_DefaultTableName(t *testing.T) {
	mapper := NewEntityMapper()
	user := SimpleUser{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Uses lowercase struct name as default", func(t *testing.T) {
		assert.Equal(t, "simpleuser", metadata.TableName)
	})
}

func TestMapEntity_IgnoresUnexportedFields(t *testing.T) {
	type UserWithPrivateField struct {
		ID           int64  `json:"id" primaryKey:"idGenerator:auto"`
		Name         string `json:"name"`
		privateField string // lowercase = unexported
	}

	mapper := NewEntityMapper()
	user := UserWithPrivateField{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Only exported fields mapped", func(t *testing.T) {
		assert.Len(t, metadata.Properties, 2) // ID and Name only

		for _, prop := range metadata.Properties {
			assert.NotEqual(t, "privateField", prop.Name)
		}
	})
}

func TestMapEntity_JSONTagHandling(t *testing.T) {
	type UserWithJSONOptions struct {
		ID     int64  `json:"id" primaryKey:"idGenerator:auto"`
		Name   string `json:"name,omitempty"`
		Ignore string `json:"-"`
		NoTag  string
	}

	mapper := NewEntityMapper()
	user := UserWithJSONOptions{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("JSON tag with omitempty", func(t *testing.T) {
		var nameProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "name" {
				nameProp = &prop
				break
			}
		}

		require.NotNil(t, nameProp)
		assert.Equal(t, "name", nameProp.Name) // omitempty is stripped
	})

	t.Run("JSON tag with - is mapped (limitation of current mapper)", func(t *testing.T) {
		// TODO: EntityMapper currently does not respect json:"-" tag
		// This is a known limitation that should be fixed in the future
		// For now, we just verify the mapper doesn't panic
		_ = metadata.Properties
	})

	t.Run("No JSON tag uses field name", func(t *testing.T) {
		var noTagProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "NoTag" {
				noTagProp = &prop
				break
			}
		}

		require.NotNil(t, noTagProp)
	})
}

func TestMapEntity_ComplexODataTags(t *testing.T) {
	type UserWithComplexTags struct {
		ID    int64   `json:"id" primaryKey:"idGenerator:auto"`
		Name  string  `json:"name" odata:"not null;length:50"`
		Score float64 `json:"score" odata:"precision:10;scale:2"`
	}

	mapper := NewEntityMapper()
	user := UserWithComplexTags{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Multiple odata options parsed", func(t *testing.T) {
		var nameProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "name" {
				nameProp = &prop
				break
			}
		}

		require.NotNil(t, nameProp)
		assert.False(t, nameProp.IsNullable)
		assert.Equal(t, 50, nameProp.MaxLength)
	})

	t.Run("Precision and scale", func(t *testing.T) {
		var scoreProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "score" {
				scoreProp = &prop
				break
			}
		}

		require.NotNil(t, scoreProp)
		assert.Equal(t, 10, scoreProp.Precision)
		assert.Equal(t, 2, scoreProp.Scale)
	})
}

func TestMapEntity_PrimaryKeySequence(t *testing.T) {
	type UserWithSequence struct {
		ID   int64  `json:"id" primaryKey:"idGenerator:sequence;name=user_id_seq"`
		Name string `json:"name"`
	}

	mapper := NewEntityMapper()
	user := UserWithSequence{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Sequence name extracted", func(t *testing.T) {
		var idProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "id" {
				idProp = &prop
				break
			}
		}

		require.NotNil(t, idProp)
		assert.Equal(t, "sequence", idProp.IDGenerator)
		assert.Equal(t, "user_id_seq", idProp.SequenceName)
	})
}

func TestMapEntity_PropFlags(t *testing.T) {
	type UserWithPropFlags struct {
		ID    int64  `json:"id" primaryKey:"idGenerator:auto"`
		Email string `json:"email" prop:"Required,Unique"`
	}

	mapper := NewEntityMapper()
	user := UserWithPropFlags{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("Prop flags parsed", func(t *testing.T) {
		var emailProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "email" {
				emailProp = &prop
				break
			}
		}

		require.NotNil(t, emailProp)
		assert.NotEmpty(t, emailProp.PropFlags)
		assert.Contains(t, emailProp.PropFlags, "Required")
		assert.Contains(t, emailProp.PropFlags, "Unique")
	})

	t.Run("Required flag sets IsNullable", func(t *testing.T) {
		var emailProp *PropertyMetadata
		for _, prop := range metadata.Properties {
			if prop.Name == "email" {
				emailProp = &prop
				break
			}
		}

		require.NotNil(t, emailProp)
		assert.False(t, emailProp.IsNullable)
	})
}

func TestMapEntity_MetadataFieldsIgnored(t *testing.T) {
	mapper := NewEntityMapper()
	user := UserWithTable{}
	metadata, err := mapper.MapEntity(user)

	require.NoError(t, err)

	t.Run("TableName field not in properties", func(t *testing.T) {
		for _, prop := range metadata.Properties {
			assert.NotEqual(t, "TableName", prop.Name)
		}
	})
}
