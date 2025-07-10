package odata

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// EntityAuthConfig configurações de autenticação por entidade
type EntityAuthConfig struct {
	RequireAuth    bool     // Se true, todas as operações requerem autenticação
	RequiredRoles  []string // Roles necessárias para acessar a entidade
	RequiredScopes []string // Scopes necessários para acessar a entidade
	RequireAdmin   bool     // Se true, apenas administradores podem acessar
	ReadOnly       bool     // Se true, apenas operações de leitura são permitidas
}

// ServerConfig representa as configurações do servidor
type ServerConfig struct {
	// Configurações básicas
	Host string
	Port int

	// Configurações de TLS
	TLSConfig   *tls.Config
	CertFile    string
	CertKeyFile string

	// Configurações de CORS
	EnableCORS       bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool

	// Configurações de log
	EnableLogging bool
	LogLevel      string
	LogFile       string

	// Configurações de middleware
	EnableCompression bool
	MaxRequestSize    int64

	// Configurações de graceful shutdown
	ShutdownTimeout time.Duration

	// Configurações de prefixo
	RoutePrefix string

	// Configurações JWT
	EnableJWT   bool
	JWTConfig   *JWTConfig
	RequireAuth bool // Se true, todas as rotas requerem autenticação por padrão
}

// DefaultServerConfig retorna uma configuração padrão do servidor
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:              "localhost",
		Port:              8080,
		EnableCORS:        true,
		AllowedOrigins:    []string{"*"},
		AllowedMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:    []string{"*"},
		ExposedHeaders:    []string{"OData-Version", "Content-Type"},
		AllowCredentials:  false,
		EnableLogging:     true,
		LogLevel:          "INFO",
		EnableCompression: false,            // Desabilitado por padrão para evitar problemas
		MaxRequestSize:    10 * 1024 * 1024, // 10MB
		ShutdownTimeout:   30 * time.Second,
		RoutePrefix:       "/odata",
	}
}

// Server representa o servidor OData
type Server struct {
	entities   map[string]EntityService
	router     *fiber.App
	parser     *ODataParser
	urlParser  *URLParser
	provider   DatabaseProvider
	config     *ServerConfig
	httpServer *fiber.App // Changed from http.Server to fiber.App
	logger     *log.Logger
	mu         sync.RWMutex
	running    bool
	jwtService *JWTService
	entityAuth map[string]EntityAuthConfig // Configurações de autenticação por entidade
}

// NewServer cria uma nova instância do servidor OData
func NewServer(provider DatabaseProvider, host string, port int, routePrefix string) *Server {
	serviceConfig := DefaultServerConfig()
	serviceConfig.Host = host
	serviceConfig.Port = port
	serviceConfig.RoutePrefix = routePrefix
	return NewServerWithConfig(provider, serviceConfig)
}

// NewServerWithConfig cria uma nova instância do servidor OData com configurações personalizadas
func NewServerWithConfig(provider DatabaseProvider, config *ServerConfig) *Server {
	server := &Server{
		entities:   make(map[string]EntityService),
		router:     fiber.New(),
		parser:     NewODataParser(),
		urlParser:  NewURLParser(),
		provider:   provider,
		config:     config,
		logger:     log.New(os.Stdout, "[OData] ", log.LstdFlags|log.Lshortfile),
		entityAuth: make(map[string]EntityAuthConfig),
	}

	// Configurar JWT se habilitado
	if config.EnableJWT {
		server.jwtService = NewJWTService(config.JWTConfig)
		server.logger.Printf("JWT habilitado com issuer: %s", config.JWTConfig.Issuer)
	}

	// Configurar middleware apenas se habilitado
	if config.EnableCORS {
		server.router.Use(cors.New(cors.Config{
			AllowOrigins:     config.AllowedOrigins,
			AllowMethods:     config.AllowedMethods,
			AllowHeaders:     config.AllowedHeaders,
			ExposeHeaders:    config.ExposedHeaders,
			AllowCredentials: config.AllowCredentials,
		}))
	}
	if config.EnableLogging {
		server.router.Use(logger.New(logger.Config{
			Format: "${time} ${method} ${path} ${status} ${latency} ${bytesReceived} ${bytesSent}\n",
			Output: os.Stdout,
		}))
	}

	// Middleware de recovery sempre ativo para segurança
	server.router.Use(recover.New())

	server.setupBaseRoutes()

	return server
}

