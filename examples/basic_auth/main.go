package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fitlcarlos/go-data/pkg/auth/basic"
	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// User modelo de exemplo
type User struct {
	ID        int       `json:"id" db:"id" odata:"key,filterable,sortable"`
	Username  string    `json:"username" db:"username" odata:"filterable,sortable,searchable"`
	Password  string    `json:"-" db:"password"` // Nunca expor senha no JSON
	Email     string    `json:"email" db:"email" odata:"filterable,sortable,searchable"`
	Role      string    `json:"role" db:"role" odata:"filterable"`
	Active    bool      `json:"active" db:"active" odata:"filterable"`
	CreatedAt time.Time `json:"created_at" db:"created_at" odata:"sortable"`
}

// Product modelo de produto (protegido por auth)
type Product struct {
	ID          int       `json:"id" db:"id" odata:"key,filterable,sortable"`
	Name        string    `json:"name" db:"name" odata:"filterable,sortable,searchable"`
	Description string    `json:"description" db:"description" odata:"searchable"`
	Price       float64   `json:"price" db:"price" odata:"filterable,sortable"`
	Stock       int       `json:"stock" db:"stock" odata:"filterable,sortable"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" odata:"sortable"`
}

var db *sql.DB

// HashPassword cria um hash bcrypt de uma senha
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash verifica se a senha corresponde ao hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// getEnv obtém uma variável de ambiente com valor padrão
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	// Carregar variáveis de ambiente
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	// Validar variáveis de ambiente críticas
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Fatal("❌ DB_PASSWORD não configurado. Defina no .env ou nas variáveis de ambiente")
	}

	// Conectar ao banco de dados
	var err error
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbName := getEnv("DB_NAME", "odata_test")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Erro ao conectar ao banco:", err)
	}
	defer db.Close()

	// Configurar pool de conexões
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verificar conexão
	if err := db.Ping(); err != nil {
		log.Fatal("Erro ao verificar conexão:", err)
	}

	// Criar tabelas se não existirem
	createTables()

	// Criar alguns usuários de exemplo
	seedUsers()

	// Configurar servidor OData
	server := odata.NewServer()

	// Configurar Basic Authentication COM AuthContext (recomendado)
	// Usa validateUserWithContext que tem acesso ao ObjectManager, Connection, etc
	basicAuth := basic.NewBasicAuthWithContext(
		server, // Passa o server para acesso ao AuthContext
		&basic.BasicAuthConfig{
			Realm: "OData API with AuthContext",
		},
		validateUserWithContext, // Nova função com contexto enriquecido
	)

	// ALTERNATIVA: Usar método legado (sem contexto)
	// basicAuth := basic.NewBasicAuth(
	//     &basic.BasicAuthConfig{Realm: "OData API"},
	//     validateUser, // Função legada sem contexto
	// )

	// Registrar entidade Users (somente leitura, protegida por auth)
	if err := server.RegisterEntity("Users", User{},
		odata.WithAuth(basicAuth),
		odata.WithReadOnly(true),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	// Registrar entidade Products (leitura/escrita, protegida por auth)
	if err := server.RegisterEntity("Products", Product{},
		odata.WithAuth(basicAuth),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	// Rota pública de informações
	server.GetRouter().Get("/api/v1/info", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":    "OData API com Basic Authentication",
			"version":    "1.0",
			"auth_type":  "Basic",
			"realm":      basicAuth.GetRealm(),
			"howto_auth": "Envie header: Authorization: Basic base64(username:password)",
			"test_users": fiber.Map{
				"admin": fiber.Map{
					"username": "admin",
					"password": "admin123",
					"role":     "admin",
				},
				"user": fiber.Map{
					"username": "user",
					"password": "user123",
					"role":     "user",
				},
			},
		})
	})

	// Rota para verificar usuário autenticado
	server.GetRouter().Get("/api/v1/me", basic.BasicAuthMiddleware(basicAuth), func(c fiber.Ctx) error {
		user := odata.GetCurrentUser(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Não autenticado",
			})
		}

		role := ""
		if len(user.Roles) > 0 {
			role = user.Roles[0]
		}

		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"id":       user.Custom["user_id"],
				"username": user.Username,
				"email":    user.Custom["email"],
				"role":     role,
				"admin":    user.Admin,
			},
		})
	})

	// Iniciar servidor
	fmt.Println("\n🚀 Servidor OData com Basic Auth + AuthContext iniciado em http://localhost:3000")
	fmt.Println("\n📋 Endpoints disponíveis:")
	fmt.Println("  GET  /api/v1/info              - Informações da API (público)")
	fmt.Println("  GET  /api/v1/me                - Usuário autenticado (requer auth)")
	fmt.Println("  GET  /api/v1/Users             - Listar usuários (requer auth)")
	fmt.Println("  GET  /api/v1/Products          - Listar produtos (requer auth)")
	fmt.Println("  POST /api/v1/Products          - Criar produto (requer auth)")
	fmt.Println("\n🔐 Credenciais de teste:")
	fmt.Println("  Admin: username=admin, password=admin123")
	fmt.Println("  User:  username=user, password=user123")
	fmt.Println("\n💡 Exemplo de uso:")
	fmt.Println(`  curl -u admin:admin123 http://localhost:3000/api/v1/Users`)
	fmt.Println(`  curl -H "Authorization: Basic YWRtaW46YWRtaW4xMjM=" http://localhost:3000/api/v1/Users`)
	fmt.Println()
	fmt.Println("✨ NOVO: AuthContext - Autenticação com Contexto Enriquecido!")
	fmt.Println("  - Acesso ao ObjectManager durante validação")
	fmt.Println("  - Conexão SQL direta para rate limiting e audit")
	fmt.Println("  - IP, Headers e outras informações do request")
	fmt.Println("  - Rate limiting automático (bloqueia após 5 tentativas)")
	fmt.Println("  - Backward compatible com validateUser() legado")
	fmt.Println()

	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

