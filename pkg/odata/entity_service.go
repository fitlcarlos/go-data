package odata

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// BaseEntityService implementa o serviço base para entidades
type BaseEntityService struct {
	provider      DatabaseProvider
	metadata      EntityMetadata
	server        *Server
	computeParser *ComputeParser
	searchParser  *SearchParser
}

// NewBaseEntityService cria uma nova instância do serviço base
func NewBaseEntityService(provider DatabaseProvider, metadata EntityMetadata, server *Server) *BaseEntityService {
	return &BaseEntityService{
		provider:      provider,
		metadata:      metadata,
		server:        server,
		computeParser: NewComputeParser(),
		searchParser:  NewSearchParser(),
	}
}

// GetMetadata retorna os metadados da entidade
func (s *BaseEntityService) GetMetadata() EntityMetadata {
	return s.metadata
}

// Query executa uma consulta OData seguindo a ordem correta de execução das query options
func (s *BaseEntityService) Query(ctx context.Context, options QueryOptions) (*ODataResponse, error) {
	// Verifica cancelamento do contexto
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// ORDEM CORRETA DE EXECUÇÃO OData v4:
	// 1. $filter – aplica filtros sobre a entidade atual
	// 2. $orderby – ordena os resultados filtrados
	// 3. $skip/$top – aplica paginação
	// 4. $compute – calcula colunas derivadas
	// 5. $select – reduz os campos retornados
	// 6. $expand – processa entidades relacionadas (recursivamente)

	// Constrói a query SQL seguindo a ordem correta
	var query string
	var args []any
	var err error

	// Aplica $filter, $orderby, $skip/$top primeiro na query SQL
	if optimizedProvider, ok := s.provider.(interface {
		BuildSelectQueryOptimized(ctx context.Context, metadata EntityMetadata, options QueryOptions) (string, []any, error)
	}); ok {
		query, args, err = optimizedProvider.BuildSelectQueryOptimized(ctx, s.metadata, options)
	} else {
		query, args, err = s.provider.BuildSelectQuery(s.metadata, options)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	// Verifica cancelamento do contexto antes da execução
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	rows, err := s.executeQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Converte os resultados, passando as propriedades expandidas
	var expandOptions []ExpandOption
	if options.Expand != nil {

		// Converte GoDataExpandQuery para []ExpandOption
		for _, item := range options.Expand.ExpandItems {
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
	}

	// Scana os resultados (aplicando paginação do SQL)
	results, err := s.scanRows(rows, expandOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %w", err)
	}

	// 4. Processa $compute DEPOIS da execução SQL básica
	if options.Compute != nil {
		err := s.processComputeOption(ctx, options.Compute)
		if err != nil {
			return nil, fmt.Errorf("failed to process compute option: %w", err)
		}

		// Aplica campos computados aos resultados
		results, err = s.applyComputeToResults(ctx, results, options.Compute)
		if err != nil {
			return nil, fmt.Errorf("failed to apply compute to results: %w", err)
		}
	}

	// 5. Processa $search se presente (integrado com $filter)
	if options.Search != nil {
		err := s.processSearchOption(ctx, options.Search)
		if err != nil {
			return nil, fmt.Errorf("failed to process search option: %w", err)
		}
	}

	// 6. Processa navegações expandidas seguindo a ordem recursivamente
	if len(expandOptions) > 0 {
		expandedResults, err := s.processExpandedNavigationWithOrder(ctx, results, expandOptions)
		if err != nil {
			// Log do erro mas tenta continuar com navigation links
			log.Printf("Warning: Failed to process expanded navigation: %v. Continuing with navigation links.", err)
		} else {
			results = expandedResults
		}
	}

	// 7. Aplica $select final se necessário (pode ser otimizado no SQL)
	if options.Select != nil {
		results, err = s.applySelectToResults(results, options.Select)
		if err != nil {
			return nil, fmt.Errorf("failed to apply select to results: %w", err)
		}
	}

	// Constrói a resposta OData
	response := &ODataResponse{
		Context: fmt.Sprintf("$metadata#%s", s.metadata.Name),
		Value:   results,
	}

	// Adiciona o count se solicitado
	if IsCountRequested(options.Count) {
		count, err := s.GetCount(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("failed to get count: %w", err)
		}
		response.Count = &count
	}

	return response, nil
}

// Get recupera uma entidade específica pelas chaves
func (s *BaseEntityService) Get(ctx context.Context, keys map[string]any) (any, error) {
	log.Printf("🔍 BaseEntityService.Get - Starting with keys: %+v", keys)

	// Log dos tipos das chaves para debug
	for k, v := range keys {
		log.Printf("🔍 BaseEntityService.Get - Key '%s': value=%v, type=%T", k, v, v)
	}

	filterQuery, err := s.BuildTypedKeyFilter(ctx, keys)
	if err != nil {
		log.Printf("❌ BaseEntityService.Get - Failed to build typed key filter: %v", err)
		return nil, fmt.Errorf("failed to build typed key filter: %w", err)
	}

	options := QueryOptions{
		Filter: filterQuery,
	}

	log.Printf("🔍 BaseEntityService.Get - Options: %+v", options)

	query, args, err := s.provider.BuildSelectQuery(s.metadata, options)
	if err != nil {
		log.Printf("❌ BaseEntityService.Get - Failed to build select query: %v", err)
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	log.Printf("🔍 BaseEntityService.Get - Query: %s", query)
	log.Printf("🔍 BaseEntityService.Get - Args: %+v", args)

	// Log dos tipos dos args para debug
	for i, arg := range args {
		log.Printf("🔍 BaseEntityService.Get - Arg[%d]: value=%v, type=%T", i, arg, arg)
	}

	// Executa a query
	rows, err := s.executeQuery(ctx, query, args)
	if err != nil {
		log.Printf("❌ BaseEntityService.Get - Failed to execute query: %v", err)
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	log.Printf("✅ BaseEntityService.Get - Query executed successfully")

	// Converte os resultados (sem expand para Get)
	results, err := s.scanRows(rows, []ExpandOption{})
	if err != nil {
		log.Printf("❌ BaseEntityService.Get - Failed to scan rows: %v", err)
		return nil, fmt.Errorf("failed to scan rows: %w", err)
	}

	if len(results) == 0 {
		log.Printf("❌ BaseEntityService.Get - Entity not found")
		return nil, fmt.Errorf("entity not found")
	}

	log.Printf("✅ BaseEntityService.Get - Entity found successfully")
	return results[0], nil
}

// buildTypedPropertyFilter constrói um filtro para uma propriedade preservando o tipo do valor
func (s *BaseEntityService) buildTypedPropertyFilter(ctx context.Context, propertyName string, propertyValue any) (*GoDataFilterQuery, error) {
	log.Printf("🔍 buildTypedPropertyFilter - Starting with property: %s, value: %v, type: %T", propertyName, propertyValue, propertyValue)

	// Cria os nós da árvore de parse preservando os tipos
	propertyNode := &ParseNode{
		Token: &Token{
			Type:  int(FilterTokenProperty),
			Value: propertyName,
		},
		Children: []*ParseNode{},
	}

	valueNode := &ParseNode{
		Token: &Token{
			Type:              s.getTokenTypeForValue(propertyValue),
			Value:             fmt.Sprintf("%v", propertyValue), // Token.Value é string
			SemanticReference: propertyValue,                    // Preserva o valor tipado original
		},
		Children: []*ParseNode{},
	}

	// Cria o nó de comparação (eq)
	comparisonNode := &ParseNode{
		Token: &Token{
			Type:  int(FilterTokenComparison),
			Value: "eq",
		},
		Children: []*ParseNode{propertyNode, valueNode},
	}

	filterQuery := &GoDataFilterQuery{
		RawValue: fmt.Sprintf("%s eq %v", propertyName, propertyValue), // Para logging
		Tree:     comparisonNode,
	}

	log.Printf("✅ buildTypedPropertyFilter - Created typed filter")
	return filterQuery, nil
}

// BuildTypedKeyFilter constrói um filtro preservando os tipos das chaves
func (s *BaseEntityService) BuildTypedKeyFilter(ctx context.Context, keys map[string]any) (*GoDataFilterQuery, error) {
	log.Printf("🔍 buildTypedKeyFilter - Starting with keys: %+v", keys)

	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys provided")
	}

	// Para uma única chave, cria um nó de comparação simples
	if len(keys) == 1 {
		for keyName, keyValue := range keys {
			log.Printf("🔍 buildTypedKeyFilter - Single key '%s': value=%v, type=%T", keyName, keyValue, keyValue)

			// Cria os nós da árvore de parse preservando os tipos
			propertyNode := &ParseNode{
				Token: &Token{
					Type:  int(FilterTokenProperty),
					Value: keyName,
				},
				Children: []*ParseNode{},
			}

			valueNode := &ParseNode{
				Token: &Token{
					Type:              s.getTokenTypeForValue(keyValue),
					Value:             fmt.Sprintf("%v", keyValue), // Token.Value é string
					SemanticReference: keyValue,                    // Preserva o valor tipado original
				},
				Children: []*ParseNode{},
			}

			// Cria o nó de comparação (eq)
			comparisonNode := &ParseNode{
				Token: &Token{
					Type:  int(FilterTokenComparison),
					Value: "eq",
				},
				Children: []*ParseNode{propertyNode, valueNode},
			}

			filterQuery := &GoDataFilterQuery{
				RawValue: fmt.Sprintf("%s eq %v", keyName, keyValue), // Para logging
				Tree:     comparisonNode,
			}

			log.Printf("✅ buildTypedKeyFilter - Created single key filter")
			return filterQuery, nil
		}
	}

	// Para múltiplas chaves, cria nós AND concatenados
	var nodes []*ParseNode
	var filterParts []string

	for keyName, keyValue := range keys {
		log.Printf("🔍 buildTypedKeyFilter - Multi key '%s': value=%v, type=%T", keyName, keyValue, keyValue)

		// Cria os nós para esta chave
		propertyNode := &ParseNode{
			Token: &Token{
				Type:  int(FilterTokenProperty),
				Value: keyName,
			},
			Children: []*ParseNode{},
		}

		valueNode := &ParseNode{
			Token: &Token{
				Type:              s.getTokenTypeForValue(keyValue),
				Value:             fmt.Sprintf("%v", keyValue), // Token.Value é string
				SemanticReference: keyValue,                    // Preserva o valor tipado original
			},
			Children: []*ParseNode{},
		}

		// Cria o nó de comparação
		comparisonNode := &ParseNode{
			Token: &Token{
				Type:  int(FilterTokenComparison),
				Value: "eq",
			},
			Children: []*ParseNode{propertyNode, valueNode},
		}

		nodes = append(nodes, comparisonNode)
		filterParts = append(filterParts, fmt.Sprintf("%s eq %v", keyName, keyValue))
	}

	// Combina os nós com AND se há múltiplas chaves
	var rootNode *ParseNode
	if len(nodes) == 1 {
		rootNode = nodes[0]
	} else {
		// Cria a cadeia de nós AND
		rootNode = nodes[0]
		for i := 1; i < len(nodes); i++ {
			andNode := &ParseNode{
				Token: &Token{
					Type:  int(FilterTokenLogical),
					Value: "and",
				},
				Children: []*ParseNode{rootNode, nodes[i]},
			}
			rootNode = andNode
		}
	}

	filterQuery := &GoDataFilterQuery{
		RawValue: strings.Join(filterParts, " and "), // Para logging
		Tree:     rootNode,
	}

	log.Printf("✅ buildTypedKeyFilter - Created multi-key filter with %d keys", len(keys))
	return filterQuery, nil
}

// getTokenTypeForValue retorna o tipo de token apropriado baseado no tipo do valor
func (s *BaseEntityService) getTokenTypeForValue(value any) int {
	switch value.(type) {
	case string:
		return int(FilterTokenString)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return int(FilterTokenNumber)
	case float32, float64:
		return int(FilterTokenNumber)
	case bool:
		return int(FilterTokenBoolean)
	default:
		// Para tipos desconhecidos, trata como string
		return int(FilterTokenString)
	}
}

// Create cria uma nova entidade
func (s *BaseEntityService) Create(ctx context.Context, entity any) (any, error) {
	// Converte a entidade para map
	data, err := s.entityToMap(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to map: %w", err)
	}

	// Constrói a query SQL
	query, args, err := s.provider.BuildInsertQuery(s.metadata, data)
	if err != nil {
		return nil, fmt.Errorf("failed to build insert query: %w", err)
	}

	// Executa a query
	result, err := s.executeExec(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute insert: %w", err)
	}

	// Verifica se a inserção foi bem-sucedida
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("no rows inserted")
	}

	// Se há chaves auto-incrementais, busca o registro inserido
	if s.hasAutoIncrementKey() {
		lastID, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}

		// Busca o registro inserido
		keyProp := s.getAutoIncrementKey()
		keys := map[string]any{
			keyProp.Name: lastID,
		}

		return s.Get(ctx, keys)
	}

	return entity, nil
}

// Update atualiza uma entidade existente
func (s *BaseEntityService) Update(ctx context.Context, keys map[string]any, entity any) (any, error) {
	// Converte a entidade para map
	data, err := s.entityToMap(entity)
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to map: %w", err)
	}

	// Remove as chaves dos dados a serem atualizados
	for key := range keys {
		delete(data, key)
	}

	// Constrói a query SQL
	query, args, err := s.provider.BuildUpdateQuery(s.metadata, data, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to build update query: %w", err)
	}

	// Executa a query
	result, err := s.executeExec(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute update: %w", err)
	}

	// Verifica se a atualização foi bem-sucedida
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("no rows updated")
	}

	// Busca o registro atualizado
	return s.Get(ctx, keys)
}

