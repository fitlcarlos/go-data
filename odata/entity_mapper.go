package odata

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// =======================================================================================
// ROW SCANNING & ENTITY MAPPING
// =======================================================================================

// scanRows escaneia rows SQL e converte para OrderedEntity
func (s *BaseEntityService) scanRows(rows *sql.Rows, expandOptions []ExpandOption) ([]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []any

	for rows.Next() {
		// Cria um slice de interfaces para os valores
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan dos valores
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		// Cria a entidade ordenada usando a ordem dos metadados
		result := NewOrderedEntity()

		// Primeiro, adiciona as propriedades normais
		for _, prop := range s.metadata.Properties {
			if !prop.IsNavigation {
				// Para propriedades normais, busca o valor na consulta SQL
				var colIndex = -1
				var colName = prop.ColumnName
				if colName == "" {
					colName = prop.Name
				}

				for i, col := range columns {
					if col == colName {
						colIndex = i
						break
					}
				}

				// Se encontrou a coluna, adiciona o valor com conversão de tipo
				if colIndex >= 0 {
					val := values[colIndex]
					if val != nil {
						// Usa convertValueToPropertyType para manter o tipo correto
						convertedVal, err := s.convertValueToPropertyType(val, prop.Name, s.metadata)
						if err != nil {
							// Em caso de erro na conversão, usa a conversão original como fallback
							switch v := val.(type) {
							case []byte:
								result.Set(prop.Name, string(v))
							default:
								result.Set(prop.Name, v)
							}
						} else {
							result.Set(prop.Name, convertedVal)
						}
					} else {
						result.Set(prop.Name, nil)
					}
				}
			}
		}

		// Depois, adiciona as propriedades de navegação (agora que as chaves estão disponíveis)
		// Só adiciona navigationLink se a propriedade NÃO está sendo expandida
		for _, prop := range s.metadata.Properties {
			if prop.IsNavigation {
				// Verifica se esta propriedade está sendo expandida (case-insensitive)
				isExpanded := false
				for _, expandOption := range expandOptions {
					if strings.EqualFold(expandOption.Property, prop.Name) {
						isExpanded = true
						break
					}
				}

				// Só adiciona navigation link se NÃO está sendo expandida
				if !isExpanded {
					result.SetNavigationProperty(prop.Name, s.buildNavigationLink(prop, result))
				}
			}
		}

		// Adiciona colunas que não estão nos metadados (caso existam)
		for i, col := range columns {
			propName := s.getPropertyNameByColumn(col)
			if propName == "" {
				propName = col
			}

			// Verifica se já foi adicionada
			if _, exists := result.Get(propName); !exists {
				val := values[i]
				if val != nil {
					// Também aplica conversão de tipo para colunas adicionais
					// Busca a propriedade nos metadados para fazer conversão correta
					var foundProp *PropertyMetadata
					for _, prop := range s.metadata.Properties {
						if strings.EqualFold(prop.Name, propName) || strings.EqualFold(prop.ColumnName, propName) {
							foundProp = &prop
							break
						}
					}

					if foundProp != nil {
						convertedVal, err := s.convertValueToPropertyType(val, foundProp.Name, s.metadata)
						if err != nil {
							// Fallback para conversão original
							switch v := val.(type) {
							case []byte:
								result.Set(propName, string(v))
							default:
								result.Set(propName, v)
							}
						} else {
							result.Set(propName, convertedVal)
						}
					} else {
						// Para colunas não mapeadas, mantém a conversão original
						switch v := val.(type) {
						case []byte:
							result.Set(propName, string(v))
						default:
							result.Set(propName, v)
						}
					}
				} else {
					result.Set(propName, nil)
				}
			}
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// entityToMap converte uma entidade para map[string]any
func (s *BaseEntityService) entityToMap(entity any) (map[string]any, error) {
	result := make(map[string]any)

	// Se já é um map, retorna diretamente
	if m, ok := entity.(map[string]any); ok {
		return m, nil
	}

	// Usa reflexão para converter struct para map
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("entity must be a struct or map")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Pula campos não exportados
		if !field.IsExported() {
			continue
		}

		// Usa a tag json ou o nome do campo
		name := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			name = strings.Split(tag, ",")[0]
		}

		result[name] = value.Interface()
	}

	return result, nil
}

// getPropertyNameByColumn encontra o nome da propriedade por nome da coluna
func (s *BaseEntityService) getPropertyNameByColumn(columnName string) string {
	for _, prop := range s.metadata.Properties {
		if prop.ColumnName == columnName {
			return prop.Name
		}
	}
	return ""
}

// buildNavigationLink constrói um navigation link para uma propriedade de navegação
func (s *BaseEntityService) buildNavigationLink(prop PropertyMetadata, entity *OrderedEntity) string {
	// Se não há chave, retorna vazio
	var keyValue any
	for _, p := range s.metadata.Properties {
		if p.IsKey {
			keyValue, _ = entity.Get(p.Name)
			break
		}
	}

	if keyValue == nil {
		return ""
	}

	// Constrói o link de navegação
	return fmt.Sprintf("/%s(%v)/%s", s.metadata.Name, keyValue, prop.Name)
}
