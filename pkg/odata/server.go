package odata

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// Server representa o servidor OData
type Server struct {
	entities  map[string]EntityService
	router    *mux.Router
	parser    *ODataParser
	urlParser *URLParser // Apenas URLParser
	provider  DatabaseProvider
}

// NewServer cria uma nova instância do servidor OData
func NewServer(provider DatabaseProvider) *Server {
	return &Server{
		entities:  make(map[string]EntityService),
		router:    mux.NewRouter(),
		parser:    NewODataParser(),
		urlParser: NewURLParser(), // Usa o constructor padrão
		provider:  provider,
	}
}

// RegisterEntity registra uma entidade no servidor usando mapeamento automático
func (s *Server) RegisterEntity(name string, entity interface{}) {
	metadata, err := MapEntityFromStruct(entity)
	if err != nil {
		log.Fatalf("Erro ao registrar entidade %s: %v", name, err)
	}

	service := NewBaseEntityService(s.provider, metadata, s)
	s.entities[name] = service
	s.setupRoutes(name)
}

// RegisterEntityWithService registra uma entidade com um serviço customizado
func (s *Server) RegisterEntityWithService(name string, service EntityService) {
	s.entities[name] = service
	s.setupRoutes(name)
}

// setupRoutes configura as rotas para uma entidade
func (s *Server) setupRoutes(entityName string) {
	// Rota para coleção de entidades (GET, POST)
	s.router.HandleFunc("/odata/"+entityName, s.handleEntityCollection).Methods("GET", "POST")

	// Rota para entidade individual (GET, PUT, PATCH, DELETE)
	s.router.HandleFunc("/odata/"+entityName+"({id})", s.handleEntityById).Methods("GET", "PUT", "PATCH", "DELETE")
	s.router.HandleFunc("/odata/"+entityName+"({id:[0-9]+})", s.handleEntityById).Methods("GET", "PUT", "PATCH", "DELETE")

	// Rota para count da coleção - deve vir ANTES da rota de entidades individuais
	s.router.HandleFunc("/odata/"+entityName+"/$count", s.handleEntityCount).Methods("GET")

	// Rota para metadados
	s.router.HandleFunc("/odata/$metadata", s.handleMetadata).Methods("GET")

	// Rota para service document
	s.router.HandleFunc("/odata/", s.handleServiceDocument).Methods("GET")
}

// GetRouter retorna o router do servidor
func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// handleEntityCollection lida com operações na coleção de entidades
func (s *Server) handleEntityCollection(w http.ResponseWriter, r *http.Request) {
	entityName := s.extractEntityName(r.URL.Path)
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetCollection(w, r, service)
	case "POST":
		s.handleCreateEntity(w, r, service)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
	}
}

// handleEntityById lida com operações em uma entidade específica
func (s *Server) handleEntityById(w http.ResponseWriter, r *http.Request) {
	entityName := s.extractEntityName(r.URL.Path)
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return
	}

	// Extrai as chaves da URL
	keys, err := s.extractKeys(r.URL.Path, service.GetMetadata())
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidKey", err.Error())
		return
	}

	switch r.Method {
	case "GET":
		s.handleGetEntity(w, r, service, keys)
	case "PUT":
		s.handleUpdateEntity(w, r, service, keys, false)
	case "PATCH":
		s.handleUpdateEntity(w, r, service, keys, true)
	case "DELETE":
		s.handleDeleteEntity(w, r, service, keys)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
	}
}

// handleGetCollection lida com GET na coleção de entidades
func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request, service EntityService) {
	var queryValues url.Values
	var err error

	queryValuesURL, parseErr := s.urlParser.ParseQueryFast(r.URL.RawQuery)
	if parseErr != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Failed to parse query: %v", parseErr))
		return
	}
	queryValues = queryValuesURL

	// Valida a query OData otimizada
	if err := s.urlParser.ValidateODataQueryFast(r.URL.RawQuery); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Invalid OData query: %v", err))
		return
	}

	// Parse das opções de consulta
	options, err := s.parser.ParseQueryOptions(queryValues)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", err.Error())
		return
	}

	// Valida as opções
	if err := s.parser.ValidateQueryOptions(options); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", err.Error())
		return
	}

	// Executa a consulta
	response, err := service.Query(r.Context(), options)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "QueryError", err.Error())
		return
	}

	s.writeJSON(w, response)
}