// setupBaseRoutes configura as rotas básicas do servidor
func (s *Server) setupBaseRoutes() {
	prefix := s.config.RoutePrefix

	// Rota para metadados
	s.router.Get(prefix+"/$metadata", s.handleMetadata)

	// Rota para service document
	s.router.Get(prefix+"/", s.handleServiceDocument)

	// Rota para health check
	s.router.Get("/health", s.handleHealth)

	// Rota para server info
	s.router.Get("/info", s.handleServerInfo)
}

// RegisterEntity registra uma entidade no servidor usando mapeamento automático
func (s *Server) RegisterEntity(name string, entity interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, err := MapEntityFromStruct(entity)
	if err != nil {
		return fmt.Errorf("erro ao registrar entidade %s: %w", name, err)
	}

	service := NewBaseEntityService(s.provider, metadata, s)
	s.entities[name] = service
	s.setupEntityRoutes(name)

	s.logger.Printf("Entidade '%s' registrada com sucesso", name)
	return nil
}

// RegisterEntityWithService registra uma entidade com um serviço customizado
func (s *Server) RegisterEntityWithService(name string, service EntityService) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entities[name] = service
	s.setupEntityRoutes(name)

	s.logger.Printf("Entidade '%s' registrada com serviço customizado", name)
	return nil
}

// AutoRegisterEntities registra múltiplas entidades automaticamente
func (s *Server) AutoRegisterEntities(entities map[string]interface{}) error {
	for name, entity := range entities {
		if err := s.RegisterEntity(name, entity); err != nil {
			return fmt.Errorf("erro ao auto-registrar entidade %s: %w", name, err)
		}
	}
	return nil
}

// setupEntityRoutes configura as rotas para uma entidade
func (s *Server) setupEntityRoutes(entityName string) {
	prefix := s.config.RoutePrefix

	// Configurar middlewares de autenticação se JWT estiver habilitado
	var authMiddleware fiber.Handler
	if s.config.EnableJWT {
		// Usar middleware de autenticação opcional para permitir acesso sem token se configurado
		authMiddleware = s.OptionalAuthMiddleware()
	}

	// Middleware para verificar autenticação específica da entidade
	entityAuthMiddleware := s.RequireEntityAuth(entityName)

	// Aplicar middlewares nas rotas
	var middlewares []fiber.Handler
	if authMiddleware != nil {
		middlewares = append(middlewares, authMiddleware)
	}
	middlewares = append(middlewares, entityAuthMiddleware)

	// Rota para coleção de entidades (GET, POST)
	getHandlers := append(middlewares, s.handleEntityCollection)
	s.router.Get(prefix+"/"+entityName, getHandlers[0], getHandlers[1:]...)

	postHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "POST"), s.handleEntityCollection)
	s.router.Post(prefix+"/"+entityName, postHandlers[0], postHandlers[1:]...)

	// Rota para entidade individual (GET, PUT, PATCH, DELETE)
	getByIdHandlers := append(middlewares, s.handleEntityById)
	s.router.Get(prefix+"/"+entityName+"({id})", getByIdHandlers[0], getByIdHandlers[1:]...)

	putHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PUT"), s.handleEntityById)
	s.router.Put(prefix+"/"+entityName+"({id})", putHandlers[0], putHandlers[1:]...)

	patchHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PATCH"), s.handleEntityById)
	s.router.Patch(prefix+"/"+entityName+"({id})", patchHandlers[0], patchHandlers[1:]...)

	deleteHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "DELETE"), s.handleEntityById)
	s.router.Delete(prefix+"/"+entityName+"({id})", deleteHandlers[0], deleteHandlers[1:]...)

	getByIdNumHandlers := append(middlewares, s.handleEntityById)
	s.router.Get(prefix+"/"+entityName+"({id:[0-9]+})", getByIdNumHandlers[0], getByIdNumHandlers[1:]...)

	putNumHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PUT"), s.handleEntityById)
	s.router.Put(prefix+"/"+entityName+"({id:[0-9]+})", putNumHandlers[0], putNumHandlers[1:]...)

	patchNumHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PATCH"), s.handleEntityById)
	s.router.Patch(prefix+"/"+entityName+"({id:[0-9]+})", patchNumHandlers[0], patchNumHandlers[1:]...)

	deleteNumHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "DELETE"), s.handleEntityById)
	s.router.Delete(prefix+"/"+entityName+"({id:[0-9]+})", deleteNumHandlers[0], deleteNumHandlers[1:]...)

	// Rota para count da coleção
	countHandlers := append(middlewares, s.handleEntityCount)
	s.router.Get(prefix+"/"+entityName+"/$count", countHandlers[0], countHandlers[1:]...)

	// Rota OPTIONS para CORS se habilitado
	if s.config.EnableCORS {
		s.router.Options(prefix+"/"+entityName, s.handleOptions)
		s.router.Options(prefix+"/"+entityName+"({id})", s.handleOptions)
		s.router.Options(prefix+"/"+entityName+"({id:[0-9]+})", s.handleOptions)
	}
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	return s.StartWithContext(context.Background())
}