// Delete remove uma entidade
func (s *BaseEntityService) Delete(ctx context.Context, keys map[string]any) error {
	// Constrói a query SQL
	query, args, err := s.provider.BuildDeleteQuery(s.metadata, keys)
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	// Executa a query
	result, err := s.executeExec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("failed to execute delete: %w", err)
	}

	// Verifica se a exclusão foi bem-sucedida
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows deleted")
	}

	return nil
}

// buildKeyFilter constrói um filtro baseado nas chaves (método legado, considerar usar BuildTypedKeyFilter)
func (s *BaseEntityService) buildKeyFilter(keys map[string]any) string {
	log.Printf("🔍 buildKeyFilter - Starting with keys: %+v", keys)

	var filters []string
	for key, value := range keys {
		log.Printf("🔍 buildKeyFilter - Processing key '%s': value=%v, type=%T", key, value, value)

		switch v := value.(type) {
		case string:
			filter := fmt.Sprintf("%s eq '%s'", key, v)
			log.Printf("🔍 buildKeyFilter - String filter: %s", filter)
			filters = append(filters, filter)
		case int, int32, int64:
			filter := fmt.Sprintf("%s eq %v", key, v)
			log.Printf("🔍 buildKeyFilter - Numeric filter: %s", filter)
			filters = append(filters, filter)
		default:
			filter := fmt.Sprintf("%s eq '%v'", key, v)
			log.Printf("🔍 buildKeyFilter - Default filter: %s", filter)
			filters = append(filters, filter)
		}
	}

	result := strings.Join(filters, " and ")
	log.Printf("🔍 buildKeyFilter - Final filter: %s", result)
	return result
}

