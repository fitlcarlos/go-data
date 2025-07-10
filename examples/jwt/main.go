package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/godata/pkg/odata"
	"github.com/fitlcarlos/godata/pkg/providers"
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

// SimpleUserAuthenticator implementa UserAuthenticator para demonstração
type SimpleUserAuthenticator struct {
	users map[string]*odata.UserIdentity
}

// NewSimpleUserAuthenticator cria um novo autenticador simples
func NewSimpleUserAuthenticator() *SimpleUserAuthenticator {
	return &SimpleUserAuthenticator{
		users: map[string]*odata.UserIdentity{
			"admin": {
				Username: "admin",
				Roles:    []string{"admin", "user"},
				Scopes:   []string{"read", "write", "delete"},
				Admin:    true,
				Custom: map[string]interface{}{
					"department": "IT",
					"level":      "senior",
				},
			},
			"manager": {
				Username: "manager",
				Roles:    []string{"manager", "user"},
				Scopes:   []string{"read", "write"},
				Admin:    false,
				Custom: map[string]interface{}{
					"department": "Sales",
					"level":      "manager",
				},
			},
			"user": {
				Username: "user",
				Roles:    []string{"user"},
				Scopes:   []string{"read"},
				Admin:    false,
				Custom: map[string]interface{}{
					"department": "Marketing",
					"level":      "junior",
				},
			},
		},
	}
}

// Authenticate autentica um usuário
func (a *SimpleUserAuthenticator) Authenticate(username, password string) (*odata.UserIdentity, error) {
	// Simulação simples: senha deve ser "password123"
	if password != "password123" {
		return nil, errors.New("senha inválida")
	}

	user, exists := a.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado")
	}

	return user, nil
}

// GetUserByUsername obtém um usuário pelo nome de usuário
func (a *SimpleUserAuthenticator) GetUserByUsername(username string) (*odata.UserIdentity, error) {
	user, exists := a.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado")
	}
	return user, nil
}

func main() {
	// Configurar provider de banco de dados
	provider := providers.NewPostgreSQLProvider()
	// Conectar ao banco (em um caso real, você configuraria a connection string)
	// provider.Connect("postgres://user:password@localhost:5432/testdb?sslmode=disable")

	// Configurar JWT
	jwtConfig := &odata.JWTConfig{
		SecretKey: "minha-chave-secreta-super-segura-123",
		Issuer:    "exemplo-godata-jwt",
		ExpiresIn: 1 * time.Hour,
		RefreshIn: 24 * time.Hour,
		Algorithm: "HS256",
	}

	// Configurar servidor com JWT
	config := odata.DefaultServerConfig()
	config.Host = "localhost"
	config.Port = 8080
	config.RoutePrefix = "/api/v1"
	config.EnableJWT = true
	config.JWTConfig = jwtConfig
	config.RequireAuth = false // Não requer autenticação global por padrão

	// Criar servidor
	server := odata.NewServerWithConfig(provider, config)

	// Configurar autenticador
	authenticator := NewSimpleUserAuthenticator()
	server.SetupAuthRoutes(authenticator)

	// Registrar entidades
	if err := server.RegisterEntity("Users", User{}); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	if err := server.RegisterEntity("Products", Product{}); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	if err := server.RegisterEntity("Orders", Order{}); err != nil {
		log.Fatal("Erro ao registrar entidade Orders:", err)
	}

	// Configurar autenticação por entidade

	// Users: Apenas administradores podem acessar
	server.SetEntityAuth("Users", odata.EntityAuthConfig{
		RequireAuth:  true,
		RequireAdmin: true,
	})

	// Products: Managers e admins podem escrever, usuários podem ler
	server.SetEntityAuth("Products", odata.EntityAuthConfig{
		RequireAuth:    true,
		RequiredRoles:  []string{"manager", "admin"},
		RequiredScopes: []string{"write"},
	})

	// Orders: Usuários autenticados podem ler, apenas managers podem escrever
	server.SetEntityAuth("Orders", odata.EntityAuthConfig{
		RequireAuth:   true,
		RequiredRoles: []string{"user"},
	})

	// Configurar algumas entidades como somente leitura para usuários comuns
	// (Esta configuração seria aplicada dinamicamente baseada no usuário)

	// Imprimir informações de configuração
	fmt.Println("🔐 Servidor JWT configurado!")
	fmt.Println("📋 Configurações de Autenticação:")
	fmt.Println("   - Users: Apenas administradores")
	fmt.Println("   - Products: Managers e admins (escrita), usuários (leitura)")
	fmt.Println("   - Orders: Usuários autenticados")
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

	// Iniciar servidor
	log.Printf("Iniciando servidor na porta %d...", config.Port)
	if err := server.StartWithContext(context.Background()); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}
