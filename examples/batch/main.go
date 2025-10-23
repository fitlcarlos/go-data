package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "modernc.org/sqlite"
)

// Product representa um produto
type Product struct {
	ID          int       `json:"id" db:"id" odata:"key,filterable,sortable"`
	Name        string    `json:"name" db:"name" odata:"filterable,sortable,searchable"`
	Description string    `json:"description" db:"description" odata:"searchable"`
	Price       float64   `json:"price" db:"price" odata:"filterable,sortable"`
	Stock       int       `json:"stock" db:"stock" odata:"filterable,sortable"`
	CategoryID  int       `json:"category_id" db:"category_id" odata:"filterable"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" odata:"sortable"`
}

// Category representa uma categoria
type Category struct {
	ID          int       `json:"id" db:"id" odata:"key,filterable,sortable"`
	Name        string    `json:"name" db:"name" odata:"filterable,sortable,searchable"`
	Description string    `json:"description" db:"description" odata:"searchable"`
	Active      bool      `json:"active" db:"active" odata:"filterable"`
	CreatedAt   time.Time `json:"created_at" db:"created_at" odata:"sortable"`
}

// Order representa um pedido
type Order struct {
	ID         int       `json:"id" db:"id" odata:"key,filterable,sortable"`
	ProductID  int       `json:"product_id" db:"product_id" odata:"filterable"`
	Quantity   int       `json:"quantity" db:"quantity" odata:"filterable,sortable"`
	TotalPrice float64   `json:"total_price" db:"total_price" odata:"filterable,sortable"`
	Status     string    `json:"status" db:"status" odata:"filterable"`
	CreatedAt  time.Time `json:"created_at" db:"created_at" odata:"sortable"`
}

var db *sql.DB

func main() {
	// Conectar ao banco de dados SQLite (in-memory para demonstração)
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal("Erro ao conectar ao banco:", err)
	}
	defer db.Close()

	// Criar tabelas
	createTables()

	// Inserir dados de exemplo
	seedData()

	// Configurar servidor OData
	server := odata.NewServer()

	// Registrar entidades
	if err := server.RegisterEntity("Products", Product{}); err != nil {
		log.Fatal("Erro ao registrar entidade Products:", err)
	}

	if err := server.RegisterEntity("Categories", Category{}); err != nil {
		log.Fatal("Erro ao registrar entidade Categories:", err)
	}

	if err := server.RegisterEntity("Orders", Order{}); err != nil {
		log.Fatal("Erro ao registrar entidade Orders:", err)
	}

	// Iniciar servidor
	fmt.Println("\n🚀 Servidor OData com $batch iniciado em http://localhost:3000")
	fmt.Println("\n📋 Endpoints disponíveis:")
	fmt.Println("  GET    /api/v1/Products           - Listar produtos")
	fmt.Println("  GET    /api/v1/Categories         - Listar categorias")
	fmt.Println("  GET    /api/v1/Orders             - Listar pedidos")
	fmt.Println("  POST   /api/v1/$batch            - Processar batch request")
	fmt.Println("\n💡 Exemplos de uso do $batch:")
	fmt.Println("\n1. BATCH COM MÚLTIPLAS LEITURAS:")
	fmt.Println(`curl -X POST http://localhost:3000/api/v1/$batch \
  -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
  --data-binary @- << 'EOF'
--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products?$top=5 HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Categories HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Orders?$filter=status eq 'pending' HTTP/1.1
Host: localhost:3000


--batch_boundary--
EOF`)

	fmt.Println("\n2. BATCH COM CHANGESET (TRANSACIONAL):")
	fmt.Println(`curl -X POST http://localhost:3000/api/v1/$batch \
  -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
  --data-binary @- << 'EOF'
--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Products HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "name": "Novo Produto",
  "description": "Criado via batch",
  "price": 99.90,
  "stock": 10,
  "category_id": 1
}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /api/v1/Orders HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "product_id": 1,
  "quantity": 5,
  "total_price": 499.50,
  "status": "pending"
}

--changeset_boundary--

--batch_boundary--
EOF`)

	fmt.Println("\n3. BATCH MISTO (LEITURA + CHANGESET):")
	fmt.Println(`curl -X POST http://localhost:3000/api/v1/$batch \
  -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
  --data-binary @- << 'EOF'
--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products?$top=3 HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Categories HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "name": "Nova Categoria",
  "description": "Categoria via batch",
  "active": true
}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /api/v1/Products(1) HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "price": 1299.99
}

--changeset_boundary--

--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Orders HTTP/1.1
Host: localhost:3000


--batch_boundary--
EOF`)

	fmt.Println("\n📚 Documentação:")
	fmt.Println("  - Batch requests permitem executar múltiplas operações em uma única requisição HTTP")
	fmt.Println("  - Operações de leitura (GET) são executadas independentemente")
	fmt.Println("  - Changesets (operações de escrita) são executados de forma transacional")
	fmt.Println("  - Se uma operação em um changeset falhar, todas as operações do changeset são revertidas")
	fmt.Println("  - Content-ID permite referenciar operações dentro do batch")
	fmt.Println("\n⚡ Benefícios do $batch:")
	fmt.Println("  - Reduz latência ao combinar múltiplas requisições")
	fmt.Println("  - Suporta transações (changesets)")
	fmt.Println("  - Reduz overhead de conexões HTTP")
	fmt.Println("  - Melhora performance em operações bulk")
	fmt.Println()

	if err := server.Start(); err != nil {
		log.Fatal("Erro ao iniciar servidor:", err)
	}
}

