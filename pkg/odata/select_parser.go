package odata

import (
	"context"
	"fmt"
	"strings"
)

// GoDataSelectQuery representa uma query de seleção OData
type GoDataSelectQuery struct {
	SelectItems []*SelectItem
	RawValue    string
}

// SelectItem representa um item individual de seleção
type SelectItem struct {
	Segments []*Token
}

// ParseSelectString faz o parsing de uma string de $select
func ParseSelectString(ctx context.Context, sel string) (*GoDataSelectQuery, error) {
	if sel == "" {
		return &GoDataSelectQuery{SelectItems: []*SelectItem{}, RawValue: sel}, nil
	}

	items := strings.Split(sel, ",")
	result := []*SelectItem{}

	for _, item := range items {
		item = strings.TrimSpace(item)

		if len(item) == 0 {
			return nil, fmt.Errorf("empty select item")
		}

		// Tokeniza o item para validação
		_, err := GlobalFilterTokenizer.Tokenize(ctx, item)
		if err != nil {
			return nil, fmt.Errorf("invalid select value '%s': %w", item, err)
		}

		// Cria os segmentos baseado nos tokens
		segments := []*Token{}
		for _, val := range strings.Split(item, "/") {
			val = strings.TrimSpace(val)
			if val != "" {
				segments = append(segments, &Token{Value: val})
			}
		}

		if len(segments) == 0 {
			return nil, fmt.Errorf("empty select segments for item '%s'", item)
		}

		result = append(result, &SelectItem{Segments: segments})
	}

	return &GoDataSelectQuery{SelectItems: result, RawValue: sel}, nil
}

// ValidateSelectQuery valida uma query de seleção contra metadados
func ValidateSelectQuery(sel *GoDataSelectQuery, service *EntityService, entity *EntityMetadata) error {
	if sel == nil {
		return nil
	}

	newItems := []*SelectItem{}

	// Substitui wildcards por todas as propriedades da entidade
	for _, item := range sel.SelectItems {
		// TODO: permitir múltiplos segmentos de path
		if len(item.Segments) > 1 {
			return fmt.Errorf("multiple path segments in select clauses are not yet supported")
		}

		if item.Segments[0].Value == "*" {
			// Adiciona todas as propriedades da entidade
			for _, prop := range entity.Properties {
				if !prop.IsNavigation { // Não inclui propriedades de navegação no wildcard
					newItems = append(newItems, &SelectItem{
						Segments: []*Token{{Value: prop.Name}},
					})
				}
			}
		} else {
			newItems = append(newItems, item)
		}
	}

	sel.SelectItems = newItems

	// Valida cada item de seleção
	for _, item := range sel.SelectItems {
		if len(item.Segments) == 0 {
			return fmt.Errorf("empty select item")
		}

		propertyName := item.Segments[0].Value
		var foundProperty *PropertyMetadata

		for _, prop := range entity.Properties {
			if strings.EqualFold(prop.Name, propertyName) {
				foundProperty = &prop
				break
			}
		}

		if foundProperty == nil {
			return fmt.Errorf("entity '%s' has no property '%s'", entity.Name, propertyName)
		}

		// Marca o tipo semântico do token
		item.Segments[0].SemanticType = SemanticTypeProperty
		item.Segments[0].SemanticReference = foundProperty
	}

	return nil
}

// GetSelectedProperties retorna as propriedades selecionadas
func GetSelectedProperties(sel *GoDataSelectQuery) []string {
	if sel == nil {
		return []string{}
	}

	var properties []string
	for _, item := range sel.SelectItems {
		if len(item.Segments) > 0 {
			properties = append(properties, item.Segments[0].Value)
		}
	}

	return properties
}

// IsSelectAll verifica se a seleção inclui todas as propriedades
func IsSelectAll(sel *GoDataSelectQuery) bool {
	if sel == nil {
		return true
	}

	for _, item := range sel.SelectItems {
		if len(item.Segments) > 0 && item.Segments[0].Value == "*" {
			return true
		}
	}

	return false
}

