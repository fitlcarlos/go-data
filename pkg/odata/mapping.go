package odata

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// EntityMapper é responsável por mapear structs para metadados OData
type EntityMapper struct{}

// NewEntityMapper cria uma nova instância do mapper
func NewEntityMapper() *EntityMapper {
	return &EntityMapper{}
}

// MapEntity mapeia uma struct para EntityMetadata usando tags
func (m *EntityMapper) MapEntity(entity interface{}) (EntityMetadata, error) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return EntityMetadata{}, fmt.Errorf("entity must be a struct, got %s", t.Kind())
	}

	metadata := EntityMetadata{
		Name:       t.Name(),
		Properties: []PropertyMetadata{},
		Keys:       []string{},
	}

	// Processa campos da struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Pula campos não exportados
		if !field.IsExported() {
			continue
		}

		// Pula campos de metadados que não são propriedades da entidade
		if m.isMetadataField(field) {
			continue
		}

		prop, err := m.mapField(field)
		if err != nil {
			return EntityMetadata{}, fmt.Errorf("error mapping field %s: %w", field.Name, err)
		}

		if prop != nil {
			metadata.Properties = append(metadata.Properties, *prop)

			// Adiciona às chaves se for primary key
			if prop.IsKey {
				metadata.Keys = append(metadata.Keys, prop.Name)
			}
		}
	}

	// Define o nome da tabela se especificado na tag
	if tableName := m.getTableName(t); tableName != "" {
		metadata.TableName = tableName
		// Processa o schema se presente
		if schema := m.getTableSchema(t); schema != "" {
			metadata.Schema = schema
		}
	} else {
		metadata.TableName = strings.ToLower(t.Name())
	}

	return metadata, nil
}

// mapField mapeia um campo da struct para PropertyMetadata
func (m *EntityMapper) mapField(field reflect.StructField) (*PropertyMetadata, error) {
	// Verifica se é um relacionamento (slice ou struct)
	if m.isRelationship(field.Type) {
		return m.mapRelationship(field)
	}

	// Determina o nome da propriedade usando a tag JSON se disponível
	propertyName := field.Name
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		// Remove opções adicionais da tag JSON (como ,omitempty)
		propertyName = strings.Split(jsonTag, ",")[0]
	}

	prop := &PropertyMetadata{
		Name: propertyName,
		Type: m.mapGoType(field.Type),
	}

	// Mapeia tags
	if err := m.mapTags(field, prop); err != nil {
		return nil, err
	}

	return prop, nil
}

// mapRelationship mapeia relacionamentos entre entidades
func (m *EntityMapper) mapRelationship(field reflect.StructField) (*PropertyMetadata, error) {
	// Determina o nome da propriedade usando a tag JSON se disponível
	propertyName := field.Name
	if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
		// Remove opções adicionais da tag JSON (como ,omitempty)
		propertyName = strings.Split(jsonTag, ",")[0]
	}

	prop := &PropertyMetadata{
		Name:         propertyName,
		Type:         "relationship",
		IsNavigation: true,
	}

	// Processa tag association (relacionamento simples)
	if association := field.Tag.Get("association"); association != "" {
		assoc, err := m.parseAssociation(association)
		if err != nil {
			return nil, err
		}
		prop.Association = assoc

		// Preenche também o campo Relationship para compatibilidade
		// Para association (N:1): A chave estrangeira está na entidade local
		prop.Relationship = &RelationshipMetadata{
			LocalProperty:      assoc.ForeignKey, // A propriedade local é a chave estrangeira
			ReferencedProperty: assoc.References, // A propriedade referenciada é a chave primária
		}
	}

	// Processa tag manyAssociation (relacionamento múltiplo)
	if manyAssociation := field.Tag.Get("manyAssociation"); manyAssociation != "" {
		manyAssoc, err := m.parseManyAssociation(manyAssociation)
		if err != nil {
			return nil, err
		}
		prop.ManyAssociation = manyAssoc

		// Preenche também o campo Relationship para compatibilidade
		// Para manyAssociation (1:N): A chave estrangeira está na entidade relacionada
		prop.Relationship = &RelationshipMetadata{
			LocalProperty:      manyAssoc.References, // A propriedade local é a chave primária
			ReferencedProperty: manyAssoc.ForeignKey, // A propriedade referenciada é a chave estrangeira
		}
	}

	// Processa tag foreignKey (compatibilidade com versão anterior)
	if foreignKey := field.Tag.Get("foreignKey"); foreignKey != "" {
		rel, err := m.parseForeignKey(foreignKey)
		if err != nil {
			return nil, err
		}
		prop.Relationship = rel
	}

	// Processa tag cascade
	if cascade := field.Tag.Get("cascade"); cascade != "" {
		cascadeFlags, err := m.parseCascade(cascade)
		if err != nil {
			return nil, err
		}
		prop.CascadeFlags = cascadeFlags
	}

	// Define se é coleção ou entidade única
	if field.Type.Kind() == reflect.Slice {
		prop.IsCollection = true
		elemType := field.Type.Elem()
		// Se o elemento é um ponteiro, pega o tipo apontado
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		prop.RelatedType = elemType.Name()
	} else {
		prop.IsCollection = false
		relatedType := field.Type
		// Se é um ponteiro, pega o tipo apontado
		if relatedType.Kind() == reflect.Ptr {
			relatedType = relatedType.Elem()
		}
		prop.RelatedType = relatedType.Name()
	}

	return prop, nil
}

