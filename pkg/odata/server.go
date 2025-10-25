package odata

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/kardianos/service"
)

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
	entityAuth        map[string]EntityAuthConfig // Configurações de autenticação por entidade
	eventManager      *EntityEventManager         // Gerenciador de eventos de entidade
	rateLimiter       *RateLimiter                // Rate limiter
	auditLogger       AuditLogger                 // Audit logger

	// Campos para gerenciamento de serviço
	serviceLogger service.Logger
	serviceCtx    context.Context
	serviceCancel context.CancelFunc
}

// NewServer cria uma nova instância do servidor OData
// Carrega automaticamente configurações multi-tenant do .env
// Se não conseguir, retorna um servidor básico para configuração manual
func NewServer() *Server {
	// Carrega configurações multi-tenant automaticamente
	multiTenantConfig := LoadMultiTenantConfig()

	// Se multi-tenant estiver habilitado, cria servidor multi-tenant
	if multiTenantConfig.Enabled {
		return newMultiTenantServer(multiTenantConfig)
	}

	// Se não está em modo multi-tenant, usa o comportamento original
	if multiTenantConfig.EnvConfig != nil {
		provider := multiTenantConfig.EnvConfig.CreateProviderFromConfig()
		if provider == nil {
			return newServerWithConfig(nil, multiTenantConfig.EnvConfig.ToServerConfig())
		}
		return newServerWithConfig(provider, multiTenantConfig.EnvConfig.ToServerConfig())
	}

	return newServerWithConfig(nil, DefaultServerConfig())
}

// NewServerWithProvider cria servidor com provider específico (mantido para compatibilidade)
// Carrega automaticamente configurações multi-tenant do .env
func NewServerWithProvider(provider DatabaseProvider, host string, port int, routePrefix string) *Server {
	// Carrega configurações multi-tenant automaticamente
	multiTenantConfig := LoadMultiTenantConfig()

	// Se multi-tenant estiver habilitado, cria servidor multi-tenant e ignora o provider fornecido
	if multiTenantConfig.Enabled {
		server := newMultiTenantServer(multiTenantConfig)
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
	return newServerWithConfig(provider, serviceConfig)
}

// NewMultiTenantServer cria um servidor multi-tenant
func newMultiTenantServer(multiTenantConfig *MultiTenantConfig) *Server {
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
		return newServerWithConfig(provider, DefaultServerConfig())
	}

	return newServerWithConfig(provider, config.ToServerConfig())
}

// newServerWithConfig cria uma nova instância do servidor OData com configurações personalizadas
func newServerWithConfig(provider DatabaseProvider, config *ServerConfig) *Server {
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

	// Configurar Rate Limit se habilitado
	if config.RateLimitConfig != nil && config.RateLimitConfig.Enabled {
		server.rateLimiter = NewRateLimiter(config.RateLimitConfig)
		server.logger.Printf("Rate limit habilitado: %d req/min, burst: %d",
			config.RateLimitConfig.RequestsPerMinute, config.RateLimitConfig.BurstSize)
	}

	// Configurar Audit Logger
	if config.AuditLogConfig != nil {
		auditLogger, err := NewAuditLogger(config.AuditLogConfig)
		if err != nil {
			server.logger.Printf("⚠️  Erro ao criar audit logger: %v (continuando sem audit logging)", err)
			server.auditLogger = &NoOpAuditLogger{}
		} else {
			server.auditLogger = auditLogger
			if config.AuditLogConfig.Enabled {
				server.logger.Printf("✅ Audit logging habilitado: tipo=%s, formato=%s",
					config.AuditLogConfig.LogType, config.AuditLogConfig.Format)
			}
		}
	} else {
		server.auditLogger = &NoOpAuditLogger{}
	}

	// Configurar Security Headers se habilitado
	if config.SecurityHeadersConfig != nil && config.SecurityHeadersConfig.Enabled {
		server.router.Use(SecurityHeadersMiddleware(config.SecurityHeadersConfig))
		server.logger.Printf("✅ Security headers habilitados")
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
	server.router.Use(fiberrecover.New())

	// Middleware que injeta o servidor no contexto Fiber
	server.router.Use(func(c fiber.Ctx) error {
		c.Locals("odata_server", server)
		return c.Next()
	})

	// Middleware de conexão de banco de dados (transparente)
	server.router.Use(server.DatabaseMiddleware())

	// Middleware de rate limit se habilitado
	if server.rateLimiter != nil {
		server.router.Use(server.RateLimitMiddleware())
	}

	server.setupBaseRoutes()

	return server
}

// setupMultiTenantMiddlewares configura middlewares específicos para multi-tenant

// setupBaseRoutes configura as rotas básicas do servidor

// RegisterEntity registra uma entidade no servidor usando mapeamento automático
// Aceita EntityOptions para configuração adicional (como WithAuth, WithReadOnly)
func (s *Server) RegisterEntity(name string, entity interface{}, opts ...EntityOption) error {
	// Cria configuração da entidade
	config := &EntityConfig{
		Name:   name,
		Entity: entity,
	}

	// Aplica options
	for _, opt := range opts {
		opt(config)
	}

	metadata, err := MapEntityFromStruct(entity)
	if err != nil {
		return fmt.Errorf("erro ao registrar entidade %s: %w", name, err)
	}

	var service EntityService

	// Se multi-tenant estiver habilitado, usa MultiTenantEntityService
	if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
		service = NewMultiTenantEntityService(metadata, s)
	} else {
		service = NewBaseEntityService(s.provider, metadata, s)
	}

	// Bloco com lock APENAS para modificar mapas
	s.mu.Lock()
	s.entities[name] = service

	// Armazena configuração de autenticação/permissões/middlewares se especificado
	if len(config.Middlewares) > 0 || config.ReadOnly || len(config.Permissions) > 0 {
		s.entityAuth[name] = EntityAuthConfig{
			RequireAuth: len(config.Middlewares) > 0,
			ReadOnly:    config.ReadOnly,
			Middlewares: config.Middlewares,
			Permissions: config.Permissions,
		}
	}
	s.mu.Unlock()

	// Configura rotas FORA do lock para evitar deadlock
	s.setupEntityRoutes(name)

	return nil
}