// handleGetEntity lida com GET de uma entidade específica
func (s *Server) handleGetEntity(w http.ResponseWriter, r *http.Request, service EntityService, keys map[string]interface{}) {
	entity, err := service.Get(r.Context(), keys)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "EntityNotFound", err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "QueryError", err.Error())
		}
		return
	}

	s.writeJSON(w, entity)
}

// handleCreateEntity lida com POST para criar uma entidade
func (s *Server) handleCreateEntity(w http.ResponseWriter, r *http.Request, service EntityService) {
	var entity map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "Invalid JSON")
		return
	}

	createdEntity, err := service.Create(r.Context(), entity)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "CreateError", err.Error())
		return
	}

	w.Header().Set("Location", s.buildEntityURL(r, service, createdEntity))
	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, createdEntity)
}

// handleUpdateEntity lida com PUT/PATCH para atualizar uma entidade
func (s *Server) handleUpdateEntity(w http.ResponseWriter, r *http.Request, service EntityService, keys map[string]interface{}, isPartial bool) {
	var entity map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidRequest", "Invalid JSON")
		return
	}

	updatedEntity, err := service.Update(r.Context(), keys, entity)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "EntityNotFound", err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "UpdateError", err.Error())
		}
		return
	}

	s.writeJSON(w, updatedEntity)
}

// handleDeleteEntity lida com DELETE para remover uma entidade
func (s *Server) handleDeleteEntity(w http.ResponseWriter, r *http.Request, service EntityService, keys map[string]interface{}) {
	err := service.Delete(r.Context(), keys)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, "EntityNotFound", err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "DeleteError", err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleMetadata lida com GET dos metadados
func (s *Server) handleMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := s.buildMetadataJSON()
	s.writeJSON(w, metadata)
}

// handleServiceDocument lida com GET do documento de serviço
func (s *Server) handleServiceDocument(w http.ResponseWriter, r *http.Request) {
	serviceDoc := map[string]interface{}{
		"@odata.context": "$metadata",
		"value":          s.buildEntitySets(),
	}

	s.writeJSON(w, serviceDoc)
}

// handleEntityCount lida com GET do count de uma coleção de entidades
func (s *Server) handleEntityCount(w http.ResponseWriter, r *http.Request) {
	entityName := s.extractEntityName(r.URL.Path)
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return
	}

	// Parse das opções de consulta usando o parser customizado
	queryValues, err := s.urlParser.ParseQuery(r.URL.RawQuery)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Failed to parse query: %v", err))
		return
	}

	// Valida a query OData
	if err := s.urlParser.ValidateODataQuery(r.URL.RawQuery); err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Invalid OData query: %v", err))
		return
	}

	// Parse das opções de consulta (principalmente $filter)
	options, err := s.parser.ParseQueryOptions(queryValues)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "InvalidQuery", err.Error())
		return
	}

	// Obtém o count através do EntityService
	count, err := s.getEntityCount(r.Context(), service, options)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "QueryError", err.Error())
		return
	}

	// Retorna apenas o número (sem envelope JSON)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("OData-Version", "4.0")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%d", count)
}

// getEntityCount obtém o count de uma entidade através do seu service
func (s *Server) getEntityCount(ctx context.Context, service EntityService, options QueryOptions) (int64, error) {
	// Usa o método do BaseEntityService se disponível
	if baseService, ok := service.(*BaseEntityService); ok {
		return baseService.GetCount(ctx, options)
	}

	// Fallback: executa uma query com count através do service
	// Força Count=true para obter o count
	countOptions := options
	countOptions.Count = SetCountValue(true)

	// Cria Top query com valor 1
	topQuery := GoDataTopQuery(1)
	countOptions.Top = &topQuery

	// Cria Skip query com valor 0
	skipQuery := GoDataSkipQuery(0)
	countOptions.Skip = &skipQuery

	response, err := service.Query(ctx, countOptions)
	if err != nil {
		return 0, err
	}

	if response.Count != nil {
		return *response.Count, nil
	}

	return 0, fmt.Errorf("count not available")
}

// extractEntityName extrai o nome da entidade da URL
func (s *Server) extractEntityName(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "odata" && i+1 < len(parts) {
			entityPart := parts[i+1]
			// Remove parênteses se existirem
			if idx := strings.Index(entityPart, "("); idx != -1 {
				entityPart = entityPart[:idx]
			}
			return entityPart
		}
	}
	return ""
}

