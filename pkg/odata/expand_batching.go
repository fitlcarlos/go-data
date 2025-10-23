package odata

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// expandWithBatching usa batching (2 queries) para relacionamentos, evitando N+1
// Estratégia: Faz uma query para buscar todas as entidades relacionadas de uma vez,
// depois agrupa em memória
func (s *BaseEntityService) expandWithBatching(
	entities []any,
	navProperty *PropertyMetadata,
	expandOption ExpandOption,
) ([]any, error) {
	if len(entities) == 0 {
		return entities, nil
	}

	log.Printf("🔍 EXPAND: Using BATCHING for %s (evitando N+1)", navProperty.Name)

	// 1. Coletar todos os IDs das entidades principais
	var parentIDs []interface{}
	parentIDSet := make(map[interface{}]bool) // Para evitar duplicatas

	for _, entity := range entities {
		orderedEntity, ok := entity.(*OrderedEntity)
		if !ok {
			continue
		}

		if id, exists := orderedEntity.Get(navProperty.Relationship.LocalProperty); exists {
			// Adiciona apenas se não existir (evita duplicatas)
			if !parentIDSet[id] {
				parentIDs = append(parentIDs, id)
				parentIDSet[id] = true
			}
		}
	}

	if len(parentIDs) == 0 {
		// Nenhuma entidade tem chave para expansão
		// Se é collection, define arrays vazios; se não, define nil
		for _, entity := range entities {
			orderedEntity, ok := entity.(*OrderedEntity)
			if !ok {
				continue
			}
			if navProperty.IsCollection {
				orderedEntity.Set(navProperty.Name, []any{})
			} else {
				orderedEntity.Set(navProperty.Name, nil)
			}
		}
		return entities, nil
	}

	// 2. Obter metadados da entidade relacionada
	relatedMetadata, err := s.getRelatedEntityMetadata(navProperty.RelatedType)
	if err != nil {
		return nil, fmt.Errorf("failed to get related entity metadata: %w", err)
	}

	// 3. Construir filtro: ReferencedProperty IN (parentIDs)
	filterParts := make([]string, len(parentIDs))
	for i, id := range parentIDs {
		// Formatar valor baseado no tipo
		switch v := id.(type) {
		case string:
			filterParts[i] = fmt.Sprintf("'%s'", v)
		default:
			filterParts[i] = fmt.Sprintf("%v", v)
		}
	}

	filterStr := fmt.Sprintf("%s in (%s)",
		navProperty.Relationship.ReferencedProperty,
		strings.Join(filterParts, ","))

	log.Printf("🔍 EXPAND BATCH: Filter = %s (querying %d related entities)", filterStr, len(parentIDs))

	// 4. Criar QueryOptions para a query em batch
	queryOptions := QueryOptions{}
	filterQuery, err := s.parseFilterWithTimeout(context.Background(), filterStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch filter: %w", err)
	}
	queryOptions.Filter = filterQuery

	// Aplicar opções adicionais do expand (filter adicional, orderby, etc)
	if expandOption.OrderBy != "" {
		queryOptions.OrderBy = expandOption.OrderBy
	}
	if expandOption.Skip > 0 {
		skip := GoDataSkipQuery(expandOption.Skip)
		queryOptions.Skip = &skip
	}
	if expandOption.Top > 0 {
		top := GoDataTopQuery(expandOption.Top)
		queryOptions.Top = &top
	}

	// 5. Executar query única para todas as entidades relacionadas (BATCHING!)
	relatedService := NewBaseEntityService(s.provider, relatedMetadata, s.server)
	response, err := relatedService.Query(context.Background(), queryOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query related entities in batch: %w", err)
	}

	relatedEntities, ok := response.Value.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from batch query")
	}

	log.Printf("✅ EXPAND BATCH: Retrieved %d related entities in 1 query", len(relatedEntities))

	// 6. Agrupar entidades relacionadas por foreign key
	grouped := make(map[interface{}][]any)

	for _, relatedEntity := range relatedEntities {
		orderedEntity, ok := relatedEntity.(*OrderedEntity)
		if !ok {
			continue
		}

		// Buscar o valor da foreign key
		fkValue, exists := orderedEntity.Get(navProperty.Relationship.ReferencedProperty)
		if !exists {
			// Tentar case-insensitive
			for _, prop := range orderedEntity.Properties {
				if strings.EqualFold(prop.Name, navProperty.Relationship.ReferencedProperty) {
					fkValue = prop.Value
					exists = true
					break
				}
			}
		}

		if exists {
			// Converter para string para comparação consistente
			fkKey := fmt.Sprintf("%v", fkValue)
			grouped[fkKey] = append(grouped[fkKey], relatedEntity)
		}
	}

	// 7. Associar entidades relacionadas às principais
	for _, entity := range entities {
		orderedEntity, ok := entity.(*OrderedEntity)
		if !ok {
			continue
		}

		pkValue, exists := orderedEntity.Get(navProperty.Relationship.LocalProperty)
		if !exists {
			// Se não tem chave, define vazio/nil
			if navProperty.IsCollection {
				orderedEntity.Set(navProperty.Name, []any{})
			} else {
				orderedEntity.Set(navProperty.Name, nil)
			}
			continue
		}

		// Converter para string para comparação consistente
		pkKey := fmt.Sprintf("%v", pkValue)

		if related, found := grouped[pkKey]; found {
			if navProperty.IsCollection {
				// 1:N - Retorna array
				orderedEntity.Set(navProperty.Name, related)
			} else {
				// N:1 ou 1:1 - Retorna primeira entidade
				if len(related) > 0 {
					orderedEntity.Set(navProperty.Name, related[0])
				} else {
					orderedEntity.Set(navProperty.Name, nil)
				}
			}
		} else {
			// Nenhuma entidade relacionada encontrada
			if navProperty.IsCollection {
				orderedEntity.Set(navProperty.Name, []any{})
			} else {
				orderedEntity.Set(navProperty.Name, nil)
			}
		}
	}

	log.Printf("✅ EXPAND BATCH: Associated related entities to %d parent entities", len(entities))

	return entities, nil
}

