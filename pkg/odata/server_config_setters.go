package odata

import "time"

// SetPort permite sobrescrever a porta do servidor
func (s *Server) SetPort(port int) *Server {
	s.config.Port = port
	return s
}

// SetHost permite sobrescrever o host do servidor
func (s *Server) SetHost(host string) *Server {
	s.config.Host = host
	return s
}

// SetRoutePrefix permite sobrescrever o prefixo das rotas
func (s *Server) SetRoutePrefix(prefix string) *Server {
	s.config.RoutePrefix = prefix
	return s
}

// SetCORS permite habilitar/desabilitar CORS
func (s *Server) SetCORS(enabled bool) *Server {
	s.config.EnableCORS = enabled
	return s
}

// SetAllowedOrigins permite configurar origens permitidas para CORS
func (s *Server) SetAllowedOrigins(origins []string) *Server {
	s.config.AllowedOrigins = origins
	return s
}

// SetAllowedMethods permite configurar métodos HTTP permitidos para CORS
func (s *Server) SetAllowedMethods(methods []string) *Server {
	s.config.AllowedMethods = methods
	return s
}

// SetAllowedHeaders permite configurar headers permitidos para CORS
func (s *Server) SetAllowedHeaders(headers []string) *Server {
	s.config.AllowedHeaders = headers
	return s
}

// SetEnableLogging permite habilitar/desabilitar logging
func (s *Server) SetEnableLogging(enabled bool) *Server {
	s.config.EnableLogging = enabled
	return s
}

// SetLogLevel permite configurar o nível de log
func (s *Server) SetLogLevel(level string) *Server {
	s.config.LogLevel = level
	return s
}

// SetMaxRequestSize permite configurar o tamanho máximo de requisição
func (s *Server) SetMaxRequestSize(size int64) *Server {
	s.config.MaxRequestSize = size
	return s
}

// SetShutdownTimeout permite configurar o timeout de shutdown
func (s *Server) SetShutdownTimeout(timeout time.Duration) *Server {
	s.config.ShutdownTimeout = timeout
	return s
}

// SetTLS permite configurar certificados TLS
func (s *Server) SetTLS(certFile, keyFile string) *Server {
	s.config.CertFile = certFile
	s.config.CertKeyFile = keyFile
	return s
}

// SetRateLimit permite configurar rate limiting
func (s *Server) SetRateLimit(requestsPerMinute, burstSize int) *Server {
	if s.config.RateLimitConfig == nil {
		s.config.RateLimitConfig = &RateLimitConfig{}
	}
	s.config.RateLimitConfig.Enabled = true
	s.config.RateLimitConfig.RequestsPerMinute = requestsPerMinute
	s.config.RateLimitConfig.BurstSize = burstSize

	// Recria o rate limiter com a nova configuração
	s.rateLimiter = NewRateLimiter(s.config.RateLimitConfig)

	return s
}

// DisableRateLimit desabilita rate limiting
func (s *Server) DisableRateLimit() *Server {
	if s.config.RateLimitConfig != nil {
		s.config.RateLimitConfig.Enabled = false
	}
	s.rateLimiter = nil
	return s
}

// SetSecurityHeaders permite habilitar/desabilitar security headers
func (s *Server) SetSecurityHeaders(enabled bool) *Server {
	if s.config.SecurityHeadersConfig == nil {
		s.config.SecurityHeadersConfig = DefaultSecurityHeadersConfig()
	}
	s.config.SecurityHeadersConfig.Enabled = enabled
	return s
}

// SetAuditLog permite configurar audit logging
func (s *Server) SetAuditLog(enabled bool, logType string) *Server {
	if s.config.AuditLogConfig == nil {
		s.config.AuditLogConfig = DefaultAuditLogConfig()
	}
	s.config.AuditLogConfig.Enabled = enabled
	s.config.AuditLogConfig.LogType = logType

	// Recria o audit logger se necessário
	if enabled {
		auditLogger, err := NewAuditLogger(s.config.AuditLogConfig)
		if err == nil {
			s.auditLogger = auditLogger
		}
	} else {
		s.auditLogger = &NoOpAuditLogger{}
	}

	return s
}

// SetProvider permite trocar o provider de banco de dados
func (s *Server) SetProvider(provider DatabaseProvider) *Server {
	s.provider = provider
	return s
}

// ApplyConfig aplica uma configuração completa ao servidor
func (s *Server) ApplyConfig(config *ServerConfig) *Server {
	s.config = config

	// Reaplica configurações que dependem de config
	if config.RateLimitConfig != nil && config.RateLimitConfig.Enabled {
		s.rateLimiter = NewRateLimiter(config.RateLimitConfig)
	}

	if config.AuditLogConfig != nil && config.AuditLogConfig.Enabled {
		auditLogger, err := NewAuditLogger(config.AuditLogConfig)
		if err == nil {
			s.auditLogger = auditLogger
		}
	}

	return s
}
