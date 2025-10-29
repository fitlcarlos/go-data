package main

import (
	"log"

	"github.com/fitlcarlos/go-data/odata"
)

// Produto representa um produto no sistema
type Produto struct {
	ID        int64   `json:"id" db:"id" odata:"key"`
	Nome      string  `json:"nome" db:"nome"`
	Descricao string  `json:"descricao" db:"descricao"`
	Preco     float64 `json:"preco" db:"preco"`
	Categoria string  `json:"categoria" db:"categoria"`
	TenantID  string  `json:"tenant_id" db:"tenant_id"`
}

// Cliente representa um cliente no sistema
type Cliente struct {
	ID       int64  `json:"id" db:"id" odata:"key"`
	Nome     string `json:"nome" db:"nome"`
	Email    string `json:"email" db:"email"`
	Telefone string `json:"telefone" db:"telefone"`
	TenantID string `json:"tenant_id" db:"tenant_id"`
}

// Pedido representa um pedido no sistema
type Pedido struct {
	ID         int64   `json:"id" db:"id" odata:"key"`
	ClienteID  int64   `json:"cliente_id" db:"cliente_id"`
	ProdutoID  int64   `json:"produto_id" db:"produto_id"`
	Quantidade int     `json:"quantidade" db:"quantidade"`
	ValorTotal float64 `json:"valor_total" db:"valor_total"`
	DataPedido string  `json:"data_pedido" db:"data_pedido"`
	TenantID   string  `json:"tenant_id" db:"tenant_id"`
}

func main() {

	// Cria o servidor OData com carregamento automático de configurações multi-tenant
	server := odata.NewServer()

	// Registra as entidades (serão automaticamente multi-tenant se configurado)
	if err := server.RegisterEntity("Produtos", &Produto{}); err != nil {
		log.Fatal("Erro ao registrar entidade Produtos:", err)
	}

	if err := server.RegisterEntity("Clientes", &Cliente{}); err != nil {
		log.Fatal("Erro ao registrar entidade Clientes:", err)
	}

	if err := server.RegisterEntity("Pedidos", &Pedido{}); err != nil {
		log.Fatal("Erro ao registrar entidade Pedidos:", err)
	}

	// Registra eventos globais
	server.OnEntityListGlobal(func(args odata.EventArgs) error {
		if listArgs, ok := args.(*odata.EntityListArgs); ok {
			// Extrai tenant_id do contexto usando odata.GetCurrentTenant
			tenantID := "default"
			if listArgs.Context != nil && listArgs.Context.FiberContext != nil {
				tenantID = odata.GetCurrentTenant(listArgs.Context.FiberContext)
			}
			log.Printf("📋 Lista de entidades acessada: %s (tenant: %s)",
				listArgs.EntityName, tenantID)
		}
		return nil
	})

	server.OnEntityGetGlobal(func(args odata.EventArgs) error {
		if getArgs, ok := args.(*odata.EntityGetArgs); ok {
			// Extrai tenant_id do contexto usando odata.GetCurrentTenant
			tenantID := "default"
			if getArgs.Context != nil && getArgs.Context.FiberContext != nil {
				tenantID = odata.GetCurrentTenant(getArgs.Context.FiberContext)
			}
			log.Printf("🔍 Entidade acessada: %s (tenant: %s)",
				getArgs.EntityName, tenantID)
		}
		return nil
	})

	// Imprime informações sobre o servidor
	printServerInfo(server)

	// Inicia o servidor
	log.Println("🚀 Iniciando servidor multi-tenant...")
	log.Fatal(server.Start())
}

func printServerInfo(server *odata.Server) {
	log.Println("🏢 Informações do Servidor Multi-Tenant:")
	log.Printf("   Endereço: %s", server.GetAddress())

	entities := server.GetEntities()
	log.Printf("   Entidades registradas: %d", len(entities))
	for name := range entities {
		log.Printf("     - %s", name)
	}

	log.Println()
	log.Println("🌐 URLs de Exemplo:")
	log.Println("   # Listar produtos (tenant padrão)")
	log.Println("   GET http://localhost:8080/api/odata/Produtos")
	log.Println()
	log.Println("   # Listar produtos (tenant específico via header)")
	log.Println("   GET http://localhost:8080/api/odata/Produtos")
	log.Println("   Header: X-Tenant-ID: empresa_a")
	log.Println()
	log.Println("   # Obter produto específico")
	log.Println("   GET http://localhost:8080/api/odata/Produtos(1)")
	log.Println("   Header: X-Tenant-ID: empresa_b")
	log.Println()
	log.Println("   # Filtrar produtos por categoria")
	log.Println("   GET http://localhost:8080/api/odata/Produtos?$filter=categoria eq 'Eletrônicos'")
	log.Println("   Header: X-Tenant-ID: empresa_c")
	log.Println()
	log.Println("   # Informações dos tenants")
	log.Println("   GET http://localhost:8080/tenants")
	log.Println()
	log.Println("   # Estatísticas dos tenants")
	log.Println("   GET http://localhost:8080/tenants/stats")
	log.Println()
	log.Println("   # Health check de um tenant específico")
	log.Println("   GET http://localhost:8080/tenants/empresa_a/health")
	log.Println()
	log.Println("📋 Métodos de Identificação de Tenant:")
	log.Println("   1. Header (padrão): X-Tenant-ID")
	log.Println("   2. Subdomain: tenant1.exemplo.com")
	log.Println("   3. Path: /api/tenant1/odata/Produtos")
	log.Println("   4. JWT: claim 'tenant_id' no token")
	log.Println()
}
