package odata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ODataResponse representa a resposta padrão do OData
type ODataResponse struct {
	Context  string      `json:"@odata.context,omitempty"`
	Count    *int64      `json:"@odata.count,omitempty"`
	NextLink string      `json:"@odata.nextLink,omitempty"`
	Value    interface{} `json:"value"`
	Error    *ODataError `json:"error,omitempty"`
}

// ODataError representa um erro OData
type ODataError struct {
	Code    string             `json:"code"`
	Message string             `json:"message"`
	Target  string             `json:"target,omitempty"`
	Details []ODataErrorDetail `json:"details,omitempty"`
}

// ODataErrorDetail representa detalhes adicionais de um erro
type ODataErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Target  string `json:"target,omitempty"`
}

// ExpandOption representa uma opção de expansão com suas sub-opções
type ExpandOption struct {
	Property string         // Nome da propriedade a ser expandida
	Filter   string         // $filter aplicado à expansão
	OrderBy  string         // $orderby aplicado à expansão
	Select   []string       // $select aplicado à expansão
	Expand   []ExpandOption // $expand recursivo aplicado à expansão
	Skip     int            // $skip aplicado à expansão
	Top      int            // $top aplicado à expansão
	Count    bool           // $count aplicado à expansão
}

// QueryOptions representa as opções de consulta OData
type QueryOptions struct {
	Filter  *GoDataFilterQuery
	OrderBy string
	Select  *GoDataSelectQuery
	Expand  *GoDataExpandQuery
	Skip    *GoDataSkipQuery
	Top     *GoDataTopQuery
	Count   *GoDataCountQuery
	Compute *ComputeOption
	Search  *SearchOption
}

// EntityMetadata representa os metadados de uma entidade
type EntityMetadata struct {
	Name       string
	TableName  string
	Schema     string // Schema da tabela
	Properties []PropertyMetadata
	Keys       []string
}

// PropertyMetadata representa os metadados de uma propriedade
type PropertyMetadata struct {
	Name         string
	Type         string
	ColumnName   string
	IsKey        bool
	IsNullable   bool
	MaxLength    int
	Precision    int
	Scale        int
	IsNavigation bool
	HasDefault   bool
	IDGenerator  string
	SequenceName string
	IsCollection bool
	RelatedType  string
	Relationship *RelationshipMetadata
	// Novas propriedades para suporte avançado
	PropFlags       []string                 // Required, NoInsert, NoUpdate, Lazy, Unique
	CascadeFlags    []string                 // SaveUpdate, Remove, Refresh, RemoveOrphan
	Schema          string                   // Schema da tabela
	Association     *AssociationMetadata     // Para associações simples
	ManyAssociation *ManyAssociationMetadata // Para associações múltiplas
}

// RelationshipMetadata representa os metadados de um relacionamento
type RelationshipMetadata struct {
	LocalProperty      string
	ReferencedProperty string
	OnDelete           string
	OnUpdate           string
}

// AssociationMetadata representa os metadados de uma associação simples (1:1 ou N:1)
type AssociationMetadata struct {
	ForeignKey    string   // Nome da chave estrangeira
	References    string   // Campo referenciado na entidade relacionada
	RelatedEntity string   // Nome da entidade relacionada
	CascadeFlags  []string // SaveUpdate, Remove, Refresh
}

// ManyAssociationMetadata representa os metadados de uma associação múltipla (1:N ou N:N)
type ManyAssociationMetadata struct {
	ForeignKey        string   // Nome da chave estrangeira
	References        string   // Campo referenciado na entidade relacionada
	RelatedEntity     string   // Nome da entidade relacionada
	CascadeFlags      []string // SaveUpdate, Remove, Refresh, RemoveOrphan
	JoinTable         string   // Nome da tabela de junção (para N:N)
	JoinColumn        string   // Coluna de junção (para N:N)
	InverseJoinColumn string   // Coluna de junção inversa (para N:N)
}

// IDGeneratorType representa os tipos de geradores de ID
type IDGeneratorType string

