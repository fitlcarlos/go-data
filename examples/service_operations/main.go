package main

import (
	"log"
	"strings"

	"github.com/fitlcarlos/go-data/odata"
)

// Product representa um produto
type Product struct {
	ID         int64   `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence"`
	Name       string  `json:"name" column:"name" prop:"[required]; length:255"`
	Price      float64 `json:"price" column:"price" prop:"[required]; precision:10; scale:2"`
	Category   string  `json:"category" column:"category" prop:"length:100"`
	SalesCount int     `json:"sales_count" column:"sales_count" prop:"default:0"`
	IsActive   bool    `json:"is_active" column:"is_active" prop:"default:true"`
}

// User representa um usuário
type User struct {
	ID       int64  `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence"`
	Email    string `json:"email" column:"email" prop:"[required, Unique]; length:255"`
	Password string `json:"password" column:"password" prop:"[required]; length:255"`
	Name     string `json:"name" column:"name" prop:"[required]; length:255"`
	IsActive bool   `json:"is_active" column:"is_active" prop:"default:true"`
}

func main() {
	log.Println("=== Go-Data Service Operations Example ===")

	// Criar servidor (carrega configurações do .env automaticamente)
	server := odata.NewServer()

	// Registrar entidades
	server.RegisterEntity("Products", Product{})
	server.RegisterEntity("Users", User{})

	// Registrar service operations
	registerServiceOperations(server)

	// Iniciar servidor
	log.Println("🚀 Servidor iniciado com Service Operations!")
	log.Println("📋 Endpoints disponíveis:")
	log.Println("   - GET  /api/odata/Service/GetTopSellingProducts")
	log.Println("   - POST /api/odata/Service/CalculateTotalPrice")
	log.Println("   - GET  /api/odata/Service/GetProductStats")
	log.Println("   - GET  /api/odata/Service/GenerateReport")
	log.Println("   - GET  /api/odata/Service/GetUserProfile")
	log.Println()
	log.Println("🆕 Novos Endpoints - Acesso ao Contexto:")
	log.Println("   - GET  /api/odata/Service/GetContextInfo (Pool, Connection, Provider)")
	log.Println("   - POST /api/odata/Service/BatchProcess (Múltiplos ObjectManagers)")
	log.Println()
	log.Println("🔐 Exemplos com autenticação:")
	log.Println("   - GET  /api/odata/Service/AdminData (requer role 'admin')")
	log.Println("   - GET  /api/odata/Service/ManagerData (requer role 'manager')")
	log.Println()

	if err := server.Start(); err != nil {
		log.Fatalf("❌ Erro ao iniciar servidor: %v", err)
	}
}

// registerServiceOperations registra todos os service operations
func registerServiceOperations(server *odata.Server) {
	log.Println("📝 Registrando Service Operations...")

	// Service sem autenticação
	server.Service("GET", "/Service/GetTopSellingProducts", GetTopSellingProducts)

	// Service com autenticação obrigatória
	server.ServiceWithAuth("POST", "/Service/CalculateTotalPrice", CalculateTotalPrice, true)

	// Service com roles específicas
	server.ServiceWithRoles("GET", "/Service/AdminData", GetAdminData, "admin")
	server.ServiceWithRoles("GET", "/Service/ManagerData", GetManagerData, "manager", "admin")

	// Service com parâmetros de rota
	server.Service("GET", "/Service/GetProductStats", GetProductStats)

	// Service que gera relatório
	server.ServiceWithAuth("GET", "/Service/GenerateReport", GenerateReport, true)

	// Service que obtém perfil do usuário
	server.ServiceWithAuth("GET", "/Service/GetUserProfile", GetUserProfile, true)

	// Exemplo usando ServiceGroup
	products := server.ServiceGroup("Products")
	products.ServiceWithAuth("GET", "GetTopSelling", GetTopSellingProducts, true)
	products.ServiceWithRoles("GET", "AdminStats", GetAdminData, "admin")

	// Novos services demonstrando acesso a Pool, Connection, Provider
	server.Service("GET", "/Service/GetContextInfo", GetContextInfo)
	server.Service("POST", "/Service/BatchProcess", BatchProcess)

	log.Println("✅ Service Operations registrados com sucesso!")
}

