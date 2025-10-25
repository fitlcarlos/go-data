package odata

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig configurações para JWT
type JWTConfig struct {
	SecretKey  string
	Issuer     string
	ExpiresIn  time.Duration
	RefreshIn  time.Duration
	Algorithm  string
	ContextKey string // Chave para armazenar o token no contexto (padrão: "user")
}

// NewRouterJWTAuth retorna middleware JWT
// Carrega config do .env se não fornecido
func (s *Server) NewRouterJWTAuth(config ...*JWTConfig) fiber.Handler {
	var jwtConfig *JWTConfig

	// Se config não foi fornecido, carrega do .env
	if len(config) == 0 || config[0] == nil {
		envConfig, err := LoadEnvOrDefault()
		if err != nil {
			s.logger.Printf("⚠️  Erro ao carregar config do .env: %v, usando padrões", err)
			jwtConfig = defaultJWTConfig()
		} else {
			jwtConfig = &JWTConfig{
				SecretKey:  envConfig.JWTSecretKey,
				Issuer:     envConfig.JWTIssuer,
				ExpiresIn:  envConfig.JWTExpiresIn,
				RefreshIn:  envConfig.JWTRefreshIn,
				Algorithm:  envConfig.JWTAlgorithm,
				ContextKey: "user",
			}
		}
	} else {
		jwtConfig = config[0]
	}

	// Validar secret key
	if jwtConfig.SecretKey == "" {
		panic("JWT SecretKey é obrigatório! Configure JWT_SECRET_KEY no arquivo .env")
	}

	// Retornar middleware customizado
	return func(c fiber.Ctx) error {
		// DEBUG: Log da requisição
		if s.config.EnableLogging {
			s.logger.Printf("🔐 JWT Middleware: %s %s", c.Method(), c.Path())
		}

		// Extrair token do header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			if s.config.EnableLogging {
				s.logger.Printf("❌ JWT: Token não fornecido para %s %s", c.Method(), c.Path())
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Token não fornecido",
			})
		}

		// Verificar se tem prefixo "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			if s.config.EnableLogging {
				s.logger.Printf("❌ JWT: Formato de token inválido para %s %s", c.Method(), c.Path())
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Formato de token inválido",
			})
		}

		// Extrair token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validar token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validar algoritmo
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtConfig.SecretKey), nil
		})

		if err != nil || !token.Valid {
			if s.config.EnableLogging {
				s.logger.Printf("❌ JWT: Token inválido ou expirado para %s %s - Erro: %v", c.Method(), c.Path(), err)
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "Token inválido ou expirado",
			})
		}

		// Extrair claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Armazenar token e claims no contexto
			c.Locals(jwtConfig.ContextKey, token)
			c.Locals("jwt_claims", claims)
			
			if s.config.EnableLogging {
				s.logger.Printf("✅ JWT: Token válido para %s %s - Usuario: %v", c.Method(), c.Path(), claims["username"])
			}
		}

		return c.Next()
	}
}

// GenerateJWT gera um token JWT (wrapper do método do servidor para compatibilidade)
func (s *Server) GenerateJWT(claims jwt.MapClaims, config ...*JWTConfig) (string, error) {
	return GenerateJWT(claims, config...)
}

// GenerateRefreshToken gera um refresh token (wrapper do método do servidor para compatibilidade)
func (s *Server) GenerateRefreshToken(claims jwt.MapClaims, config ...*JWTConfig) (string, error) {
	return GenerateRefreshToken(claims, config...)
}

// ValidateJWT valida um token JWT (wrapper do método do servidor para compatibilidade)
func (s *Server) ValidateJWT(tokenString string, config ...*JWTConfig) (jwt.MapClaims, error) {
	return ValidateJWT(tokenString, config...)
}

// GetJWTClaims retorna os claims do token JWT do contexto
func GetJWTClaims(c fiber.Ctx) jwt.MapClaims {
	if claims := c.Locals("jwt_claims"); claims != nil {
		if mapClaims, ok := claims.(jwt.MapClaims); ok {
			return mapClaims
		}
	}
	return nil
}

// defaultJWTConfig retorna configuração padrão
func defaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		SecretKey:  "change-me-in-production",
		Issuer:     "go-data-server",
		ExpiresIn:  24 * time.Hour,
		RefreshIn:  7 * 24 * time.Hour,
		Algorithm:  "HS256",
		ContextKey: "user",
	}
}

// =======================================================================================
// FUNÇÕES STANDALONE (não dependem do servidor)
// =======================================================================================

// GenerateJWT gera um token JWT (função standalone)
func GenerateJWT(claims jwt.MapClaims, config ...*JWTConfig) (string, error) {
	var jwtConfig *JWTConfig

	// Carregar config
	if len(config) == 0 || config[0] == nil {
		envConfig, err := LoadEnvOrDefault()
		if err != nil {
			return "", err
		}
		jwtConfig = &JWTConfig{
			SecretKey: envConfig.JWTSecretKey,
			Issuer:    envConfig.JWTIssuer,
			ExpiresIn: envConfig.JWTExpiresIn,
			Algorithm: envConfig.JWTAlgorithm,
		}
	} else {
		jwtConfig = config[0]
	}

	// Adicionar claims padrão
	now := time.Now()
	if claims["iss"] == nil {
		claims["iss"] = jwtConfig.Issuer
	}
	if claims["iat"] == nil {
		claims["iat"] = now.Unix()
	}
	if claims["exp"] == nil {
		claims["exp"] = now.Add(jwtConfig.ExpiresIn).Unix()
	}

	// Criar token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assinar token
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

// GenerateRefreshToken gera um refresh token (função standalone)
func GenerateRefreshToken(claims jwt.MapClaims, config ...*JWTConfig) (string, error) {
	var jwtConfig *JWTConfig

	// Carregar config
	if len(config) == 0 || config[0] == nil {
		envConfig, err := LoadEnvOrDefault()
		if err != nil {
			return "", err
		}
		jwtConfig = &JWTConfig{
			SecretKey: envConfig.JWTSecretKey,
			Issuer:    envConfig.JWTIssuer,
			RefreshIn: envConfig.JWTRefreshIn,
			Algorithm: envConfig.JWTAlgorithm,
		}
	} else {
		jwtConfig = config[0]
	}

	// Adicionar claims padrão
	now := time.Now()
	if claims["iss"] == nil {
		claims["iss"] = jwtConfig.Issuer
	}
	if claims["iat"] == nil {
		claims["iat"] = now.Unix()
	}
	if claims["exp"] == nil {
		claims["exp"] = now.Add(jwtConfig.RefreshIn).Unix()
	}

	// Criar token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Assinar token
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

// ValidateJWT valida um token JWT (função standalone)
func ValidateJWT(tokenString string, config ...*JWTConfig) (jwt.MapClaims, error) {
	var jwtConfig *JWTConfig

	// Carregar config
	if len(config) == 0 || config[0] == nil {
		envConfig, err := LoadEnvOrDefault()
		if err != nil {
			return nil, err
		}
		jwtConfig = &JWTConfig{
			SecretKey: envConfig.JWTSecretKey,
		}
	} else {
		jwtConfig = config[0]
	}

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}