const (
	IDGeneratorNone      IDGeneratorType = "none"
	IDGeneratorSequence  IDGeneratorType = "sequence"
	IDGeneratorGuid      IDGeneratorType = "guid"
	IDGeneratorUuid38    IDGeneratorType = "uuid38"
	IDGeneratorUuid36    IDGeneratorType = "uuid36"
	IDGeneratorUuid32    IDGeneratorType = "uuid32"
	IDGeneratorSmartGuid IDGeneratorType = "smartGuid"
)

// DatabaseProvider interface para os providers de banco
type DatabaseProvider interface {
	Connect(connectionString string) error
	Close() error
	GetConnection() *sql.DB
	GetDriverName() string
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	BuildSelectQuery(entity EntityMetadata, options QueryOptions) (string, []interface{}, error)
	BuildInsertQuery(entity EntityMetadata, data map[string]interface{}) (string, []interface{}, error)
	BuildUpdateQuery(entity EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error)
	BuildDeleteQuery(entity EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error)
	BuildWhereClause(filter string, metadata EntityMetadata) (string, []interface{}, error)
	BuildOrderByClause(orderBy string, metadata EntityMetadata) (string, error)
	MapGoTypeToSQL(goType string) string
	FormatDateTime(t time.Time) string
}

// EntityService interface para serviços de entidade
type EntityService interface {
	GetMetadata() EntityMetadata
	Query(ctx context.Context, options QueryOptions) (*ODataResponse, error)
	Get(ctx context.Context, keys map[string]interface{}) (interface{}, error)
	Create(ctx context.Context, entity interface{}) (interface{}, error)
	Update(ctx context.Context, keys map[string]interface{}, entity interface{}) (interface{}, error)
	Delete(ctx context.Context, keys map[string]interface{}) error
}

// ODataService é o serviço principal
type ODataService struct {
	Provider DatabaseProvider
	Entities map[string]EntityService
}

// ConnectionConfig representa a configuração de conexão
type ConnectionConfig struct {
	Driver           string
	ConnectionString string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
}

// FilterOperator representa os operadores de filtro OData
type FilterOperator string

const (
	FilterEq         FilterOperator = "eq"
	FilterNe         FilterOperator = "ne"
	FilterGt         FilterOperator = "gt"
	FilterGe         FilterOperator = "ge"
	FilterLt         FilterOperator = "lt"
	FilterLe         FilterOperator = "le"
	FilterIn         FilterOperator = "in"
	FilterContains   FilterOperator = "contains"
	FilterStartsWith FilterOperator = "startswith"
	FilterEndsWith   FilterOperator = "endswith"
	// Funções de string
	FilterLength    FilterOperator = "length"
	FilterToLower   FilterOperator = "tolower"
	FilterToUpper   FilterOperator = "toupper"
	FilterTrim      FilterOperator = "trim"
	FilterConcat    FilterOperator = "concat"
	FilterIndexOf   FilterOperator = "indexof"
	FilterSubstring FilterOperator = "substring"
	// Operadores matemáticos
	FilterAdd FilterOperator = "add"
	FilterSub FilterOperator = "sub"
	FilterMul FilterOperator = "mul"
	FilterDiv FilterOperator = "div"
	FilterMod FilterOperator = "mod"
	// Funções de data/hora
	FilterYear   FilterOperator = "year"
	FilterMonth  FilterOperator = "month"
	FilterDay    FilterOperator = "day"
	FilterHour   FilterOperator = "hour"
	FilterMinute FilterOperator = "minute"
	FilterSecond FilterOperator = "second"
	FilterNow    FilterOperator = "now"
)

// FilterExpression representa uma expressão de filtro
type FilterExpression struct {
	Property string
	Operator FilterOperator
	Value    interface{}
	// Para funções que precisam de múltiplos argumentos
	Arguments []interface{}
}

// OrderByDirection representa a direção da ordenação
type OrderByDirection string

const (
	OrderAsc  OrderByDirection = "asc"
	OrderDesc OrderByDirection = "desc"
)

// OrderByExpression representa uma expressão de ordenação
type OrderByExpression struct {
	Property  string
	Direction OrderByDirection
}

// MetadataResponse representa a resposta de metadados em JSON
type MetadataResponse struct {
	Context    string               `json:"@odata.context"`
	Version    string               `json:"@odata.version"`
	Entities   []EntityTypeMetadata `json:"entities"`
	EntitySets []EntitySetMetadata  `json:"entitySets"`
	Schemas    []SchemaMetadata     `json:"schemas"`
}

