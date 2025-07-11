package odata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
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
	var args []interface{}
	var err error

	// Aplica $filter, $orderby, $skip/$top primeiro na query SQL
	if optimizedProvider, ok := s.provider.(interface {
		BuildSelectQueryOptimized(ctx context.Context, metadata EntityMetadata, options QueryOptions) (string, []interface{}, error)
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
func (s *BaseEntityService) Get(ctx context.Context, keys map[string]interface{}) (interface{}, error) {
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
		// Top: removido para evitar subquery desnecessária no Oracle
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
func (s *BaseEntityService) buildTypedPropertyFilter(ctx context.Context, propertyName string, propertyValue interface{}) (*GoDataFilterQuery, error) {
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
func (s *BaseEntityService) BuildTypedKeyFilter(ctx context.Context, keys map[string]interface{}) (*GoDataFilterQuery, error) {
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
func (s *BaseEntityService) getTokenTypeForValue(value interface{}) int {
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
func (s *BaseEntityService) Create(ctx context.Context, entity interface{}) (interface{}, error) {
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
		keys := map[string]interface{}{
			keyProp.Name: lastID,
		}

		return s.Get(ctx, keys)
	}

	return entity, nil
}

// Update atualiza uma entidade existente
func (s *BaseEntityService) Update(ctx context.Context, keys map[string]interface{}, entity interface{}) (interface{}, error) {
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
func (s *BaseEntityService) Delete(ctx context.Context, keys map[string]interface{}) error {
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

// scanRows converte os resultados SQL para maps
func (s *BaseEntityService) scanRows(rows *sql.Rows, expandOptions []ExpandOption) ([]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []interface{}

	for rows.Next() {
		// Cria um slice de interfaces para os valores
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

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
						// CORREÇÃO: Usa convertValueToPropertyType para manter o tipo correto
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
					// CORREÇÃO: Também aplica conversão de tipo para colunas adicionais
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

// GetCount obtém o total de registros
func (s *BaseEntityService) GetCount(ctx context.Context, options QueryOptions) (int64, error) {
	// Constrói a query de count usando o provider
	tableName := s.metadata.TableName
	if tableName == "" {
		tableName = s.metadata.Name
	}

	// Usa o provider para construir a cláusula WHERE corretamente
	var whereClause string
	var args []interface{}
	var err error

	if options.Filter != nil && options.Filter.Tree != nil {
		whereClause, args, err = ConvertFilterToSQL(ctx, options.Filter, s.metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to build where clause for count: %w", err)
		}
	}

	// Constrói a query COUNT com o provider
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	// Executa a query
	row := s.provider.GetConnection().QueryRowContext(ctx, query, args...)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return count, nil
}

// buildKeyFilter constrói um filtro baseado nas chaves
func (s *BaseEntityService) buildKeyFilter(keys map[string]interface{}) string {
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

// entityToMap converte uma entidade para map
func (s *BaseEntityService) entityToMap(entity interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Se já é um map, retorna diretamente
	if m, ok := entity.(map[string]interface{}); ok {
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

// processExpandedNavigation processa as navegações expandidas
func (s *BaseEntityService) processExpandedNavigation(ctx context.Context, results []interface{}, expandOptions []ExpandOption) ([]interface{}, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Para cada resultado, processa as navegações
	for i, result := range results {
		orderedEntity, ok := result.(*OrderedEntity)
		if !ok {
			continue
		}

		// Processa cada opção de expansão
		for _, expandOption := range expandOptions {
			expandedResult, err := s.expandNavigationProperty(ctx, orderedEntity, expandOption)
			if err != nil {
				// Log detalhado do erro para debug
				errorMsg := fmt.Sprintf("%v", err)

				// Log do erro mas tenta continuar processando outras propriedades
				log.Printf("Warning: Failed to expand navigation property %s: %v. Property will remain as navigation link.", expandOption.Property, err)

				// Se o erro for crítico de estrutura (não de conexão), falha
				if strings.Contains(errorMsg, "navigation property") && strings.Contains(errorMsg, "not found") {
					// Erro de estrutura - propriedade não existe, isso é crítico
					return nil, fmt.Errorf("critical error expanding navigation property %s: %w", expandOption.Property, err)
				}

				// Para outros erros (incluindo conexão), continua com navigation link
				continue
			}

			results[i] = expandedResult
		}
	}

	return results, nil
}

// expandNavigationProperty expande uma propriedade de navegação específica
func (s *BaseEntityService) expandNavigationProperty(ctx context.Context, entity *OrderedEntity, expandOption ExpandOption) (*OrderedEntity, error) {
	// Encontra a propriedade de navegação nos metadados (comparação case-insensitive)
	var navProperty *PropertyMetadata
	for _, prop := range s.metadata.Properties {
		if strings.EqualFold(prop.Name, expandOption.Property) && prop.IsNavigation {
			navProperty = &prop
			break
		}
	}

	if navProperty == nil {
		// Debug: adiciona informações sobre as propriedades disponíveis
		var availableProps []string
		for _, prop := range s.metadata.Properties {
			if prop.IsNavigation {
				availableProps = append(availableProps, prop.Name)
			}
		}
		return entity, fmt.Errorf("navigation property %s not found. Available navigation properties: %v", expandOption.Property, availableProps)
	}

	// Obtém o valor da chave para fazer a busca relacionada
	if navProperty.Relationship == nil {
		return entity, fmt.Errorf("navigation property %s has no relationship metadata", expandOption.Property)
	}

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
			// Debug: mostra quais propriedades estão disponíveis na entidade
			var availableEntityProps []string
			for _, prop := range entity.Properties {
				availableEntityProps = append(availableEntityProps, prop.Name)
			}
			return entity, fmt.Errorf("local key property %s not found in entity. Available entity properties: %v", navProperty.Relationship.LocalProperty, availableEntityProps)
		}
	}

	// Busca entidades relacionadas
	relatedEntities, err := s.findRelatedEntities(ctx, navProperty, localKeyValue, expandOption)
	if err != nil {
		return entity, fmt.Errorf("failed to find related entities for property %s: %w", expandOption.Property, err)
	}

	// Adiciona as entidades relacionadas ao resultado
	if navProperty.IsCollection {
		// Para collections, usa a lista retornada (pode ser vazia se nenhuma entidade passar no filtro)
		if relatedEntities == nil {
			entity.Set(navProperty.Name, []interface{}{})
		} else {
			entity.Set(navProperty.Name, relatedEntities)
		}
	} else {
		// Para propriedades únicas, verifica se há resultado
		if relatedEntities == nil {
			// findRelatedEntities retornou nil - entidade não passa no filtro do expand
			entity.Set(navProperty.Name, nil)
		} else if len(relatedEntities) > 0 {
			// Entidade encontrada e passa no filtro
			entity.Set(navProperty.Name, relatedEntities[0])
		} else {
			// Nenhuma entidade encontrada
			entity.Set(navProperty.Name, nil)
		}
	}

	return entity, nil
}

// findRelatedEntities encontra entidades relacionadas baseado na propriedade de navegação
func (s *BaseEntityService) findRelatedEntities(ctx context.Context, navProperty *PropertyMetadata, keyValue interface{}, expandOption ExpandOption) ([]interface{}, error) {
	if navProperty.Relationship == nil {
		return nil, fmt.Errorf("navigation property has no relationship metadata")
	}

	// Primeiro, obtém os metadados da entidade relacionada para determinar o nome correto das propriedades
	relatedMetadata, err := s.getRelatedEntityMetadata(navProperty.RelatedType)
	if err != nil {
		return nil, fmt.Errorf("failed to get related entity metadata: %w", err)
	}

	// Constrói filtro para buscar entidades relacionadas
	// A lógica é diferente para association vs manyAssociation:
	//
	// - association (N:1): A chave estrangeira está na entidade atual
	//   Ex: FabTarefa -> FabOperacao
	//   Filtro: <chave_primaria_FabOperacao> eq <valor_de_ID_OPERACAO>
	//
	// - manyAssociation (1:N): A chave estrangeira está na entidade relacionada
	//   Ex: FabOperacao -> FabTarefa
	//   Filtro: <chave_estrangeira_FabTarefa> eq <valor_de_ID>
	var filterProperty string
	var filterValue interface{}

	if navProperty.IsCollection {
		// manyAssociation: filtro pela chave estrangeira na entidade relacionada
		// Usa o nome da propriedade conforme definido no relacionamento
		filterProperty = navProperty.Relationship.ReferencedProperty
		filterValue = keyValue
	} else {
		// association: filtro pela chave primária na entidade relacionada
		// Busca a chave primária real na entidade relacionada
		var primaryKeyProperty string
		for _, prop := range relatedMetadata.Properties {
			if prop.IsKey {
				primaryKeyProperty = prop.Name
				break
			}
		}
		if primaryKeyProperty == "" {
			return nil, fmt.Errorf("no primary key found in related entity %s", navProperty.RelatedType)
		}
		filterProperty = primaryKeyProperty
		filterValue = keyValue
	}

	// Converte o valor para o tipo correto baseado nos metadados da propriedade
	convertedValue, err := s.convertValueToPropertyType(filterValue, filterProperty, relatedMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to convert filter value: %w", err)
	}

	// CORREÇÃO: Cria um filtro tipado preservando os tipos das propriedades
	// similar ao buildTypedKeyFilter usado no método Get
	filterQuery, err := s.buildTypedPropertyFilter(ctx, filterProperty, convertedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to build typed property filter: %w", err)
	}

	// Converte expandOption.Count para GoDataCountQuery
	var countQuery *GoDataCountQuery
	if expandOption.Count {
		countQuery = SetCountValue(true)
	}

	// Converte expandOption.Select para GoDataSelectQuery
	var selectQuery *GoDataSelectQuery
	if len(expandOption.Select) > 0 {
		selectStr := strings.Join(expandOption.Select, ",")
		var err error
		selectQuery, err = ParseSelectString(ctx, selectStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse select: %w", err)
		}
	}

	// Converte expandOption.Expand para GoDataExpandQuery
	var expandQuery *GoDataExpandQuery
	if len(expandOption.Expand) > 0 {
		// Constrói string de expand recursiva a partir das opções
		var expandParts []string
		for _, exp := range expandOption.Expand {
			expandPart := exp.Property

			// Adiciona filtros e outros parâmetros se presentes
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

			// Adiciona expand recursivo se presente - RECURSIVO COMPLETO
			if len(exp.Expand) > 0 {
				var nestedExpands []string
				for _, nestedExp := range exp.Expand {
					nestedExpandPart := nestedExp.Property

					// Adiciona filtros e outros parâmetros nos expand aninhados se presentes
					var nestedSubOptions []string
					if nestedExp.Filter != "" {
						nestedSubOptions = append(nestedSubOptions, fmt.Sprintf("$filter=%s", nestedExp.Filter))
					}
					if nestedExp.OrderBy != "" {
						nestedSubOptions = append(nestedSubOptions, fmt.Sprintf("$orderby=%s", nestedExp.OrderBy))
					}
					if len(nestedExp.Select) > 0 {
						nestedSubOptions = append(nestedSubOptions, fmt.Sprintf("$select=%s", strings.Join(nestedExp.Select, ",")))
					}
					if nestedExp.Skip > 0 {
						nestedSubOptions = append(nestedSubOptions, fmt.Sprintf("$skip=%d", nestedExp.Skip))
					}
					if nestedExp.Top > 0 {
						nestedSubOptions = append(nestedSubOptions, fmt.Sprintf("$top=%d", nestedExp.Top))
					}

					// Combina propriedade com suas opções aninhadas
					if len(nestedSubOptions) > 0 {
						nestedExpandPart = fmt.Sprintf("%s(%s)", nestedExpandPart, strings.Join(nestedSubOptions, ";"))
					}

					nestedExpands = append(nestedExpands, nestedExpandPart)
				}
				subOptions = append(subOptions, fmt.Sprintf("$expand=%s", strings.Join(nestedExpands, ",")))
			}

			// Combina propriedade com suas opções
			if len(subOptions) > 0 {
				expandPart = fmt.Sprintf("%s(%s)", expandPart, strings.Join(subOptions, ";"))
			}

			expandParts = append(expandParts, expandPart)
		}
		expandStr := strings.Join(expandParts, ",")
		var err error
		expandQuery, err = ParseExpandString(ctx, expandStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expand: %w", err)
		}
	}

	// Converte expandOption.Skip para GoDataSkipQuery
	var skipQuery *GoDataSkipQuery
	if expandOption.Skip > 0 {
		skip := GoDataSkipQuery(expandOption.Skip)
		skipQuery = &skip
	}

	// Converte expandOption.Top para GoDataTopQuery
	var topQuery *GoDataTopQuery
	if expandOption.Top > 0 {
		top := GoDataTopQuery(expandOption.Top)
		topQuery = &top
	}

	// Primeiro, busca as entidades relacionadas SEM o filtro do expand
	// O filtro do expand será aplicado posteriormente para determinar quais expandir
	options := QueryOptions{
		Filter:  filterQuery, // Apenas o filtro de relacionamento
		OrderBy: expandOption.OrderBy,
		Select:  selectQuery,
		Expand:  expandQuery,
		Skip:    skipQuery,
		Top:     topQuery,
		Count:   countQuery,
	}

	// Constrói query SQL para buscar entidades relacionadas com proteção de timeout
	query, args, err := s.buildQueryWithTimeoutProtection(ctx, relatedMetadata, options)
	if err != nil {
		return nil, fmt.Errorf("failed to build select query for related entities: %w", err)
	}

	// Verifica se o contexto já foi cancelado antes de executar
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before query execution: %w", ctx.Err())
	default:
	}

	rows, err := s.executeQuery(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query for related entities: %w", err)
	}
	defer rows.Close()

	// Converte os resultados usando os metadados da entidade relacionada
	relatedService := NewBaseEntityService(s.provider, relatedMetadata, s.server)
	allResults, err := relatedService.scanRows(rows, expandOption.Expand)
	if err != nil {
		return nil, fmt.Errorf("failed to scan related entity rows: %w", err)
	}

	// Se não há filtro do expand, processa todas as entidades normalmente
	if expandOption.Filter == "" {
		// Processa expansões recursivas se houver
		if len(expandOption.Expand) > 0 {
			allResults, err = relatedService.processExpandedNavigation(ctx, allResults, expandOption.Expand)
			if err != nil {
				return nil, fmt.Errorf("failed to process recursive expanded navigation: %w", err)
			}
		}
		return allResults, nil
	}

	// Parse o filtro do expand
	expandFilterQuery, err := s.parseFilterWithTimeout(ctx, expandOption.Filter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expand filter: %w", err)
	}

	// Aplica o filtro do expand de forma diferente dependendo se é collection ou não
	if navProperty.IsCollection {
		// Para collections (1:N), filtra apenas as entidades que passam no filtro
		var filteredResults []interface{}
		for _, result := range allResults {
			if s.entityMatchesFilter(result, expandFilterQuery, relatedMetadata) {
				filteredResults = append(filteredResults, result)
			}
		}

		// Processa expansões recursivas apenas nas entidades que passaram no filtro
		if len(expandOption.Expand) > 0 && len(filteredResults) > 0 {
			filteredResults, err = relatedService.processExpandedNavigation(ctx, filteredResults, expandOption.Expand)
			if err != nil {
				return nil, fmt.Errorf("failed to process recursive expanded navigation: %w", err)
			}
		}

		return filteredResults, nil
	} else {
		// Para propriedades únicas (N:1), verifica se a entidade encontrada passa no filtro
		if len(allResults) == 0 {
			return nil, nil // Não há entidade relacionada
		}

		// Verifica se a entidade encontrada passa no filtro do expand
		entity := allResults[0]
		if s.entityMatchesFilter(entity, expandFilterQuery, relatedMetadata) {
			// Entidade passa no filtro - processa expansões recursivas
			if len(expandOption.Expand) > 0 {
				expandedResults, err := relatedService.processExpandedNavigation(ctx, []interface{}{entity}, expandOption.Expand)
				if err != nil {
					return nil, fmt.Errorf("failed to process recursive expanded navigation: %w", err)
				}
				return expandedResults, nil
			}
			return []interface{}{entity}, nil
		} else {
			// Entidade NÃO passa no filtro - retorna nil para indicar que deve ser null
			return nil, nil
		}
	}
}

// convertValueToPropertyType converte o valor para o tipo correto baseado nos metadados da propriedade
func (s *BaseEntityService) convertValueToPropertyType(value interface{}, propertyName string, metadata EntityMetadata) (interface{}, error) {
	// Se o valor é nil, retorna nil
	if value == nil {
		return nil, nil
	}

	// Encontra a propriedade nos metadados
	for _, prop := range metadata.Properties {
		if strings.EqualFold(prop.Name, propertyName) {
			// Converte o valor para o tipo correto
			switch prop.Type {
			case "int64":
				switch v := value.(type) {
				case int64:
					return v, nil
				case int:
					return int64(v), nil
				case int32:
					return int64(v), nil
				case string:
					parsed, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as int64: %w", v, err)
					}
					return parsed, nil
				case float64:
					return int64(v), nil
				case float32:
					return int64(v), nil
				case []byte:
					str := string(v)
					parsed, err := strconv.ParseInt(str, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse []byte %s as int64: %w", str, err)
					}
					return parsed, nil
				default:
					return nil, fmt.Errorf("cannot convert %T to int64", value)
				}
			case "int32", "int":
				switch v := value.(type) {
				case int32:
					return v, nil
				case int:
					return int32(v), nil
				case int64:
					return int32(v), nil
				case string:
					parsed, err := strconv.ParseInt(v, 10, 32)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as int32: %w", v, err)
					}
					return int32(parsed), nil
				case float64:
					return int32(v), nil
				case float32:
					return int32(v), nil
				case []byte:
					str := string(v)
					parsed, err := strconv.ParseInt(str, 10, 32)
					if err != nil {
						return nil, fmt.Errorf("failed to parse []byte %s as int32: %w", str, err)
					}
					return int32(parsed), nil
				default:
					return nil, fmt.Errorf("cannot convert %T to int32", value)
				}
			case "float64", "double":
				switch v := value.(type) {
				case float64:
					return v, nil
				case float32:
					return float64(v), nil
				case int:
					return float64(v), nil
				case int32:
					return float64(v), nil
				case int64:
					return float64(v), nil
				case string:
					parsed, err := strconv.ParseFloat(v, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as float64: %w", v, err)
					}
					return parsed, nil
				case []byte:
					str := string(v)
					parsed, err := strconv.ParseFloat(str, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse []byte %s as float64: %w", str, err)
					}
					return parsed, nil
				default:
					return nil, fmt.Errorf("cannot convert %T to float64", value)
				}
			case "float32", "single":
				switch v := value.(type) {
				case float32:
					return v, nil
				case float64:
					return float32(v), nil
				case int:
					return float32(v), nil
				case int32:
					return float32(v), nil
				case int64:
					return float32(v), nil
				case string:
					parsed, err := strconv.ParseFloat(v, 32)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as float32: %w", v, err)
					}
					return float32(parsed), nil
				case []byte:
					str := string(v)
					parsed, err := strconv.ParseFloat(str, 32)
					if err != nil {
						return nil, fmt.Errorf("failed to parse []byte %s as float32: %w", str, err)
					}
					return float32(parsed), nil
				default:
					return nil, fmt.Errorf("cannot convert %T to float32", value)
				}
			case "string":
				switch v := value.(type) {
				case string:
					return v, nil
				case []byte:
					return string(v), nil
				default:
					return fmt.Sprintf("%v", value), nil
				}
			case "bool", "boolean":
				switch v := value.(type) {
				case bool:
					return v, nil
				case string:
					parsed, err := strconv.ParseBool(v)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as bool: %w", v, err)
					}
					return parsed, nil
				case []byte:
					str := string(v)
					parsed, err := strconv.ParseBool(str)
					if err != nil {
						return nil, fmt.Errorf("failed to parse []byte %s as bool: %w", str, err)
					}
					return parsed, nil
				case int:
					return v != 0, nil
				case int32:
					return v != 0, nil
				case int64:
					return v != 0, nil
				default:
					return nil, fmt.Errorf("cannot convert %T to bool", value)
				}
			case "[]byte", "binary":
				switch v := value.(type) {
				case []byte:
					return v, nil
				case string:
					return []byte(v), nil
				default:
					return nil, fmt.Errorf("cannot convert %T to []byte", value)
				}
			default:
				// Para tipos não mapeados ou personalizados, aplica conversão básica
				switch v := value.(type) {
				case []byte:
					// Por padrão, converte []byte para string se não for tipo binário
					return string(v), nil
				default:
					return value, nil
				}
			}
		}
	}

	// Se não encontrar a propriedade, aplica conversão básica
	switch v := value.(type) {
	case []byte:
		return string(v), nil
	default:
		return value, nil
	}
}

// getRelatedEntityMetadata obtém metadados da entidade relacionada
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

// buildNavigationLink constrói um navigation link para uma propriedade de navegação
func (s *BaseEntityService) buildNavigationLink(prop PropertyMetadata, entity *OrderedEntity) string {
	// Constrói o URL do navigation link no formato OData
	// Ex: "FabTarefa(53)/FabOperacao"

	// Obtém as chaves primárias da entidade atual
	var keyValues []interface{}

	// Busca as chaves primárias de forma mais robusta
	for _, metaProp := range s.metadata.Properties {
		if metaProp.IsKey {
			var found bool
			var keyValue interface{}

			// Tenta várias formas de encontrar a chave
			searchNames := []string{
				metaProp.Name,                        // Nome original da propriedade
				strings.ToUpper(metaProp.Name),       // Nome em maiúsculas
				strings.ToLower(metaProp.Name),       // Nome em minúsculas
				metaProp.ColumnName,                  // Nome da coluna do banco
				strings.ToUpper(metaProp.ColumnName), // Nome da coluna em maiúsculas
			}

			// Remove duplicatas e entradas vazias
			uniqueNames := make(map[string]bool)
			var finalNames []string
			for _, name := range searchNames {
				if name != "" && !uniqueNames[name] {
					uniqueNames[name] = true
					finalNames = append(finalNames, name)
				}
			}

			// Busca exata primeiro
			for _, searchName := range finalNames {
				if value, exists := entity.Get(searchName); exists {
					keyValue = value
					found = true
					break
				}
			}

			// Se não encontrou com busca exata, tenta case-insensitive
			if !found {
				for _, entityProp := range entity.Properties {
					for _, searchName := range finalNames {
						if strings.EqualFold(entityProp.Name, searchName) {
							keyValue = entityProp.Value
							found = true
							break
						}
					}
					if found {
						break
					}
				}
			}

			if found {
				keyValues = append(keyValues, keyValue)
			}
		}
	}

	if len(keyValues) == 0 {
		return ""
	}

	// Constrói a parte da chave: para chave simples "53", para chave composta "53,2"
	var keyPart string
	if len(keyValues) == 1 {
		keyPart = fmt.Sprintf("%v", keyValues[0])
	} else {
		var keyStrings []string
		for _, key := range keyValues {
			keyStrings = append(keyStrings, fmt.Sprintf("%v", key))
		}
		keyPart = strings.Join(keyStrings, ",")
	}

	// Constrói o link: EntitySet(key)/NavigationProperty
	return fmt.Sprintf("%s(%s)/%s", s.metadata.Name, keyPart, prop.Name)
}

// getJSONTagName extrai o nome do tag JSON de uma propriedade
func getJSONTagName(propertyName string) string {
	// Converte de PascalCase para camelCase
	if len(propertyName) == 0 {
		return propertyName
	}

	// Se o primeiro caractere é maiúsculo, converte para minúsculo
	if propertyName[0] >= 'A' && propertyName[0] <= 'Z' {
		return strings.ToLower(string(propertyName[0])) + propertyName[1:]
	}

	return propertyName
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

// formatFilterValue retorna o valor formatado de acordo com o tipo
func (s *BaseEntityService) formatFilterValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

// getArgsTypes retorna os tipos dos argumentos para debug
func (s *BaseEntityService) getArgsTypes(args []interface{}) []string {
	var types []string
	for _, arg := range args {
		types = append(types, fmt.Sprintf("%T", arg))
	}
	return types
}

// createSanitizedFilter cria um filtro sanitizado para evitar caracteres inválidos
func (s *BaseEntityService) createSanitizedFilter(property string, value interface{}) string {
	// Sanitiza o nome da propriedade removendo caracteres inválidos
	sanitizedProperty := strings.ReplaceAll(property, "'", "''") // Escape de aspas simples

	// Formata o valor de forma segura
	formattedValue := s.formatFilterValueSafe(value)

	return fmt.Sprintf("%s eq %s", sanitizedProperty, formattedValue)
}

// parseFilterWithTimeout faz parse do filter com timeout apropriado
func (s *BaseEntityService) parseFilterWithTimeout(ctx context.Context, filter string) (*GoDataFilterQuery, error) {
	return ParseFilterString(ctx, filter)
}

// buildQueryWithTimeoutProtection constrói query com proteção contra timeout
func (s *BaseEntityService) buildQueryWithTimeoutProtection(ctx context.Context, metadata EntityMetadata, options QueryOptions) (string, []interface{}, error) {
	// Verifica se o provider suporta query otimizada com contexto
	if optimizedProvider, ok := s.provider.(interface {
		BuildSelectQueryOptimized(ctx context.Context, metadata EntityMetadata, options QueryOptions) (string, []interface{}, error)
	}); ok {
		return optimizedProvider.BuildSelectQueryOptimized(ctx, metadata, options)
	}

	// Fallback para método padrão
	return s.provider.BuildSelectQuery(metadata, options)
}

// formatFilterValueSafe retorna o valor formatado de forma segura para Oracle
func (s *BaseEntityService) formatFilterValueSafe(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape de aspas simples e remoção de caracteres de controle
		escaped := strings.ReplaceAll(v, "'", "''")
		// Remove caracteres de controle que podem causar ORA-00911
		sanitized := strings.Map(func(r rune) rune {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' {
				return -1 // Remove caractere
			}
			return r
		}, escaped)
		return fmt.Sprintf("'%s'", sanitized)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%v", v)
	default:
		// Para outros tipos, converte para string e sanitiza
		str := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(str, "'", "''")
		sanitized := strings.Map(func(r rune) rune {
			if r < 32 && r != '\t' && r != '\n' && r != '\r' {
				return -1
			}
			return r
		}, escaped)
		return fmt.Sprintf("'%s'", sanitized)
	}
}

// executeQuery executa uma query com os argumentos apropriados para o provider
func (s *BaseEntityService) executeQuery(ctx context.Context, query string, args []interface{}) (*sql.Rows, error) {
	log.Printf("🔍 executeQuery - Starting execution")
	log.Printf("🔍 executeQuery - Query: %s", query)
	log.Printf("🔍 executeQuery - Args: %+v", args)
	log.Printf("🔍 executeQuery - Provider driver: %s", s.provider.GetDriverName())

	// Verifica se a conexão está disponível
	conn := s.provider.GetConnection()
	if conn == nil {
		log.Printf("❌ executeQuery - Database connection is nil")
		return nil, fmt.Errorf("database connection is nil - make sure the provider is properly connected")
	}

	log.Printf("✅ executeQuery - Database connection OK")

	// O driver Oracle irá reconhecer automaticamente os argumentos nomeados
	for i, arg := range args {
		log.Printf("🔍 executeQuery - Arg[%d]: value=%v, type=%T", i, arg, arg)
	}

	log.Printf("🔍 executeQuery - Using sql.Named arguments")

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("❌ executeQuery - Error with sql.Named arguments: %v", err)
		return nil, err
	}
	log.Printf("✅ executeQuery - sql.Named arguments query executed successfully")
	return rows, nil
}

// executeExec executa um comando com os argumentos apropriados para o provider
func (s *BaseEntityService) executeExec(ctx context.Context, query string, args []interface{}) (sql.Result, error) {
	// Verifica se a conexão está disponível
	conn := s.provider.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("database connection is nil - make sure the provider is properly connected")
	}

	return conn.ExecContext(ctx, query, args...)
}

// entityMatchesFilter verifica se uma entidade atende ao filtro especificado
func (s *BaseEntityService) entityMatchesFilter(entity interface{}, filter *GoDataFilterQuery, metadata EntityMetadata) bool {
	if filter == nil || filter.Tree == nil {
		return true
	}

	// Converte a entidade para OrderedEntity se necessário
	var orderedEntity *OrderedEntity
	if oe, ok := entity.(*OrderedEntity); ok {
		orderedEntity = oe
	} else {
		// Se não é OrderedEntity, tenta converter
		return false
	}

	// Avalia o filtro recursivamente
	return s.evaluateFilterNode(orderedEntity, filter.Tree, metadata)
}

// evaluateFilterNode avalia um nó do filtro recursivamente
func (s *BaseEntityService) evaluateFilterNode(entity *OrderedEntity, node *ParseNode, metadata EntityMetadata) bool {
	if node == nil {
		return true
	}

	switch node.Token.Type {
	case int(FilterTokenLogical):
		// Operadores lógicos: and, or, not
		switch node.Token.Value {
		case "and":
			if len(node.Children) != 2 {
				return false
			}
			return s.evaluateFilterNode(entity, node.Children[0], metadata) && s.evaluateFilterNode(entity, node.Children[1], metadata)
		case "or":
			if len(node.Children) != 2 {
				return false
			}
			return s.evaluateFilterNode(entity, node.Children[0], metadata) || s.evaluateFilterNode(entity, node.Children[1], metadata)
		case "not":
			if len(node.Children) != 1 {
				return false
			}
			return !s.evaluateFilterNode(entity, node.Children[0], metadata)
		}
	case int(FilterTokenComparison):
		// Operadores de comparação: eq, ne, gt, lt, ge, le
		if len(node.Children) != 2 {
			return false
		}

		leftValue := s.evaluateFilterValue(entity, node.Children[0], metadata)
		rightValue := s.evaluateFilterValue(entity, node.Children[1], metadata)

		return s.compareValues(leftValue, rightValue, node.Token.Value)
	}

	return false
}

// evaluateFilterValue avalia um valor no filtro (propriedade ou literal)
func (s *BaseEntityService) evaluateFilterValue(entity *OrderedEntity, node *ParseNode, metadata EntityMetadata) interface{} {
	if node == nil {
		return nil
	}

	switch node.Token.Type {
	case int(FilterTokenString), int(FilterTokenNumber), int(FilterTokenBoolean), int(FilterTokenNull):
		// Valor literal (string, número, booleano)
		return s.parseFilterLiteral(node.Token.Value)
	case int(FilterTokenProperty):
		// Nome de propriedade
		propertyName := node.Token.Value

		// Busca o valor na entidade
		if value, exists := entity.Get(propertyName); exists {
			return value
		}

		// Se não encontrou, tenta busca case-insensitive
		for _, prop := range entity.Properties {
			if strings.EqualFold(prop.Name, propertyName) {
				return prop.Value
			}
		}

		return nil
	}

	return nil
}

// parseFilterLiteral converte um literal string para o tipo apropriado
func (s *BaseEntityService) parseFilterLiteral(literal string) interface{} {
	// Remove aspas se for string
	if len(literal) >= 2 && literal[0] == '\'' && literal[len(literal)-1] == '\'' {
		return literal[1 : len(literal)-1]
	}

	// Tenta converter para número
	if intVal, err := strconv.ParseInt(literal, 10, 64); err == nil {
		return intVal
	}

	// Tenta converter para float
	if floatVal, err := strconv.ParseFloat(literal, 64); err == nil {
		return floatVal
	}

	// Tenta converter para booleano
	if boolVal, err := strconv.ParseBool(literal); err == nil {
		return boolVal
	}

	// Retorna como string se não conseguir converter
	return literal
}

// compareValues compara dois valores usando o operador especificado
func (s *BaseEntityService) compareValues(left, right interface{}, operator string) bool {
	// Converte para string para comparação
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch operator {
	case "eq":
		return leftStr == rightStr
	case "ne":
		return leftStr != rightStr
	case "gt":
		return leftStr > rightStr
	case "lt":
		return leftStr < rightStr
	case "ge":
		return leftStr >= rightStr
	case "le":
		return leftStr <= rightStr
	}

	return false
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

// applyComputeToResults aplica campos computados aos resultados seguindo a ordem OData v4
func (s *BaseEntityService) applyComputeToResults(ctx context.Context, results []interface{}, computeOption *ComputeOption) ([]interface{}, error) {
	if computeOption == nil || len(computeOption.Expressions) == 0 {
		return results, nil
	}

	// Para cada resultado, calcula os campos computados
	for i, result := range results {
		orderedEntity, ok := result.(*OrderedEntity)
		if !ok {
			continue
		}

		// Calcula cada expressão computada
		for _, expr := range computeOption.Expressions {
			computedValue, err := s.evaluateComputeExpression(ctx, expr, orderedEntity)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate compute expression '%s': %w", expr.Expression, err)
			}

			// Adiciona o campo computado ao resultado
			orderedEntity.Set(expr.Alias, computedValue)
		}

		results[i] = orderedEntity
	}

	return results, nil
}

// evaluateComputeExpression avalia uma expressão computada
func (s *BaseEntityService) evaluateComputeExpression(ctx context.Context, expr ComputeExpression, entity *OrderedEntity) (interface{}, error) {
	if expr.ParseTree == nil {
		return nil, fmt.Errorf("compute expression has no parse tree")
	}

	return s.evaluateComputeNode(ctx, expr.ParseTree, entity)
}

// evaluateComputeNode avalia um nó da árvore de compute
func (s *BaseEntityService) evaluateComputeNode(ctx context.Context, node *ParseNode, entity *OrderedEntity) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("compute node is nil")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	switch node.Token.Type {
	case int(FilterTokenProperty):
		// Obtém valor da propriedade
		value, exists := entity.Get(node.Token.Value)
		if !exists {
			return nil, fmt.Errorf("property %s not found", node.Token.Value)
		}
		return value, nil

	case int(FilterTokenString):
		// String literal
		value := node.Token.Value
		if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1] // Remove aspas
		}
		return value, nil

	case int(FilterTokenNumber):
		// Número literal
		return node.Token.Value, nil

	case int(FilterTokenArithmetic):
		// Operador aritmético
		if len(node.Children) != 2 {
			return nil, fmt.Errorf("arithmetic operator requires 2 operands")
		}

		left, err := s.evaluateComputeNode(ctx, node.Children[0], entity)
		if err != nil {
			return nil, err
		}

		right, err := s.evaluateComputeNode(ctx, node.Children[1], entity)
		if err != nil {
			return nil, err
		}

		return s.evaluateArithmeticOperation(node.Token.Value, left, right)

	default:
		return nil, fmt.Errorf("unsupported compute token type: %v", node.Token.Type)
	}
}

