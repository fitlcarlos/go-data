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

// UserAuthenticator implementa autenticação com banco de dados
type UserAuthenticator struct {
}

// NewUserAuthenticator cria um novo autenticador com banco de dados
func NewUserAuthenticator() (*UserAuthenticator, error) {
	return &UserAuthenticator{}, nil
}

// Authenticate valida credenciais do usuário no banco
func (a *UserAuthenticator) Authenticate(username, password string) (*odata.UserIdentity, error) {
	// Buscar usuário no banco

	if username != "teste" {
		return nil, errors.New("usuário não encontrado ou inativo")
	}

	err := bcrypt.CompareHashAndPassword([]byte("password123"), []byte(password))

	if err != nil {
		return nil, errors.New("senha inválida")
	}

	// Criar UserIdentity
	userIdentity := &odata.UserIdentity{
		Username: username,
		Admin:    false,
	}

	return userIdentity, nil
}

// GetUserByUsername obtém usuário por username
func (a *UserAuthenticator) GetUserByUsername(username string) (*odata.UserIdentity, error) {
	// Buscar usuário no banco
	/*
		var userID int
		var email, fullName string
		var isActive, isAdmin bool

		err := a.db.QueryRow(`
			SELECT id, email, full_name, is_active, is_admin
			FROM users
			WHERE username = ? AND is_active = TRUE
		`, username).Scan(&userID, &email, &fullName, &isActive, &isAdmin)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.New("usuário não encontrado ou inativo")
			}
			return nil, fmt.Errorf("erro ao buscar usuário: %v", err)
		}
	*/

	// Criar UserIdentity
	userIdentity := &odata.UserIdentity{
		Username: "teste",
		Admin:    false,
	}

	return userIdentity, nil
}

func main() {
	// Criar servidor
	server := odata.NewServer()

	// Criar autenticador
	authenticator, err := NewUserAuthenticator()
	if err != nil {
		log.Fatal("Erro ao criar autenticador:", err)
	}

	// Configurar autenticação
	server.SetupAuthRoutes(authenticator)

	// Imprimir informações
	fmt.Println("🔐 Servidor JWT com Banco de Dados configurado!")
	fmt.Println("📋 Configurações de Autenticação:")
	fmt.Println("   - Users: Apenas administradores")
	fmt.Println("   - Products: Managers e admins (escrita), usuários (leitura)")
	fmt.Println("   - Orders: Usuários autenticados")
	fmt.Println()
	fmt.Println("👥 Usuários de teste (senha: password123):")
	fmt.Println("   - admin (Admin)")
	fmt.Println("   - manager (Manager)")
	fmt.Println("   - user (User)")
	fmt.Println()
	fmt.Println("🔗 Endpoints de autenticação:")
	fmt.Println("   - POST /auth/login - Fazer login")
	fmt.Println("   - POST /auth/refresh - Renovar token")
	fmt.Println("   - POST /auth/logout - Fazer logout")
	fmt.Println("   - GET /auth/me - Informações do usuário atual")
	fmt.Println()
	fmt.Println("🔗 Endpoints OData:")
	fmt.Println("   - GET /api/v1/Users - Lista usuários (Admin)")
	fmt.Println("   - GET /api/v1/Products - Lista produtos (Autenticado)")
	fmt.Println("   - POST /api/v1/Products - Criar produto (Manager/Admin)")
	fmt.Println("   - GET /api/v1/Orders - Lista pedidos (Autenticado)")
	fmt.Println()
	fmt.Println("📖 Exemplo de uso:")
	fmt.Println("   1. POST /auth/login com {\"username\":\"admin\",\"password\":\"password123\"}")
	fmt.Println("   2. Usar o access_token retornado no header: Authorization: Bearer <token>")
	fmt.Println("   3. Acessar endpoints protegidos")
	fmt.Println()
	fmt.Printf("💾 Banco de dados: %s\n", "jwt_example.db")

	// Adicionar endpoints de teste para demonstrar uso da conexão do contexto
	server.GetRouter().Get("/api/custom/users", func(c fiber.Ctx) error {
		// Obter conexão do contexto
		db := odata.GetDBFromContext(c)
		if db == nil {
			return c.Status(500).JSON(map[string]string{
				"error": "Conexão de banco não disponível",
			})
		}

		// Usar a conexão para fazer uma consulta
		rows, err := db.Query("SELECT id, username, email, full_name FROM users LIMIT 10")
		if err != nil {
			return c.Status(500).JSON(map[string]string{
				"error": "Erro na consulta: " + err.Error(),
			})
		}
		defer rows.Close()

		var users []map[string]interface{}
		for rows.Next() {
			var id int
			var username, email, fullName string
			if err := rows.Scan(&id, &username, &email, &fullName); err != nil {
				continue
			}
			users = append(users, map[string]interface{}{
				"id":        id,
				"username":  username,
				"email":     email,
				"full_name": fullName,
			})
		}

		return c.JSON(map[string]interface{}{
			"message": "Conexão obtida com sucesso do contexto",
			"users":   users,
			"count":   len(users),
		})
	})

	// Endpoint para testar multi-tenant (se habilitado)
	server.GetRouter().Get("/api/custom/tenant-info", func(c fiber.Ctx) error {
		// Obter conexão do contexto
		db := odata.GetDBFromContext(c)
		if db == nil {
			return c.Status(500).JSON(map[string]string{
				"error": "Conexão de banco não disponível",
			})
		}

		// Obter informações do tenant atual
		tenantID := odata.GetCurrentTenant(c)

		return c.JSON(map[string]interface{}{
			"message":            "Informações do tenant",
			"tenant_id":          tenantID,
			"database_available": db != nil,
		})
	})

	fmt.Println("🔗 Endpoints de teste adicionados:")
	fmt.Println("   - GET /api/custom/users - Testa conexão do contexto")
	fmt.Println("   - GET /api/custom/tenant-info - Testa multi-tenant")
	fmt.Println()

	// Iniciar servidor
	log.Println("🚀 Servidor iniciado com JWT e banco de dados")
	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}