// EntityTypeMetadata representa os metadados de um tipo de entidade
type EntityTypeMetadata struct {
	Name       string                       `json:"name"`
	Namespace  string                       `json:"namespace"`
	Keys       []string                     `json:"keys"`
	Properties []PropertyTypeMetadata       `json:"properties"`
	Navigation []NavigationPropertyMetadata `json:"navigation,omitempty"`
}

// PropertyTypeMetadata representa os metadados de uma propriedade
type PropertyTypeMetadata struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	MaxLength  int    `json:"maxLength,omitempty"`
	Precision  int    `json:"precision,omitempty"`
	Scale      int    `json:"scale,omitempty"`
	IsKey      bool   `json:"isKey"`
	HasDefault bool   `json:"hasDefault"`
}

// NavigationPropertyMetadata representa os metadados de uma propriedade de navegação
type NavigationPropertyMetadata struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsCollection bool   `json:"isCollection"`
	Partner      string `json:"partner,omitempty"`
}

// EntitySetMetadata representa os metadados de um conjunto de entidades
type EntitySetMetadata struct {
	Name       string `json:"name"`
	EntityType string `json:"entityType"`
	Kind       string `json:"kind"`
	URL        string `json:"url"`
}

// SchemaMetadata representa os metadados de um schema
type SchemaMetadata struct {
	Namespace       string                  `json:"namespace"`
	Alias           string                  `json:"alias,omitempty"`
	EntityTypes     []EntityTypeMetadata    `json:"entityTypes"`
	EntitySets      []EntitySetMetadata     `json:"entitySets"`
	EntityContainer EntityContainerMetadata `json:"entityContainer"`
}

// EntityContainerMetadata representa os metadados de um container de entidades
type EntityContainerMetadata struct {
	Name       string              `json:"name"`
	EntitySets []EntitySetMetadata `json:"entitySets"`
}

// OrderedEntity representa uma entidade com propriedades ordenadas
type OrderedEntity struct {
	Properties      []OrderedProperty `json:"-"`
	NavigationLinks []NavigationLink  `json:"-"`
	data            map[string]interface{}
	navigationData  map[string]string
}

// NavigationLink representa um link de navegação OData
type NavigationLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// OrderedProperty representa uma propriedade ordenada
type OrderedProperty struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// NewOrderedEntity cria uma nova entidade ordenada
func NewOrderedEntity() *OrderedEntity {
	return &OrderedEntity{
		Properties:      make([]OrderedProperty, 0),
		NavigationLinks: make([]NavigationLink, 0),
		data:            make(map[string]interface{}),
		navigationData:  make(map[string]string),
	}
}

// Set adiciona uma propriedade mantendo a ordem
func (e *OrderedEntity) Set(name string, value interface{}) {
	// Verifica se a propriedade já existe
	for i, prop := range e.Properties {
		if prop.Name == name {
			e.Properties[i].Value = value
			e.data[name] = value
			return
		}
	}

	// Adiciona nova propriedade
	e.Properties = append(e.Properties, OrderedProperty{
		Name:  name,
		Value: value,
	})
	e.data[name] = value
}

// SetNavigationProperty adiciona uma propriedade de navegação como link
func (e *OrderedEntity) SetNavigationProperty(name string, navigationURL string) {
	// Verifica se o navigation link já existe
	for i, link := range e.NavigationLinks {
		if link.Name == name {
			e.NavigationLinks[i].URL = navigationURL
			e.navigationData[name] = navigationURL
			return
		}
	}

	// Adiciona novo navigation link
	e.NavigationLinks = append(e.NavigationLinks, NavigationLink{
		Name: name,
		URL:  navigationURL,
	})
	e.navigationData[name] = navigationURL
}

// Get obtém o valor de uma propriedade
func (e *OrderedEntity) Get(name string) (interface{}, bool) {
	value, exists := e.data[name]
	return value, exists
}

// ToMap converte para map (pode perder a ordem)
func (e *OrderedEntity) ToMap() map[string]interface{} {
	return e.data
}