// StartWithContext inicia o servidor com contexto
func (s *Server) StartWithContext(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("servidor já está rodando")
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.httpServer = s.router // Use the router as the server

	s.running = true
	s.mu.Unlock()

	// Determina se está usando HTTPS ou HTTP
	scheme := "http"
	if s.config.TLSConfig != nil || (s.config.CertFile != "" && s.config.CertKeyFile != "") {
		scheme = "https"
	}

	s.logger.Printf("🚀 Servidor OData iniciado em %s://%s", scheme, addr)
	s.logger.Printf("📋 Entidades registradas: %d", len(s.entities))
	for name := range s.entities {
		s.logger.Printf("   - %s", name)
	}
	s.logger.Printf("🔗 Endpoints disponíveis:")
	s.logger.Printf("   - Service Document: %s://%s%s/", scheme, addr, s.config.RoutePrefix)
	s.logger.Printf("   - Metadata: %s://%s%s/$metadata", scheme, addr, s.config.RoutePrefix)
	s.logger.Printf("   - Health Check: %s://%s/health", scheme, addr)
	s.logger.Printf("   - Server Info: %s://%s/info", scheme, addr)

	// Configurar shutdown graceful em goroutine separada
	go s.setupGracefulShutdown(ctx)

	// Inicia o servidor (bloqueante)
	if s.config.TLSConfig != nil || (s.config.CertFile != "" && s.config.CertKeyFile != "") {
		if s.config.CertFile != "" && s.config.CertKeyFile != "" {
			// No Fiber v3, configuramos TLS com certificado e chave
			return s.httpServer.Listen(addr, fiber.ListenConfig{
				CertFile:    s.config.CertFile,
				CertKeyFile: s.config.CertKeyFile,
			})
		}
		// Se temos TLSConfig personalizado, usamos TLSConfigFunc
		return s.httpServer.Listen(addr, fiber.ListenConfig{
			TLSConfigFunc: func(tlsConfig *tls.Config) {
				// Copia as configurações do nosso TLS config para o config do Fiber
				if s.config.TLSConfig != nil {
					tlsConfig.Certificates = s.config.TLSConfig.Certificates
					tlsConfig.ServerName = s.config.TLSConfig.ServerName
					tlsConfig.MinVersion = s.config.TLSConfig.MinVersion
					tlsConfig.MaxVersion = s.config.TLSConfig.MaxVersion
					tlsConfig.CipherSuites = s.config.TLSConfig.CipherSuites
				}
			},
		})
	}

	return s.httpServer.Listen(addr)
}

// setupGracefulShutdown configura o shutdown graceful em goroutine separada
func (s *Server) setupGracefulShutdown(ctx context.Context) {
	// Channel para capturar sinais do sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Aguarda cancelamento do contexto ou sinal do sistema
	select {
	case <-ctx.Done():
		s.logger.Printf("Contexto cancelado, parando servidor...")
	case sig := <-sigChan:
		s.logger.Printf("Sinal recebido: %v, parando servidor...", sig)
	}

	// Executa shutdown graceful
	if err := s.Shutdown(); err != nil {
		s.logger.Printf("Erro durante shutdown: %v", err)
	}
}

// Shutdown para o servidor gracefully
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("servidor não está rodando")
	}

	if s.httpServer == nil {
		return fmt.Errorf("servidor HTTP não inicializado")
	}

	s.logger.Printf("Parando servidor...")

	// Context com timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Shutdown graceful
	if err := s.httpServer.ShutdownWithContext(ctx); err != nil {
		s.logger.Printf("Erro durante shutdown: %v", err)
		return err
	}

	// Fechar provider se necessário
	if s.provider != nil {
		if err := s.provider.Close(); err != nil {
			s.logger.Printf("Erro ao fechar provider: %v", err)
		}
	}

	s.running = false
	s.logger.Printf("Servidor parado com sucesso")
	return nil
}

