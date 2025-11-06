package odata

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// =======================================================================================
// HEALTH & INFO HANDLERS
// =======================================================================================

// handleHealth lida com requisições de health check
func (s *Server) handleHealth(c fiber.Ctx) error {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"entities":  len(s.entities),
	}

	// Testa conexão com banco se possível
	if s.provider != nil {
		if db := s.provider.GetConnection(); db != nil {
			if err := db.Ping(); err != nil {
				health["database"] = "error"
				health["database_error"] = err.Error()
			} else {
				health["database"] = "healthy"
			}
		}
	}

	return c.JSON(health)
}

// handleServerInfo lida com requisições de informações do servidor
func (s *Server) handleServerInfo(c fiber.Ctx) error {
	info := map[string]interface{}{
		"name":          "Go-Data OData Server",
		"version":       "1.0.0",
		"odata_version": "4.0",
		"description":   "Servidor OData v4 completo em Go",
		"address":       s.GetAddress(),
		"entities":      len(s.entities),
		"entity_list":   s.getEntityList(),
		"endpoints": map[string]string{
			"service_document": s.config.RoutePrefix + "/",
			"metadata":         s.config.RoutePrefix + "/$metadata",
			"health":           "/health",
			"info":             "/info",
		},
		"features": []string{
			"CRUD Operations",
			"Query Options ($filter, $orderby, $select, $expand, $top, $skip, $count)",
			"Computed Fields ($compute)",
			"Search ($search)",
			"Relationships (association, manyAssociation)",
			"Cascade Operations",
			"Nullable Types",
			"Auto Schema Generation",
			"Multi-database Support",
			"JSON Responses",
			"CORS Support",
			"Graceful Shutdown",
			"Health Checks",
		},
	}

	return c.JSON(info)
}

// getEntityList retorna lista de entidades registradas
func (s *Server) getEntityList() []string {
	var entities []string
	for name := range s.entities {
		entities = append(entities, name)
	}
	return entities
}

