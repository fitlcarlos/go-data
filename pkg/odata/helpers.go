package odata

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// =======================================================================================
// QUERY PARSING & EXECUTION
// =======================================================================================

// getEntityCount obtém a contagem de entidades com base nas opções de consulta
func (s *Server) getEntityCount(ctx context.Context, service EntityService, options QueryOptions) (int64, error) {
	// Cria novas opções apenas com filtro para contagem
	countOptions := QueryOptions{
		Filter: options.Filter,
		Search: options.Search,
	}

	// Executa a consulta para contagem
	response, err := service.Query(ctx, countOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	// Extrai contagem da resposta
	if response != nil {
		if response.Count != nil {
			return *response.Count, nil
		}

		// Se não tem Count, conta os itens na resposta
		if response.Value != nil {
			if items, ok := response.Value.([]interface{}); ok {
				return int64(len(items)), nil
			}
		}
	}

	return 0, nil
}

// parseQueryOptions analisa as opções de consulta OData da URL
func (s *Server) parseQueryOptions(c fiber.Ctx) (QueryOptions, error) {
	var queryValues url.Values
	var err error

	// Extrai query string
	queryString := string(c.Request().URI().QueryString())

	// Parse rápido da query string
	queryValuesURL, parseErr := s.urlParser.ParseQueryFast(queryString)
	if parseErr != nil {
		return QueryOptions{}, fmt.Errorf("failed to parse query: %w", parseErr)
	}
	queryValues = queryValuesURL

	// Valida a query OData
	if err := s.urlParser.ValidateODataQueryFast(queryString); err != nil {
		return QueryOptions{}, fmt.Errorf("invalid OData query: %w", err)
	}

	// Parse das opções de consulta
	options, err := s.parser.ParseQueryOptions(queryValues)
	if err != nil {
		return QueryOptions{}, fmt.Errorf("failed to parse query options: %w", err)
	}

	// Valida as opções
	if err := s.parser.ValidateQueryOptions(options); err != nil {
		return QueryOptions{}, fmt.Errorf("invalid query options: %w", err)
	}

	return options, nil
}

// executeEntityQuery centraliza a execução de consultas para entidades
func (s *Server) executeEntityQuery(ctx context.Context, service EntityService, options QueryOptions, entityName string) (*ODataResponse, error) {
	// Log da consulta para debug
	s.logger.Printf("🔍 Executando consulta para entidade: %s", entityName)
	if options.Expand != nil {
		s.logger.Printf("🔍 Expand solicitado: %v", options.Expand)
	}
	if options.Filter != nil {
		s.logger.Printf("🔍 Filtro aplicado: %s", options.Filter.RawValue)
	}

	// Executa a consulta
	response, err := service.Query(ctx, options)
	if err != nil {
		s.logger.Printf("❌ Erro na consulta: %v", err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	s.logger.Printf("✅ Consulta executada com sucesso")
	return response, nil
}

// handleEntityQueryWithEvents executa consulta e dispara eventos apropriados
func (s *Server) handleEntityQueryWithEvents(ctx context.Context, service EntityService, options QueryOptions, entityName string, isCollection bool) (*ODataResponse, error) {
	// Executa a consulta
	response, err := s.executeEntityQuery(ctx, service, options, entityName)
	if err != nil {
		return nil, err
	}

	// Dispara eventos apropriados
	if response != nil && response.Value != nil {
		// Extrai Fiber Context do contexto para eventos
		var fiberCtx fiber.Ctx
		if fc, ok := ctx.Value(FiberContextKey).(fiber.Ctx); ok {
			fiberCtx = fc
		}

		if fiberCtx != nil {
			eventCtx := createEventContext(fiberCtx, entityName)

			if isCollection {
				// Para collections, dispara evento OnEntityList
				if results, ok := response.Value.([]interface{}); ok {
					args := NewEntityListArgs(eventCtx, options, results)

					// Definir TotalCount corretamente
					if response.Count != nil {
						args.TotalCount = *response.Count
					} else {
						args.TotalCount = int64(len(results))
					}

					// Definir se filtro foi aplicado
					args.FilterApplied = options.Filter != nil

					if err := s.eventManager.Emit(args); err != nil {
						s.logger.Printf("❌ Erro no evento OnEntityList: %v", err)
					}
				}
			} else {
				// Para entidades específicas, dispara evento OnEntityGet
				if results, ok := response.Value.([]interface{}); ok && len(results) > 0 {
					// Extrai chaves da URL para o evento
					keys := make(map[string]interface{})
					if options.Filter != nil {
						// Tenta extrair chaves do filtro (implementação básica)
						keys["extracted_from_filter"] = options.Filter.RawValue
					}

					args := NewEntityGetArgs(eventCtx, keys, results[0])
					if err := s.eventManager.Emit(args); err != nil {
						s.logger.Printf("❌ Erro no evento OnEntityGet: %v", err)
					}
				}
			}
		}
	}

	return response, nil
}

// =======================================================================================
// URL & KEY EXTRACTION
// =======================================================================================

// extractEntityName extrai o nome da entidade da URL
func (s *Server) extractEntityName(path string) string {
	// Remove o prefixo da rota
	prefix := s.config.RoutePrefix
	if strings.HasPrefix(path, prefix+"/") {
		path = strings.TrimPrefix(path, prefix+"/")
	}

	// Remove barra inicial
	path = strings.TrimPrefix(path, "/")

	// Remove parâmetros de ID se presentes
	if idx := strings.Index(path, "("); idx != -1 {
		path = path[:idx]
	}

	// Remove $count se presente
	path = strings.TrimSuffix(path, "/$count")

	return path
}

// extractKeys extrai as chaves da URL para operações em entidades específicas
func (s *Server) extractKeys(path string, metadata EntityMetadata) (map[string]interface{}, error) {
	keys := make(map[string]interface{})

	s.logger.Printf("🔍 extractKeys - Path: %s", path)

	// Encontra a parte entre parênteses
	start := strings.Index(path, "(")
	end := strings.LastIndex(path, ")")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("invalid key format in path: %s", path)
	}

	keyString := path[start+1 : end]
	s.logger.Printf("🔍 extractKeys - KeyString: %s", keyString)

	// Identifica as chaves primárias dos metadados
	var primaryKeys []PropertyMetadata
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			primaryKeys = append(primaryKeys, prop)
		}
	}

	s.logger.Printf("🔍 extractKeys - Primary keys: %+v", primaryKeys)

	if len(primaryKeys) == 0 {
		return nil, fmt.Errorf("no primary keys defined for entity")
	}

	// Se há apenas uma chave primária, assume que o valor é para ela
	if len(primaryKeys) == 1 {
		key := primaryKeys[0]
		value, err := s.parseKeyValue(keyString, key.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key value for %s: %w", key.Name, err)
		}
		keys[key.Name] = value
		s.logger.Printf("🔍 extractKeys - Single key result: %+v", keys)
		return keys, nil
	}

	// Para chaves compostas, precisa analisar pares chave=valor
	// Implementação básica para chaves compostas
	pairs := strings.Split(keyString, ",")
	for _, pair := range pairs {
		kv := strings.Split(strings.TrimSpace(pair), "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid key-value pair: %s", pair)
		}

		keyName := strings.TrimSpace(kv[0])
		keyValue := strings.TrimSpace(kv[1])

		// Encontra a propriedade correspondente
		var keyProp *PropertyMetadata
		for _, prop := range primaryKeys {
			if prop.Name == keyName {
				keyProp = &prop
				break
			}
		}

		if keyProp == nil {
			return nil, fmt.Errorf("unknown key: %s", keyName)
		}

		value, err := s.parseKeyValue(keyValue, keyProp.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key value for %s: %w", keyName, err)
		}

		keys[keyName] = value
	}

	s.logger.Printf("🔍 extractKeys - Composite key result: %+v", keys)
	return keys, nil
}

