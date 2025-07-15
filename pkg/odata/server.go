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
	fiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
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
	entities          map[string]EntityService
	router            *fiber.App
	parser            *ODataParser
	urlParser         *URLParser
	provider          DatabaseProvider         // Provider padrão
	multiTenantPool   *MultiTenantProviderPool // Pool multi-tenant
	multiTenantConfig *MultiTenantConfig       // Configurações multi-tenant
	config            *ServerConfig
	httpServer        *fiber.App // Changed from http.Server to fiber.App
	logger            *log.Logger
	mu                sync.RWMutex
	running           bool
	jwtService        *JWTService
	entityAuth        map[string]EntityAuthConfig // Configurações de autenticação por entidade
	eventManager      *EntityEventManager         // Gerenciador de eventos de entidade
}

// NewServer cria uma nova instância do servidor OData
// Carrega automaticamente configurações multi-tenant do .env
// Se não conseguir, retorna um servidor básico para configuração manual
func NewServer() *Server {
	// Carrega configurações multi-tenant automaticamente
	multiTenantConfig := LoadMultiTenantConfig()

	// Se multi-tenant estiver habilitado, cria servidor multi-tenant
	if multiTenantConfig.Enabled {
		return NewMultiTenantServer(multiTenantConfig)
	}

	// Se não está em modo multi-tenant, usa o comportamento original
	if multiTenantConfig.EnvConfig != nil {
		provider := multiTenantConfig.EnvConfig.CreateProviderFromConfig()
		if provider == nil {
			return NewServerWithConfig(nil, multiTenantConfig.EnvConfig.ToServerConfig())
		}
		return NewServerWithConfig(provider, multiTenantConfig.EnvConfig.ToServerConfig())
	}

	return NewServerWithConfig(nil, DefaultServerConfig())
}

// NewServerWithProvider cria servidor com provider específico (mantido para compatibilidade)
// Carrega automaticamente configurações multi-tenant do .env
func NewServerWithProvider(provider DatabaseProvider, host string, port int, routePrefix string) *Server {
	// Carrega configurações multi-tenant automaticamente
	multiTenantConfig := LoadMultiTenantConfig()

	// Se multi-tenant estiver habilitado, cria servidor multi-tenant e ignora o provider fornecido
	if multiTenantConfig.Enabled {
		server := NewMultiTenantServer(multiTenantConfig)
		// Sobrescreve configurações básicas do servidor
		server.config.Host = host
		server.config.Port = port
		server.config.RoutePrefix = routePrefix
		return server
	}

	// Se não está em modo multi-tenant, usa o comportamento original
	serviceConfig := DefaultServerConfig()
	serviceConfig.Host = host
	serviceConfig.Port = port
	serviceConfig.RoutePrefix = routePrefix
	return NewServerWithConfig(provider, serviceConfig)
}

// NewMultiTenantServer cria um servidor multi-tenant
func NewMultiTenantServer(multiTenantConfig *MultiTenantConfig) *Server {
	logger := log.New(os.Stdout, "[OData-MultiTenant] ", log.LstdFlags|log.Lshortfile)

	server := &Server{
		entities:          make(map[string]EntityService),
		router:            fiber.New(),
		parser:            NewODataParser(),
		urlParser:         NewURLParser(),
		multiTenantConfig: multiTenantConfig,
		config:            multiTenantConfig.EnvConfig.ToServerConfig(),
		logger:            logger,
		entityAuth:        make(map[string]EntityAuthConfig),
		eventManager:      NewEntityEventManager(logger),
	}

	// Inicializa pool multi-tenant
	server.multiTenantPool = NewMultiTenantProviderPool(multiTenantConfig, logger)
	if err := server.multiTenantPool.InitializeProviders(); err != nil {
		logger.Printf("❌ Erro ao inicializar pool multi-tenant: %v", err)
	}

	// Configura middlewares específicos para multi-tenant
	server.setupMultiTenantMiddlewares()
	server.setupBaseRoutes()

	// Imprime informações sobre configuração multi-tenant
	multiTenantConfig.PrintMultiTenantConfig()

	return server
}

// NewServerWithEnv cria uma nova instância do servidor OData carregando configurações do .env
func NewServerWithEnv(provider DatabaseProvider) *Server {
	config, err := LoadEnvOrDefault()
	if err != nil {
		log.Printf("Aviso: Não foi possível carregar configurações do .env: %v", err)
		return NewServerWithConfig(provider, DefaultServerConfig())
	}

	// Imprime configurações carregadas
	config.PrintLoadedConfig()

	return NewServerWithConfig(provider, config.ToServerConfig())
}