// RegisterEntityWithService registra uma entidade com um serviço customizado
func (s *Server) RegisterEntityWithService(name string, service EntityService) error {
	s.mu.Lock()
	s.entities[name] = service
	s.mu.Unlock()

	// Configura rotas FORA do lock para evitar deadlock
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

// Start inicia o servidor HTTP
// Detecta automaticamente se deve executar como serviço ou normalmente
func (s *Server) Start() error {
	// Detecta se está sendo executado como serviço
	if s.IsRunningAsService() {
		s.logger.Printf("🔧 Detectado execução como serviço, iniciando com RunService...")
		return s.run()
	}

	s.logger.Printf("🔧 Detectado execução normal, iniciando servidor HTTP...")
	return s.startWithContext(context.Background())
}

// StartWithContext inicia o servidor com contexto
func (s *Server) startWithContext(ctx context.Context) error {
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
	s.logger.Println("")

	// Imprimir configurações carregadas
	s.printServerConfig()

	// Imprimir middlewares ativos
	s.printActiveMiddlewares()

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

// =================================================================================================
// IMPLEMENTAÇÃO DOS METODOS PARA EXECUTAR O SERVIDOR COMO SERVIÇO
// =================================================================================================

// Run executa o servidor como serviço (usado pelo gerenciador de serviços)
func (s *Server) run() error {
	wrapper := &ServiceWrapper{server: s}
	svc, err := service.New(wrapper, s.createServiceConfig())
	if err != nil {
		return fmt.Errorf("erro ao criar serviço: %w", err)
	}

	// Configura logger do serviço
	s.serviceLogger, err = svc.Logger(nil)
	if err != nil {
		s.logger.Printf("Aviso: Não foi possível configurar logger do serviço: %v", err)
	}

	err = svc.Run()
	if err != nil {
		if s.serviceLogger != nil {
			s.serviceLogger.Error(err)
		}
		return err
	}

	return nil
}

// Stop para o servidor gracefully
// Unifica StopService e Shutdown em um único método
func (s *Server) Stop() error {
	// Se está rodando como serviço, usa a lógica de serviço
	if s.serviceCancel != nil {
		if s.serviceLogger != nil {
			s.serviceLogger.Info("⏹️ Parando serviço GoData...")
		}

		// Cancela o contexto para sinalizar shutdown
		s.serviceCancel()

		// Aguarda um tempo para shutdown graceful
		time.Sleep(2 * time.Second)
	}

	// Para o servidor HTTP
	return s.Shutdown()
}

// Restart reinicia o serviço do sistema
func (s *Server) Restart() error {
	wrapper := &ServiceWrapper{server: s}
	svc, err := service.New(wrapper, s.createServiceConfig())
	if err != nil {
		return fmt.Errorf("erro ao criar serviço: %w", err)
	}

	s.logger.Printf("🔄 Reiniciando serviço '%s'...", s.config.Name)

	// Para o serviço
	if err := svc.Stop(); err != nil {
		s.logger.Printf("Aviso: Erro ao parar serviço: %v", err)
	}

	// Aguarda um momento
	time.Sleep(3 * time.Second)

	// Inicia o serviço
	if err := svc.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar serviço: %w", err)
	}

	s.logger.Printf("✅ Serviço '%s' reiniciado com sucesso!", s.config.Name)
	return nil
}

// Status retorna o status do serviço do sistema
func (s *Server) Status() (service.Status, error) {
	wrapper := &ServiceWrapper{server: s}
	svc, err := service.New(wrapper, s.createServiceConfig())
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("erro ao criar serviço: %w", err)
	}

	status, err := svc.Status()
	if err != nil {
		return service.StatusUnknown, fmt.Errorf("erro ao verificar status: %w", err)
	}

	var statusText string
	switch status {
	case service.StatusRunning:
		statusText = "🟢 Executando"
	case service.StatusStopped:
		statusText = "🔴 Parado"
	case service.StatusUnknown:
		statusText = "❓ Desconhecido"
	default:
		statusText = "❓ Status desconhecido"
	}

	s.logger.Printf("📊 Status do serviço '%s': %s", s.config.Name, statusText)
	return status, nil
}

