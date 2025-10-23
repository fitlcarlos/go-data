package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "github.com/fitlcarlos/go-data/pkg/providers" // Importa providers para registrar factories
	"github.com/joho/godotenv"
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
	users map[string]*odata.UserIdentity
}

// NewSimpleUserStore cria um novo store de usuários
func NewSimpleUserStore() *SimpleUserStore {
	return &SimpleUserStore{
		users: map[string]*odata.UserIdentity{
			"admin": {
				Username: "admin",
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

// AuthenticateWithContext autentica usuário durante login com acesso ao contexto enriquecido
// Demonstra uso do AuthContext para acesso a ObjectManager, Connection, etc
func (s *SimpleUserStore) AuthenticateWithContext(ctx *odata.AuthContext, username, password string) (*odata.UserIdentity, error) {
	// Acesso ao ObjectManager durante autenticação
	manager := ctx.GetManager()

	log.Printf("🔐 Login attempt for user: %s from IP: %s", username, ctx.IP())

	// EXEMPLO 1: Buscar usuário no banco via ObjectManager
	// Descomente se tiver a tabela Users configurada no banco
	/*
		userFromDB, err := manager.Find("Users", username)
		if err != nil {
			log.Printf("❌ User not found in database: %s", username)
			// Fallback para users em memória
		} else {
			log.Printf("✅ User found in database")
			// Aqui você validaria o hash da senha contra o banco
			if userMap, ok := userFromDB.(map[string]interface{}); ok {
				// Valide senha hash aqui
				passwordHash := userMap["password"].(string)
				// bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
			}
		}
	*/

	// EXEMPLO 2: Rate limiting usando conexão SQL direta
	conn := ctx.GetConnection()
	if conn != nil {
		// Exemplo: verificar tentativas de login
		// Descomente se tiver a tabela user_security
		/*
			var attempts int
			err := conn.QueryRowContext(ctx.FiberContext.Context(),
				"SELECT COALESCE(login_attempts, 0) FROM user_security WHERE username = ?",
				username).Scan(&attempts)

			if err == nil && attempts > 5 {
				log.Printf("🚫 Account locked due to too many attempts: %s", username)
				return nil, errors.New("conta bloqueada por múltiplas tentativas")
			}
		*/
		log.Printf("📊 Connection available for rate limiting/audit")
	}

	// EXEMPLO 3: Audit log de tentativa de login
	if manager != nil {
		auditLog := map[string]any{
			"event":      "login_attempt",
			"username":   username,
			"ip":         ctx.IP(),
			"tenant":     ctx.GetTenantID(),
			"user_agent": ctx.GetHeader("User-Agent"),
		}
		// Descomente se tiver tabela de audit log
		// manager.Save(auditLog)
		log.Printf("📝 Audit log: %+v", auditLog)
	}

	// Validação simples de senha (para demo)
	if password != "password123" {
		// Incrementar contador de tentativas falhas
		// Descomente se tiver a tabela user_security
		/*
			if conn != nil {
				conn.ExecContext(ctx.FiberContext.Context(),
					"UPDATE user_security SET login_attempts = login_attempts + 1 WHERE username = ?",
					username)
			}
		*/
		return nil, errors.New("senha inválida")
	}

	// Buscar usuário da lista em memória
	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado")
	}

	// Resetar contador de tentativas em caso de sucesso
	// Descomente se tiver a tabela user_security
	/*
		if conn != nil {
			conn.ExecContext(ctx.FiberContext.Context(),
				"UPDATE user_security SET login_attempts = 0, last_login = NOW() WHERE username = ?",
				username)
		}
	*/

	log.Printf("✅ Login successful for user: %s", username)
	return user, nil
}

// RefreshToken recarrega/valida dados do usuário durante refresh token
// O contexto está disponível para validar no banco, mas não é obrigatório usar
func (s *SimpleUserStore) RefreshToken(ctx *odata.AuthContext, username string) (*odata.UserIdentity, error) {
	log.Printf("🔄 Refreshing token for user: %s from IP: %s", username, ctx.IP())

	// OPCIONAL: Buscar dados atualizados do banco
	manager := ctx.GetManager()
	if manager != nil {
		// Exemplo: você poderia recarregar roles/permissions do banco
		// userFromDB, err := manager.Find("Users", username)
		// if err == nil {
		//     log.Printf("✅ User data refreshed from database")
		//     // Converter userFromDB para UserIdentity
		// }
	}

	// Fallback para usuário em memória
	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("usuário não encontrado")
	}

	// Audit log do refresh
	log.Printf("📝 Token refreshed: user=%s, ip=%s, tenant=%s",
		username, ctx.IP(), ctx.GetTenantID())

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
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	log.Println("✅ Configurações carregadas do .env")

	// Cria o servidor
	server := odata.NewServer()

	// Cria o store de usuários
	userStore := NewSimpleUserStore()

	// ============================================================================
	// OPÇÃO 1: Usar apenas .env (RECOMENDADO)
	// ============================================================================
	// JWT lê automaticamente JWT_SECRET, JWT_EXPIRATION, etc do .env
	jwtAuth := odata.NewJwtAuth(nil)
	log.Println("✅ JWT configurado via .env")

	// ============================================================================
	// OPÇÃO 2: Override parcial (usa .env + customizações)
	// ============================================================================
	// Mantém JWT_SECRET do .env, mas override expiration
	customJwtAuth := odata.NewJwtAuth(&odata.JWTConfig{
		ExpiresIn: 2 * time.Hour, // Override apenas isso
	})

	// ============================================================================
	// OPÇÃO 3: Override total (ignora .env)
	// ============================================================================
	// Configuração completamente manual (não recomendado para produção)
	// manualJwtAuth := odata.NewJwtAuth(&odata.JWTConfig{
	//     SecretKey: "your-custom-secret-key-min-32-chars",
	//     Issuer:    "custom-issuer",
	//     ExpiresIn: 30 * time.Minute,
	//     RefreshIn: 7 * 24 * time.Hour,
	//     Algorithm: "HS256",
	// })

	// Registrar entidades com autenticação
	// Users: Protegido com JWT (usa .env)
	if err := server.RegisterEntity("Users", User{}, odata.WithAuth(jwtAuth)); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	// Products: Protegido com JWT customizado (override expiration)
	if err := server.RegisterEntity("Products", Product{}, odata.WithAuth(customJwtAuth)); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	// Orders: Protegido com mesmo JWT
	if err := server.RegisterEntity("Orders", Order{},
		odata.WithAuth(jwtAuth),
		odata.WithReadOnly(false),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Orders:", err)
	}

	// 4. Configurar rotas de autenticação
	// SetupAuthRoutes detecta automaticamente se o authenticator implementa
	// ContextAuthenticator e usa AuthContext quando disponível
	server.SetupAuthRoutes(userStore)

	// Imprimir informações de configuração
	fmt.Println("🔐 Servidor JWT com AuthContext configurado!")
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
	fmt.Println("✨ NOVO: AuthContext - Autenticação com Contexto Enriquecido!")
	fmt.Println("   - Acesso ao ObjectManager durante login")
	fmt.Println("   - Conexão SQL direta para rate limiting e audit")
	fmt.Println("   - Suporte a multi-tenant na autenticação")
	fmt.Println("   - IP, Headers e outras informações do request")
	fmt.Println("   - Backward compatible com método Authenticate() legado")
	fmt.Println()
	fmt.Println("🔧 Recursos disponíveis no AuthContext:")
	fmt.Println("   - ctx.GetManager() - ObjectManager para ORM")
	fmt.Println("   - ctx.GetConnection() - Conexão SQL direta")
	fmt.Println("   - ctx.GetProvider() - DatabaseProvider")
	fmt.Println("   - ctx.GetPool() - Pool multi-tenant")
	fmt.Println("   - ctx.IP() - Endereço IP do cliente")
	fmt.Println("   - ctx.GetHeader() - Headers da requisição")
	fmt.Println()

	// Iniciar servidor
	log.Println("🚀 Servidor iniciado com configurações automaticamente carregadas")
	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}