// GetTopSellingProducts service operation sem autenticação
func GetTopSellingProducts(ctx *odata.ServiceContext) error {
	// Usar ObjectManager diretamente (equivale ao TXDataOperationContext.Current.GetManager())
	manager := ctx.GetManager()

	// Query usando OData
	products, err := manager.Query("Products").
		Where("sales_count gt 100").
		OrderBy("sales_count desc").
		Top(10).
		List()

	if err != nil {
		return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
	}

	// Retornar resposta com informações do contexto
	return ctx.JSON(map[string]interface{}{
		"top_selling":   products,
		"count":         len(products),
		"tenant":        ctx.GetTenantID(),
		"authenticated": ctx.IsAuthenticated(),
	})
}

// CalculateTotalPrice service operation com autenticação
func CalculateTotalPrice(ctx *odata.ServiceContext) error {
	// Verificar se está autenticado (garantido pelo ServiceWithAuth)
	if !ctx.IsAuthenticated() {
		return ctx.Status(401).JSON(map[string]string{"error": "Authentication required"})
	}

	// Obter parâmetros da query
	productIDs := ctx.Query("product_ids")
	if productIDs == "" {
		return ctx.Status(400).JSON(map[string]string{"error": "product_ids parameter required"})
	}

	// Processar IDs
	ids := strings.Split(productIDs, ",")
	manager := ctx.GetManager()

	total := 0.0
	processedCount := 0

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		// Buscar produto usando ObjectManager
		product, err := manager.Find("Products", id)
		if err != nil {
			continue // Ignorar produtos não encontrados
		}

		if productMap, ok := product.(map[string]interface{}); ok {
			if price, exists := productMap["price"]; exists {
				if priceFloat, ok := price.(float64); ok {
					total += priceFloat
					processedCount++
				}
			}
		}
	}

	// Retornar resultado
	return ctx.JSON(map[string]interface{}{
		"total_price":   total,
		"product_count": processedCount,
		"user":          ctx.GetUser().Username,
		"tenant":        ctx.GetTenantID(),
	})
}

// GetAdminData service operation que requer role admin
func GetAdminData(ctx *odata.ServiceContext) error {
	// Verificar se é admin (garantido pelo ServiceWithRoles)
	if !ctx.IsAdmin() {
		return ctx.Status(403).JSON(map[string]string{"error": "Admin access required"})
	}

	manager := ctx.GetManager()

	// Query administrativa
	stats, err := manager.Query("Products").
		Select("category, count(*) as total, sum(price) as total_value").
		GroupBy("category").
		List()

	if err != nil {
		return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
	}

	return ctx.JSON(map[string]interface{}{
		"admin_stats": stats,
		"admin_user":  ctx.GetUser().Username,
		"tenant":      ctx.GetTenantID(),
	})
}

// GetManagerData service operation que requer role manager ou admin
func GetManagerData(ctx *odata.ServiceContext) error {
	// Verificar roles (garantido pelo ServiceWithRoles)
	if !ctx.HasAnyRole("manager", "admin") {
		return ctx.Status(403).JSON(map[string]string{"error": "Manager or Admin access required"})
	}

	manager := ctx.GetManager()

	// Query de manager
	products, err := manager.Query("Products").
		Where("is_active eq true").
		OrderBy("sales_count desc").
		List()

	if err != nil {
		return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
	}

	return ctx.JSON(map[string]interface{}{
		"active_products": products,
		"manager":         ctx.GetUser().Username,
		"tenant":          ctx.GetTenantID(),
	})
}

// GetProductStats service operation com parâmetros
func GetProductStats(ctx *odata.ServiceContext) error {
	manager := ctx.GetManager()

	// Obter parâmetros
	category := ctx.Query("category")
	minPrice := ctx.Query("min_price")
	maxPrice := ctx.Query("max_price")

	// Construir query baseada nos parâmetros
	query := manager.Query("Products")

	if category != "" {
		query = query.Where("category eq '" + category + "'")
	}

	if minPrice != "" {
		query = query.Where("price ge " + minPrice)
	}

	if maxPrice != "" {
		query = query.Where("price le " + maxPrice)
	}

	products, err := query.List()
	if err != nil {
		return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
	}

	return ctx.JSON(map[string]interface{}{
		"products": products,
		"filters": map[string]string{
			"category":  category,
			"min_price": minPrice,
			"max_price": maxPrice,
		},
		"tenant": ctx.GetTenantID(),
	})
}

