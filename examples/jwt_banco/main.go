package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/go-data/odata"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User representa um usuário do sistema
type User struct {
	ID       int    `json:"id" odata:"key"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

// Product representa um produto
type Product struct {
	ID          int     `json:"id" odata:"key"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	CreatedBy   string  `json:"created_by"`
}

// Order representa um pedido
type Order struct {
	ID         int       `json:"id" odata:"key"`
	UserID     int       `json:"user_id"`
	ProductID  int       `json:"product_id"`
	Quantity   int       `json:"quantity"`
	TotalPrice float64   `json:"total_price"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserStore armazena usuários (demonstração - em produção use banco de dados)
type UserStore struct {
	users map[string]*UserData
}

type UserData struct {
	Username     string
	PasswordHash string
	Admin        bool
	Roles        []string
	Scopes       []string
	Custom       map[string]interface{}
}

// NewUserStore cria um novo store de usuários
func NewUserStore() (*UserStore, error) {
	// Hash da senha "teste123" para demonstração
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("teste123"), bcrypt.DefaultCost)

	store := &UserStore{
		users: map[string]*UserData{
			"teste": {
				Username:     "teste",
				PasswordHash: string(passwordHash),
				Admin:        false,
				Roles:        []string{"user"},
				Scopes:       []string{"read"},
				Custom: map[string]interface{}{
					"name":  "Usuário Teste",
					"email": "teste@example.com",
				},
			},
			"admin": {
				Username:     "admin",
				PasswordHash: string(passwordHash),
				Admin:        true,
				Roles:        []string{"admin", "user"},
				Scopes:       []string{"read", "write", "delete"},
				Custom: map[string]interface{}{
					"name":  "Administrador",
					"email": "admin@example.com",
				},
			},
		},
	}
	return store, nil
}

// Authenticate valida credenciais do usuário
func (s *UserStore) Authenticate(username, password string) (*UserData, error) {
	userData, exists := s.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado ou inativo")
	}

	// Verificar senha usando bcrypt
	err := bcrypt.CompareHashAndPassword([]byte(userData.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("senha inválida")
	}

	return userData, nil
}

// LoginRequest representa os dados de login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RefreshRequest representa os dados de refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func main() {
	// Criar servidor
	server := odata.NewServer()

	// Criar user store
	userStore, err := NewUserStore()
	if err != nil {
		log.Fatal("Erro ao criar user store:", err)
	}

	// Criar middleware JWT (carrega configurações do .env automaticamente)
	jwtMiddleware := server.NewRouterJWTAuth()
	log.Println("✅ JWT configurado")

	// Registrar entidades com autenticação JWT
	if err := server.RegisterEntity("Users", User{}, odata.WithMiddleware(jwtMiddleware)); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	if err := server.RegisterEntity("Products", Product{}, odata.WithMiddleware(jwtMiddleware)); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	if err := server.RegisterEntity("Orders", Order{}, odata.WithMiddleware(jwtMiddleware)); err != nil {
		log.Fatal("Erro ao registrar entidade Orders:", err)
	}

	// Configurar rotas de autenticação
	setupAuthRoutes(server, userStore)

	// Imprimir informações
	printInfo()

	// Iniciar servidor
	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

func setupAuthRoutes(server *odata.Server, userStore *UserStore) {
	// POST /auth/login
	server.Post("/auth/login", func(c fiber.Ctx) error {
		var req LoginRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Dados de login inválidos"})
		}

		if req.Username == "" || req.Password == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Username e password são obrigatórios"})
		}

		log.Printf("🔐 Login attempt for user: %s from IP: %s", req.Username, c.IP())

		// Autenticar usuário
		userData, err := userStore.Authenticate(req.Username, req.Password)
		if err != nil {
			log.Printf("❌ Authentication failed: %v", err)
			return c.Status(401).JSON(fiber.Map{"error": "Credenciais inválidas"})
		}

		// Criar claims JWT
		claims := jwt.MapClaims{
			"username": userData.Username,
			"roles":    userData.Roles,
			"scopes":   userData.Scopes,
			"admin":    userData.Admin,
		}

		// Adicionar custom fields
		for k, v := range userData.Custom {
			claims[k] = v
		}

		// Gerar tokens
		accessToken, err := odata.GenerateJWT(claims)
		if err != nil {
			log.Printf("❌ Error generating access token: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar token de acesso"})
		}

		refreshToken, err := odata.GenerateRefreshToken(claims)
		if err != nil {
			log.Printf("❌ Error generating refresh token: %v", err)
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar refresh token"})
		}

		log.Printf("✅ Login successful: %s (Admin: %v)", userData.Username, userData.Admin)

		return c.JSON(fiber.Map{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    86400, // 24h
			"user": fiber.Map{
				"username": userData.Username,
				"roles":    userData.Roles,
				"scopes":   userData.Scopes,
				"admin":    userData.Admin,
				"custom":   userData.Custom,
			},
		})
	})

	// POST /auth/refresh
	server.Post("/auth/refresh", func(c fiber.Ctx) error {
		var req RefreshRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Dados de refresh inválidos"})
		}

		if req.RefreshToken == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Refresh token é obrigatório"})
		}

		// Validar refresh token
		claims, err := odata.ValidateJWT(req.RefreshToken)
		if err != nil {
			log.Printf("❌ Invalid refresh token: %v", err)
			return c.Status(401).JSON(fiber.Map{"error": "Refresh token inválido"})
		}

		username, _ := claims["username"].(string)
		log.Printf("🔄 Refresh token for user: %s", username)

		// Recarregar dados do usuário
		userData, exists := userStore.users[username]
		if !exists {
			return c.Status(401).JSON(fiber.Map{"error": "Usuário não encontrado"})
		}

		// Criar novos claims
		newClaims := jwt.MapClaims{
			"username": userData.Username,
			"roles":    userData.Roles,
			"scopes":   userData.Scopes,
			"admin":    userData.Admin,
		}

		for k, v := range userData.Custom {
			newClaims[k] = v
		}

		// Gerar novo access token
		newAccessToken, err := odata.GenerateJWT(newClaims)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar novo token"})
		}

		return c.JSON(fiber.Map{
			"access_token": newAccessToken,
			"token_type":   "Bearer",
			"expires_in":   86400,
		})
	})

	// POST /auth/logout
	server.Post("/auth/logout", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Logout realizado com sucesso"})
	})

	// GET /auth/me
	server.Get("/auth/me", func(c fiber.Ctx) error {
		claims := odata.GetJWTClaims(c)
		if claims == nil {
			return c.Status(401).JSON(fiber.Map{"error": "Token não fornecido ou inválido"})
		}

		return c.JSON(fiber.Map{
			"username": claims["username"],
			"roles":    claims["roles"],
			"scopes":   claims["scopes"],
			"admin":    claims["admin"],
			"custom":   claims,
		})
	})
}

func printInfo() {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println("  🔐 Servidor JWT com Banco de Dados")
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("👥 Usuários de teste:")
	fmt.Println("   - teste/teste123 (User)")
	fmt.Println("   - admin/teste123 (Admin)")
	fmt.Println()
	fmt.Println("🔗 Endpoints de autenticação:")
	fmt.Println("   POST /auth/login - Fazer login")
	fmt.Println("   POST /auth/refresh - Renovar token")
	fmt.Println("   POST /auth/logout - Fazer logout")
	fmt.Println("   GET /auth/me - Info do usuário")
	fmt.Println()
	fmt.Println("🔗 Endpoints OData (Protegidos):")
	fmt.Println("   GET /odata/Users")
	fmt.Println("   GET /odata/Products")
	fmt.Println("   GET /odata/Orders")
	fmt.Println()
	fmt.Println("📖 Exemplo:")
	fmt.Println("   1. POST /auth/login com {\"username\":\"teste\",\"password\":\"teste123\"}")
	fmt.Println("   2. Usar token: Authorization: Bearer <token>")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println()
}
