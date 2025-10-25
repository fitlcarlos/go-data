package odata

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ProviderFactory é uma função que cria um provider de banco de dados
type ProviderFactory func() DatabaseProvider

// Registry global de providers
var providerRegistry = make(map[string]ProviderFactory)

// RegisterProvider registra uma factory de provider para um tipo específico
func RegisterProvider(dbType string, factory ProviderFactory) {
	providerRegistry[dbType] = factory
}

// CreateProviderFromConfig cria um provider baseado no config
func (c *EnvConfig) CreateProviderFromConfig() DatabaseProvider {
	if factory, exists := providerRegistry[c.DBDriver]; exists {
		return factory()
	}
	return nil
}

// EnvConfig representa as configurações carregadas do arquivo .env
type EnvConfig struct {
	// Configurações do banco de dados
	DBDriver           string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	DBSchema           string
	DBConnectionString string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration

	// Configurações do servidor OData
	ServerHost              string
	ServerPort              int
	ServerRoutePrefix       string
	ServerEnableCORS        bool
	ServerAllowedOrigins    []string
	ServerAllowedMethods    []string
	ServerAllowedHeaders    []string
	ServerExposedHeaders    []string
	ServerAllowCredentials  bool
	ServerEnableLogging     bool
	ServerLogLevel          string
	ServerLogFile           string
	ServerEnableCompression bool
	ServerMaxRequestSize    int64
	ServerShutdownTimeout   time.Duration

	// Configurações TLS
	ServerTLSCertFile string
	ServerTLSKeyFile  string

	// Configurações JWT
	JWTSecretKey   string
	JWTIssuer      string
	JWTExpiresIn   time.Duration
	JWTRefreshIn   time.Duration
	JWTAlgorithm   string
	JWTEnabled     bool
	JWTRequireAuth bool

	// Configurações do serviço
	ServiceName        string
	ServiceDisplayName string
	ServiceDescription string

	// Configurações de Rate Limit
	RateLimitEnabled           bool
	RateLimitRequestsPerMinute int
	RateLimitBurstSize         int
	RateLimitWindowSize        time.Duration
	RateLimitHeaders           bool

	// Mapa de todas as variáveis para acesso direto
	Variables map[string]string
}

// LoadEnvConfig carrega configurações do arquivo .env
func LoadEnvConfig() (*EnvConfig, error) {
	// Busca o arquivo .env no diretório atual e nos diretórios pai
	envPath := findEnvFile()
	if envPath == "" {
		return nil, fmt.Errorf("arquivo .env não encontrado")
	}

	// Carrega as variáveis do arquivo .env
	variables, err := loadEnvFile(envPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar arquivo .env: %w", err)
	}

	// ✅ INJETAR TODAS as variáveis no ambiente global
	// Isso permite que os.Getenv() funcione para todas as variáveis
	// APENAS se a variável NÃO existir já no ambiente (não sobrescrever variáveis do sistema)
	for key, value := range variables {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}

	// Cria a configuração com valores padrão
	config := &EnvConfig{
		Variables: variables,
	}

	// Preenche as configurações a partir das variáveis
	config.parseVariables()

	return config, nil
}

// findEnvFile busca o arquivo .env no diretório atual e nos diretórios pai
func findEnvFile() string {
	// Obtém o diretório atual
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Busca o arquivo .env no diretório atual e nos diretórios pai
	for {
		envPath := filepath.Join(currentDir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}

		// Vai para o diretório pai
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Chegou na raiz do sistema
			break
		}
		currentDir = parentDir
	}

	return ""
}

// loadEnvFile carrega as variáveis do arquivo .env
func loadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	variables := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignora linhas vazias e comentários
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Processa a linha no formato CHAVE=VALOR
		if !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove aspas se presentes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		variables[key] = value
	}

	return variables, scanner.Err()
}

