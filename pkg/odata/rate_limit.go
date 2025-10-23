package odata

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// RateLimitConfig representa as configurações do rate limit
type RateLimitConfig struct {
	Enabled           bool                     // Se o rate limit está habilitado
	RequestsPerMinute int                      // Número de requisições por minuto
	BurstSize         int                      // Tamanho do burst (requisições simultâneas)
	WindowSize        time.Duration            // Tamanho da janela de tempo
	KeyGenerator      func(c fiber.Ctx) string // Função para gerar chave única
	SkipSuccessful    bool                     // Pular requisições bem-sucedidas
	SkipFailed        bool                     // Pular requisições com falha
	Headers           bool                     // Incluir headers de rate limit na resposta
}

// DefaultRateLimitConfig retorna uma configuração padrão de rate limit
// NOTA: Rate limiting está HABILITADO por padrão para proteção contra abuso.
// Para desabilitar, defina Enabled = false na configuração.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:           true, // ⚠️ HABILITADO POR PADRÃO para segurança
		RequestsPerMinute: DefaultRateLimitPerMinute,
		BurstSize:         DefaultRateLimitBurstSize,
		WindowSize:        DefaultRateLimitWindow,
		KeyGenerator:      defaultKeyGenerator,
		SkipSuccessful:    false,
		SkipFailed:        false,
		Headers:           true,
	}
}

// RateLimiter implementa o controle de rate limit
type RateLimiter struct {
	config        *RateLimitConfig
	clients       map[string]*ClientLimiter
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// ClientLimiter representa o limitador para um cliente específico
type ClientLimiter struct {
	requests   []time.Time
	lastAccess time.Time
	mu         sync.Mutex
}

// NewRateLimiter cria uma nova instância do rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      config,
		clients:     make(map[string]*ClientLimiter),
		stopCleanup: make(chan bool),
	}

	// Inicia limpeza automática de clientes inativos
	if config.Enabled {
		rl.startCleanup()
	}

	return rl
}

// startCleanup inicia a limpeza automática de clientes inativos
func (rl *RateLimiter) startCleanup() {
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-rl.cleanupTicker.C:
				rl.cleanupInactiveClients()
			case <-rl.stopCleanup:
				rl.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// cleanupInactiveClients remove clientes inativos há mais de 10 minutos
func (rl *RateLimiter) cleanupInactiveClients() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, client := range rl.clients {
		client.mu.Lock()
		if client.lastAccess.Before(cutoff) {
			delete(rl.clients, key)
		}
		client.mu.Unlock()
	}
}

// Stop para o rate limiter e limpa recursos
func (rl *RateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.stopCleanup <- true
	}
}

// Allow verifica se uma requisição é permitida
func (rl *RateLimiter) Allow(key string) (bool, RateLimitInfo) {
	if !rl.config.Enabled {
		return true, RateLimitInfo{}
	}

	rl.mu.Lock()
	client, exists := rl.clients[key]
	if !exists {
		client = &ClientLimiter{
			requests: make([]time.Time, 0),
		}
		rl.clients[key] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	client.lastAccess = now

	// Remove requisições antigas (fora da janela)
	cutoff := now.Add(-rl.config.WindowSize)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range client.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	client.requests = validRequests

	// Verifica se pode fazer nova requisição
	if len(client.requests) >= rl.config.RequestsPerMinute {
		// Se o limite é 0, bloqueia imediatamente sem tentar acessar requests[0]
		if rl.config.RequestsPerMinute == 0 || len(client.requests) == 0 {
			info := RateLimitInfo{
				Allowed:    false,
				Limit:      rl.config.RequestsPerMinute,
				Remaining:  0,
				ResetTime:  now.Add(rl.config.WindowSize),
				RetryAfter: int(rl.config.WindowSize.Seconds()),
			}
			return false, info
		}

		// Calcula quando a próxima requisição será permitida
		oldestRequest := client.requests[0]
		resetTime := oldestRequest.Add(rl.config.WindowSize)
		remainingTime := resetTime.Sub(now)

		info := RateLimitInfo{
			Allowed:    false,
			Limit:      rl.config.RequestsPerMinute,
			Remaining:  0,
			ResetTime:  resetTime,
			RetryAfter: int(remainingTime.Seconds()),
		}

		return false, info
	}

	// Adiciona a nova requisição
	client.requests = append(client.requests, now)

	// Calcula informações do rate limit
	remaining := rl.config.RequestsPerMinute - len(client.requests)
	resetTime := now.Add(rl.config.WindowSize)

	info := RateLimitInfo{
		Allowed:    true,
		Limit:      rl.config.RequestsPerMinute,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: 0,
	}

	return true, info
}

// RateLimitInfo contém informações sobre o status do rate limit
type RateLimitInfo struct {
	Allowed    bool      // Se a requisição é permitida
	Limit      int       // Limite total de requisições
	Remaining  int       // Requisições restantes
	ResetTime  time.Time // Quando o limite será resetado
	RetryAfter int       // Segundos para tentar novamente (se bloqueado)
}

// defaultKeyGenerator gera uma chave baseada no IP do cliente
func defaultKeyGenerator(c fiber.Ctx) string {
	// Tenta obter o IP real (considerando proxies)
	ip := c.IP()

	// Se há header X-Forwarded-For, usa o primeiro IP
	if forwardedFor := c.Get("X-Forwarded-For"); forwardedFor != "" {
		ip = forwardedFor
	}

	// Se há header X-Real-IP, usa esse IP
	if realIP := c.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	}

	return fmt.Sprintf("rate_limit:%s", ip)
}

