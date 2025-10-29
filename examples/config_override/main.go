package main

import (
	"log"
	"os"

	"github.com/fitlcarlos/go-data/odata"
	"github.com/gofiber/fiber/v3"
)

type Product struct {
	ID    int     `json:"id" db:"id" odata:"key"`
	Name  string  `json:"name" db:"name"`
	Price float64 `json:"price" db:"price"`
}

func main() {
	log.Println("=== Demonstração: Carregamento Automático do .env ===")
	log.Println()

	// ✅ ANTES: Você precisava fazer isso
	// if err := godotenv.Load(); err != nil {
	//     log.Println("Erro ao carregar .env")
	// }

	// ✅ AGORA: odata.NewServer() carrega AUTOMATICAMENTE e injeta no ambiente!
	log.Println("1️⃣  Criando servidor (carrega .env automaticamente)...")
	server := odata.NewServer()

	// ✅ Todas as variáveis do .env estão disponíveis via os.Getenv()
	log.Println("\n2️⃣  Variáveis do .env agora estão disponíveis:")
	log.Printf("   DB_DRIVER: %s", os.Getenv("DB_DRIVER"))
	log.Printf("   DB_HOST: %s", os.Getenv("DB_HOST"))
	log.Printf("   DB_PORT: %s", os.Getenv("DB_PORT"))
	log.Printf("   SERVER_PORT: %s", os.Getenv("SERVER_PORT"))
	log.Printf("   JWT_SECRET_KEY: %s", os.Getenv("JWT_SECRET_KEY"))

	// Suas variáveis personalizadas TAMBÉM estão disponíveis!
	customApiKey := os.Getenv("MY_CUSTOM_API_KEY")
	if customApiKey != "" {
		log.Printf("   MY_CUSTOM_API_KEY: %s", customApiKey)
	}

	// ✅ Sobrescrever configurações via código (prioridade sobre .env)
	log.Println("\n3️⃣  Sobrescrevendo configurações via código...")
	server.SetPort(9000). // Sobrescreve SERVER_PORT do .env
				SetHost("0.0.0.0").               // Sobrescreve SERVER_HOST do .env
				SetRoutePrefix("/api/v2").        // Sobrescreve SERVER_ROUTE_PREFIX do .env
				SetCORS(true).                    // Habilita CORS
				SetAllowedOrigins([]string{"*"}). // Permite todas as origens
				SetEnableLogging(true).           // Habilita logging
				SetRateLimit(100, 20)             // 100 req/min, burst de 20

	log.Println("   ✅ Porta alterada para: 9000")
	log.Println("   ✅ Host alterado para: 0.0.0.0")
	log.Println("   ✅ Route prefix alterado para: /api/v2")
	log.Println("   ✅ Rate limit configurado: 100 req/min")

	// Registrar entidades
	log.Println("\n4️⃣  Registrando entidades...")
	server.RegisterEntity("Products", Product{})

	// Endpoint customizado que usa variável do .env
	server.Get("/test-env", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":     "Todas as variáveis do .env estão disponíveis!",
			"db_driver":   os.Getenv("DB_DRIVER"),
			"db_host":     os.Getenv("DB_HOST"),
			"server_port": os.Getenv("SERVER_PORT"),
			"custom_var":  os.Getenv("MY_CUSTOM_API_KEY"),
			"note":        "Você NÃO precisou usar godotenv.Load()!",
		})
	})

	// Configuração final
	config := server.GetConfig()
	log.Println("\n5️⃣  Configuração final do servidor:")
	log.Printf("   Host: %s", config.Host)
	log.Printf("   Port: %d", config.Port)
	log.Printf("   RoutePrefix: %s", config.RoutePrefix)
	log.Printf("   EnableCORS: %v", config.EnableCORS)
	log.Printf("   EnableLogging: %v", config.EnableLogging)
	if config.RateLimitConfig != nil && config.RateLimitConfig.Enabled {
		log.Printf("   RateLimit: %d req/min", config.RateLimitConfig.RequestsPerMinute)
	}

	log.Println("\n6️⃣  Iniciando servidor...")
	log.Println("   💡 Acesse: http://0.0.0.0:9000/test-env")
	log.Println("   💡 Metadata: http://0.0.0.0:9000/api/v2/$metadata")
	log.Println()

	if err := server.Start(); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
