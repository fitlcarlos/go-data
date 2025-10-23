package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "github.com/fitlcarlos/go-data/pkg/providers" // Importa providers para registrar factories          // Driver SQLite
	"github.com/gofiber/fiber/v3"
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
	users map[string]*odata.UserIdentity
}

// NewUserStore cria um novo store de usuários
func NewUserStore() (*UserStore, error) {
	// Em produção, você buscaria do banco de dados
	store := &UserStore{
		users: map[string]*odata.UserIdentity{
			"teste": {
				Username: "teste",
				Admin:    false,
				Roles:    []string{"user"},
				Scopes:   []string{"read"},
			},
		},
	}
	return store, nil
}

// Authenticate valida credenciais do usuário
func (s *UserStore) Authenticate(username, password string) (*odata.UserIdentity, error) {
	// Em produção, você buscaria do banco de dados
	// Exemplo:
	// var user UserRow
	// err := db.QueryRow("SELECT id, username, password_hash, role FROM users WHERE username = ?", username).Scan(...)

	if username != "teste" {
		return nil, errors.New("usuário não encontrado ou inativo")
	}

	err := bcrypt.CompareHashAndPassword([]byte("$2a$10$...hash..."), []byte(password))
	if err != nil {
		// Para demonstração, aceita qualquer senha
		// return nil, errors.New("senha inválida")
	}

	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado")
	}

	return user, nil
}

// GetUserByUsername obtém usuário por username
func (s *UserStore) GetUserByUsername(username string) (*odata.UserIdentity, error) {
	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado ou inativo")
	}
	return user, nil
}

// LoginRequest representa os dados de login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse representa a resposta de login
type LoginResponse struct {
	AccessToken  string              `json:"access_token"`
	RefreshToken string              `json:"refresh_token"`
	TokenType    string              `json:"token_type"`
	ExpiresIn    int64               `json:"expires_in"`
	User         *odata.UserIdentity `json:"user"`
}