// MarshalJSON implementa json.Marshaler mantendo a ordem
func (e *OrderedEntity) MarshalJSON() ([]byte, error) {
	// Constrói o JSON mantendo a ordem das propriedades
	var pairs []string

	// Adiciona propriedades normais
	for _, prop := range e.Properties {
		// Serializa o valor
		valueJSON, err := json.Marshal(prop.Value)
		if err != nil {
			return nil, err
		}

		// Adiciona o par chave-valor
		pairs = append(pairs, fmt.Sprintf(`"%s":%s`, prop.Name, string(valueJSON)))
	}

	// Adiciona navigation links no formato OData
	for _, navLink := range e.NavigationLinks {
		// Adiciona o navigation link no formato @odata.navigationLink
		linkJSON, err := json.Marshal(navLink.URL)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, fmt.Sprintf(`"%s@odata.navigationLink":%s`, navLink.Name, string(linkJSON)))
	}

	return []byte(fmt.Sprintf("{%s}", strings.Join(pairs, ","))), nil
}

// UnmarshalJSON implementa json.Unmarshaler
func (e *OrderedEntity) UnmarshalJSON(data []byte) error {
	// Parse do JSON em um map temporário
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Reinicializa os dados
	e.data = make(map[string]interface{})
	e.Properties = make([]OrderedProperty, 0)
	e.NavigationLinks = make([]NavigationLink, 0)
	e.navigationData = make(map[string]string)

	// Adiciona as propriedades na ordem que aparecem no JSON
	for key, value := range temp {
		if strings.HasSuffix(key, "@odata.navigationLink") {
			// É um navigation link
			propName := strings.TrimSuffix(key, "@odata.navigationLink")
			e.SetNavigationProperty(propName, value.(string))
		} else {
			// É uma propriedade normal
			e.Set(key, value)
		}
	}

	return nil
}

// OrderedEntityResponse representa uma resposta de entidade única mantendo a ordem dos campos
type OrderedEntityResponse struct {
	Context         string                   `json:"@odata.context"`
	Fields          []ResponseField          `json:"-"`
	NavigationLinks []ResponseNavigationLink `json:"-"`
	entityMetadata  EntityMetadata           `json:"-"`
}

// ResponseField representa um campo na resposta
type ResponseField struct {
	Name  string      `json:"-"`
	Value interface{} `json:"-"`
}

// ResponseNavigationLink representa um navigation link na resposta
type ResponseNavigationLink struct {
	Name string `json:"-"`
	URL  string `json:"-"`
}

// NewOrderedEntityResponse cria uma nova resposta de entidade ordenada
func NewOrderedEntityResponse(context string, metadata EntityMetadata) *OrderedEntityResponse {
	return &OrderedEntityResponse{
		Context:         context,
		Fields:          make([]ResponseField, 0),
		NavigationLinks: make([]ResponseNavigationLink, 0),
		entityMetadata:  metadata,
	}
}

// AddField adiciona um campo à resposta
func (r *OrderedEntityResponse) AddField(name string, value interface{}) {
	r.Fields = append(r.Fields, ResponseField{
		Name:  name,
		Value: value,
	})
}

// AddNavigationLink adiciona um navigation link à resposta
func (r *OrderedEntityResponse) AddNavigationLink(name, url string) {
	r.NavigationLinks = append(r.NavigationLinks, ResponseNavigationLink{
		Name: name,
		URL:  url,
	})
}