// IsRunning verifica se o servidor está rodando
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetConfig retorna a configuração do servidor
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// GetRouter retorna o router do servidor
func (s *Server) GetRouter() *fiber.App {
	return s.router
}

// GetHandler retorna o handler HTTP do servidor (para compatibilidade)
func (s *Server) GetHandler() *fiber.App {
	return s.router
}

// GetAddress retorna o endereço do servidor
func (s *Server) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// GetEntities retorna a lista de entidades registradas
func (s *Server) GetEntities() map[string]EntityService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities := make(map[string]EntityService)
	for name, service := range s.entities {
		entities[name] = service
	}
	return entities
}

// isOriginAllowed verifica se uma origem é permitida
func (s *Server) isOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}

	for _, allowed := range s.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// Health check handler
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

// Server info handler
func (s *Server) handleServerInfo(c fiber.Ctx) error {
	info := map[string]interface{}{
		"name":          "GoData OData Server",
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

// getEntityList retorna lista de entidades
func (s *Server) getEntityList() []string {
	var entities []string
	for name := range s.entities {
		entities = append(entities, name)
	}
	return entities
}

// handleOptions lida com requisições OPTIONS
func (s *Server) handleOptions(c fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// handleEntityCollection lida com operações na coleção de entidades
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

// handleEntityById lida com operações em uma entidade específica
func (s *Server) handleEntityById(c fiber.Ctx) error {
	entityName := s.extractEntityName(c.Path())
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(c, fiber.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return nil
	}

	// Extrai as chaves da URL
	keys, err := s.extractKeys(c.Path(), service.GetMetadata())
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidKey", err.Error())
		return nil
	}

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

// handleGetCollection lida com GET na coleção de entidades
func (s *Server) handleGetCollection(c fiber.Ctx, service EntityService) error {
	var queryValues url.Values
	var err error

	// Extrai o nome da entidade para logging
	entityName := s.extractEntityName(c.Path())

	queryString := string(c.Request().URI().QueryString())
	queryValuesURL, parseErr := s.urlParser.ParseQueryFast(queryString)
	if parseErr != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Failed to parse query: %v", parseErr))
		return nil
	}
	queryValues = queryValuesURL

	// Valida a query OData otimizada
	if err := s.urlParser.ValidateODataQueryFast(queryString); err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Invalid OData query: %v", err))
		return nil
	}

	// Parse das opções de consulta
	options, err := s.parser.ParseQueryOptions(queryValues)
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Valida as opções
	if err := s.parser.ValidateQueryOptions(options); err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Log da consulta para debug
	s.logger.Printf("🔍 Executando consulta para entidade: %s", entityName)
	if options.Expand != nil {
		s.logger.Printf("🔍 Expand solicitado: %v", options.Expand)
	}
	if options.Filter != nil {
		s.logger.Printf("🔍 Filtro aplicado: %s", options.Filter.RawValue)
	}

	// Executa a consulta
	response, err := service.Query(c.Context(), options)
	if err != nil {
		s.logger.Printf("❌ Erro na consulta: %v", err)
		s.writeError(c, fiber.StatusInternalServerError, "QueryError", err.Error())
		return nil
	}

	s.logger.Printf("✅ Consulta executada com sucesso")
	return c.JSON(response)
}

// handleGetEntity lida com GET de uma entidade específica
func (s *Server) handleGetEntity(c fiber.Ctx, service EntityService, keys map[string]interface{}) error {
	entity, err := service.Get(c.Context(), keys)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(c, fiber.StatusNotFound, "EntityNotFound", err.Error())
		} else {
			s.writeError(c, fiber.StatusInternalServerError, "QueryError", err.Error())
		}
		return nil
	}

	return c.JSON(entity)
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

// handleMetadata lida com GET dos metadados
func (s *Server) handleMetadata(c fiber.Ctx) error {
	metadata := s.buildMetadataJSON()
	return c.JSON(metadata)
}

// handleServiceDocument lida com GET do documento de serviço
func (s *Server) handleServiceDocument(c fiber.Ctx) error {
	serviceDoc := map[string]interface{}{
		"@odata.context": "$metadata",
		"value":          s.buildEntitySets(),
	}

	return c.JSON(serviceDoc)
}