// handleOptions lida com requisições OPTIONS para CORS
func (s *Server) handleOptions(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// =======================================================================================
// ENTITY COLLECTION HANDLERS
// =======================================================================================

// handleEntityCollection lida com operações na coleção de entidades (GET, POST)
func (s *Server) handleEntityCollection(c fiber.Ctx) error {
	entityName := s.extractEntityName(c.Path())
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(c, fiber.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return nil
	}

	switch c.Method() {
	case "GET":
		return s.handleGetCollection(c, service)
	case "POST":
		return s.handleCreateEntity(c, service)
	default:
		s.writeError(c, fiber.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return nil
	}
}

// handleGetCollection lida com GET na coleção de entidades
func (s *Server) handleGetCollection(c fiber.Ctx, service EntityService) error {
	// Cria contexto com referência ao Fiber Context para multi-tenant
	// Usa background context para garantir que não seja cancelado quando a requisição HTTP terminar
	ctx := context.WithValue(context.Background(), FiberContextKey, c)

	// Extrai o nome da entidade
	entityName := s.extractEntityName(c.Path())

	// Parse centralizado das opções de consulta
	options, err := s.parseQueryOptions(c)
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Executa consulta centralizada com eventos
	response, err := s.handleEntityQueryWithEvents(ctx, service, options, entityName, true)
	if err != nil {
		s.writeError(c, fiber.StatusInternalServerError, "QueryError", err.Error())
		return nil
	}

	// Constrói resposta OData centralizada
	odataResponse := s.buildODataResponse(response, true, service.GetMetadata())

	return c.JSON(odataResponse)
}

// handleCreateEntity lida com POST para criar uma entidade
func (s *Server) handleCreateEntity(c fiber.Ctx, service EntityService) error {
	var entity map[string]interface{}
	if err := c.Bind().Body(&entity); err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidRequest", "Invalid JSON")
		return nil
	}

	createdEntity, err := service.Create(c.Context(), entity)
	if err != nil {
		s.writeError(c, fiber.StatusInternalServerError, "CreateError", err.Error())
		return nil
	}

	c.Set("Location", s.buildEntityURL(c, service, createdEntity))
	c.Status(fiber.StatusCreated)
	return c.JSON(createdEntity)
}

// =======================================================================================
// ENTITY BY ID HANDLERS
// =======================================================================================

// handleEntityById lida com operações em uma entidade específica (GET, PUT, PATCH, DELETE)
func (s *Server) handleEntityById(c fiber.Ctx) error {
	path := c.Path()
	s.logger.Printf("🔍 handleEntityById - Path: %s", path)

	entityName := s.extractEntityName(path)
	s.logger.Printf("🔍 handleEntityById - EntityName: %s", entityName)

	service, exists := s.entities[entityName]
	if !exists {
		s.logger.Printf("❌ handleEntityById - Entity '%s' not found", entityName)
		s.writeError(c, fiber.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return nil
	}

	// Verifica se o path tem parênteses para distinguir de collection request
	if !strings.Contains(path, "(") {
		s.logger.Printf("❌ handleEntityById - Path sem parênteses, redirecionando para collection")
		return s.handleEntityCollection(c)
	}

	// Extrai as chaves da URL
	keys, err := s.extractKeys(path, service.GetMetadata())
	if err != nil {
		s.logger.Printf("❌ handleEntityById - Erro ao extrair chaves: %v", err)
		s.writeError(c, fiber.StatusBadRequest, "InvalidKey", err.Error())
		return nil
	}

	s.logger.Printf("🔍 handleEntityById - Keys extraídas: %+v", keys)

	switch c.Method() {
	case "GET":
		return s.handleGetEntity(c, service, keys)
	case "PUT":
		return s.handleUpdateEntity(c, service, keys)
	case "PATCH":
		return s.handleUpdateEntity(c, service, keys)
	case "DELETE":
		return s.handleDeleteEntity(c, service, keys)
	default:
		s.writeError(c, fiber.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return nil
	}
}

// handleGetEntity lida com GET de uma entidade específica
func (s *Server) handleGetEntity(c fiber.Ctx, service EntityService, keys map[string]interface{}) error {
	s.logger.Printf("🔍 handleGetEntity - Starting with keys: %+v", keys)

	// Log dos tipos das chaves para debug
	for k, v := range keys {
		s.logger.Printf("🔍 handleGetEntity - Key '%s': value=%v, type=%T", k, v, v)
	}

	// Cria contexto com referência ao Fiber Context para multi-tenant
	// Usa background context para garantir que não seja cancelado quando a requisição HTTP terminar
	ctx := context.WithValue(context.Background(), FiberContextKey, c)

	// Extrai o nome da entidade
	entityName := s.extractEntityName(c.Path())

	// Parse das opções de consulta da URL (caso existam)
	options, err := s.parseQueryOptions(c)
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Constrói filtro para as chaves específicas usando o método centralizado do BaseEntityService
	baseService, ok := service.(*BaseEntityService)
	if !ok {
		// Tenta com MultiTenantEntityService
		if mtService, ok := service.(*MultiTenantEntityService); ok {
			baseService = mtService.BaseEntityService
		} else {
			s.writeError(c, fiber.StatusInternalServerError, "ServiceError", "Service type not supported")
			return nil
		}
	}

	// Constrói filtro tipado para as chaves
	keyFilter, err := baseService.BuildTypedKeyFilter(ctx, keys)
	if err != nil {
		s.logger.Printf("❌ handleGetEntity - Failed to build key filter: %v", err)
		s.writeError(c, fiber.StatusBadRequest, "InvalidKey", err.Error())
		return nil
	}

	// Combina filtro de chaves com filtro da query (se houver)
	if options.Filter != nil {
		// Se já há um filtro na query, combina com AND
		s.logger.Printf("🔍 handleGetEntity - Combining key filter with existing filter")
		combinedFilter := fmt.Sprintf("(%s) and (%s)", keyFilter.RawValue, options.Filter.RawValue)

		// Cria novo filtro combinado (implementação básica - idealmente deveria combinar as árvores)
		keyFilter.RawValue = combinedFilter
	}

	// Aplica o filtro de chaves
	options.Filter = keyFilter

	// Executa consulta centralizada com eventos
	response, err := s.handleEntityQueryWithEvents(ctx, service, options, entityName, false)
	if err != nil {
		s.writeError(c, fiber.StatusInternalServerError, "QueryError", err.Error())
		return nil
	}

	// Verifica se a entidade foi encontrada
	if response == nil || response.Value == nil {
		s.writeError(c, fiber.StatusNotFound, "EntityNotFound", "Entity not found")
		return nil
	}

	if results, ok := response.Value.([]interface{}); ok {
		if len(results) == 0 {
			s.writeError(c, fiber.StatusNotFound, "EntityNotFound", "Entity not found")
			return nil
		}
	}

	s.logger.Printf("✅ handleGetEntity - Entity retrieved successfully")

	// Dispara evento OnEntityGet específico com as chaves reais
	eventCtx := createEventContext(c, entityName)
	if results, ok := response.Value.([]interface{}); ok && len(results) > 0 {
		args := NewEntityGetArgs(eventCtx, keys, results[0])
		if err := s.eventManager.Emit(args); err != nil {
			s.logger.Printf("❌ Erro no evento OnEntityGet: %v", err)
		}
	}

	// Constrói resposta OData centralizada
	odataResponse := s.buildODataResponse(response, false, service.GetMetadata())

	return c.JSON(odataResponse)
}

// handleUpdateEntity lida com PUT/PATCH para atualizar uma entidade
func (s *Server) handleUpdateEntity(c fiber.Ctx, service EntityService, keys map[string]interface{}) error {
	var entity map[string]interface{}
	if err := c.Bind().Body(&entity); err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidRequest", "Invalid JSON")
		return nil
	}

	updatedEntity, err := service.Update(c.Context(), keys, entity)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(c, fiber.StatusNotFound, "EntityNotFound", err.Error())
		} else {
			s.writeError(c, fiber.StatusInternalServerError, "UpdateError", err.Error())
		}
		return nil
	}

	return c.JSON(updatedEntity)
}