// parseKeyValue converte uma string em valor do tipo apropriado
func (s *Server) parseKeyValue(value, dataType string) (interface{}, error) {
	s.logger.Printf("🔍 parseKeyValue - Original value: '%s', dataType: '%s'", value, dataType)

	// Remove aspas se presentes
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
		s.logger.Printf("🔍 parseKeyValue - Removed quotes, new value: '%s'", value)
	}

	var result interface{}
	var err error

	switch dataType {
	case "string":
		result = value
	case "int32", "int":
		// Converte para int mas garante que seja tratado como int64 internamente
		intVal, parseErr := strconv.ParseInt(value, 10, 32)
		if parseErr != nil {
			err = parseErr
		} else {
			result = intVal // Retorna int64 para compatibilidade
		}
	case "int64":
		result, err = strconv.ParseInt(value, 10, 64)
	case "float32":
		val, parseErr := strconv.ParseFloat(value, 32)
		if parseErr != nil {
			err = parseErr
		} else {
			result = float64(val) // Converte para float64 para compatibilidade
		}
	case "float64":
		result, err = strconv.ParseFloat(value, 64)
	case "bool":
		result, err = strconv.ParseBool(value)
	default:
		s.logger.Printf("⚠️ parseKeyValue - Unknown dataType '%s', treating as string", dataType)
		result = value
	}

	if err != nil {
		s.logger.Printf("❌ parseKeyValue - Error converting '%s' to %s: %v", value, dataType, err)
		return nil, fmt.Errorf("failed to parse key value '%s' as %s: %w", value, dataType, err)
	}

	s.logger.Printf("✅ parseKeyValue - Converted to: %v (type: %T)", result, result)
	return result, nil
}