// handleEntityCount lida com GET do count de uma coleção de entidades
func (s *Server) handleEntityCount(c fiber.Ctx) error {
	entityName := s.extractEntityName(c.Path())
	service, exists := s.entities[entityName]
	if !exists {
		s.writeError(c, fiber.StatusNotFound, "EntityNotFound", fmt.Sprintf("Entity '%s' not found", entityName))
		return nil
	}

	// Parse das opções de consulta usando o parser customizado
	queryString := string(c.Request().URI().QueryString())
	queryValues, err := s.urlParser.ParseQuery(queryString)
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Failed to parse query: %v", err))
		return nil
	}

	// Valida a query OData
	if err := s.urlParser.ValidateODataQuery(queryString); err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", fmt.Sprintf("Invalid OData query: %v", err))
		return nil
	}

	// Parse das opções de consulta (principalmente $filter)
	options, err := s.parser.ParseQueryOptions(queryValues)
	if err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Valida as opções
	if err := s.parser.ValidateQueryOptions(options); err != nil {
		s.writeError(c, fiber.StatusBadRequest, "InvalidQuery", err.Error())
		return nil
	}

	// Obtém a contagem usando o método específico
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

// extractEntityName extrai o nome da entidade da URL
func (s *Server) extractEntityName(path string) string {
	// Remove o prefixo da rota
	prefix := s.config.RoutePrefix
	if strings.HasPrefix(path, prefix+"/") {
		path = strings.TrimPrefix(path, prefix+"/")
	}

	// Remove parâmetros de ID se presentes
	if idx := strings.Index(path, "("); idx != -1 {
		path = path[:idx]
	}

	// Remove $count se presente
	if strings.HasSuffix(path, "/$count") {
		path = strings.TrimSuffix(path, "/$count")
	}

	return path
}

// extractKeys extrai as chaves da URL para operações em entidades específicas
func (s *Server) extractKeys(path string, metadata EntityMetadata) (map[string]interface{}, error) {
	keys := make(map[string]interface{})

	// Encontra a parte entre parênteses
	start := strings.Index(path, "(")
	end := strings.LastIndex(path, ")")
	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("invalid key format in path: %s", path)
	}

	keyString := path[start+1 : end]

	// Identifica as chaves primárias dos metadados
	var primaryKeys []PropertyMetadata
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			primaryKeys = append(primaryKeys, prop)
		}
	}

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

	return keys, nil
}

// parseKeyValue converte uma string em valor do tipo apropriado
func (s *Server) parseKeyValue(value, dataType string) (interface{}, error) {
	// Remove aspas se presentes
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		value = value[1 : len(value)-1]
	}

	switch dataType {
	case "string":
		return value, nil
	case "int32", "int":
		return strconv.Atoi(value)
	case "int64":
		return strconv.ParseInt(value, 10, 64)
	case "float32":
		val, err := strconv.ParseFloat(value, 32)
		return float32(val), err
	case "float64":
		return strconv.ParseFloat(value, 64)
	case "bool":
		return strconv.ParseBool(value)
	default:
		return value, nil
	}
}

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

// writeJSON escreve uma resposta JSON
func (s *Server) writeJSON(c fiber.Ctx, data interface{}) error {
	c.Set("Content-Type", "application/json")
	c.Set("OData-Version", "4.0")

	return c.JSON(data)
}

// writeError escreve uma resposta de erro
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

// parseQueryOptionsWithCustomParser faz o parsing das opções usando o parser customizado
func (s *Server) parseQueryOptionsWithCustomParser(c fiber.Ctx) (QueryOptions, error) {
	// Parse usando o parser customizado
	queryString := string(c.Request().URI().QueryString())
	queryValues, err := s.urlParser.ParseQuery(queryString)
	if err != nil {
		return QueryOptions{}, fmt.Errorf("failed to parse query: %w", err)
	}

	// Valida a query OData
	if err := s.urlParser.ValidateODataQuery(queryString); err != nil {
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
func (s *Server) debugQueryParsing(c fiber.Ctx) {
	log.Printf("🔍 DEBUG Query Parsing:")
	log.Printf("   Raw Query: %s", string(c.Request().URI().QueryString()))
	log.Printf("   Standard Query: %v", c.Queries())

	// Testa o parser customizado
	customValues, err := s.urlParser.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		log.Printf("   Custom Parser Error: %v", err)
	} else {
		log.Printf("   Custom Query: %v", customValues)
	}

	// Compara os resultados
	log.Printf("   Differences:")
	for key, vals := range customValues {
		standardVal := c.Query(key)
		if len(vals) > 0 && vals[0] != standardVal {
			log.Printf("     %s: standard=%v, custom=%v", key, standardVal, vals)
		}
	}
}