// extractKeys extrai as chaves da URL
func (s *Server) extractKeys(path string, metadata EntityMetadata) (map[string]interface{}, error) {
	keys := make(map[string]interface{})

	// Procura por parênteses na URL
	start := strings.Index(path, "(")
	end := strings.LastIndex(path, ")")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("invalid key format")
	}

	keyStr := path[start+1 : end]

	// Se há apenas uma chave numérica
	if val, err := strconv.Atoi(keyStr); err == nil {
		// Encontra a primeira chave nos metadados
		for _, prop := range metadata.Properties {
			if prop.IsKey {
				keys[prop.Name] = val
				break
			}
		}
		return keys, nil
	}

	// Se há chaves nomeadas (ex: ID=1,Name='Test')
	keyPairs := strings.Split(keyStr, ",")
	for _, pair := range keyPairs {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove aspas se existirem
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = value[1 : len(value)-1]
			keys[key] = value
		} else if val, err := strconv.Atoi(value); err == nil {
			keys[key] = val
		} else {
			keys[key] = value
		}
	}

	return keys, nil
}

// buildEntityURL constrói a URL da entidade criada
func (s *Server) buildEntityURL(r *http.Request, service EntityService, entity interface{}) string {
	metadata := service.GetMetadata()

	// Assume que a entidade é um map
	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return ""
	}

	// Constrói a URL com as chaves
	var keyParts []string
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			if val, exists := entityMap[prop.Name]; exists {
				keyParts = append(keyParts, fmt.Sprintf("%v", val))
			}
		}
	}

	if len(keyParts) == 0 {
		return ""
	}

	return fmt.Sprintf("%s://%s/odata/%s(%s)",
		"http", r.Host, metadata.Name, strings.Join(keyParts, ","))
}

// buildMetadataJSON constrói a estrutura JSON dos metadados
func (s *Server) buildMetadataJSON() MetadataResponse {
	entities := []EntityTypeMetadata{}
	entitySets := []EntitySetMetadata{}

	// Constrói entidades e entity sets
	for name, service := range s.entities {
		metadata := service.GetMetadata()

		// Constrói o tipo de entidade
		entityType := EntityTypeMetadata{
			Name:       name,
			Namespace:  "Default",
			Keys:       metadata.Keys,
			Properties: []PropertyTypeMetadata{},
			Navigation: []NavigationPropertyMetadata{},
		}

		// Adiciona propriedades
		for _, prop := range metadata.Properties {
			if prop.IsNavigation {
				// Propriedade de navegação
				navProp := NavigationPropertyMetadata{
					Name:         prop.Name,
					Type:         prop.RelatedType,
					IsCollection: prop.IsCollection,
				}
				entityType.Navigation = append(entityType.Navigation, navProp)
			} else {
				// Propriedade regular
				propType := PropertyTypeMetadata{
					Name:       prop.Name,
					Type:       s.mapODataType(prop.Type),
					Nullable:   prop.IsNullable,
					MaxLength:  prop.MaxLength,
					Precision:  prop.Precision,
					Scale:      prop.Scale,
					IsKey:      prop.IsKey,
					HasDefault: prop.HasDefault,
				}
				entityType.Properties = append(entityType.Properties, propType)
			}
		}

		entities = append(entities, entityType)

		// Constrói entity set
		entitySet := EntitySetMetadata{
			Name:       name,
			EntityType: "Default." + name,
			Kind:       "EntitySet",
			URL:        name,
		}
		entitySets = append(entitySets, entitySet)
	}

	// Constrói schema
	schema := SchemaMetadata{
		Namespace:   "Default",
		EntityTypes: entities,
		EntitySets:  entitySets,
		EntityContainer: EntityContainerMetadata{
			Name:       "Container",
			EntitySets: entitySets,
		},
	}

	return MetadataResponse{
		Context:    "$metadata",
		Version:    "4.0",
		Entities:   entities,
		EntitySets: entitySets,
		Schemas:    []SchemaMetadata{schema},
	}
}

