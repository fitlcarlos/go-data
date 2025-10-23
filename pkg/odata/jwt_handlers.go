package odata

import (
	"github.com/gofiber/fiber/v3"
)

// LoginRequest representa os dados de login
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse representa a resposta de login
type LoginResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	TokenType    string        `json:"token_type"`
	ExpiresIn    int64         `json:"expires_in"`
	User         *UserIdentity `json:"user"`
}

// RefreshRequest representa os dados de refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// SetupAuthRoutes configura as rotas de autenticação
func (s *Server) SetupAuthRoutes(authenticator ContextAuthenticator) {
	if !s.config.EnableJWT {
		s.logger.Printf("JWT não habilitado, rotas de autenticação não serão configuradas")
		return
	}

	authGroup := s.router.Group("/auth")

	// Rota de login
	authGroup.Post("/login", s.handleLogin(authenticator))

	// Rota de refresh token
	authGroup.Post("/refresh", s.handleRefresh(authenticator))

	// Rota de logout
	authGroup.Post("/logout", s.handleLogout())

	// Rota para obter informações do usuário atual
	authGroup.Get("/me", s.AuthMiddleware(), s.handleMe())

	s.logger.Printf("Rotas de autenticação configuradas")
}

// handleLogin handler para login
func (s *Server) handleLogin(authenticator ContextAuthenticator) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req LoginRequest
		if err := c.Bind().JSON(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Dados de login inválidos")
		}

		if req.Username == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Username e password são obrigatórios")
		}

		// Autenticar usando ContextAuthenticator
		authCtx := s.createAuthContext(c)
		user, err := authenticator.AuthenticateWithContext(authCtx, req.Username, req.Password)

		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Credenciais inválidas")
		}

		// Gerar tokens
		accessToken, err := s.jwtService.GenerateToken(user)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Erro ao gerar token de acesso")
		}

		refreshToken, err := s.jwtService.GenerateRefreshToken(user)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Erro ao gerar refresh token")
		}

		response := LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int64(s.jwtService.config.ExpiresIn.Seconds()),
			User:         user,
		}

		return c.JSON(response)
	}
}

// handleRefresh handler para refresh token
func (s *Server) handleRefresh(authenticator ContextAuthenticator) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req RefreshRequest
		if err := c.Bind().JSON(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Dados de refresh inválidos")
		}

		if req.RefreshToken == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Refresh token é obrigatório")
		}

		// Validar refresh token e extrair username
		claims, err := s.jwtService.ValidateToken(req.RefreshToken)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Refresh token inválido")
		}

		// Chamar RefreshToken do authenticator (contexto disponível se necessário)
		authCtx := s.createAuthContext(c)
		user, err := authenticator.RefreshToken(authCtx, claims.Username)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Não foi possível renovar token")
		}

		// Gerar novo access token
		accessToken, err := s.jwtService.GenerateToken(user)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Erro ao gerar token")
		}

		response := map[string]interface{}{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   int64(s.jwtService.config.ExpiresIn.Seconds()),
		}

		return c.JSON(response)
	}
}

// handleLogout handler para logout
func (s *Server) handleLogout() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Por enquanto, apenas retorna sucesso
		// Em uma implementação real, você poderia invalidar o token
		// mantendo uma blacklist de tokens
		return c.JSON(map[string]string{
			"message": "Logout realizado com sucesso",
		})
	}
}

// handleMe handler para obter informações do usuário atual
func (s *Server) handleMe() fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Usuário não autenticado")
		}

		return c.JSON(user)
	}
}

// SetEntityAuth configura autenticação para uma entidade específica
func (s *Server) SetEntityAuth(entityName string, config EntityAuthConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entityAuth[entityName] = config
	s.logger.Printf("Configuração de autenticação definida para entidade '%s'", entityName)
}

// GetEntityAuth obtém configuração de autenticação para uma entidade
func (s *Server) GetEntityAuth(entityName string) (EntityAuthConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, exists := s.entityAuth[entityName]
	return config, exists
}