// RateLimitMiddleware cria um middleware de rate limit
func (s *Server) RateLimitMiddleware() fiber.Handler {
	if s.rateLimiter == nil {
		// Se não há rate limiter configurado, retorna middleware que não faz nada
		return func(c fiber.Ctx) error {
			return c.Next()
		}
	}

	return func(c fiber.Ctx) error {
		// Gera chave única para o cliente
		key := s.rateLimiter.config.KeyGenerator(c)

		// Verifica se a requisição é permitida
		allowed, info := s.rateLimiter.Allow(key)

		// Adiciona headers de rate limit se habilitado
		if s.rateLimiter.config.Headers {
			c.Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
			c.Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
			c.Set("X-RateLimit-Reset", strconv.FormatInt(info.ResetTime.Unix(), 10))

			if !allowed {
				c.Set("X-RateLimit-Retry-After", strconv.Itoa(info.RetryAfter))
			}
		}

		// Se não é permitido, retorna erro 429
		if !allowed {
			errorResponse := ODataResponse{
				Error: &ODataError{
					Code:    "RateLimitExceeded",
					Message: fmt.Sprintf("Rate limit exceeded. Try again in %d seconds.", info.RetryAfter),
					Target:  "rate_limit",
				},
			}

			c.Set("Content-Type", "application/json")
			c.Status(http.StatusTooManyRequests)
			return c.JSON(errorResponse)
		}

		// Continua para o próximo middleware/handler
		return c.Next()
	}
}

// SetRateLimitConfig configura o rate limiter do servidor
func (s *Server) SetRateLimitConfig(config *RateLimitConfig) {
	if config.Enabled {
		s.rateLimiter = NewRateLimiter(config)
		s.logger.Printf("Rate limit habilitado: %d req/min, burst: %d",
			config.RequestsPerMinute, config.BurstSize)
	} else {
		s.rateLimiter = nil
		s.logger.Printf("Rate limit desabilitado")
	}
}

// GetRateLimitConfig retorna a configuração atual do rate limit
func (s *Server) GetRateLimitConfig() *RateLimitConfig {
	if s.rateLimiter != nil {
		return s.rateLimiter.config
	}
	return nil
}

// CustomKeyGenerator cria um gerador de chave customizado
func CustomKeyGenerator(fn func(c fiber.Ctx) string) func(c fiber.Ctx) string {
	return fn
}

// UserBasedKeyGenerator gera chave baseada no usuário autenticado
func UserBasedKeyGenerator(c fiber.Ctx) string {
	// Tenta obter o ID do usuário do contexto JWT
	if userID := c.Locals("user_id"); userID != nil {
		return fmt.Sprintf("rate_limit:user:%v", userID)
	}

	// Fallback para IP se não há usuário autenticado
	return defaultKeyGenerator(c)
}

// APIKeyBasedKeyGenerator gera chave baseada na API key
func APIKeyBasedKeyGenerator(c fiber.Ctx) string {
	apiKey := c.Get("X-API-Key")
	if apiKey != "" {
		return fmt.Sprintf("rate_limit:api_key:%s", apiKey)
	}

	// Fallback para IP se não há API key
	return defaultKeyGenerator(c)
}

// TenantBasedKeyGenerator gera chave baseada no tenant (para multi-tenant)
func TenantBasedKeyGenerator(c fiber.Ctx) string {
	tenantID := GetCurrentTenant(c)
	if tenantID != "" {
		ip := c.IP()
		return fmt.Sprintf("rate_limit:tenant:%s:%s", tenantID, ip)
	}

	// Fallback para IP se não há tenant
	return defaultKeyGenerator(c)
}