// =======================================================================================
// RESPONSE BUILDERS
// =======================================================================================

// buildEntityURL constrói a URL para uma entidade específica
func (s *Server) buildEntityURL(c fiber.Ctx, service EntityService, entity interface{}) string {
	metadata := service.GetMetadata()

	// Encontra as chaves primárias
	var keyValues []string
	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return ""
	}

	for _, prop := range metadata.Properties {
		if prop.IsKey {
			if value, exists := entityMap[prop.Name]; exists {
				keyValues = append(keyValues, fmt.Sprintf("%v", value))
			}
		}
	}

	if len(keyValues) == 0 {
		return ""
	}

	scheme := "http"
	if c.Protocol() == "https" {
		scheme = "https"
	}

	baseURL := fmt.Sprintf("%s://%s%s/%s", scheme, c.Hostname(), s.config.RoutePrefix, metadata.Name)

	if len(keyValues) == 1 {
		return fmt.Sprintf("%s(%s)", baseURL, keyValues[0])
	}

	// Para chaves compostas, usar formato chave=valor
	var keyPairs []string
	i := 0
	for _, prop := range metadata.Properties {
		if prop.IsKey && i < len(keyValues) {
			keyPairs = append(keyPairs, fmt.Sprintf("%s=%s", prop.Name, keyValues[i]))
			i++
		}
	}

	return fmt.Sprintf("%s(%s)", baseURL, strings.Join(keyPairs, ","))
}

// buildMetadataJSON constrói os metadados em formato JSON
func (s *Server) buildMetadataJSON() MetadataResponse {
	metadata := MetadataResponse{
		Context: "$metadata",
		Version: "4.0",
	}

	// Adiciona as entidades
	var entities []EntityTypeMetadata
	var entitySets []EntitySetMetadata

	for name, service := range s.entities {
		entityMetadata := service.GetMetadata()

		// Constrói as propriedades
		var properties []PropertyTypeMetadata
		for _, prop := range entityMetadata.Properties {
			property := PropertyTypeMetadata{
				Name:       prop.Name,
				Type:       s.mapODataType(prop.Type),
				Nullable:   prop.IsNullable,
				IsKey:      prop.IsKey,
				HasDefault: prop.HasDefault,
				MaxLength:  prop.MaxLength,
			}

			properties = append(properties, property)
		}

		// Entidade
		entity := EntityTypeMetadata{
			Name:       name,
			Namespace:  "Default",
			Keys:       s.getEntityKeys(entityMetadata),
			Properties: properties,
		}

		entities = append(entities, entity)

		// Entity Set
		entitySet := EntitySetMetadata{
			Name:       name,
			EntityType: "Default." + name,
			Kind:       "EntitySet",
			URL:        name,
		}

		entitySets = append(entitySets, entitySet)
	}

	metadata.Entities = entities
	metadata.EntitySets = entitySets

	return metadata
}

// getEntityKeys retorna as chaves primárias de uma entidade
func (s *Server) getEntityKeys(metadata EntityMetadata) []string {
	var keys []string
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			keys = append(keys, prop.Name)
		}
	}
	return keys
}

// mapODataType mapeia tipos internos para tipos OData
func (s *Server) mapODataType(internalType string) string {
	typeMap := map[string]string{
		"string":    "Edm.String",
		"int":       "Edm.Int32",
		"int32":     "Edm.Int32",
		"int64":     "Edm.Int64",
		"float32":   "Edm.Single",
		"float64":   "Edm.Double",
		"bool":      "Edm.Boolean",
		"time.Time": "Edm.DateTimeOffset",
		"[]byte":    "Edm.Binary",
		"object":    "Edm.ComplexType",
		"array":     "Collection(Edm.String)",
	}

	if mappedType, exists := typeMap[internalType]; exists {
		return mappedType
	}
	return "Edm.String" // Default
}