// Install instala o servidor como serviço do sistema
func (s *Server) Install() error {
	wrapper := &ServiceWrapper{server: s}
	svc, err := service.New(wrapper, s.createServiceConfig())
	if err != nil {
		return fmt.Errorf("erro ao criar serviço: %w", err)
	}

	// Configura logger do serviço
	s.serviceLogger, err = svc.Logger(nil)
	if err != nil {
		s.logger.Printf("Aviso: Não foi possível configurar logger do serviço: %v", err)
	}

	err = svc.Install()
	if err != nil {
		return fmt.Errorf("erro ao instalar serviço: %w", err)
	}

	s.logger.Printf("✅ Serviço '%s' instalado com sucesso!", s.config.Name)
	return nil
}

// Uninstall remove o servidor como serviço do sistema
func (s *Server) Uninstall() error {
	wrapper := &ServiceWrapper{server: s}
	svc, err := service.New(wrapper, s.createServiceConfig())
	if err != nil {
		return fmt.Errorf("erro ao criar serviço: %w", err)
	}

	// Tenta parar o serviço antes de desinstalar
	if status, _ := svc.Status(); status == service.StatusRunning {
		s.logger.Println("⏹️ Parando serviço antes de desinstalar...")
		if err := svc.Stop(); err != nil {
			s.logger.Printf("Aviso: Erro ao parar serviço: %v", err)
		}
		time.Sleep(2 * time.Second)
	}

	err = svc.Uninstall()
	if err != nil {
		return fmt.Errorf("erro ao desinstalar serviço: %w", err)
	}

	s.logger.Printf("✅ Serviço '%s' removido com sucesso!", s.config.Name)
	return nil
}

// createServiceConfig cria a configuração do serviço baseada na configuração do servidor
func (s *Server) createServiceConfig() *service.Config {
	svcConfig := &service.Config{
		Name:        s.config.Name,
		DisplayName: s.config.DisplayName,
		Description: s.config.Description,
		Arguments:   []string{"run"},
	}

	// Adiciona configurações específicas por plataforma
	if runtime.GOOS == "windows" {
		svcConfig.Dependencies = []string{
			"Tcpip",
			"Dhcp",
		}
		svcConfig.Option = service.KeyValue{
			"StartType":              "automatic",
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "5s",
			"OnFailureResetPeriod":   10,
		}
	} else {
		// Linux/Unix
		svcConfig.Dependencies = []string{
			"Requires=network.target",
			"After=network-online.target syslog.target",
		}
		svcConfig.Option = service.KeyValue{
			"Restart":        "always",
			"RestartSec":     "5",
			"User":           "godata",
			"Group":          "godata",
			"LimitNOFILE":    "65536",
			"Type":           "notify",
			"KillMode":       "mixed",
			"TimeoutStopSec": "30",
		}
	}

	return svcConfig
}

