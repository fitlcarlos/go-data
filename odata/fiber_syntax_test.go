package odata

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestFiberCorrectSyntax(t *testing.T) {
	executed := []string{}

	middleware1 := func(c fiber.Ctx) error {
		executed = append(executed, "middleware1")
		t.Log("🔹 Middleware 1 executado")
		return c.Next()
	}

	middleware2 := func(c fiber.Ctx) error {
		executed = append(executed, "middleware2")
		t.Log("🔹 Middleware 2 executado")
		return c.Next()
	}

	handler := func(c fiber.Ctx) error {
		executed = append(executed, "handler")
		t.Log("🔹 Handler executado")
		return c.JSON(fiber.Map{"message": "success"})
	}

	// TESTE 1: Sintaxe literal - passar handlers individuais
	t.Run("Literal: middleware1, middleware2, handler", func(t *testing.T) {
		app := fiber.New()
		executed = []string{}

		app.Get("/test1", middleware1, middleware2, handler)

		req := httptest.NewRequest("GET", "/test1", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		t.Logf("✅ Execução: %v", executed)

		if len(executed) != 3 {
			t.Errorf("Esperado 3, recebeu %d: %v", len(executed), executed)
		}

		if executed[0] != "middleware1" || executed[1] != "middleware2" || executed[2] != "handler" {
			t.Errorf("Ordem incorreta: %v", executed)
		}
	})

	// TESTE 2: Usando handlers[0], handlers[1:]...
	t.Run("handlers[0], handlers[1:]...", func(t *testing.T) {
		app := fiber.New()
		executed = []string{}

		handlers := []fiber.Handler{middleware1, middleware2, handler}
		// Converter []fiber.Handler para []any para compatibilidade com Fiber v3
		handlersAny := make([]any, len(handlers))
		for i, h := range handlers {
			handlersAny[i] = h
		}
		// Converter handlersAny[1:] para []any
		remainingHandlers := make([]any, len(handlersAny)-1)
		for i := 1; i < len(handlersAny); i++ {
			remainingHandlers[i-1] = handlersAny[i]
		}
		app.Get("/test2", handlersAny[0], remainingHandlers...)

		req := httptest.NewRequest("GET", "/test2", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		t.Logf("✅ Execução: %v", executed)

		if len(executed) != 3 {
			t.Errorf("Esperado 3, recebeu %d: %v", len(executed), executed)
		}
	})

	// TESTE 3: Verificar se a ordem importa
	t.Run("Ordem Invertida", func(t *testing.T) {
		app := fiber.New()
		executed = []string{}

		// Passar handler PRIMEIRO, middlewares DEPOIS
		app.Get("/test3", handler, middleware1, middleware2)

		req := httptest.NewRequest("GET", "/test3", nil)
		resp, _ := app.Test(req)
		defer resp.Body.Close()

		t.Logf("✅ Execução com ordem invertida: %v", executed)
	})
}
