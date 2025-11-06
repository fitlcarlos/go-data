package odata

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// =======================================================================================
// EXPAND PROCESSING
// =======================================================================================

// processExpandedNavigationWithOrder processa navegações expandidas seguindo a ordem OData v4
// Com otimização para evitar problema N+1 usando batching
func (s *BaseEntityService) processExpandedNavigationWithOrder(results []any, expandOptions []ExpandOption) ([]any, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Para cada opção de expand, processa em batch para todas as entidades
	for _, expandOption := range expandOptions {
		// Encontrar propriedade de navegação
		var navProperty *PropertyMetadata
		for _, prop := range s.metadata.Properties {
			if strings.EqualFold(prop.Name, expandOption.Property) && prop.IsNavigation {
				navProperty = &prop
				break
			}
		}

		if navProperty == nil {
			// Debug: propriedade não encontrada
			var availableProps []string
			for _, prop := range s.metadata.Properties {
				if prop.IsNavigation {
					availableProps = append(availableProps, prop.Name)
				}
			}
			log.Printf("Warning: Navigation property %s not found. Available: %v", expandOption.Property, availableProps)
			continue
		}

		// Decidir estratégia baseada no tipo de relacionamento e configuração
		var err error

		if s.server != nil && s.server.config.DisableJoinForExpand {
			// Usuário forçou batching para tudo
			log.Printf("🔍 EXPAND: Forced batching for %s (DisableJoinForExpand=true)", navProperty.Name)
			results, err = s.expandWithBatching(results, navProperty, expandOption)
		} else {
			// Usa batching (resolve N+1 problem)
			// TODO: Implementar JOIN otimizado para N:1 no futuro
			results, err = s.expandWithBatching(results, navProperty, expandOption)
		}

		if err != nil {
			// Log detalhado do erro
			errorMsg := fmt.Sprintf("%v", err)

			log.Printf("⚠️ Warning: Failed to expand %s: %v", expandOption.Property, err)

			// Se erro crítico de estrutura, falha
			if strings.Contains(errorMsg, "navigation property") && strings.Contains(errorMsg, "not found") {
				return nil, fmt.Errorf("critical error expanding navigation property %s: %w", expandOption.Property, err)
			}

			// Para outros erros, continua (entidades já têm navigation links)
			continue
		}
	}

	return results, nil
}

// findRelatedEntitiesWithOrder encontra entidades relacionadas seguindo a ordem OData v4
func (s *BaseEntityService) findRelatedEntitiesWithOrder(ctx context.Context, navProperty *PropertyMetadata, entity *OrderedEntity, expandOption ExpandOption) ([]any, error) {
	if navProperty.Relationship == nil {
		return nil, fmt.Errorf("navigation property has no relationship metadata")
	}

	// Obtém o valor da chave para fazer a busca relacionada
	localKeyValue, exists := entity.Get(navProperty.Relationship.LocalProperty)
	if !exists {
		// Tenta procurar por variações do nome
		for _, prop := range entity.Properties {
			if strings.EqualFold(prop.Name, navProperty.Relationship.LocalProperty) {
				localKeyValue = prop.Value
				exists = true
				break
			}
		}

		if !exists {
			return nil, fmt.Errorf("local key property %s not found in entity", navProperty.Relationship.LocalProperty)
		}
	}

	// Obtém os metadados da entidade relacionada
	relatedMetadata, err := s.getRelatedEntityMetadata(navProperty.RelatedType)
	if err != nil {
		return nil, fmt.Errorf("failed to get related entity metadata: %w", err)
	}

	// Constrói QueryOptions seguindo a ordem OData v4
	queryOptions := QueryOptions{}

	// 1. $filter – aplica filtros sobre a entidade relacionada
	if expandOption.Filter != "" {
		filterQuery, err := s.parseFilterWithTimeout(ctx, expandOption.Filter)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expand filter: %w", err)
		}
		queryOptions.Filter = filterQuery
	}

	// 2. $orderby – ordena os resultados filtrados
	if expandOption.OrderBy != "" {
		queryOptions.OrderBy = expandOption.OrderBy
	}

	// 3. $skip/$top – aplica paginação
	if expandOption.Skip > 0 {
		skip := GoDataSkipQuery(expandOption.Skip)
		queryOptions.Skip = &skip
	}
	if expandOption.Top > 0 {
		top := GoDataTopQuery(expandOption.Top)
		queryOptions.Top = &top
	}

	// 4. $compute seria aplicado aqui se suportado no expand
	// 5. $select – reduz os campos retornados
	if len(expandOption.Select) > 0 {
		selectStr := strings.Join(expandOption.Select, ",")
		selectQuery, err := ParseSelectString(ctx, selectStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expand select: %w", err)
		}
		queryOptions.Select = selectQuery
	}

	// 6. $expand – processa entidades relacionadas recursivamente
	if len(expandOption.Expand) > 0 {
		// Constrói string de expand recursiva
		var expandParts []string
		for _, exp := range expandOption.Expand {
			expandPart := exp.Property
			var subOptions []string

			if exp.Filter != "" {
				subOptions = append(subOptions, fmt.Sprintf("$filter=%s", exp.Filter))
			}
			if exp.OrderBy != "" {
				subOptions = append(subOptions, fmt.Sprintf("$orderby=%s", exp.OrderBy))
			}
			if len(exp.Select) > 0 {
				subOptions = append(subOptions, fmt.Sprintf("$select=%s", strings.Join(exp.Select, ",")))
			}
			if exp.Skip > 0 {
				subOptions = append(subOptions, fmt.Sprintf("$skip=%d", exp.Skip))
			}
			if exp.Top > 0 {
				subOptions = append(subOptions, fmt.Sprintf("$top=%d", exp.Top))
			}

			if len(subOptions) > 0 {
				expandPart = fmt.Sprintf("%s(%s)", expandPart, strings.Join(subOptions, ";"))
			}

			expandParts = append(expandParts, expandPart)
		}

		expandStr := strings.Join(expandParts, ",")
		expandQuery, err := ParseExpandString(ctx, expandStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse nested expand: %w", err)
		}
		queryOptions.Expand = expandQuery
	}

	// Cria serviço para a entidade relacionada
	relatedService := NewBaseEntityService(s.provider, relatedMetadata, s.server)

	// Executa a consulta seguindo a ordem OData v4
	response, err := relatedService.Query(ctx, queryOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to query related entities: %w", err)
	}

	// Converte response.Value para []any
	entities, ok := response.Value.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response.Value)
	}

	// Filtra resultados pelo relacionamento
	return s.filterRelatedEntities(entities, navProperty, localKeyValue, relatedMetadata)
}

