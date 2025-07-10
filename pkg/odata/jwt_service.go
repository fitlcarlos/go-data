package odata

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService gerencia operações JWT
type JWTService struct {
	config *JWTConfig
}

// NewJWTService cria uma nova instância do serviço JWT
func NewJWTService(config *JWTConfig) *JWTService {
	if config == nil {
		config = DefaultJWTConfig()
	}
	if config.Algorithm == "" {
		config.Algorithm = "HS256"
	}
	return &JWTService{
		config: config,
	}
}

// GenerateToken gera um token JWT para o usuário
func (s *JWTService) GenerateToken(user *UserIdentity) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		Username: user.Username,
		Roles:    user.Roles,
		Scopes:   user.Scopes,
		Admin:    user.Admin,
		Custom:   user.Custom,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.ExpiresIn)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.SecretKey))
}

// GenerateRefreshToken gera um refresh token
func (s *JWTService) GenerateRefreshToken(user *UserIdentity) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.config.RefreshIn)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.SecretKey))
}

// ValidateToken valida um token JWT e retorna as claims
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", token.Header["alg"])
		}
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token inválido")
}

// ExtractUserIdentity extrai a identidade do usuário das claims
func (s *JWTService) ExtractUserIdentity(claims *JWTClaims) *UserIdentity {
	return &UserIdentity{
		Username: claims.Username,
		Roles:    claims.Roles,
		Scopes:   claims.Scopes,
		Admin:    claims.Admin,
		Custom:   claims.Custom,
	}
}

// ValidateAndExtractUser valida o token e extrai a identidade do usuário
func (s *JWTService) ValidateAndExtractUser(tokenString string) (*UserIdentity, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return s.ExtractUserIdentity(claims), nil
}

// IsTokenExpired verifica se o token está expirado
func (s *JWTService) IsTokenExpired(tokenString string) bool {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.SecretKey), nil
	})

	if err != nil {
		return true
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims.ExpiresAt.Time.Before(time.Now())
	}

	return true
}

// RefreshToken gera um novo token a partir de um refresh token válido
func (s *JWTService) RefreshToken(refreshTokenString string) (string, error) {
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Criar nova identidade do usuário apenas com username
	// (outras informações podem ter mudado)
	user := &UserIdentity{
		Username: claims.Username,
		Roles:    claims.Roles,
		Scopes:   claims.Scopes,
		Admin:    claims.Admin,
		Custom:   claims.Custom,
	}

	return s.GenerateToken(user)
}