// hasAutoIncrementKey verifica se há chaves auto-incrementais
func (s *BaseEntityService) hasAutoIncrementKey() bool {
	for _, prop := range s.metadata.Properties {
		if prop.IsKey && (prop.Type == "int" || prop.Type == "int32" || prop.Type == "int64") {
			return true
		}
	}
	return false
}

// getAutoIncrementKey retorna a primeira chave auto-incremental
func (s *BaseEntityService) getAutoIncrementKey() *PropertyMetadata {
	for _, prop := range s.metadata.Properties {
		if prop.IsKey && (prop.Type == "int" || prop.Type == "int32" || prop.Type == "int64") {
			return &prop
		}
	}
	return nil
}

// processComputeOption processa e valida uma opção $compute
func (s *BaseEntityService) processComputeOption(ctx context.Context, computeOption *ComputeOption) error {
	if computeOption == nil {
		return nil
	}

	// Valida cada expressão de compute
	for _, expr := range computeOption.Expressions {
		err := s.computeParser.ValidateComputeExpression(expr, s.metadata)
		if err != nil {
			return fmt.Errorf("invalid compute expression '%s': %w", expr.Expression, err)
		}
	}

	return nil
}

// processSearchOption processa e valida uma opção $search
func (s *BaseEntityService) processSearchOption(ctx context.Context, searchOption *SearchOption) error {
	if searchOption == nil || searchOption.Expression == nil {
		return nil
	}

	// Valida a expressão de search
	err := s.searchParser.ValidateSearchExpression(searchOption.Expression)
	if err != nil {
		return fmt.Errorf("invalid search expression: %w", err)
	}

	// Verifica se há propriedades pesquisáveis
	searchableProps := s.searchParser.GetSearchableProperties(s.metadata)
	if len(searchableProps) == 0 {
		return fmt.Errorf("no searchable properties found in entity %s", s.metadata.Name)
	}

	return nil
}

// ParseComputeQuery analisa uma string $compute e retorna ComputeOption
func (s *BaseEntityService) ParseComputeQuery(ctx context.Context, computeStr string) (*ComputeOption, error) {
	if computeStr == "" {
		return nil, nil
	}

	return s.computeParser.ParseCompute(ctx, computeStr)
}

// ParseSearchQuery analisa uma string $search e retorna SearchOption
func (s *BaseEntityService) ParseSearchQuery(ctx context.Context, searchStr string) (*SearchOption, error) {
	if searchStr == "" {
		return nil, nil
	}

	return s.searchParser.ParseSearch(ctx, searchStr)
}

// GetComputeFields retorna os campos computados como metadados
func (s *BaseEntityService) GetComputeFields(computeOption *ComputeOption) []PropertyMetadata {
	if computeOption == nil {
		return []PropertyMetadata{}
	}

	return s.computeParser.GetComputeFields(computeOption)
}

// GetSearchableProperties retorna as propriedades pesquisáveis da entidade
func (s *BaseEntityService) GetSearchableProperties() []PropertyMetadata {
	return s.searchParser.GetSearchableProperties(s.metadata)
}