// mapTags processa todas as tags do campo
func (m *EntityMapper) mapTags(field reflect.StructField, prop *PropertyMetadata) error {
	// Tag column
	if column := field.Tag.Get("column"); column != "" {
		prop.ColumnName = column
	}

	// Tag primaryKey
	if pk := field.Tag.Get("primaryKey"); pk != "" {
		prop.IsKey = true
		if err := m.parsePrimaryKey(pk, prop); err != nil {
			return err
		}
	}

	// Tag prop (nova tag para propriedades avançadas)
	if propTag := field.Tag.Get("prop"); propTag != "" {
		propFlags, err := m.parseProp(propTag)
		if err != nil {
			return err
		}
		prop.PropFlags = propFlags

		// Processa as flags para definir propriedades do metadata
		for _, flag := range propFlags {
			switch flag {
			case "Required":
				prop.IsNullable = false
			case "Unique":
				// Será processado pelos providers
			}
		}
	}

	// Tag odata
	if odata := field.Tag.Get("odata"); odata != "" {
		if err := m.parseODataTag(odata, prop); err != nil {
			return err
		}
	} else {
		// Se não há tag odata, define como nullable para tipos nullable
		if m.isNullableType(field.Type) {
			prop.IsNullable = true
		}
	}

	// Define column name padrão se não especificado
	if prop.ColumnName == "" {
		prop.ColumnName = strings.ToLower(field.Name)
	}

	return nil
}

// parseODataTag processa a tag odata
func (m *EntityMapper) parseODataTag(odata string, prop *PropertyMetadata) error {
	parts := strings.Split(odata, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		switch {
		case part == "not null":
			prop.IsNullable = false
		case part == "null":
			prop.IsNullable = true
		case part == "default":
			prop.HasDefault = true
		case strings.HasPrefix(part, "length:"):
			if length, err := strconv.Atoi(strings.TrimPrefix(part, "length:")); err == nil {
				prop.MaxLength = length
			}
		case strings.HasPrefix(part, "precision:"):
			if precision, err := strconv.Atoi(strings.TrimPrefix(part, "precision:")); err == nil {
				prop.Precision = precision
			}
		case strings.HasPrefix(part, "scale:"):
			if scale, err := strconv.Atoi(strings.TrimPrefix(part, "scale:")); err == nil {
				prop.Scale = scale
			}
		}
	}

	return nil
}

// parsePrimaryKey processa a tag primaryKey
func (m *EntityMapper) parsePrimaryKey(pk string, prop *PropertyMetadata) error {
	parts := strings.Split(pk, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.HasPrefix(part, "idGenerator:") {
			generator := strings.TrimPrefix(part, "idGenerator:")
			prop.IDGenerator = generator
		}

		if strings.HasPrefix(part, "name=") {
			seqName := strings.TrimPrefix(part, "name=")
			prop.SequenceName = seqName
		}
	}

	return nil
}

