package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "github.com/fitlcarlos/go-data/pkg/providers" // Importa providers para registrar factories
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// User representa uma entidade de usuário
type User struct {
	ID       int64     `json:"id" odata:"key"`
	Name     string    `json:"name" odata:"required"`
	Email    string    `json:"email" odata:"required"`
	IsActive bool      `json:"is_active"`
	Created  time.Time `json:"created"`
	Updated  time.Time `json:"updated"`
}

// Product representa uma entidade de produto
type Product struct {
	ID          int64     `json:"id" odata:"key"`
	Name        string    `json:"name" odata:"required"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	CreatedBy   int64     `json:"created_by"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

func main() {
	// Cria o servidor (carrega automaticamente configurações do .env se disponível)
	server := odata.NewServer()

	// Registra as entidades
	if err := server.RegisterEntity("Users", User{}); err != nil {
		log.Fatal(err)
	}
	if err := server.RegisterEntity("Products", Product{}); err != nil {
		log.Fatal(err)
	}

	// Configura eventos específicos para usuários
	setupUserEvents(server)

	// Configura eventos específicos para produtos
	setupProductEvents(server)

	// Configura eventos globais
	setupGlobalEvents(server)

	// Inicia o servidor
	log.Println("🚀 Servidor iniciado com eventos configurados")
	log.Fatal(server.Start())
}

// setupUserEvents configura eventos específicos para a entidade Users
func setupUserEvents(server *odata.Server) {
	// Evento OnEntityInserting - Validação antes da inserção
	server.OnEntityInserting("Users", func(args odata.EventArgs) error {
		insertArgs := args.(*odata.EntityInsertingArgs)

		log.Printf("🔍 [Users] Inserindo usuário: %+v", insertArgs.Data)

		// Validação customizada
		if name, ok := insertArgs.Data["name"].(string); ok && len(name) < 2 {
			args.Cancel("Nome deve ter pelo menos 2 caracteres")
			return nil
		}

		if email, ok := insertArgs.Data["email"].(string); ok && !isValidEmail(email) {
			args.Cancel("Email inválido")
			return nil
		}

		// Adiciona timestamp de criação
		insertArgs.Data["created"] = time.Now()
		insertArgs.Data["updated"] = time.Now()
		insertArgs.Data["is_active"] = true

		return nil
	})

	// Evento OnEntityInserted - Ação após inserção
	server.OnEntityInserted("Users", func(args odata.EventArgs) error {
		insertedArgs := args.(*odata.EntityInsertedArgs)

		log.Printf("✅ [Users] Usuário inserido com sucesso: %+v", insertedArgs.CreatedEntity)

		// Aqui você poderia enviar um email de boas-vindas, criar auditoria, etc.
		// sendWelcomeEmail(insertedArgs.CreatedEntity)

		return nil
	})

	// Evento OnEntityModifying - Validação antes da atualização
	server.OnEntityModifying("Users", func(args odata.EventArgs) error {
		modifyArgs := args.(*odata.EntityModifyingArgs)

		log.Printf("🔍 [Users] Modificando usuário: %+v", modifyArgs.Data)

		// Impede alteração do email se o usuário não for admin
		if _, emailChanged := modifyArgs.Data["email"]; emailChanged {
			// Aqui você poderia verificar se o usuário atual é admin
			if !isCurrentUserAdmin(modifyArgs.GetContext()) {
				args.Cancel("Apenas administradores podem alterar email")
				return nil
			}
		}

		// Atualiza timestamp
		modifyArgs.Data["updated"] = time.Now()

		return nil
	})

	// Evento OnEntityDeleting - Validação antes da exclusão
	server.OnEntityDeleting("Users", func(args odata.EventArgs) error {
		deleteArgs := args.(*odata.EntityDeletingArgs)

		log.Printf("🗑️ [Users] Deletando usuário: %+v", deleteArgs.Keys)

		// Impede exclusão se o usuário tem produtos
		if hasUserProducts(deleteArgs.Keys) {
			args.Cancel("Não é possível excluir usuário com produtos associados")
			return nil
		}

		return nil
	})

	// Evento OnEntityGet - Filtro após recuperação
	server.OnEntityGet("Users", func(args odata.EventArgs) error {
		getArgs := args.(*odata.EntityGetArgs)

		log.Printf("👀 [Users] Recuperando usuário: %+v", getArgs.Keys)

		// Aqui você poderia filtrar dados sensíveis baseado nas permissões
		if entity, ok := getArgs.GetEntity().(map[string]interface{}); ok {
			// Remove dados sensíveis se não for admin
			if !isCurrentUserAdmin(getArgs.GetContext()) {
				delete(entity, "email")
			}
		}

		return nil
	})
}

// setupProductEvents configura eventos específicos para a entidade Products
func setupProductEvents(server *odata.Server) {
	// Evento OnEntityInserting - Validação de produtos
	server.OnEntityInserting("Products", func(args odata.EventArgs) error {
		insertArgs := args.(*odata.EntityInsertingArgs)

		log.Printf("🔍 [Products] Inserindo produto: %+v", insertArgs.Data)

		// Validação de preço
		if price, ok := insertArgs.Data["price"].(float64); ok && price < 0 {
			args.Cancel("Preço não pode ser negativo")
			return nil
		}

		// Adiciona usuário criador
		if userID := getCurrentUserID(insertArgs.GetContext()); userID != "" {
			insertArgs.Data["created_by"] = userID
		}

		// Adiciona timestamps
		insertArgs.Data["created"] = time.Now()
		insertArgs.Data["updated"] = time.Now()

		return nil
	})

	// Evento OnEntityList - Filtro de listagem
	server.OnEntityList("Products", func(args odata.EventArgs) error {
		listArgs := args.(*odata.EntityListArgs)

		log.Printf("📋 [Products] Listando produtos, total: %d", listArgs.TotalCount)

		// Aqui você poderia adicionar filtros customizados
		// Por exemplo, mostrar apenas produtos ativos

		return nil
	})
}

// setupGlobalEvents configura eventos globais para todas as entidades
func setupGlobalEvents(server *odata.Server) {
	// Evento global OnEntityInserting - Auditoria
	server.OnEntityInsertingGlobal(func(args odata.EventArgs) error {
		log.Printf("🔍 [GLOBAL] Inserindo entidade: %s", args.GetEntityName())

		// Aqui você poderia adicionar auditoria global
		// auditLog.Log("INSERT", args.GetEntityName(), args.GetContext().UserID)

		return nil
	})

	// Evento global OnEntityModifying - Auditoria
	server.OnEntityModifyingGlobal(func(args odata.EventArgs) error {
		log.Printf("🔍 [GLOBAL] Modificando entidade: %s", args.GetEntityName())

		// Auditoria de modificação
		// auditLog.Log("UPDATE", args.GetEntityName(), args.GetContext().UserID)

		return nil
	})

	// Evento global OnEntityDeleting - Auditoria
	server.OnEntityDeletingGlobal(func(args odata.EventArgs) error {
		log.Printf("🔍 [GLOBAL] Deletando entidade: %s", args.GetEntityName())

		// Auditoria de exclusão
		// auditLog.Log("DELETE", args.GetEntityName(), args.GetContext().UserID)

		return nil
	})

	// Evento global OnEntityError - Tratamento de erros
	server.OnEntityErrorGlobal(func(args odata.EventArgs) error {
		errorArgs := args.(*odata.EntityErrorArgs)

		log.Printf("❌ [GLOBAL] Erro na entidade %s: %v", args.GetEntityName(), errorArgs.Error)

		// Aqui você poderia enviar notificações, logs detalhados, etc.
		// errorNotification.Send(errorArgs.Error, errorArgs.Operation)

		return nil
	})
}

// Funções auxiliares para demonstração
func isValidEmail(email string) bool {
	// Implementação simples de validação de email
	return len(email) > 0 && fmt.Sprintf("%s", email) != ""
}

func isCurrentUserAdmin(ctx *odata.EventContext) bool {
	// Aqui você verificaria se o usuário atual é admin
	// Por exemplo, verificando roles ou permissions
	for _, role := range ctx.UserRoles {
		if role == "admin" {
			return true
		}
	}
	return false
}

func hasUserProducts(keys map[string]interface{}) bool {
	// Aqui você verificaria se o usuário tem produtos
	// Esta seria uma consulta ao banco de dados
	return false
}

func getCurrentUserID(ctx *odata.EventContext) string {
	// Retorna o ID do usuário atual
	return ctx.UserID
}

// Exemplo de uso com Fiber customizado
func createFiberApp() *fiber.App {
	app := fiber.New()

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	}))

	return app
}