// validateUser valida credenciais do usuário no banco de dados usando bcrypt (legado)
func validateUser(username, password string) (*odata.UserIdentity, error) {
	log.Printf("⚠️  Using legacy validateUser method (without context)")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	var passwordHash string

	// Buscar usuário e hash da senha
	query := `SELECT id, username, password, email, role, active FROM users WHERE username = ? AND active = 1`
	err := db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&passwordHash,
		&user.Email,
		&user.Role,
		&user.Active,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("credenciais inválidas")
		}
		return nil, fmt.Errorf("erro ao consultar usuário: %w", err)
	}

	// Verificar senha usando bcrypt
	if !CheckPasswordHash(password, passwordHash) {
		return nil, fmt.Errorf("credenciais inválidas")
	}

	// Converter para UserIdentity
	return &odata.UserIdentity{
		Username: user.Username,
		Roles:    []string{user.Role},
		Admin:    user.Role == "admin",
		Custom: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"active":  user.Active,
		},
	}, nil
}

// validateUserWithContext - NOVA função com contexto enriquecido
// Demonstra uso do AuthContext para acesso a ObjectManager, Connection, etc
func validateUserWithContext(authCtx basic.AuthContext, username, password string) (*odata.UserIdentity, error) {
	log.Printf("🔐 Basic Auth: %s from IP: %s", username, authCtx.IP())

	// EXEMPLO 1: Usar ObjectManager para buscar usuário (opcional)
	// manager := authCtx.GetManager()
	// user, err := manager.Find("Users", username)

	// EXEMPLO 2: Usar conexão SQL direta com informações do contexto
	connInterface := authCtx.GetConnection()
	conn, ok := connInterface.(*sql.DB)
	if !ok || conn == nil {
		return nil, fmt.Errorf("conexão não disponível")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	var passwordHash string
	var loginAttempts int

	// Buscar usuário, hash da senha e tentativas de login
	query := `SELECT id, username, password, email, role, active, 
	          COALESCE((SELECT attempts FROM user_security WHERE user_id = users.id), 0) as attempts
	          FROM users WHERE username = ? AND active = 1`
	err := conn.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&passwordHash,
		&user.Email,
		&user.Role,
		&user.Active,
		&loginAttempts,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ User not found: %s", username)
			return nil, fmt.Errorf("credenciais inválidas")
		}
		log.Printf("❌ Database error: %v", err)
		return nil, fmt.Errorf("erro ao consultar usuário: %w", err)
	}

	// EXEMPLO 3: Rate limiting - bloquear após muitas tentativas
	if loginAttempts > 5 {
		log.Printf("🚫 Account locked due to too many attempts: %s", username)
		return nil, fmt.Errorf("conta bloqueada por múltiplas tentativas")
	}

	// Verificar senha usando bcrypt
	if !CheckPasswordHash(password, passwordHash) {
		// Incrementar contador de tentativas falhas
		_, err = conn.ExecContext(ctx,
			`INSERT INTO user_security (user_id, attempts, last_attempt) 
			 VALUES (?, 1, NOW()) 
			 ON DUPLICATE KEY UPDATE attempts = attempts + 1, last_attempt = NOW()`,
			user.ID)
		if err != nil {
			log.Printf("⚠️  Failed to update login attempts: %v", err)
		}

		log.Printf("❌ Invalid password for user: %s", username)
		return nil, fmt.Errorf("credenciais inválidas")
	}

	// EXEMPLO 4: Audit log - registrar login bem-sucedido
	providerInterface := authCtx.GetProvider()
	tenantID := authCtx.GetTenantID()
	userAgent := authCtx.GetHeader("User-Agent")

	providerName := "unknown"
	if provider, ok := providerInterface.(odata.DatabaseProvider); ok && provider != nil {
		providerName = provider.GetDriverName()
	}

	log.Printf("✅ Login successful: %s (ID: %d, Role: %s, Tenant: %s, Provider: %s, IP: %s, UA: %s)",
		user.Username, user.ID, user.Role, tenantID, providerName, authCtx.IP(), userAgent)

	// Resetar contador de tentativas
	_, err = conn.ExecContext(ctx,
		`INSERT INTO user_security (user_id, attempts, last_attempt, last_success) 
		 VALUES (?, 0, NOW(), NOW()) 
		 ON DUPLICATE KEY UPDATE attempts = 0, last_success = NOW()`,
		user.ID)
	if err != nil {
		log.Printf("⚠️  Failed to reset login attempts: %v", err)
	}

	// Converter para UserIdentity
	return &odata.UserIdentity{
		Username: user.Username,
		Roles:    []string{user.Role},
		Admin:    user.Role == "admin",
		Custom: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"active":  user.Active,
			"ip":      authCtx.IP(),
			"tenant":  tenantID,
		},
	}, nil
}