// parseForeignKey processa a tag foreignKey
func (m *EntityMapper) parseForeignKey(fk string) (*RelationshipMetadata, error) {
	rel := &RelationshipMetadata{}

	parts := strings.Split(fk, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, ":") {
			keyValue := strings.Split(part, ":")
			if len(keyValue) == 2 {
				key := strings.TrimSpace(keyValue[0])
				value := strings.TrimSpace(keyValue[1])

				switch key {
				case "references":
					rel.ReferencedProperty = value
				case "OnDelete":
					rel.OnDelete = value
				case "OnUpdate":
					rel.OnUpdate = value
				}
			}
		} else {
			// Assume que é a propriedade local
			rel.LocalProperty = part
		}
	}

	return rel, nil
}

// parseAssociation processa a tag association
func (m *EntityMapper) parseAssociation(association string) (*AssociationMetadata, error) {
	assoc := &AssociationMetadata{}

	parts := strings.Split(association, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, ":") {
			keyValue := strings.Split(part, ":")
			if len(keyValue) == 2 {
				key := strings.TrimSpace(keyValue[0])
				value := strings.TrimSpace(keyValue[1])

				switch key {
				case "foreignKey":
					assoc.ForeignKey = value
				case "references":
					assoc.References = value
				case "entity":
					assoc.RelatedEntity = value
				}
			}
		}
	}

	return assoc, nil
}

// parseManyAssociation processa a tag manyAssociation
func (m *EntityMapper) parseManyAssociation(manyAssociation string) (*ManyAssociationMetadata, error) {
	manyAssoc := &ManyAssociationMetadata{}

	parts := strings.Split(manyAssociation, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, ":") {
			keyValue := strings.Split(part, ":")
			if len(keyValue) == 2 {
				key := strings.TrimSpace(keyValue[0])
				value := strings.TrimSpace(keyValue[1])

				switch key {
				case "foreignKey":
					manyAssoc.ForeignKey = value
				case "references":
					manyAssoc.References = value
				case "entity":
					manyAssoc.RelatedEntity = value
				case "joinTable":
					manyAssoc.JoinTable = value
				case "joinColumn":
					manyAssoc.JoinColumn = value
				case "inverseJoinColumn":
					manyAssoc.InverseJoinColumn = value
				}
			}
		}
	}

	return manyAssoc, nil
}

// parseCascade processa a tag cascade
func (m *EntityMapper) parseCascade(cascade string) ([]string, error) {
	// Remove colchetes se existirem
	cascade = strings.Trim(cascade, "[]")

	// Divide por vírgulas
	parts := strings.Split(cascade, ",")

	var cascadeFlags []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cascadeFlags = append(cascadeFlags, part)
		}
	}

	return cascadeFlags, nil
}

// parseProp processa a tag prop
func (m *EntityMapper) parseProp(propTag string) ([]string, error) {
	// Remove colchetes se existirem
	propTag = strings.Trim(propTag, "[]")

	// Divide por vírgulas
	parts := strings.Split(propTag, ",")

	var propFlags []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			propFlags = append(propFlags, part)
		}
	}

	return propFlags, nil
}

// parseTableName processa o nome da tabela e schema
func (m *EntityMapper) parseTableName(tableName string) string {
	// Remove schema se presente (será processado separadamente)
	if strings.Contains(tableName, ";") {
		parts := strings.Split(tableName, ";")
		return strings.TrimSpace(parts[0])
	}
	return tableName
}

// parseTableSchema extrai o schema da tag table
func (m *EntityMapper) parseTableSchema(tableName string) string {
	if strings.Contains(tableName, ";") {
		parts := strings.Split(tableName, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "schema=") {
				return strings.TrimPrefix(part, "schema=")
			}
		}
	}
	return ""
}