func main() {
	// Criar servidor
	server := odata.NewServer()

	// Criar store de usuários
	userStore, err := NewUserStore()
	if err != nil {
		log.Fatal("Erro ao criar user store:", err)
	}

	// 1. Criar JwtAuth (usa .env automaticamente)
	// Você pode passar nil para usar apenas .env, ou passar config para override
	jwtAuth := odata.NewJwtAuth(nil) // Lê JWT_SECRET do .env

	// 2. (Opcional) Customizar geração de token
	// Exemplo: Adicionar claims customizados e usar método padrão
	jwtAuth.TokenGenerator = func(user *odata.UserIdentity) (string, error) {
		// Adicionar informações extras nos custom claims
		if user.Custom == nil {
			user.Custom = make(map[string]interface{})
		}
		user.Custom["generated_at"] = time.Now().Unix()
		user.Custom["server"] = "jwt-banco-example"

		// ✅ Chama o método padrão PÚBLICO
		return jwtAuth.DefaultGenerateToken(user)
	}

	// Exemplo de validação customizada com verificações extras
	jwtAuth.TokenValidator = func(tokenString string) (*odata.UserIdentity, error) {
		// Validações extras (exemplo: blacklist)
		// if isTokenBlacklisted(tokenString) {
		//     return nil, errors.New("token revogado")
		// }

		// ✅ Chama o método padrão PÚBLICO
		user, err := jwtAuth.DefaultValidateToken(tokenString)
		if err != nil {
			return nil, err
		}

		// Validações pós-parse (exemplo: verificar se usuário está ativo)
		// if !isUserActive(user.Username) {
		//     return nil, errors.New("usuário inativo")
		// }

		return user, nil
	}

	// 3. Registrar entidades com autenticação
	if err := server.RegisterEntity("Users", User{}, odata.WithAuth(jwtAuth)); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	if err := server.RegisterEntity("Products", Product{}, odata.WithAuth(jwtAuth)); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	if err := server.RegisterEntity("Orders", Order{}, odata.WithAuth(jwtAuth)); err != nil {
		log.Fatal("Erro ao registrar entidade Orders:", err)
	}

	// 4. Criar rotas de autenticação customizadas
	router := server.GetRouter()

	// Rota de login
	router.Post("/auth/login", func(c fiber.Ctx) error {
		var req LoginRequest
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Dados de login inválidos",
			})
		}

		// Valida campos
		if req.Username == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Username e password são obrigatórios",
			})
		}

		// Autentica usuário
		user, err := userStore.Authenticate(req.Username, req.Password)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Credenciais inválidas",
			})
		}

		// Gera tokens
		accessToken, err := jwtAuth.GenerateToken(user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erro ao gerar token de acesso",
			})
		}

		refreshToken, err := jwtAuth.GenerateRefreshToken(user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erro ao gerar refresh token",
			})
		}

		response := LoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    int64(jwtAuth.GetConfig().ExpiresIn.Seconds()),
			User:         user,
		}

		return c.JSON(response)
	})

	// Rota de refresh token
	router.Post("/auth/refresh", func(c fiber.Ctx) error {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}

		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Dados de refresh inválidos",
			})
		}

		if req.RefreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Refresh token é obrigatório",
			})
		}

		// Gera novo token de acesso
		newAccessToken, err := jwtAuth.RefreshToken(req.RefreshToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Refresh token inválido",
			})
		}

		return c.JSON(fiber.Map{
			"access_token": newAccessToken,
			"token_type":   "Bearer",
			"expires_in":   int64(jwtAuth.GetConfig().ExpiresIn.Seconds()),
		})
	})

	// Rota de logout
	router.Post("/auth/logout", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Logout realizado com sucesso",
		})
	})

	// Rota para obter informações do usuário atual
	router.Get("/auth/me", odata.AuthMiddleware(jwtAuth), func(c fiber.Ctx) error {
		user := odata.GetCurrentUser(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Usuário não autenticado",
			})
		}

		return c.JSON(user)
	})

	// Endpoint customizado usando conexão do banco de dados do contexto
	router.Get("/api/custom/users", odata.AuthMiddleware(jwtAuth), func(c fiber.Ctx) error {
		// Obter conexão do contexto
		db := odata.GetDBFromContext(c)
		if db == nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Conexão de banco não disponível",
			})
		}

		// Usar a conexão para fazer uma consulta
		// rows, err := db.Query("SELECT id, username, email, full_name FROM users LIMIT 10")
		// ...

		return c.JSON(fiber.Map{
			"message": "Conexão obtida com sucesso do contexto",
			"info":    "Em produção, aqui você faria consultas ao banco",
		})
	})

	// Imprimir informações
	fmt.Println("🔐 Servidor JWT com Banco de Dados configurado!")
	fmt.Println("📋 Configurações de Autenticação:")
	fmt.Println("   - Todas as entidades requerem autenticação")
	fmt.Println()
	fmt.Println("👥 Usuários de teste (senha: qualquer):")
	fmt.Println("   - teste (User)")
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
	fmt.Println("🔗 Endpoints de teste:")
	fmt.Println("   - GET /api/custom/users - Testa conexão do contexto (Autenticado)")
	fmt.Println()
	fmt.Println("📖 Exemplo de uso:")
	fmt.Println("   1. POST /auth/login com {\"username\":\"teste\",\"password\":\"qualquer\"}")
	fmt.Println("   2. Usar o access_token retornado no header: Authorization: Bearer <token>")
	fmt.Println("   3. Acessar endpoints protegidos")
	fmt.Println()
	fmt.Println("✨ Modelo JWT Desacoplado com Banco:")
	fmt.Println("   - JWT customizado com claims extras")
	fmt.Println("   - Integração com banco de dados")
	fmt.Println("   - Conexão disponível no contexto via GetDBFromContext()")
	fmt.Println()

	// Iniciar servidor
	log.Println("🚀 Servidor iniciado com JWT e banco de dados")
	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}
