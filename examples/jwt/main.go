package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "github.com/fitlcarlos/go-data/pkg/providers" // Importa providers para registrar factories
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
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

// SimpleUserStore armazena usuários em memória para demonstração
type SimpleUserStore struct {
	users map[string]*UserData
}

type UserData struct {
	Username string
	Password string
	Roles    []string
	Scopes   []string
	Admin    bool
	Custom   map[string]interface{}
}

// NewSimpleUserStore cria um novo store de usuários
func NewSimpleUserStore() *SimpleUserStore {
	return &SimpleUserStore{
		users: map[string]*UserData{
			"admin": {
				Username: "admin",
				Password: "password123",
				Roles:    []string{"admin", "user"},
				Scopes:   []string{"read", "write", "delete"},
				Admin:    true,
				Custom: map[string]interface{}{
					"name":       "Administrator",
					"email":      "admin@example.com",
					"department": "IT",
					"level":      "senior",
				},
			},
			"manager": {
				Username: "manager",
				Password: "password123",
				Roles:    []string{"manager", "user"},
				Scopes:   []string{"read", "write"},
				Admin:    false,
				Custom: map[string]interface{}{
					"name":       "Manager User",
					"email":      "manager@example.com",
					"department": "Sales",
					"level":      "intermediate",
				},
			},
			"user": {
				Username: "user",
				Password: "password123",
				Roles:    []string{"user"},
				Scopes:   []string{"read"},
				Admin:    false,
				Custom: map[string]interface{}{
					"name":       "Regular User",
					"email":      "user@example.com",
					"department": "Customer Service",
					"level":      "junior",
				},
			},
		},
	}
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
	// Cria o servidor
	server := odata.NewServer()

	// Cria o store de usuários
	userStore := NewSimpleUserStore()

	// Criar middleware JWT (carrega configurações do .env automaticamente)
	jwtMiddleware := server.NewRouterJWTAuth()
	log.Println("✅ JWT configurado via .env")

	// Registrar entidades com autenticação
	if err := server.RegisterEntity("Users", User{}, odata.WithMiddleware(jwtMiddleware)); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	if err := server.RegisterEntity("Products", Product{}, odata.WithMiddleware(jwtMiddleware)); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	if err := server.RegisterEntity("Orders", Order{},
		odata.WithMiddleware(jwtMiddleware),
		odata.WithReadOnly(false),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Orders:", err)
	}

	// Configurar rotas de autenticação manualmente
	setupAuthRoutes(server, userStore)

	// Imprimir informações de configuração
	fmt.Println("🔐 Servidor JWT configurado!")
	fmt.Println("📋 Configurações de Autenticação:")
	fmt.Println("   - Users: Autenticação requerida")
	fmt.Println("   - Products: Autenticação requerida")
	fmt.Println("   - Orders: Autenticação requerida")
	fmt.Println()
	fmt.Println("👥 Usuários de teste:")
	fmt.Println("   - admin/password123 (Admin)")
	fmt.Println("   - manager/password123 (Manager)")
	fmt.Println("   - user/password123 (User)")
	fmt.Println()
	fmt.Println("🔗 Endpoints de autenticação:")
	fmt.Println("   - POST /auth/login - Fazer login")
	fmt.Println("   - POST /auth/refresh - Renovar token")
	fmt.Println("   - POST /auth/logout - Fazer logout")
	fmt.Println("   - GET /auth/me - Informações do usuário atual")
	fmt.Println()
	fmt.Println("🔗 Endpoints OData:")
	fmt.Println("   - GET /odata/Users - Lista usuários (Autenticado)")
	fmt.Println("   - GET /odata/Products - Lista produtos (Autenticado)")
	fmt.Println("   - POST /odata/Products - Criar produto (Autenticado)")
	fmt.Println("   - GET /odata/Orders - Lista pedidos (Autenticado)")
	fmt.Println()
	fmt.Println("📖 Exemplo de uso:")
	fmt.Println("   1. POST /auth/login com {\"username\":\"admin\",\"password\":\"password123\"}")
	fmt.Println("   2. Usar o access_token retornado no header: Authorization: Bearer <token>")
	fmt.Println("   3. Acessar endpoints protegidos")
	fmt.Println()

	// Iniciar servidor
	log.Println("🚀 Servidor iniciado com configurações automaticamente carregadas")
	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

func setupAuthRoutes(server *odata.Server, userStore *SimpleUserStore) {
	// POST /auth/login - Login e geração de token
	server.Post("/auth/login", func(c fiber.Ctx) error {
		var req LoginRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Dados de login inválidos"})
		}

		if req.Username == "" || req.Password == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Username e password são obrigatórios"})
		}

		log.Printf("🔐 Login attempt for user: %s from IP: %s", req.Username, c.IP())

		// Validar usuário
		userData, exists := userStore.users[req.Username]
		if !exists || userData.Password != req.Password {
			log.Printf("❌ Invalid credentials for user: %s", req.Username)
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
			"expires_in":    86400, // 24h em segundos
			"user": fiber.Map{
				"username": userData.Username,
				"roles":    userData.Roles,
				"scopes":   userData.Scopes,
				"admin":    userData.Admin,
				"custom":   userData.Custom,
			},
		})
	})

	// POST /auth/refresh - Renovar access token
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
		log.Printf("🔄 Refresh token for user: %s from IP: %s", username, c.IP())

		// Recarregar dados do usuário
		userData, exists := userStore.users[username]
		if !exists {
			log.Printf("❌ User not found during refresh: %s", username)
			return c.Status(401).JSON(fiber.Map{"error": "Usuário não encontrado"})
		}

		// Criar novos claims
		newClaims := jwt.MapClaims{
			"username": userData.Username,
			"roles":    userData.Roles,
			"scopes":   userData.Scopes,
			"admin":    userData.Admin,
		}

		// Adicionar custom fields
		for k, v := range userData.Custom {
			newClaims[k] = v
		}

		// Gerar novo access token
		newAccessToken, err := odata.GenerateJWT(newClaims)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao gerar novo token"})
		}

		log.Printf("✅ Token refreshed: %s (Admin: %v)", username, userData.Admin)

		return c.JSON(fiber.Map{
			"access_token": newAccessToken,
			"token_type":   "Bearer",
			"expires_in":   86400, // 24h em segundos
		})
	})

	// POST /auth/logout - Logout
	server.Post("/auth/logout", func(c fiber.Ctx) error {
		log.Printf("👋 Logout from IP: %s", c.IP())
		return c.JSON(fiber.Map{
			"message": "Logout realizado com sucesso",
		})
	})

	// GET /auth/me - Informações do usuário autenticado
	server.Get("/auth/me", func(c fiber.Ctx) error {
		// Extrair claims do JWT
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