// parseVariables preenche as configurações a partir das variáveis carregadas
func (c *EnvConfig) parseVariables() {
	// Configurações do banco de dados
	c.DBDriver = c.getEnvString("DB_DRIVER", "oracle")
	c.DBHost = c.getEnvString("DB_HOST", "localhost")
	c.DBPort = c.getEnvString("DB_PORT", "1521")
	c.DBName = c.getEnvString("DB_NAME", "")
	c.DBUser = c.getEnvString("DB_USER", "")
	c.DBPassword = c.getEnvString("DB_PASSWORD", "")
	c.DBSchema = c.getEnvString("DB_SCHEMA", "")
	c.DBConnectionString = c.getEnvString("DB_CONNECTION_STRING", "")
	c.DBMaxOpenConns = c.getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxConnections)
	c.DBMaxIdleConns = c.getEnvInt("DB_MAX_IDLE_CONNS", DefaultMinConnections)
	c.DBConnMaxLifetime = c.getEnvDuration("DB_CONN_MAX_LIFETIME", DefaultMaxIdleTime)

	// Configurações do servidor OData
	c.ServerHost = c.getEnvString("SERVER_HOST", "localhost")
	c.ServerPort = c.getEnvInt("SERVER_PORT", 8080)
	c.ServerRoutePrefix = c.getEnvString("SERVER_ROUTE_PREFIX", "/odata")
	c.ServerEnableCORS = c.getEnvBool("SERVER_ENABLE_CORS", true)
	c.ServerAllowedOrigins = c.getEnvStringSlice("SERVER_ALLOWED_ORIGINS", []string{"*"})
	c.ServerAllowedMethods = c.getEnvStringSlice("SERVER_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	c.ServerAllowedHeaders = c.getEnvStringSlice("SERVER_ALLOWED_HEADERS", []string{"*"})
	c.ServerExposedHeaders = c.getEnvStringSlice("SERVER_EXPOSED_HEADERS", []string{"OData-Version", "Content-Type"})
	c.ServerAllowCredentials = c.getEnvBool("SERVER_ALLOW_CREDENTIALS", false)
	c.ServerEnableLogging = c.getEnvBool("SERVER_ENABLE_LOGGING", true)
	c.ServerLogLevel = c.getEnvString("SERVER_LOG_LEVEL", "INFO")
	c.ServerLogFile = c.getEnvString("SERVER_LOG_FILE", "")
	c.ServerEnableCompression = c.getEnvBool("SERVER_ENABLE_COMPRESSION", false)
	c.ServerMaxRequestSize = c.getEnvInt64("SERVER_MAX_REQUEST_SIZE", 10*1024*1024)
	c.ServerShutdownTimeout = c.getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second)

	// Configurações TLS
	c.ServerTLSCertFile = c.getEnvString("SERVER_TLS_CERT_FILE", "")
	c.ServerTLSKeyFile = c.getEnvString("SERVER_TLS_KEY_FILE", "")

	// Configurações JWT
	c.JWTSecretKey = c.getEnvString("JWT_SECRET_KEY", "")
	c.JWTIssuer = c.getEnvString("JWT_ISSUER", "go-data-server")
	c.JWTExpiresIn = c.getEnvDuration("JWT_EXPIRES_IN", 1*time.Hour)
	c.JWTRefreshIn = c.getEnvDuration("JWT_REFRESH_IN", 24*time.Hour)
	c.JWTAlgorithm = c.getEnvString("JWT_ALGORITHM", "HS256")
	c.JWTEnabled = c.getEnvBool("JWT_ENABLED", false)
	c.JWTRequireAuth = c.getEnvBool("JWT_REQUIRE_AUTH", false)

	// Configurações do serviço
	c.ServiceName = c.getEnvString("SERVICE_NAME", "godata-service")
	c.ServiceDisplayName = c.getEnvString("SERVICE_DISPLAY_NAME", "GoData OData Service")
	c.ServiceDescription = c.getEnvString("SERVICE_DESCRIPTION", "Serviço GoData OData v4 para APIs RESTful")

	// Configurações de Rate Limit
	c.RateLimitEnabled = c.getEnvBool("RATE_LIMIT_ENABLED", false)
	c.RateLimitRequestsPerMinute = c.getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", DefaultRateLimitPerMinute)
	c.RateLimitBurstSize = c.getEnvInt("RATE_LIMIT_BURST_SIZE", DefaultRateLimitBurstSize)
	c.RateLimitWindowSize = c.getEnvDuration("RATE_LIMIT_WINDOW_SIZE", DefaultRateLimitWindow)
	c.RateLimitHeaders = c.getEnvBool("RATE_LIMIT_HEADERS", true)
}

// getEnvString retorna uma string do ambiente ou valor padrão
func (c *EnvConfig) getEnvString(key, defaultValue string) string {
	if value, exists := c.Variables[key]; exists {
		return value
	}
	return defaultValue
}