// printActiveMiddlewares imprime os middlewares globais ativos
func (s *Server) printActiveMiddlewares() {
	s.logger.Println("⚙️  Middlewares Globais Ativos:")

	middlewares := []string{}

	// Middlewares sempre ativos
	middlewares = append(middlewares, "✅ Recover (panic recovery)")
	middlewares = append(middlewares, "✅ Server Context Injection")
	middlewares = append(middlewares, "✅ Database Connection")

	// Middlewares condicionais
	if s.config.EnableCORS {
		middlewares = append(middlewares, fmt.Sprintf("✅ CORS (Origins: %v)", s.config.AllowedOrigins))
	}
	if s.config.EnableLogging {
		middlewares = append(middlewares, "✅ Logger (HTTP request logging)")
	}
	if s.rateLimiter != nil {
		middlewares = append(middlewares, fmt.Sprintf("✅ Rate Limit (%d req/min, burst: %d)",
			s.config.RateLimitConfig.RequestsPerMinute, s.config.RateLimitConfig.BurstSize))
	}
	if s.config.SecurityHeadersConfig != nil && s.config.SecurityHeadersConfig.Enabled {
		middlewares = append(middlewares, "✅ Security Headers")
	}
	if s.auditLogger != nil && s.config.AuditLogConfig != nil && s.config.AuditLogConfig.Enabled {
		middlewares = append(middlewares, fmt.Sprintf("✅ Audit Logger (tipo: %s)", s.config.AuditLogConfig.LogType))
	}

	// Multi-tenant
	if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
		middlewares = append(middlewares, "✅ Multi-Tenant (tenant detection)")
	}

	for _, mw := range middlewares {
		s.logger.Printf("   %s", mw)
	}
}

// printServerConfig imprime as configurações do servidor
func (s *Server) printServerConfig() {
	s.logger.Println("📋 Configurações carregadas:")

	// Informações do banco de dados
	if s.provider != nil {
		s.logger.Printf("   Database: %s", s.provider.GetDriverName())
	} else if s.multiTenantPool != nil {
		s.logger.Println("   Database: Multi-Tenant")
	}

	// Informações do servidor
	s.logger.Printf("   Server: %s:%d%s", s.config.Host, s.config.Port, s.config.RoutePrefix)
	s.logger.Printf("   CORS: %v", s.config.EnableCORS)

	// Informações JWT (se houver)
	if len(os.Getenv("JWT_SECRET_KEY")) > 0 {
		s.logger.Printf("   JWT: true")
		if issuer := os.Getenv("JWT_ISSUER"); issuer != "" {
			s.logger.Printf("   JWT Issuer: %s", issuer)
		}
	} else {
		s.logger.Printf("   JWT: false")
	}

	// TLS
	tlsEnabled := s.config.TLSConfig != nil || (s.config.CertFile != "" && s.config.CertKeyFile != "")
	s.logger.Printf("   TLS: %v", tlsEnabled)
	s.logger.Println("")
}

// isSystemRoute verifica se o path é uma rota de sistema
func (s *Server) isSystemRoute(path string) bool {
	systemPaths := []string{"/health", "/info", "/$metadata", s.config.RoutePrefix + "/", "/"}
	for _, sp := range systemPaths {
		if path == sp || path == s.config.RoutePrefix+sp {
			return true
		}
	}
	return false
}

// isEntityRoute verifica se o path é uma rota de entidade OData
func (s *Server) isEntityRoute(path string) bool {
	for name := range s.entities {
		entityPath := fmt.Sprintf("%s/%s", s.config.RoutePrefix, name)

		// Verifica se o path corresponde a qualquer padrão de rota de entidade
		if path == entityPath || // Coleção (GET, POST)
			path == entityPath+"(*)" || // Entidade individual (GET, PUT, PATCH, DELETE) - Fiber syntax
			path == entityPath+"/:id" || // Entidade individual alternativo
			path == entityPath+"(/:id)" || // OData syntax
			path == entityPath+"/$count" || // Contagem OData
			strings.HasPrefix(path, entityPath+"/") { // Qualquer subrota da entidade
			return true
		}
	}
	return false
}

// isProtectedRoute tenta detectar se uma rota é protegida (heurística)
func (s *Server) isProtectedRoute(path string) bool {
	// Heurística: rotas com /auth/ normalmente são de autenticação
	// mas podem não ser protegidas (como /auth/login)
	// Rotas protegidas geralmente incluem /me, /profile, etc
	protectedPaths := []string{"/me", "/profile", "/dashboard"}
	for _, pp := range protectedPaths {
		if len(path) >= len(pp) && path[len(path)-len(pp):] == pp {
			return true
		}
	}
	return false
}