// evaluateArithmeticOperation avalia operações aritméticas
func (s *BaseEntityService) evaluateArithmeticOperation(operator string, left, right interface{}) (interface{}, error) {
	// Converte para números
	leftNum, err := s.convertToNumber(left)
	if err != nil {
		return nil, fmt.Errorf("left operand is not a number: %w", err)
	}

	rightNum, err := s.convertToNumber(right)
	if err != nil {
		return nil, fmt.Errorf("right operand is not a number: %w", err)
	}

	switch operator {
	case "add":
		return leftNum + rightNum, nil
	case "sub":
		return leftNum - rightNum, nil
	case "mul":
		return leftNum * rightNum, nil
	case "div":
		if rightNum == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return leftNum / rightNum, nil
	default:
		return nil, fmt.Errorf("unsupported arithmetic operator: %s", operator)
	}
}

// convertToNumber converte um valor para número
func (s *BaseEntityService) convertToNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, nil
		}
		return 0, fmt.Errorf("cannot convert string '%s' to number", v)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// processExpandedNavigationWithOrder processa navegações expandidas seguindo a ordem OData v4
func (s *BaseEntityService) processExpandedNavigationWithOrder(ctx context.Context, results []interface{}, expandOptions []ExpandOption) ([]interface{}, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Para cada resultado, processa as navegações seguindo a ordem
	for i, result := range results {
		orderedEntity, ok := result.(*OrderedEntity)
		if !ok {
			continue
		}

		// Processa cada opção de expansão seguindo a ordem OData v4
		for _, expandOption := range expandOptions {
			expandedResult, err := s.expandNavigationPropertyWithOrder(ctx, orderedEntity, expandOption)
			if err != nil {
				// Log detalhado do erro para debug
				errorMsg := fmt.Sprintf("%v", err)

				// Log do erro mas tenta continuar processando outras propriedades
				log.Printf("Warning: Failed to expand navigation property %s: %v. Property will remain as navigation link.", expandOption.Property, err)

				// Se o erro for crítico de estrutura (não de conexão), falha
				if strings.Contains(errorMsg, "navigation property") && strings.Contains(errorMsg, "not found") {
					// Erro de estrutura - propriedade não existe, isso é crítico
					return nil, fmt.Errorf("critical error expanding navigation property %s: %w", expandOption.Property, err)
				}

				// Para outros erros (incluindo conexão), continua com navigation link
				continue
			}

			results[i] = expandedResult
		}
	}

	return results, nil
}