// getEnvInt retorna um int do ambiente ou valor padrão
func (c *EnvConfig) getEnvInt(key string, defaultValue int) int {
	if value, exists := c.Variables[key]; exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvInt64 retorna um int64 do ambiente ou valor padrão
func (c *EnvConfig) getEnvInt64(key string, defaultValue int64) int64 {
	if value, exists := c.Variables[key]; exists {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool retorna um bool do ambiente ou valor padrão
func (c *EnvConfig) getEnvBool(key string, defaultValue bool) bool {
	if value, exists := c.Variables[key]; exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvDuration retorna uma duração do ambiente ou valor padrão
func (c *EnvConfig) getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := c.Variables[key]; exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvStringSlice retorna um slice de strings do ambiente ou valor padrão
func (c *EnvConfig) getEnvStringSlice(key string, defaultValue []string) []string {
	if value, exists := c.Variables[key]; exists {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// BuildConnectionString constrói a string de conexão baseada nas configurações
func (c *EnvConfig) BuildConnectionString() string {
	// Se a string de conexão completa foi fornecida, usa ela
	if c.DBConnectionString != "" {
		return c.DBConnectionString
	}

	// Constrói a string de conexão baseada no driver
	switch c.DBDriver {
	case "oracle":
		return fmt.Sprintf("oracle://%s:%s@%s:%s/%s",
			c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
	case "postgres", "postgresql":
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
	default:
		return c.DBConnectionString
	}
}

// ToServerConfig converte a configuração .env para ServerConfig
func (c *EnvConfig) ToServerConfig() *ServerConfig {
	config := &ServerConfig{
		// Configurações do serviço
		Name:        c.ServiceName,
		DisplayName: c.ServiceDisplayName,
		Description: c.ServiceDescription,

		// Configurações do servidor
		Host:              c.ServerHost,
		Port:              c.ServerPort,
		RoutePrefix:       c.ServerRoutePrefix,
		EnableCORS:        c.ServerEnableCORS,
		AllowedOrigins:    c.ServerAllowedOrigins,
		AllowedMethods:    c.ServerAllowedMethods,
		AllowedHeaders:    c.ServerAllowedHeaders,
		ExposedHeaders:    c.ServerExposedHeaders,
		AllowCredentials:  c.ServerAllowCredentials,
		EnableLogging:     c.ServerEnableLogging,
		LogLevel:          c.ServerLogLevel,
		LogFile:           c.ServerLogFile,
		EnableCompression: c.ServerEnableCompression,
		MaxRequestSize:    c.ServerMaxRequestSize,
		ShutdownTimeout:   c.ServerShutdownTimeout,
		CertFile:          c.ServerTLSCertFile,
		CertKeyFile:       c.ServerTLSKeyFile,
		EnableJWT:         c.JWTEnabled,
		RequireAuth:       c.JWTRequireAuth,
	}

	// Configura JWT se habilitado
	if c.JWTEnabled && c.JWTSecretKey != "" {
		config.JWTConfig = &JWTConfig{
			SecretKey: c.JWTSecretKey,
			Issuer:    c.JWTIssuer,
			ExpiresIn: c.JWTExpiresIn,
			RefreshIn: c.JWTRefreshIn,
			Algorithm: c.JWTAlgorithm,
		}
	}

	// Configurações de Rate Limit
	if c.RateLimitEnabled {
		config.RateLimitConfig = &RateLimitConfig{
			Enabled:           c.RateLimitEnabled,
			RequestsPerMinute: c.RateLimitRequestsPerMinute,
			BurstSize:         c.RateLimitBurstSize,
			WindowSize:        c.RateLimitWindowSize,
			KeyGenerator:      defaultKeyGenerator,
			Headers:           c.RateLimitHeaders,
		}
	}

	return config
}

// LoadEnvOrDefault carrega configurações do .env ou retorna configurações padrão
func LoadEnvOrDefault() (*EnvConfig, error) {
	config, err := LoadEnvConfig()
	if err != nil {
		// Se não encontrar .env, cria configuração padrão
		config = &EnvConfig{
			Variables: make(map[string]string),
		}
		config.parseVariables()
	}
	return config, nil
}

// PrintLoadedConfig imprime as configurações carregadas para debug
func (c *EnvConfig) PrintLoadedConfig() {
	fmt.Println("📋 Configurações carregadas do .env:")
	fmt.Printf("   Database: %s://%s:%s/%s\n", c.DBDriver, c.DBHost, c.DBPort, c.DBName)
	fmt.Printf("   Server: %s:%d%s\n", c.ServerHost, c.ServerPort, c.ServerRoutePrefix)
	fmt.Printf("   CORS: %v\n", c.ServerEnableCORS)
	fmt.Printf("   JWT: %v\n", c.JWTEnabled)
	if c.JWTEnabled {
		fmt.Printf("   JWT Issuer: %s\n", c.JWTIssuer)
	}
	fmt.Printf("   TLS: %v\n", c.ServerTLSCertFile != "" && c.ServerTLSKeyFile != "")
}