// mapODataType mapeia tipos internos para tipos OData
func (s *Server) mapODataType(internalType string) string {
	typeMap := map[string]string{
		"string":    "Edm.String",
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

// buildMetadataXML constrói o XML dos metadados (mantido para compatibilidade)
func (s *Server) buildMetadataXML() string {
	var xml strings.Builder
	xml.WriteString(`<?xml version="1.0" encoding="utf-8"?>`)
	xml.WriteString(`<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">`)
	xml.WriteString(`<edmx:DataServices>`)
	xml.WriteString(`<Schema Namespace="Default" xmlns="http://docs.oasis-open.org/odata/ns/edm">`)

	// Adiciona as entidades
	for name, service := range s.entities {
		metadata := service.GetMetadata()
		xml.WriteString(fmt.Sprintf(`<EntityType Name="%s">`, name))

		// Adiciona as chaves
		xml.WriteString(`<Key>`)
		for _, prop := range metadata.Properties {
			if prop.IsKey {
				xml.WriteString(fmt.Sprintf(`<PropertyRef Name="%s"/>`, prop.Name))
			}
		}
		xml.WriteString(`</Key>`)

		// Adiciona as propriedades
		for _, prop := range metadata.Properties {
			if !prop.IsNavigation {
				xml.WriteString(fmt.Sprintf(`<Property Name="%s" Type="%s"/>`, prop.Name, s.mapODataType(prop.Type)))
			}
		}

		xml.WriteString(`</EntityType>`)
	}

	// Adiciona o container
	xml.WriteString(`<EntityContainer Name="Container">`)
	for name := range s.entities {
		xml.WriteString(fmt.Sprintf(`<EntitySet Name="%s" EntityType="Default.%s"/>`, name, name))
	}
	xml.WriteString(`</EntityContainer>`)

	xml.WriteString(`</Schema>`)
	xml.WriteString(`</edmx:DataServices>`)
	xml.WriteString(`</edmx:Edmx>`)

	return xml.String()
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

// writeJSON escreve uma resposta JSON
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("OData-Version", "4.0")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.writeError(w, http.StatusInternalServerError, "SerializationError", err.Error())
	}
}

// writeError escreve uma resposta de erro
func (s *Server) writeError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := ODataResponse{
		Error: &ODataError{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// Start inicia o servidor HTTP
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting OData server on %s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// StartWithContext inicia o servidor com contexto
func (s *Server) StartWithContext(ctx context.Context, port int) error {
	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	fmt.Printf("Starting OData server on %s\n", addr)
	return server.ListenAndServe()
}

// parseQueryOptionsWithCustomParser faz o parsing das opções usando o parser customizado
func (s *Server) parseQueryOptionsWithCustomParser(r *http.Request) (QueryOptions, error) {
	// Parse usando o parser customizado
	queryValues, err := s.urlParser.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return QueryOptions{}, fmt.Errorf("failed to parse query: %w", err)
	}

	// Valida a query OData
	if err := s.urlParser.ValidateODataQuery(r.URL.RawQuery); err != nil {
		return QueryOptions{}, fmt.Errorf("invalid OData query: %w", err)
	}

	// Extrai parâmetros do sistema OData
	systemParams := s.urlParser.ExtractODataSystemParams(queryValues)

	// Processa valores específicos do OData
	processedValues := queryValues

	// Processa valores $expand se presente
	if expandValue, exists := systemParams["$expand"]; exists {
		cleanedExpand, err := s.urlParser.ParseExpandValue(expandValue)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid $expand value: %w", err)
		}
		processedValues.Set("$expand", cleanedExpand)
	}

	// Processa valores $filter se presente
	if filterValue, exists := systemParams["$filter"]; exists {
		cleanedFilter, err := s.urlParser.ParseFilterValue(filterValue)
		if err != nil {
			return QueryOptions{}, fmt.Errorf("invalid $filter value: %w", err)
		}
		processedValues.Set("$filter", cleanedFilter)
	}

	// Parse das opções de consulta
	return s.parser.ParseQueryOptions(processedValues)
}

// debugQueryParsing adiciona debug detalhado do parsing de query
func (s *Server) debugQueryParsing(r *http.Request) {
	log.Printf("🔍 DEBUG Query Parsing:")
	log.Printf("   Raw Query: %s", r.URL.RawQuery)
	log.Printf("   Standard Query: %v", r.URL.Query())

	// Testa o parser customizado
	customValues, err := s.urlParser.ParseQuery(r.URL.RawQuery)
	if err != nil {
		log.Printf("   Custom Parser Error: %v", err)
	} else {
		log.Printf("   Custom Query: %v", customValues)
	}

	// Compara os resultados
	standardValues := r.URL.Query()
	log.Printf("   Differences:")
	for key, vals := range customValues {
		if standardVals, exists := standardValues[key]; exists {
			if len(vals) != len(standardVals) || (len(vals) > 0 && vals[0] != standardVals[0]) {
				log.Printf("     %s: standard=%v, custom=%v", key, standardVals, vals)
			}
		} else {
			log.Printf("     %s: missing in standard, custom=%v", key, vals)
		}
	}
}