// NewServerWithConfig cria uma nova instância do servidor OData com configurações personalizadas
func NewServerWithConfig(provider DatabaseProvider, config *ServerConfig) *Server {
	logger := log.New(os.Stdout, "[OData] ", log.LstdFlags|log.Lshortfile)

	server := &Server{
		entities:     make(map[string]EntityService),
		router:       fiber.New(),
		parser:       NewODataParser(),
		urlParser:    NewURLParser(),
		provider:     provider,
		config:       config,
		logger:       logger,
		entityAuth:   make(map[string]EntityAuthConfig),
		eventManager: NewEntityEventManager(logger),
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
		server.router.Use(fiberlogger.New(fiberlogger.Config{
			Format: "${time} ${method} ${path} ${status} ${latency} ${bytesReceived} ${bytesSent}\n",
			Output: os.Stdout,
		}))
	}

	// Middleware de recovery sempre ativo para segurança
	server.router.Use(recover.New())

	server.setupBaseRoutes()

	return server
}

// setupMultiTenantMiddlewares configura middlewares específicos para multi-tenant
func (s *Server) setupMultiTenantMiddlewares() {
	// Middleware de identificação de tenant (deve ser o primeiro)
	s.router.Use(s.TenantMiddleware())

	// Middleware de informações do tenant
	s.router.Use(s.TenantInfo())

	// Demais middlewares...
	if s.config.EnableCORS {
		s.router.Use(cors.New(cors.Config{
			AllowOrigins:     s.config.AllowedOrigins,
			AllowMethods:     s.config.AllowedMethods,
			AllowHeaders:     s.config.AllowedHeaders,
			ExposeHeaders:    s.config.ExposedHeaders,
			AllowCredentials: s.config.AllowCredentials,
		}))
	}

	if s.config.EnableLogging {
		s.router.Use(fiberlogger.New(fiberlogger.Config{
			Format: "${time} ${method} ${path} ${status} ${latency} [${locals:tenant_id}]\n",
			Output: os.Stdout,
		}))
	}

	s.router.Use(recover.New())
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

	// Rotas específicas para multi-tenant
	if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
		// Rota para informações dos tenants
		s.router.Get("/tenants", s.handleTenantList)

		// Rota para estatísticas dos tenants
		s.router.Get("/tenants/stats", s.handleTenantStats)

		// Rota para health check específico de tenant
		s.router.Get("/tenants/:tenantId/health", s.handleTenantHealth)
	}
}

// RegisterEntity registra uma entidade no servidor usando mapeamento automático
func (s *Server) RegisterEntity(name string, entity interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata, err := MapEntityFromStruct(entity)
	if err != nil {
		return fmt.Errorf("erro ao registrar entidade %s: %w", name, err)
	}

	var service EntityService

	// Se multi-tenant estiver habilitado, usa MultiTenantEntityService
	if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
		service = NewMultiTenantEntityService(metadata, s)
		s.logger.Printf("Entidade '%s' registrada com suporte multi-tenant", name)
	} else {
		service = NewBaseEntityService(s.provider, metadata, s)
		s.logger.Printf("Entidade '%s' registrada com provider único", name)
	}

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
	// Usando padrão wildcard para capturar URLs como /odata/FabTarefa(53)
	getByIdHandlers := append(middlewares, s.handleEntityById)
	s.router.Get(prefix+"/"+entityName+"(*)", getByIdHandlers[0], getByIdHandlers[1:]...)

	putHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PUT"), s.handleEntityById)
	s.router.Put(prefix+"/"+entityName+"(*)", putHandlers[0], putHandlers[1:]...)

	patchHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "PATCH"), s.handleEntityById)
	s.router.Patch(prefix+"/"+entityName+"(*)", patchHandlers[0], patchHandlers[1:]...)

	deleteHandlers := append(middlewares, s.CheckEntityReadOnly(entityName, "DELETE"), s.handleEntityById)
	s.router.Delete(prefix+"/"+entityName+"(*)", deleteHandlers[0], deleteHandlers[1:]...)

	// Rota para count da coleção
	countHandlers := append(middlewares, s.handleEntityCount)
	s.router.Get(prefix+"/"+entityName+"/$count", countHandlers[0], countHandlers[1:]...)

	// Rota OPTIONS para CORS se habilitado
	if s.config.EnableCORS {
		s.router.Options(prefix+"/"+entityName, s.handleOptions)
		s.router.Options(prefix+"/"+entityName+"(*)", s.handleOptions)
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

// GetEventManager retorna o gerenciador de eventos
func (s *Server) GetEventManager() *EntityEventManager {
	return s.eventManager
}

// OnEntityGet registra um handler para o evento EntityGet
func (s *Server) OnEntityGet(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityGet, entityName, handler)
}

