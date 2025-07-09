package odata

import (
	"reflect"
	"testing"
	"time"

	"github.com/godata/odata/pkg/nullable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Structs de teste
type TestUser struct {
	TableName string           `table:"test_user;schema=dbo"`
	ID        int64            `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_user_id"`
	Nome      string           `json:"nome" column:"nome" prop:"[required]; length:100"`
	Email     string           `json:"email" column:"email" prop:"[required, Unique]; length:255"`
	Idade     nullable.Int64   `json:"idade" column:"idade"`
	Ativo     bool             `json:"ativo" column:"ativo" prop:"[required]; default"`
	DtInc     time.Time        `json:"dt_inc" column:"dt_inc" prop:"[required, NoUpdate]; default"`
	DtAlt     nullable.Time    `json:"dt_alt" column:"dt_alt"`
	Salario   nullable.Float64 `json:"salario" column:"salario" prop:"precision:10; scale:2"`
}

type TestProduct struct {
	TableName string           `table:"test_product;schema=dbo"`
	ID        int64            `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:identity"`
	Nome      string           `json:"nome" column:"nome" prop:"[required]; length:100"`
	Descricao nullable.String  `json:"descricao" column:"descricao" prop:"length:500"`
	Preco     nullable.Float64 `json:"preco" column:"preco" prop:"precision:8; scale:2"`
	Ativo     bool             `json:"ativo" column:"ativo" prop:"[required]; default"`
}

type TestOrder struct {
	TableName string    `table:"test_order;schema=dbo"`
	ID        int64     `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:none"`
	UserID    int64     `json:"user_id" column:"user_id" prop:"[required]"`
	DtPedido  time.Time `json:"dt_pedido" column:"dt_pedido" prop:"[required]"`

	// Relacionamentos
	User  *TestUser       `json:"User" association:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Refresh]"`
	Items []TestOrderItem `json:"Items" manyAssociation:"foreignKey:order_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
}

type TestOrderItem struct {
	TableName string `table:"test_order_item;schema=dbo"`
	ID        int64  `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence"`
	OrderID   int64  `json:"order_id" column:"order_id" prop:"[required, NoUpdate]"`
	ProductID int64  `json:"product_id" column:"product_id" prop:"[required]"`
	Quantity  int32  `json:"quantity" column:"quantity" prop:"[required]"`

	// Relacionamentos
	Order   *TestOrder   `json:"Order" association:"foreignKey:order_id; references:id" cascade:"[SaveUpdate, Refresh]"`
	Product *TestProduct `json:"Product" association:"foreignKey:product_id; references:id" cascade:"[SaveUpdate, Refresh]"`
}

func TestEntityMapper_MapEntity(t *testing.T) {
	mapper := NewEntityMapper()

	t.Run("MapUser", func(t *testing.T) {
		metadata, err := mapper.MapEntity(TestUser{})
		require.NoError(t, err)

		assert.Equal(t, "TestUser", metadata.Name)
		assert.Equal(t, "test_user", metadata.TableName)
		assert.Equal(t, "dbo", metadata.Schema)
		assert.Equal(t, []string{"id"}, metadata.Keys)
		assert.Len(t, metadata.Properties, 8)

		// Verifica propriedade ID
		idProp := findProperty(metadata.Properties, "ID")
		require.NotNil(t, idProp)
		assert.Equal(t, "int64", idProp.Type)
		assert.Equal(t, "id", idProp.ColumnName)
		assert.True(t, idProp.IsKey)
		assert.False(t, idProp.IsNullable)
		assert.Equal(t, "sequence", idProp.IDGenerator)
		assert.Equal(t, "seq_user_id", idProp.SequenceName)

		// Verifica propriedade Nome
		nomeProp := findProperty(metadata.Properties, "Nome")
		require.NotNil(t, nomeProp)
		assert.Equal(t, "string", nomeProp.Type)
		assert.Equal(t, "nome", nomeProp.ColumnName)
		assert.False(t, nomeProp.IsKey)
		assert.False(t, nomeProp.IsNullable)
		assert.Contains(t, nomeProp.PropFlags, "required")

		// Verifica propriedade Email (com Unique)
		emailProp := findProperty(metadata.Properties, "Email")
		require.NotNil(t, emailProp)
		assert.Equal(t, "string", emailProp.Type)
		assert.Equal(t, "email", emailProp.ColumnName)
		assert.False(t, emailProp.IsNullable)
		assert.Contains(t, emailProp.PropFlags, "required")
		assert.Contains(t, emailProp.PropFlags, "Unique")

		// Verifica propriedade nullable
		idadeProp := findProperty(metadata.Properties, "Idade")
		require.NotNil(t, idadeProp)
		assert.Equal(t, "int64", idadeProp.Type)
		assert.Equal(t, "idade", idadeProp.ColumnName)
		assert.True(t, idadeProp.IsNullable)

		// Verifica propriedade com default
		ativoProp := findProperty(metadata.Properties, "Ativo")
		require.NotNil(t, ativoProp)
		assert.Equal(t, "bool", ativoProp.Type)
		assert.True(t, ativoProp.HasDefault)
		assert.False(t, ativoProp.IsNullable)
		assert.Contains(t, ativoProp.PropFlags, "required")

		// Verifica propriedade com NoUpdate
		dtIncProp := findProperty(metadata.Properties, "DtInc")
		require.NotNil(t, dtIncProp)
		assert.Equal(t, "time.Time", dtIncProp.Type)
		assert.Contains(t, dtIncProp.PropFlags, "required")
		assert.Contains(t, dtIncProp.PropFlags, "NoUpdate")

		// Verifica propriedade com precisão
		salarioProp := findProperty(metadata.Properties, "Salario")
		require.NotNil(t, salarioProp)
		assert.Equal(t, "float64", salarioProp.Type)
		assert.Equal(t, 10, salarioProp.Precision)
		assert.Equal(t, 2, salarioProp.Scale)
		assert.True(t, salarioProp.IsNullable)
	})

	t.Run("MapProduct", func(t *testing.T) {
		metadata, err := mapper.MapEntity(TestProduct{})
		require.NoError(t, err)

		assert.Equal(t, "TestProduct", metadata.Name)
		assert.Equal(t, "test_product", metadata.TableName)
		assert.Equal(t, "dbo", metadata.Schema)
		assert.Equal(t, []string{"id"}, metadata.Keys)

		// Verifica gerador identity
		idProp := findProperty(metadata.Properties, "ID")
		require.NotNil(t, idProp)
		assert.Equal(t, "identity", idProp.IDGenerator)
		assert.Equal(t, "", idProp.SequenceName)

		// Verifica propriedade nullable string
		descProp := findProperty(metadata.Properties, "Descricao")
		require.NotNil(t, descProp)
		assert.Equal(t, "string", descProp.Type)
		assert.Equal(t, 500, descProp.MaxLength)
		assert.True(t, descProp.IsNullable)
	})

	t.Run("MapOrderWithRelationships", func(t *testing.T) {
		metadata, err := mapper.MapEntity(TestOrder{})
		require.NoError(t, err)

		assert.Equal(t, "TestOrder", metadata.Name)
		assert.Equal(t, "test_order", metadata.TableName)
		assert.Equal(t, "dbo", metadata.Schema)
		assert.Equal(t, []string{"id"}, metadata.Keys)

		// Verifica relacionamento singular (association)
		userProp := findProperty(metadata.Properties, "User")
		require.NotNil(t, userProp)
		assert.Equal(t, "relationship", userProp.Type)
		assert.True(t, userProp.IsNavigation)
		assert.False(t, userProp.IsCollection)
		assert.Equal(t, "TestUser", userProp.RelatedType)
		require.NotNil(t, userProp.Association)
		assert.Equal(t, "user_id", userProp.Association.ForeignKey)
		assert.Equal(t, "id", userProp.Association.References)
		assert.Contains(t, userProp.CascadeFlags, "SaveUpdate")
		assert.Contains(t, userProp.CascadeFlags, "Refresh")

		// Verifica relacionamento de coleção (manyAssociation)
		itemsProp := findProperty(metadata.Properties, "Items")
		require.NotNil(t, itemsProp)
		assert.Equal(t, "relationship", itemsProp.Type)
		assert.True(t, itemsProp.IsNavigation)
		assert.True(t, itemsProp.IsCollection)
		assert.Equal(t, "TestOrderItem", itemsProp.RelatedType)
		require.NotNil(t, itemsProp.ManyAssociation)
		assert.Equal(t, "order_id", itemsProp.ManyAssociation.ForeignKey)
		assert.Equal(t, "id", itemsProp.ManyAssociation.References)
		assert.Contains(t, itemsProp.CascadeFlags, "SaveUpdate")
		assert.Contains(t, itemsProp.CascadeFlags, "Remove")
		assert.Contains(t, itemsProp.CascadeFlags, "Refresh")
		assert.Contains(t, itemsProp.CascadeFlags, "RemoveOrphan")
	})
}

func TestEntityMapper_MapGoType(t *testing.T) {
	mapper := NewEntityMapper()

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "", "string"},
		{"int", int(0), "int32"},
		{"int32", int32(0), "int32"},
		{"int64", int64(0), "int64"},
		{"float32", float32(0), "float32"},
		{"float64", float64(0), "float64"},
		{"bool", false, "bool"},
		{"time.Time", time.Time{}, "time.Time"},
		{"[]byte", []byte{}, "[]byte"},
		{"nullable.Int64", nullable.Int64{}, "int64"},
		{"nullable.String", nullable.String{}, "string"},
		{"nullable.Bool", nullable.Bool{}, "bool"},
		{"nullable.Time", nullable.Time{}, "time.Time"},
		{"nullable.Float64", nullable.Float64{}, "float64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.mapGoType(reflect.TypeOf(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntityMapper_ParseTags(t *testing.T) {
	mapper := NewEntityMapper()

	t.Run("ParsePropTag", func(t *testing.T) {
		propFlags, err := mapper.parseProp("[required, NoInsert, NoUpdate, Unique]")
		require.NoError(t, err)

		assert.Contains(t, propFlags, "required")
		assert.Contains(t, propFlags, "NoInsert")
		assert.Contains(t, propFlags, "NoUpdate")
		assert.Contains(t, propFlags, "Unique")
		assert.Len(t, propFlags, 4)
	})

	t.Run("ParsePrimaryKey", func(t *testing.T) {
		prop := &PropertyMetadata{}

		err := mapper.parsePrimaryKey("idGenerator:sequence;name=seq_test_id", prop)
		require.NoError(t, err)

		assert.Equal(t, "sequence", prop.IDGenerator)
		assert.Equal(t, "seq_test_id", prop.SequenceName)
	})

	t.Run("ParseAssociation", func(t *testing.T) {
		association, err := mapper.parseAssociation("foreignKey:user_id; references:id")
		require.NoError(t, err)

		assert.Equal(t, "user_id", association.ForeignKey)
		assert.Equal(t, "id", association.References)
	})

	t.Run("ParseManyAssociation", func(t *testing.T) {
		manyAssoc, err := mapper.parseManyAssociation("foreignKey:order_id; references:id")
		require.NoError(t, err)

		assert.Equal(t, "order_id", manyAssoc.ForeignKey)
		assert.Equal(t, "id", manyAssoc.References)
	})

	t.Run("ParseCascade", func(t *testing.T) {
		cascadeFlags, err := mapper.parseCascade("[SaveUpdate, Remove, Refresh, RemoveOrphan]")
		require.NoError(t, err)

		assert.Contains(t, cascadeFlags, "SaveUpdate")
		assert.Contains(t, cascadeFlags, "Remove")
		assert.Contains(t, cascadeFlags, "Refresh")
		assert.Contains(t, cascadeFlags, "RemoveOrphan")
		assert.Len(t, cascadeFlags, 4)
	})
}

func TestEntityMapper_IsRelationship(t *testing.T) {
	mapper := NewEntityMapper()

	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"string", "", false},
		{"int", int(0), false},
		{"time.Time", time.Time{}, false},
		{"nullable.Int64", nullable.Int64{}, false},
		{"TestUser", TestUser{}, true},
		{"[]TestUser", []TestUser{}, true},
		{"TestProduct", TestProduct{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapper.isRelationship(reflect.TypeOf(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapEntityFromStruct(t *testing.T) {
	metadata, err := MapEntityFromStruct(TestUser{})
	require.NoError(t, err)

	assert.Equal(t, "TestUser", metadata.Name)
	assert.Equal(t, "test_user", metadata.TableName)
	assert.Equal(t, "dbo", metadata.Schema)
	assert.Equal(t, []string{"id"}, metadata.Keys)
	assert.Len(t, metadata.Properties, 8)
}

func TestMapEntityFromStruct_InvalidInput(t *testing.T) {
	_, err := MapEntityFromStruct("not a struct")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a struct")
}

// Helper function para encontrar propriedade por nome
func findProperty(properties []PropertyMetadata, name string) *PropertyMetadata {
	for i := range properties {
		if properties[i].Name == name {
			return &properties[i]
		}
	}
	return nil
}