func createTables() {
	// Tabela de categorias
	categoriesTable := `
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(categoriesTable); err != nil {
		log.Printf("Erro ao criar tabela categories: %v", err)
	}

	// Tabela de produtos
	productsTable := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		price REAL NOT NULL,
		stock INTEGER DEFAULT 0,
		category_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_id) REFERENCES categories(id)
	);
	`

	if _, err := db.Exec(productsTable); err != nil {
		log.Printf("Erro ao criar tabela products: %v", err)
	}

	// Tabela de pedidos
	ordersTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL,
		total_price REAL NOT NULL,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (product_id) REFERENCES products(id)
	);
	`

	if _, err := db.Exec(ordersTable); err != nil {
		log.Printf("Erro ao criar tabela orders: %v", err)
	}

	log.Println("✅ Tabelas criadas com sucesso")
}

func seedData() {
	// Inserir categorias
	categories := []struct {
		name        string
		description string
	}{
		{"Eletrônicos", "Produtos eletrônicos e gadgets"},
		{"Livros", "Livros e publicações"},
		{"Roupas", "Vestuário e acessórios"},
	}

	for _, cat := range categories {
		_, err := db.Exec("INSERT INTO categories (name, description) VALUES (?, ?)",
			cat.name, cat.description)
		if err != nil {
			log.Printf("Erro ao inserir categoria: %v", err)
		}
	}

	// Inserir produtos
	products := []struct {
		name        string
		description string
		price       float64
		stock       int
		categoryID  int
	}{
		{"Notebook Dell", "Notebook Dell Inspiron 15", 3500.00, 10, 1},
		{"Mouse Logitech", "Mouse Logitech MX Master", 450.00, 50, 1},
		{"Teclado Mecânico", "Teclado Mecânico RGB", 350.00, 30, 1},
		{"Clean Code", "Livro Clean Code - Robert Martin", 89.90, 20, 2},
		{"Camiseta Go", "Camiseta Golang Developer", 59.90, 100, 3},
	}

	for _, prod := range products {
		_, err := db.Exec("INSERT INTO products (name, description, price, stock, category_id) VALUES (?, ?, ?, ?, ?)",
			prod.name, prod.description, prod.price, prod.stock, prod.categoryID)
		if err != nil {
			log.Printf("Erro ao inserir produto: %v", err)
		}
	}

	// Inserir pedidos
	orders := []struct {
		productID  int
		quantity   int
		totalPrice float64
		status     string
	}{
		{1, 2, 7000.00, "pending"},
		{2, 5, 2250.00, "completed"},
		{3, 1, 350.00, "pending"},
	}

	for _, order := range orders {
		_, err := db.Exec("INSERT INTO orders (product_id, quantity, total_price, status) VALUES (?, ?, ?, ?)",
			order.productID, order.quantity, order.totalPrice, order.status)
		if err != nil {
			log.Printf("Erro ao inserir pedido: %v", err)
		}
	}

	log.Println("✅ Dados de exemplo inseridos com sucesso")
}
