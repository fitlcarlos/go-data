package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fitlcarlos/go-data/pkg/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

// JwtAuth implementação de AuthProvider usando JWT
type JwtAuth struct {
	config *JWTConfig

	// TokenGenerator função customizável para gerar tokens
	// Se nil, usa geração padrão
	TokenGenerator func(*auth.UserIdentity) (string, error)

	// TokenValidator função customizável para validar tokens
	// Se nil, usa validação padrão
	TokenValidator func(string) (*auth.UserIdentity, error)

	// TokenExtractor função customizável para extrair tokens do contexto
	// Se nil, usa extração padrão (Bearer token)
	TokenExtractor func(fiber.Ctx) string
}

// NewJwtAuth cria uma nova instância de JwtAuth
// Ordem de prioridade de configuração:
// 1. Parâmetro config (override manual)
// 2. Variáveis de ambiente (.env)
// 3. Valores padrão
func NewJwtAuth(config *JWTConfig) *JwtAuth {
	// 1. Carrega configuração do .env
	envConfig := LoadConfigFromEnv()

	// 2. Faz merge com config fornecido (se houver)
	finalConfig := MergeConfig(envConfig, config)

	// 3. Valida configuração final
	if err := finalConfig.Validate(); err != nil {
		panic(fmt.Sprintf("JWT configuration error: %v", err))
	}

	auth := &JwtAuth{
		config: finalConfig,
	}

	// Define geradores/validadores padrão
	auth.TokenGenerator = auth.DefaultGenerateToken
	auth.TokenValidator = auth.DefaultValidateToken
	auth.TokenExtractor = auth.DefaultExtractToken

	return auth
}

// ValidateToken implementa AuthProvider.ValidateToken
func (j *JwtAuth) ValidateToken(token string) (*auth.UserIdentity, error) {
	if j.TokenValidator != nil {
		return j.TokenValidator(token)
	}
	return j.DefaultValidateToken(token)
}

// GenerateToken implementa AuthProvider.GenerateToken
func (j *JwtAuth) GenerateToken(user *auth.UserIdentity) (string, error) {
	if j.TokenGenerator != nil {
		return j.TokenGenerator(user)
	}
	return j.DefaultGenerateToken(user)
}

// ExtractToken implementa AuthProvider.ExtractToken
func (j *JwtAuth) ExtractToken(c fiber.Ctx) string {
	if j.TokenExtractor != nil {
		return j.TokenExtractor(c)
	}
	return j.DefaultExtractToken(c)
}

// DefaultGenerateToken geração padrão de token JWT (público para reutilização)
func (j *JwtAuth) DefaultGenerateToken(user *auth.UserIdentity) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		Username: user.Username,
		Roles:    user.Roles,
		Scopes:   user.Scopes,
		Admin:    user.Admin,
		Custom:   user.Custom,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.config.ExpiresIn)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.SecretKey))
}

// DefaultValidateToken validação padrão de token JWT (público para reutilização)
func (j *JwtAuth) DefaultValidateToken(tokenString string) (*auth.UserIdentity, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return []byte(j.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return &auth.UserIdentity{
			Username: claims.Username,
			Roles:    claims.Roles,
			Scopes:   claims.Scopes,
			Admin:    claims.Admin,
			Custom:   claims.Custom,
		}, nil
	}

	return nil, errors.New("token inválido")
}

// DefaultExtractToken extração padrão de token do header Authorization (público para reutilização)
func (j *JwtAuth) DefaultExtractToken(c fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}

	return strings.TrimPrefix(authHeader, "Bearer ")
}

// GenerateRefreshToken gera um refresh token
func (j *JwtAuth) GenerateRefreshToken(user *auth.UserIdentity) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.config.RefreshIn)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.config.SecretKey))
}

// RefreshToken gera um novo token a partir de um refresh token válido
func (j *JwtAuth) RefreshToken(refreshTokenString string) (string, error) {
	user, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	return j.GenerateToken(user)
}

// GetConfig retorna a configuração JWT
func (j *JwtAuth) GetConfig() *JWTConfig {
	return j.config
}