// GenerateReport service operation que gera relatório
func GenerateReport(ctx *odata.ServiceContext) error {
	// Verificar autenticação
	if !ctx.IsAuthenticated() {
		return ctx.Status(401).JSON(map[string]string{"error": "Authentication required"})
	}

	// Obter formato do relatório
	format := ctx.Query("format")
	if format == "" {
		format = "json"
	}

	manager := ctx.GetManager()

	// Query para relatório
	reportData, err := manager.Query("Products").
		Where("is_active eq true").
		OrderBy("sales_count desc").
		Select("name, price, sales_count, category").
		List()

	if err != nil {
		return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
	}

	// Configurar headers baseado no formato
	switch format {
	case "pdf":
		ctx.SetHeader("Content-Type", "application/pdf")
		ctx.SetHeader("Content-Disposition", "attachment; filename=products_report.pdf")
	case "excel":
		ctx.SetHeader("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		ctx.SetHeader("Content-Disposition", "attachment; filename=products_report.xlsx")
	case "csv":
		ctx.SetHeader("Content-Type", "text/csv")
		ctx.SetHeader("Content-Disposition", "attachment; filename=products_report.csv")
	default:
		ctx.SetHeader("Content-Type", "application/json")
	}

	return ctx.JSON(map[string]interface{}{
		"report_data":  reportData,
		"format":       format,
		"generated_by": ctx.GetUser().Username,
		"tenant":       ctx.GetTenantID(),
		"generated_at": "2024-01-01T00:00:00Z",
	})
}

// GetUserProfile service operation que obtém perfil do usuário
func GetUserProfile(ctx *odata.ServiceContext) error {
	// Verificar autenticação
	if !ctx.IsAuthenticated() {
		return ctx.Status(401).JSON(map[string]string{"error": "Authentication required"})
	}

	user := ctx.GetUser()

	// Retornar perfil do usuário
	return ctx.JSON(map[string]interface{}{
		"user": map[string]interface{}{
			"username": user.Username,
			"roles":    user.Roles,
			"scopes":   user.Scopes,
			"admin":    user.Admin,
			"custom":   user.Custom,
		},
		"tenant":        ctx.GetTenantID(),
		"authenticated": true,
	})
}

// GetContextInfo demonstra acesso aos recursos do contexto
func GetContextInfo(ctx *odata.ServiceContext) error {
	info := map[string]interface{}{
		"tenant":         ctx.GetTenantID(),
		"authenticated":  ctx.IsAuthenticated(),
		"has_pool":       ctx.GetPool() != nil,
		"has_provider":   ctx.GetProvider() != nil,
		"has_connection": ctx.GetConnection() != nil,
	}

	// Usar conexão SQL direta
	conn := ctx.GetConnection()
	if conn != nil {
		var count int
		err := conn.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
		if err == nil {
			info["total_products"] = count
		}
	}

	// Pool info (se multi-tenant habilitado)
	if pool := ctx.GetPool(); pool != nil {
		info["pool_active"] = true
	}

	return ctx.JSON(info)
}

// BatchProcess demonstra criação de múltiplos ObjectManagers
func BatchProcess(ctx *odata.ServiceContext) error {
	// Manager principal
	mainManager := ctx.GetManager()

	// Criar managers isolados
	manager1 := ctx.CreateObjectManager()
	manager2 := ctx.CreateObjectManager()

	return ctx.JSON(map[string]interface{}{
		"message":                     "Batch processed with isolated managers",
		"managers_created":            3,
		"main_manager_available":      mainManager != nil,
		"isolated_manager1_available": manager1 != nil,
		"isolated_manager2_available": manager2 != nil,
	})
}