// buildEntitySets constrói a lista de entity sets
func (s *Server) buildEntitySets() []map[string]interface{} {
	var entitySets []map[string]interface{}

	for name := range s.entities {
		entitySets = append(entitySets, map[string]interface{}{
			"name": name,
			"kind": "EntitySet",
			"url":  name,
		})
	}

	return entitySets
}

// buildSingleEntityResponse constrói resposta OData para uma entidade única
func (s *Server) buildSingleEntityResponse(entity interface{}, metadata EntityMetadata) map[string]interface{} {
	// Cria um map para a resposta
	response := make(map[string]interface{})

	// Adiciona o contexto OData
	response["@odata.context"] = fmt.Sprintf("$metadata#%s", metadata.Name)

	// Se a entidade é um OrderedEntity, preserva a ordem e navigation links
	if orderedEntity, ok := entity.(*OrderedEntity); ok {
		// Adiciona todas as propriedades da entidade
		for _, prop := range orderedEntity.Properties {
			response[prop.Name] = prop.Value
		}

		// Adiciona navigation links
		for _, navLink := range orderedEntity.NavigationLinks {
			response[fmt.Sprintf("%s@odata.navigationLink", navLink.Name)] = navLink.URL
		}
	} else if entityMap, ok := entity.(map[string]interface{}); ok {
		// Para maps regulares, copia todas as propriedades
		for key, value := range entityMap {
			response[key] = value
		}
	}

	return response
}

// buildODataResponse centraliza a construção de respostas OData
func (s *Server) buildODataResponse(response *ODataResponse, isCollection bool, metadata EntityMetadata) interface{} {
	if response == nil {
		return nil
	}

	if isCollection {
		// Para collections, retorna a resposta completa
		return response
	} else {
		// Para entidades específicas, extrai a primeira entidade e adiciona contexto
		if results, ok := response.Value.([]interface{}); ok && len(results) > 0 {
			entity := results[0]

			// Se é OrderedEntity, cria resposta ordenada com contexto
			if orderedEntity, ok := entity.(*OrderedEntity); ok {
				// Cria resposta ordenada seguindo a ordem dos metadados
				entityResponse := NewOrderedEntityResponse(
					fmt.Sprintf("$metadata#%s", metadata.Name),
					metadata,
				)

				// Adiciona propriedades na ordem dos metadados da entidade
				for _, metaProp := range metadata.Properties {
					if !metaProp.IsNavigation {
						if value, exists := orderedEntity.Get(metaProp.Name); exists {
							entityResponse.AddField(metaProp.Name, value)
						}
					}
				}

				// Adiciona propriedades que não estão nos metadados (na ordem original da entidade)
				addedFields := make(map[string]bool)
				for _, metaProp := range metadata.Properties {
					if !metaProp.IsNavigation {
						addedFields[metaProp.Name] = true
					}
				}

				for _, prop := range orderedEntity.Properties {
					if !addedFields[prop.Name] {
						entityResponse.AddField(prop.Name, prop.Value)
					}
				}

				// Adiciona navigation links na ordem dos metadados
				for _, metaProp := range metadata.Properties {
					if metaProp.IsNavigation {
						for _, navLink := range orderedEntity.NavigationLinks {
							if navLink.Name == metaProp.Name {
								entityResponse.AddNavigationLink(navLink.Name, navLink.URL)
								break
							}
						}
					}
				}

				return entityResponse
			}

			// Para outros tipos, usa o método buildSingleEntityResponse
			return s.buildSingleEntityResponse(entity, metadata)
		}

		// Se não há resultados, retorna nil
		return nil
	}
}

// =======================================================================================
// ERROR HANDLING
// =======================================================================================

// writeError escreve uma resposta de erro OData
func (s *Server) writeError(c fiber.Ctx, statusCode int, code, message string) {
	c.Set("Content-Type", "application/json")
	c.Status(statusCode)

	errorResponse := ODataResponse{
		Error: &ODataError{
			Code:    code,
			Message: message,
		},
	}

	c.JSON(errorResponse)
}

// =======================================================================================
// MULTI-TENANT HELPERS
// =======================================================================================

// getCurrentProvider retorna o provider para o tenant atual
func (s *Server) getCurrentProvider(c fiber.Ctx) DatabaseProvider {
	if s.multiTenantPool == nil {
		return s.provider
	}

	tenantID := GetCurrentTenant(c)
	return s.multiTenantPool.GetProvider(tenantID)
}