// expandNavigationPropertyWithOrder expande uma propriedade de navegação seguindo a ordem OData v4
func (s *BaseEntityService) expandNavigationPropertyWithOrder(ctx context.Context, entity *OrderedEntity, expandOption ExpandOption) (*OrderedEntity, error) {
	// Encontra a propriedade de navegação nos metadados (comparação case-insensitive)
	var navProperty *PropertyMetadata
	for _, prop := range s.metadata.Properties {
		if strings.EqualFold(prop.Name, expandOption.Property) && prop.IsNavigation {
			navProperty = &prop
			break
		}
	}

	if navProperty == nil {
		// Debug: adiciona informações sobre as propriedades disponíveis
		var availableProps []string
		for _, prop := range s.metadata.Properties {
			if prop.IsNavigation {
				availableProps = append(availableProps, prop.Name)
			}
		}
		return entity, fmt.Errorf("navigation property %s not found. Available navigation properties: %v", expandOption.Property, availableProps)
	}

	// Busca entidades relacionadas aplicando a ordem OData v4
	relatedEntities, err := s.findRelatedEntitiesWithOrder(ctx, navProperty, entity, expandOption)
	if err != nil {
		return entity, fmt.Errorf("failed to find related entities for property %s: %w", expandOption.Property, err)
	}

	// Adiciona as entidades relacionadas ao resultado
	if navProperty.IsCollection {
		if relatedEntities == nil {
			entity.Set(navProperty.Name, []interface{}{})
		} else {
			entity.Set(navProperty.Name, relatedEntities)
		}
	} else {
		if relatedEntities == nil {
			entity.Set(navProperty.Name, nil)
		} else if len(relatedEntities) > 0 {
			entity.Set(navProperty.Name, relatedEntities[0])
		} else {
			entity.Set(navProperty.Name, nil)
		}
	}

	return entity, nil
}