// handleDeleteEntity lida com DELETE para remover uma entidade
func (s *Server) handleDeleteEntity(c fiber.Ctx, service EntityService, keys map[string]interface{}) error {
	err := service.Delete(c.Context(), keys)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(c, fiber.StatusNotFound, "EntityNotFound", err.Error())
		} else {
			s.writeError(c, fiber.StatusInternalServerError, "DeleteError", err.Error())
		}
		return nil
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// =======================================================================================
// METADATA & SERVICE DOCUMENT HANDLERS
// =======================================================================================

// handleMetadata lida com GET dos metadados OData
func (s *Server) handleMetadata(c fiber.Ctx) error {
	metadata := s.buildMetadataJSON()
	return c.JSON(metadata)
}

// handleServiceDocument lida com GET do documento de serviço OData
func (s *Server) handleServiceDocument(c fiber.Ctx) error {
	serviceDoc := map[string]interface{}{
		"@odata.context": "$metadata",
		"value":          s.buildEntitySets(),
	}

	return c.JSON(serviceDoc)
}

// =======================================================================================
// COUNT HANDLER
// =======================================================================================

// handleEntityCount lida com GET do count de uma coleção de entidades
func (s *Server) handleEntityCount(c fiber.Ctx) error {
	entityName := s.extractEntityName(c.Path())
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(c, fiber.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return nil
	}

	// Parse centralizado das opções de consulta
	options, err := s.parseQueryOptions(c)
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Obtém a contagem usando o método centralizado
	count, err := s.getEntityCount(c.Context(), service, options)
	if err != nil {
		s.writeError(c, fiber.StatusInternalServerError, "CountError", err.Error())
		return nil
	}

	// Retorna apenas o valor numérico para count
	c.Set("Content-Type", "text/plain")
	c.Status(fiber.StatusOK)
	return c.SendString(fmt.Sprintf("%d", count))
}

// =======================================================================================
// MULTI-TENANT HANDLERS
// =======================================================================================

// handleTenantList lista todos os tenants disponíveis
func (s *Server) handleTenantList(c fiber.Ctx) error {
	if s.multiTenantPool == nil {
		return c.JSON(map[string]interface{}{
			"multi_tenant": false,
			"tenants":      []string{"default"},
		})
	}

	tenants := s.multiTenantPool.GetTenantList()
	return c.JSON(map[string]interface{}{
		"multi_tenant": true,
		"tenants":      tenants,
		"total_count":  len(tenants),
	})
}

// handleTenantStats retorna estatísticas de todos os tenants
func (s *Server) handleTenantStats(c fiber.Ctx) error {
	if s.multiTenantPool == nil {
		return c.JSON(map[string]interface{}{
			"multi_tenant": false,
			"message":      "Multi-tenant não habilitado",
		})
	}

	stats := s.multiTenantPool.GetAllStats()
	return c.JSON(stats)
}

// handleTenantHealth retorna informações de saúde de um tenant específico
func (s *Server) handleTenantHealth(c fiber.Ctx) error {
	tenantID := c.Params("tenantId")

	if s.multiTenantPool == nil {
		return c.JSON(map[string]interface{}{
			"tenant_id":    tenantID,
			"multi_tenant": false,
			"status":       "not_applicable",
		})
	}

	if !s.multiTenantConfig.TenantExists(tenantID) {
		return c.Status(fiber.StatusNotFound).JSON(map[string]interface{}{
			"tenant_id": tenantID,
			"status":    "not_found",
			"message":   "Tenant não encontrado",
		})
	}

	provider := s.multiTenantPool.GetProvider(tenantID)
	if provider == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(map[string]interface{}{
			"tenant_id": tenantID,
			"status":    "no_provider",
			"message":   "Provider não disponível",
		})
	}

	health := map[string]interface{}{
		"tenant_id": tenantID,
		"status":    "healthy",
	}

	// Testa a conexão
	if db := provider.GetConnection(); db != nil {
		if err := db.Ping(); err != nil {
			health["status"] = "unhealthy"
			health["error"] = err.Error()
			return c.Status(fiber.StatusServiceUnavailable).JSON(health)
		}

		// Adiciona estatísticas da conexão
		stats := db.Stats()
		health["connection_stats"] = map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
		}
	}

	return c.JSON(health)
}