// filterRelatedEntities filtra entidades relacionadas baseado no relacionamento
func (s *BaseEntityService) filterRelatedEntities(entities []any, navProperty *PropertyMetadata, keyValue any, relatedMetadata EntityMetadata) ([]any, error) {
	var filtered []any

	for _, entity := range entities {
		orderedEntity, ok := entity.(*OrderedEntity)
		if !ok {
			continue
		}

		// Obtém o valor da chave estrangeira na entidade relacionada
		foreignKeyValue, exists := orderedEntity.Get(navProperty.Relationship.ReferencedProperty)
		if !exists {
			// Tenta procurar por variações do nome
			for _, prop := range orderedEntity.Properties {
				if strings.EqualFold(prop.Name, navProperty.Relationship.ReferencedProperty) {
					foreignKeyValue = prop.Value
					exists = true
					break
				}
			}

			if !exists {
				continue
			}
		}

		// Compara os valores
		if fmt.Sprintf("%v", foreignKeyValue) == fmt.Sprintf("%v", keyValue) {
			filtered = append(filtered, orderedEntity)
		}
	}

	return filtered, nil
}

// getRelatedEntityMetadata obtém os metadados da entidade relacionada
func (s *BaseEntityService) getRelatedEntityMetadata(relatedType string) (EntityMetadata, error) {
	if s.server == nil {
		return EntityMetadata{}, fmt.Errorf("server reference not available")
	}

	// Busca no registry de entidades do servidor
	for entityName, service := range s.server.entities {
		metadata := service.GetMetadata()
		// Verifica se o nome da entidade ou o tipo corresponde
		if entityName == relatedType || metadata.Name == relatedType {
			return metadata, nil
		}
	}

	// Debug: mostra as entidades disponíveis
	var availableEntities []string
	for entityName, service := range s.server.entities {
		metadata := service.GetMetadata()
		availableEntities = append(availableEntities, fmt.Sprintf("%s (%s)", entityName, metadata.Name))
	}

	return EntityMetadata{}, fmt.Errorf("related entity metadata not found for type: %s. Available entities: %v", relatedType, availableEntities)
}

// convertExpandItemsToExpandOptions converte ExpandItems para ExpandOptions
func (s *BaseEntityService) convertExpandItemsToExpandOptions(items []*ExpandItem) []ExpandOption {
	var expandOptions []ExpandOption
	for _, item := range items {
		if len(item.Path) > 0 {
			expandOption := ExpandOption{
				Property: item.Path[0].Value,
			}

			// Converte opções de filtro
			if item.Filter != nil {
				expandOption.Filter = item.Filter.RawValue
			}

			// Converte opções de ordenação
			if item.OrderBy != nil {
				expandOption.OrderBy = item.OrderBy.RawValue
			}

			// Converte opções de select
			if item.Select != nil {
				expandOption.Select = GetSelectedProperties(item.Select)
			}

			// Converte opções de skip
			if item.Skip != nil {
				expandOption.Skip = GetSkipValue(item.Skip)
			}

			// Converte opções de top
			if item.Top != nil {
				expandOption.Top = GetTopValue(item.Top)
			}

			// Converte expansões recursivas
			if item.Expand != nil {
				expandOption.Expand = s.convertExpandItemsToExpandOptions(item.Expand.ExpandItems)
			}

			expandOptions = append(expandOptions, expandOption)
		}
	}
	return expandOptions
}