// createTables cria as tabelas necessárias
func createTables() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tabela de usuários
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		role VARCHAR(20) NOT NULL DEFAULT 'user',
		active BOOLEAN NOT NULL DEFAULT true,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	if _, err := db.ExecContext(ctx, usersTable); err != nil {
		log.Printf("Aviso ao criar tabela users: %v", err)
	}

	// Tabela de produtos
	productsTable := `
	CREATE TABLE IF NOT EXISTS products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		description TEXT,
		price DECIMAL(10,2) NOT NULL,
		stock INT NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	if _, err := db.ExecContext(ctx, productsTable); err != nil {
		log.Printf("Aviso ao criar tabela products: %v", err)
	}

	log.Println("✅ Tabelas verificadas/criadas")
}

// seedUsers cria usuários de exemplo com senhas hasheadas
func seedUsers() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	users := []struct {
		username string
		password string
		email    string
		role     string
	}{
		{"admin", "admin123", "admin@example.com", "admin"},
		{"user", "user123", "user@example.com", "user"},
		{"manager", "manager123", "manager@example.com", "manager"},
	}

	for _, u := range users {
		// Gerar hash bcrypt da senha
		passwordHash, err := HashPassword(u.password)
		if err != nil {
			log.Printf("Erro ao gerar hash para usuário %s: %v", u.username, err)
			continue
		}

		query := `INSERT INTO users (username, password, email, role) VALUES (?, ?, ?, ?)
				  ON DUPLICATE KEY UPDATE email=?, role=?`
		_, err = db.ExecContext(ctx, query, u.username, passwordHash, u.email, u.role, u.email, u.role)
		if err != nil {
			log.Printf("Aviso ao criar usuário %s: %v", u.username, err)
		}
	}

	// Criar alguns produtos de exemplo
	products := []struct {
		name        string
		description string
		price       float64
		stock       int
	}{
		{"Notebook", "Notebook Dell Inspiron 15", 3500.00, 10},
		{"Mouse", "Mouse Logitech MX Master", 450.00, 50},
		{"Teclado", "Teclado Mecânico RGB", 350.00, 30},
	}

	for _, p := range products {
		query := `INSERT INTO products (name, description, price, stock) VALUES (?, ?, ?, ?)
				  ON DUPLICATE KEY UPDATE name=name`
		_, err := db.ExecContext(ctx, query, p.name, p.description, p.price, p.stock)
		if err != nil {
			log.Printf("Aviso ao criar produto %s: %v", p.name, err)
		}
	}

	log.Println("✅ Dados de exemplo criados")
}