// findRelatedEntitiesWithOrder encontra entidades relacionadas seguindo a ordem OData v4
func (s *BaseEntityService) findRelatedEntitiesWithOrder(ctx context.Context, navProperty *PropertyMetadata, entity *OrderedEntity, expandOption ExpandOption) ([]interface{}, error) {
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

	// Converte response.Value para []interface{}
	entities, ok := response.Value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response type: %T", response.Value)
	}

	// Filtra resultados pelo relacionamento
	return s.filterRelatedEntities(entities, navProperty, localKeyValue, relatedMetadata)
}

// filterRelatedEntities filtra entidades relacionadas baseado no relacionamento
func (s *BaseEntityService) filterRelatedEntities(entities []interface{}, navProperty *PropertyMetadata, keyValue interface{}, relatedMetadata EntityMetadata) ([]interface{}, error) {
	var filtered []interface{}

	for _, entity := range entities {
		orderedEntity, ok := entity.(*OrderedEntity)
		if !ok {
			continue
		}

		// Obtém o valor da propriedade de relacionamento
		var relationshipValue interface{}
		var exists bool

		if navProperty.IsCollection {
			// Para collections (1:N), verifica a chave estrangeira
			relationshipValue, exists = orderedEntity.Get(navProperty.Relationship.ReferencedProperty)
		} else {
			// Para associações (N:1), verifica a chave primária
			for _, prop := range relatedMetadata.Properties {
				if prop.IsKey {
					relationshipValue, exists = orderedEntity.Get(prop.Name)
					break
				}
			}
		}

		if exists && s.compareValues(relationshipValue, keyValue, "eq") {
			filtered = append(filtered, entity)
		}
	}

	return filtered, nil
}

// applySelectToResults aplica seleção de campos aos resultados
func (s *BaseEntityService) applySelectToResults(results []interface{}, selectQuery *GoDataSelectQuery) ([]interface{}, error) {
	if selectQuery == nil {
		return results, nil
	}

	selectedFields := GetSelectedProperties(selectQuery)
	if len(selectedFields) == 0 {
		return results, nil
	}

	// Para cada resultado, filtra apenas os campos selecionados
	for i, result := range results {
		orderedEntity, ok := result.(*OrderedEntity)
		if !ok {
			continue
		}

		// Cria nova entidade com apenas os campos selecionados
		filteredEntity := NewOrderedEntity()
		for _, field := range selectedFields {
			if value, exists := orderedEntity.Get(field); exists {
				filteredEntity.Set(field, value)
			}
		}

		results[i] = filteredEntity
	}

	return results, nil
}