// OnEntityList registra um handler para o evento EntityList
func (s *Server) OnEntityList(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityList, entityName, handler)
}

// OnEntityInserting registra um handler para o evento EntityInserting
func (s *Server) OnEntityInserting(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityInserting, entityName, handler)
}

// OnEntityInserted registra um handler para o evento EntityInserted
func (s *Server) OnEntityInserted(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityInserted, entityName, handler)
}

// OnEntityModifying registra um handler para o evento EntityModifying
func (s *Server) OnEntityModifying(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityModifying, entityName, handler)
}

// OnEntityModified registra um handler para o evento EntityModified
func (s *Server) OnEntityModified(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityModified, entityName, handler)
}

// OnEntityDeleting registra um handler para o evento EntityDeleting
func (s *Server) OnEntityDeleting(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityDeleting, entityName, handler)
}

// OnEntityDeleted registra um handler para o evento EntityDeleted
func (s *Server) OnEntityDeleted(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityDeleted, entityName, handler)
}

// OnEntityError registra um handler para o evento EntityError
func (s *Server) OnEntityError(entityName string, handler func(args EventArgs) error) {
	s.eventManager.SubscribeFunc(EventEntityError, entityName, handler)
}

// OnEntityGetGlobal registra um handler global para o evento EntityGet
func (s *Server) OnEntityGetGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityGet, handler)
}

// OnEntityListGlobal registra um handler global para o evento EntityList
func (s *Server) OnEntityListGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityList, handler)
}

// OnEntityInsertingGlobal registra um handler global para o evento EntityInserting
func (s *Server) OnEntityInsertingGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityInserting, handler)
}

// OnEntityInsertedGlobal registra um handler global para o evento EntityInserted
func (s *Server) OnEntityInsertedGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityInserted, handler)
}

// OnEntityModifyingGlobal registra um handler global para o evento EntityModifying
func (s *Server) OnEntityModifyingGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityModifying, handler)
}

// OnEntityModifiedGlobal registra um handler global para o evento EntityModified
func (s *Server) OnEntityModifiedGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityModified, handler)
}

// OnEntityDeletingGlobal registra um handler global para o evento EntityDeleting
func (s *Server) OnEntityDeletingGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityDeleting, handler)
}

// OnEntityDeletedGlobal registra um handler global para o evento EntityDeleted
func (s *Server) OnEntityDeletedGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityDeleted, handler)
}

// OnEntityErrorGlobal registra um handler global para o evento EntityError
func (s *Server) OnEntityErrorGlobal(handler func(args EventArgs) error) {
	s.eventManager.SubscribeGlobalFunc(EventEntityError, handler)
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

// handleGetCollection lida com GET na coleção de entidades
func (s *Server) handleGetCollection(c fiber.Ctx, service EntityService) error {
	// Cria contexto com referência ao Fiber Context para multi-tenant
	ctx := context.WithValue(c.Context(), FiberContextKey, c)

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

// handleGetEntity lida com GET de uma entidade específica
func (s *Server) handleGetEntity(c fiber.Ctx, service EntityService, keys map[string]interface{}) error {
	s.logger.Printf("🔍 handleGetEntity - Starting with keys: %+v", keys)

	// Log dos tipos das chaves para debug
	for k, v := range keys {
		s.logger.Printf("🔍 handleGetEntity - Key '%s': value=%v, type=%T", k, v, v)
	}

	// Cria contexto com referência ao Fiber Context para multi-tenant
	ctx := context.WithValue(c.Context(), FiberContextKey, c)

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
		// Se é um map, copia todas as propriedades
		for key, value := range entityMap {
			response[key] = value
		}
	} else {
		// Para outros tipos, tenta converter usando reflection
		// Mas por enquanto, apenas adiciona como "data"
		response["data"] = entity
	}

	return response
}

// parseQueryOptions centraliza o parse das opções de consulta OData
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

// getCurrentProvider retorna o provider para o tenant atual
func (s *Server) getCurrentProvider(c fiber.Ctx) DatabaseProvider {
	if s.multiTenantPool == nil {
		return s.provider
	}

	tenantID := GetCurrentTenant(c)
	return s.multiTenantPool.GetProvider(tenantID)
}

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