// isRelationship verifica se o tipo representa um relacionamento
func (m *EntityMapper) isRelationship(t reflect.Type) bool {
	// Se é slice, verifica o tipo do elemento
	if t.Kind() == reflect.Slice {
		elemType := t.Elem()
		// Se o elemento é um ponteiro, pega o tipo apontado
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		return elemType.Kind() == reflect.Struct && !m.isPrimitiveType(elemType)
	}

	// Se é ponteiro, pega o tipo apontado
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Se é struct, verifica se não é um tipo primitivo
	if t.Kind() == reflect.Struct {
		return !m.isPrimitiveType(t)
	}

	return false
}

// isNullableType verifica se é um tipo nullable
func (m *EntityMapper) isNullableType(t reflect.Type) bool {
	return t.PkgPath() == "github.com/godata/odata/pkg/nullable"
}

// isPrimitiveType verifica se é um tipo primitivo
func (m *EntityMapper) isPrimitiveType(t reflect.Type) bool {
	// Tipos nullable são considerados primitivos
	if m.isNullableType(t) {
		return true
	}

	switch t {
	case reflect.TypeOf(time.Time{}):
		return true
	default:
		return false
	}
}

// mapGoType mapeia tipos Go para tipos OData
func (m *EntityMapper) mapGoType(t reflect.Type) string {
	// Remove ponteiros
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Verifica tipos nullable
	if t.PkgPath() == "github.com/godata/odata/pkg/nullable" {
		switch t.Name() {
		case "Int64":
			return "int64"
		case "String":
			return "string"
		case "Bool":
			return "bool"
		case "Time":
			return "time.Time"
		case "Float64":
			return "float64"
		}
	}

	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "int32"
	case reflect.Int64:
		return "int64"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "int64"
	case reflect.Float32:
		return "float32"
	case reflect.Float64:
		return "float64"
	case reflect.Bool:
		return "bool"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 { // []byte
			return "[]byte"
		}
		return "array"
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return "time.Time"
		}
		return "object"
	default:
		return "string"
	}
}

// getTableName extrai o nome da tabela da tag table ou do nome da struct
func (m *EntityMapper) getTableName(t reflect.Type) string {
	// Verifica se há um campo TableName com tag table
	if tag, ok := t.FieldByName("TableName"); ok {
		if tableName := tag.Tag.Get("table"); tableName != "" {
			// Processa schema se presente
			return m.parseTableName(tableName)
		}
	}

	// Verifica se há uma tag table na struct (usando reflect)
	// Procura por um campo _table_name que pode ser usado para definir o nome da tabela
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "_table_name" || field.Name == "TableName" {
			if tableName := field.Tag.Get("table"); tableName != "" {
				return m.parseTableName(tableName)
			}
		}
	}

	return ""
}

// getTableSchema extrai o schema da tag table
func (m *EntityMapper) getTableSchema(t reflect.Type) string {
	// Verifica se há um campo TableName com tag table
	if tag, ok := t.FieldByName("TableName"); ok {
		if tableName := tag.Tag.Get("table"); tableName != "" {
			return m.parseTableSchema(tableName)
		}
	}

	// Verifica se há uma tag table na struct (usando reflect)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "_table_name" || field.Name == "TableName" {
			if tableName := field.Tag.Get("table"); tableName != "" {
				return m.parseTableSchema(tableName)
			}
		}
	}

	return ""
}

// MapEntityFromStruct é uma função helper para mapear rapidamente uma struct
func MapEntityFromStruct(entity interface{}) (EntityMetadata, error) {
	mapper := NewEntityMapper()
	return mapper.MapEntity(entity)
}

// AutoRegisterEntities registra múltiplas entidades automaticamente
func (s *Server) AutoRegisterEntities(entities map[string]interface{}) error {
	for name, entity := range entities {
		s.RegisterEntity(name, entity)
	}
	return nil
}

// isMetadataField verifica se um campo é de metadados
func (m *EntityMapper) isMetadataField(field reflect.StructField) bool {
	// Lista de nomes de campos de metadados
	metadataFields := []string{"TableName", "_table_name"}

	// Verifica se o nome do campo está na lista de campos de metadados
	for _, metadataField := range metadataFields {
		if field.Name == metadataField {
			return true
		}
	}

	return false
}