// MarshalJSON implementa json.Marshaler mantendo a ordem dos campos conforme os metadados
func (r *OrderedEntityResponse) MarshalJSON() ([]byte, error) {
	var pairs []string

	// Adiciona o contexto primeiro
	contextJSON, err := json.Marshal(r.Context)
	if err != nil {
		return nil, err
	}
	pairs = append(pairs, fmt.Sprintf(`"@odata.context":%s`, string(contextJSON)))

	// Adiciona campos na ordem dos metadados da entidade
	fieldsMap := make(map[string]interface{})
	for _, field := range r.Fields {
		fieldsMap[field.Name] = field.Value
	}

	// Itera sobre as propriedades na ordem definida nos metadados
	for _, metaProp := range r.entityMetadata.Properties {
		if !metaProp.IsNavigation {
			if value, exists := fieldsMap[metaProp.Name]; exists {
				// Serializa o valor
				valueJSON, err := json.Marshal(value)
				if err != nil {
					return nil, err
				}
				pairs = append(pairs, fmt.Sprintf(`"%s":%s`, metaProp.Name, string(valueJSON)))
			}
		}
	}

	// Adiciona campos que não estão nos metadados (na ordem que foram adicionados)
	addedFields := make(map[string]bool)
	for _, metaProp := range r.entityMetadata.Properties {
		if !metaProp.IsNavigation {
			addedFields[metaProp.Name] = true
		}
	}

	for _, field := range r.Fields {
		if !addedFields[field.Name] {
			valueJSON, err := json.Marshal(field.Value)
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, fmt.Sprintf(`"%s":%s`, field.Name, string(valueJSON)))
			addedFields[field.Name] = true
		}
	}

	// Adiciona navigation links na ordem dos metadados
	for _, metaProp := range r.entityMetadata.Properties {
		if metaProp.IsNavigation {
			for _, navLink := range r.NavigationLinks {
				if navLink.Name == metaProp.Name {
					linkJSON, err := json.Marshal(navLink.URL)
					if err != nil {
						return nil, err
					}
					pairs = append(pairs, fmt.Sprintf(`"%s@odata.navigationLink":%s`, navLink.Name, string(linkJSON)))
					break
				}
			}
		}
	}

	return []byte(fmt.Sprintf("{%s}", strings.Join(pairs, ","))), nil
}

// Funções de conveniência para criação de erros

// NewODataError cria um novo erro OData
func NewODataError(code, message string) *ODataError {
	return &ODataError{
		Code:    code,
		Message: message,
	}
}

// NewODataErrorWithTarget cria um novo erro OData com target
func NewODataErrorWithTarget(code, message, target string) *ODataError {
	return &ODataError{
		Code:    code,
		Message: message,
		Target:  target,
	}
}

// BadRequestError cria um erro de requisição inválida
func BadRequestError(message string) *ODataError {
	return NewODataError("BadRequest", message)
}

// EntityNotFoundError cria um erro de entidade não encontrada
func EntityNotFoundError(entityName string) *ODataError {
	return NewODataErrorWithTarget(
		"EntityNotFound",
		fmt.Sprintf("Entity '%s' not found", entityName),
		entityName,
	)
}

// PropertyNotFoundError cria um erro de propriedade não encontrada
func PropertyNotFoundError(propertyName, entityName string) *ODataError {
	return NewODataErrorWithTarget(
		"PropertyNotFound",
		fmt.Sprintf("Property '%s' not found in entity '%s'", propertyName, entityName),
		fmt.Sprintf("%s.%s", entityName, propertyName),
	)
}

// InvalidFilterError cria um erro de filtro inválido
func InvalidFilterError(filter string) *ODataError {
	return NewODataErrorWithTarget(
		"InvalidFilter",
		fmt.Sprintf("Invalid filter expression: %s", filter),
		"$filter",
	)
}

// =======================================================================================
// USER IDENTITY
// =======================================================================================

// UserIdentity representa a identidade do usuário autenticado
type UserIdentity struct {
	Username string                 `json:"username"`
	Roles    []string               `json:"roles"`
	Scopes   []string               `json:"scopes"`
	Admin    bool                   `json:"admin"`
	Custom   map[string]interface{} `json:"custom"`
}

// HasRole verifica se o usuário possui uma role específica
func (u *UserIdentity) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasScope verifica se o usuário possui um scope específico
func (u *UserIdentity) HasScope(scope string) bool {
	for _, s := range u.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// HasAnyRole verifica se o usuário possui pelo menos uma das roles
func (u *UserIdentity) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAnyScope verifica se o usuário possui pelo menos um dos scopes
func (u *UserIdentity) HasAnyScope(scopes ...string) bool {
	for _, scope := range scopes {
		if u.HasScope(scope) {
			return true
		}
	}
	return false
}

// GetCustomClaim retorna um valor custom do usuário
func (u *UserIdentity) GetCustomClaim(key string) (interface{}, bool) {
	if u.Custom == nil {
		return nil, false
	}
	val, ok := u.Custom[key]
	return val, ok
}

// Constantes para chaves do contexto
const (
	UserContextKey = "user" // Chave para armazenar usuário no contexto
)