// GetSelectComplexity calcula a complexidade de uma query de seleção
func GetSelectComplexity(sel *GoDataSelectQuery) int {
	if sel == nil {
		return 0
	}

	complexity := 0
	for _, item := range sel.SelectItems {
		complexity += len(item.Segments) // Cada segmento adiciona complexidade
	}

	return complexity
}

// ValidateSelectProperty valida se uma propriedade pode ser selecionada
func ValidateSelectProperty(propertyName string, entity *EntityMetadata) error {
	for _, prop := range entity.Properties {
		if strings.EqualFold(prop.Name, propertyName) {
			return nil
		}
	}

	return fmt.Errorf("property '%s' not found in entity '%s'", propertyName, entity.Name)
}

// ConvertSelectToSQL converte uma query de seleção para SQL
func ConvertSelectToSQL(sel *GoDataSelectQuery, entity *EntityMetadata) ([]string, error) {
	if sel == nil || len(sel.SelectItems) == 0 {
		// Retorna todas as propriedades não-navegação
		var columns []string
		for _, prop := range entity.Properties {
			if !prop.IsNavigation {
				columns = append(columns, fmt.Sprintf("%s.%s", entity.TableName, prop.ColumnName))
			}
		}
		return columns, nil
	}

	var columns []string
	for _, item := range sel.SelectItems {
		if len(item.Segments) > 0 {
			propertyName := item.Segments[0].Value

			// Encontra a propriedade
			for _, prop := range entity.Properties {
				if strings.EqualFold(prop.Name, propertyName) {
					if !prop.IsNavigation {
						columns = append(columns, fmt.Sprintf("%s.%s", entity.TableName, prop.ColumnName))
					}
					break
				}
			}
		}
	}

	return columns, nil
}

// FormatSelectExpression formata uma expressão de seleção para exibição
func FormatSelectExpression(sel *GoDataSelectQuery) string {
	if sel == nil {
		return ""
	}

	var parts []string
	for _, item := range sel.SelectItems {
		if len(item.Segments) > 0 {
			var segments []string
			for _, segment := range item.Segments {
				segments = append(segments, segment.Value)
			}
			parts = append(parts, strings.Join(segments, "/"))
		}
	}

	return strings.Join(parts, ", ")
}

// String retorna a representação em string da query de seleção
func (s *GoDataSelectQuery) String() string {
	if s == nil {
		return ""
	}
	return s.RawValue
}

// HasProperty verifica se uma propriedade específica está selecionada
func (s *GoDataSelectQuery) HasProperty(propertyName string) bool {
	if s == nil {
		return true // Se não há seleção, todas as propriedades são incluídas
	}

	for _, item := range s.SelectItems {
		if len(item.Segments) > 0 {
			if strings.EqualFold(item.Segments[0].Value, propertyName) || item.Segments[0].Value == "*" {
				return true
			}
		}
	}

	return false
}

// GetSelectItemByProperty retorna o item de seleção para uma propriedade específica
func (s *GoDataSelectQuery) GetSelectItemByProperty(propertyName string) *SelectItem {
	if s == nil {
		return nil
	}

	for _, item := range s.SelectItems {
		if len(item.Segments) > 0 && strings.EqualFold(item.Segments[0].Value, propertyName) {
			return item
		}
	}

	return nil
}

// AddSelectItem adiciona um item de seleção
func (s *GoDataSelectQuery) AddSelectItem(propertyName string) {
	if s == nil {
		return
	}

	// Verifica se já existe
	if s.HasProperty(propertyName) {
		return
	}

	item := &SelectItem{
		Segments: []*Token{{Value: propertyName}},
	}

	s.SelectItems = append(s.SelectItems, item)
}

// RemoveSelectItem remove um item de seleção
func (s *GoDataSelectQuery) RemoveSelectItem(propertyName string) {
	if s == nil {
		return
	}

	var newItems []*SelectItem
	for _, item := range s.SelectItems {
		if len(item.Segments) == 0 || !strings.EqualFold(item.Segments[0].Value, propertyName) {
			newItems = append(newItems, item)
		}
	}

	s.SelectItems = newItems
}
