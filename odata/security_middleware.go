package odata

import (
	"github.com/gofiber/fiber/v3"
)

// SecurityHeadersConfig configurações de security headers
type SecurityHeadersConfig struct {
	// Habilita/desabilita todos os headers de segurança
	Enabled bool

	// Headers individuais (podem ser habilitados/desabilitados individualmente)
	XFrameOptions           string // DENY, SAMEORIGIN, ALLOW-FROM uri
	XContentTypeOptions     string // nosniff
	XXSSProtection          string // 1; mode=block
	ContentSecurityPolicy   string // default-src 'self'; ...
	StrictTransportSecurity string // max-age=31536000; includeSubDomains
	ReferrerPolicy          string // strict-origin-when-cross-origin, no-referrer, etc
	PermissionsPolicy       string // camera=(), microphone=(), geolocation=(), etc

	// Headers customizados adicionais
	CustomHeaders map[string]string
}

// DefaultSecurityHeadersConfig retorna configuração padrão de security headers
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled:                 true,
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy:       "camera=(), microphone=(), geolocation=(), payment=()",
		CustomHeaders:           make(map[string]string),
	}
}

// SecurityHeadersMiddleware cria middleware de security headers
func SecurityHeadersMiddleware(config *SecurityHeadersConfig) fiber.Handler {
	// Se config for nil ou desabilitado, retorna handler vazio
	if config == nil || !config.Enabled {
		return func(c fiber.Ctx) error {
			return c.Next()
		}
	}

	return func(c fiber.Ctx) error {
		// X-Frame-Options: previne clickjacking
		if config.XFrameOptions != "" {
			c.Set("X-Frame-Options", config.XFrameOptions)
		}

		// X-Content-Type-Options: previne MIME type sniffing
		if config.XContentTypeOptions != "" {
			c.Set("X-Content-Type-Options", config.XContentTypeOptions)
		}

		// X-XSS-Protection: habilita proteção XSS do browser (deprecated em favor de CSP, mas ainda útil)
		if config.XXSSProtection != "" {
			c.Set("X-XSS-Protection", config.XXSSProtection)
		}

		// Content-Security-Policy: controla quais recursos podem ser carregados
		if config.ContentSecurityPolicy != "" {
			c.Set("Content-Security-Policy", config.ContentSecurityPolicy)
		}

		// Strict-Transport-Security: força HTTPS (só adiciona se conexão for HTTPS)
		if config.StrictTransportSecurity != "" && c.Protocol() == "https" {
			c.Set("Strict-Transport-Security", config.StrictTransportSecurity)
		}

		// Referrer-Policy: controla informações de referrer
		if config.ReferrerPolicy != "" {
			c.Set("Referrer-Policy", config.ReferrerPolicy)
		}

		// Permissions-Policy: controla acesso a features do browser
		if config.PermissionsPolicy != "" {
			c.Set("Permissions-Policy", config.PermissionsPolicy)
		}

		// Headers customizados adicionais
		for key, value := range config.CustomHeaders {
			c.Set(key, value)
		}

		// Headers adicionais de segurança recomendados
		c.Set("X-Permitted-Cross-Domain-Policies", "none")
		c.Set("X-Download-Options", "noopen")

		return c.Next()
	}
}

// DisableSecurityHeaders desabilita headers de segurança
func DisableSecurityHeaders() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled: false,
	}
}

// StrictSecurityHeadersConfig retorna configuração mais restritiva
func StrictSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled:                 true,
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';",
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		ReferrerPolicy:          "no-referrer",
		PermissionsPolicy:       "accelerometer=(), autoplay=(), camera=(), cross-origin-isolated=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(), usb=(), web-share=(), xr-spatial-tracking=()",
		CustomHeaders:           make(map[string]string),
	}
}

// RelaxedSecurityHeadersConfig retorna configuração mais permissiva (para desenvolvimento)
func RelaxedSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled:                 true,
		XFrameOptions:           "SAMEORIGIN",
		XContentTypeOptions:     "nosniff",
		XXSSProtection:          "1; mode=block",
		ContentSecurityPolicy:   "default-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' ws: wss:;",
		StrictTransportSecurity: "",
		ReferrerPolicy:          "origin-when-cross-origin",
		PermissionsPolicy:       "",
		CustomHeaders:           make(map[string]string),
	}
}
