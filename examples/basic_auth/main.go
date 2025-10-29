package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/fitlcarlos/go-data/odata"
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
		log.Println("⚠️  DB_PASSWORD não configurado. Usando banco in-memory para demo.")
		// Para demonstração, vamos usar um mapa em memória
		runInMemoryDemo()
		return
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

	// Executar servidor com banco de dados
	runWithDatabase()
}

func runInMemoryDemo() {
	log.Println("🚀 Executando demo com usuários em memória")

	// Configurar servidor OData
	server := odata.NewServer()

	// Criar middleware Basic Auth com validator customizado
	basicAuthMiddleware := server.NewRouterBasicAuth(func(username, password string) bool {
		// Validar credenciais (em produção, use banco de dados e bcrypt)
		validUsers := map[string]string{
			"admin": "admin123",
			"user":  "user123",
		}

		expectedPassword, exists := validUsers[username]
		return exists && expectedPassword == password
	}, &odata.BasicAuthConfig{
		Realm: "OData API Demo",
	})

	// Registrar entidade Users (somente leitura, protegida por auth)
	if err := server.RegisterEntity("Users", User{},
		odata.WithMiddleware(basicAuthMiddleware),
		odata.WithReadOnly(true),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	// Registrar entidade Products (leitura/escrita, protegida por auth)
	if err := server.RegisterEntity("Products", Product{},
		odata.WithMiddleware(basicAuthMiddleware),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	// Rota pública de informações
	server.Get("/api/v1/info", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":    "OData API com Basic Authentication (In-Memory)",
			"version":    "1.0",
			"auth_type":  "Basic",
			"realm":      "OData API Demo",
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

	// Rota para verificar usuário autenticado (protegida)
	server.Get("/api/v1/me", basicAuthMiddleware, func(c fiber.Ctx) error {
		username := odata.GetBasicAuthUsername(c)
		if username == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Não autenticado",
			})
		}

		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"username":  username,
				"auth_type": "basic",
			},
		})
	})

	printInfo("In-Memory Demo")

	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

func runWithDatabase() {
	// Configurar servidor OData
	server := odata.NewServer()

	// Criar middleware Basic Auth com validator que consulta o banco
	basicAuthMiddleware := server.NewRouterBasicAuth(func(username, password string) bool {
		return validateUserWithDatabase(username, password)
	}, &odata.BasicAuthConfig{
		Realm: "OData API with Database",
	})

	// Registrar entidade Users (somente leitura, protegida por auth)
	if err := server.RegisterEntity("Users", User{},
		odata.WithMiddleware(basicAuthMiddleware),
		odata.WithReadOnly(true),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Users:", err)
	}

	// Registrar entidade Products (leitura/escrita, protegida por auth)
	if err := server.RegisterEntity("Products", Product{},
		odata.WithMiddleware(basicAuthMiddleware),
	); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	// Rota pública de informações
	server.Get("/api/v1/info", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":    "OData API com Basic Authentication",
			"version":    "1.0",
			"auth_type":  "Basic",
			"realm":      "OData API with Database",
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

	// Rota para verificar usuário autenticado (protegida)
	server.Get("/api/v1/me", basicAuthMiddleware, func(c fiber.Ctx) error {
		username := odata.GetBasicAuthUsername(c)
		if username == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Não autenticado",
			})
		}

		// Buscar informações adicionais do usuário no banco
		user, err := getUserInfo(username)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Erro ao buscar informações do usuário",
			})
		}

		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		})
	})

	printInfo("Database Mode")

	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

// validateUserWithDatabase valida credenciais do usuário no banco de dados
func validateUserWithDatabase(username, password string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var passwordHash string

	// Buscar hash da senha do usuário
	query := `SELECT password FROM users WHERE username = ? AND active = 1`
	err := db.QueryRowContext(ctx, query, username).Scan(&passwordHash)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("❌ User not found: %s", username)
		} else {
			log.Printf("❌ Database error: %v", err)
		}
		return false
	}

	// Verificar senha usando bcrypt
	if !CheckPasswordHash(password, passwordHash) {
		log.Printf("❌ Invalid password for user: %s", username)
		return false
	}

	log.Printf("✅ Login successful: %s", username)
	return true
}

// getUserInfo busca informações do usuário no banco
func getUserInfo(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User

	query := `SELECT id, username, email, role, active, created_at FROM users WHERE username = ? AND active = 1`
	err := db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.Active,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func printInfo(mode string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Printf("  🔐 Servidor OData com Basic Auth (%s)\n", mode)
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("📋 Endpoints disponíveis:")
	fmt.Println("  GET  /api/v1/info              - Informações da API (público)")
	fmt.Println("  GET  /api/v1/me                - Usuário autenticado (requer auth)")
	fmt.Println("  GET  /api/v1/Users             - Listar usuários (requer auth)")
	fmt.Println("  GET  /api/v1/Products          - Listar produtos (requer auth)")
	fmt.Println("  POST /api/v1/Products          - Criar produto (requer auth)")
	fmt.Println()
	fmt.Println("🔐 Credenciais de teste:")
	fmt.Println("  Admin: username=admin, password=admin123")
	fmt.Println("  User:  username=user, password=user123")
	fmt.Println()
	fmt.Println("💡 Exemplo de uso:")
	fmt.Println(`  curl -u admin:admin123 http://localhost:3000/api/v1/Users`)
	fmt.Println(`  curl -H "Authorization: Basic YWRtaW46YWRtaW4xMjM=" http://localhost:3000/api/v1/Users`)
	fmt.Println()
	fmt.Println("✨ Novo sistema de autenticação:")
	fmt.Println("  - Basic Auth nativo do Fiber v3")
	fmt.Println("  - Validator customizado com acesso ao banco")
	fmt.Println("  - Proteção via middleware nas entidades")
	fmt.Println("  - Suporte a bcrypt para senhas")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Println()
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
