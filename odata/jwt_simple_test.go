package odata

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestMiddlewareExecution(t *testing.T) {
	app := fiber.New()

	// Middleware simples que deve bloquear todas as requisições
	simpleMiddleware := func(c fiber.Ctx) error {
		t.Log("❌ Middleware BLOQUEANDO requisição")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "bloqueado pelo middleware"})
	}

	// Rota com middleware
	app.Get("/test", simpleMiddleware, func(c fiber.Ctx) error {
		t.Log("✅ Handler PRINCIPAL executado")
		return c.JSON(fiber.Map{"message": "success"})
	})

	// Fazer requisição
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Logf("Status Code: %d", resp.StatusCode)

	if resp.StatusCode != 403 {
		t.Errorf("Esperado 403 (middleware deve bloquear), recebeu %d", resp.StatusCode)
	}
}

func TestMiddlewareChaining(t *testing.T) {
	app := fiber.New()

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

	// Registrar rota com múltiplos middlewares
	app.Get("/test", middleware1, middleware2, handler)

	// Fazer requisição
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Logf("Ordem de execução: %v", executed)

	if len(executed) != 3 {
		t.Errorf("Esperado 3 handlers executados, recebeu %d: %v", len(executed), executed)
	}

	if executed[0] != "middleware1" || executed[1] != "middleware2" || executed[2] != "handler" {
		t.Errorf("Ordem de execução incorreta: %v", executed)
	}
}
