package odata

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTMiddleware(t *testing.T) {
	// Criar servidor de teste
	app := fiber.New()

	// Criar configuração JWT
	config := &JWTConfig{
		SecretKey:  "test-secret-key",
		Issuer:     "test-issuer",
		ExpiresIn:  24 * time.Hour,
		RefreshIn:  168 * time.Hour,
		Algorithm:  "HS256",
		ContextKey: "user",
	}

	// Criar configuração do servidor
	serverConfig := DefaultServerConfig()
	serverConfig.EnableLogging = false // Desabilitar logs nos testes

	// Criar servidor OData mock
	server := &Server{
		router: app,
		config: serverConfig,
		logger: nil,
	}

	// 🔍 DEBUG: Adicionar um middleware de teste antes do JWT para verificar se está sendo executado
	t.Log("Configurando middleware JWT...")

	// Criar middleware JWT
	jwtMiddleware := server.NewRouterJWTAuth(config)

	// 🔍 DEBUG: Verificar se o middleware foi criado
	if jwtMiddleware == nil {
		t.Fatal("Middleware JWT não foi criado!")
	}
	t.Log("Middleware JWT criado com sucesso")

	// Rota protegida
	// Fiber v3: handler PRIMEIRO, middlewares DEPOIS
	handler := func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	}
	app.Get("/protected", handler, jwtMiddleware)

	// 🔍 DEBUG: Rota de teste simples sem middleware
	app.Get("/public", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "public"})
	})

	// 🔍 DEBUG: Rota com middleware simples que sempre bloqueia
	app.Get("/blocked", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "blocked"})
	})

	// Teste 0: Verificar se rotas públicas funcionam
	t.Run("Rota Pública", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/public", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Esperado status 200, recebeu %d", resp.StatusCode)
		}
	})

	// Teste -1: Verificar se middleware bloqueador funciona
	t.Run("Rota Bloqueada", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/blocked", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Esperado status 403, recebeu %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	// Teste 1: Sem token - deve retornar 401
	t.Run("Sem Token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Esperado status 401, recebeu %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	// Teste 2: Token inválido - deve retornar 401
	t.Run("Token Inválido", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Esperado status 401, recebeu %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	// Teste 3: Token válido - deve retornar 200
	t.Run("Token Válido", func(t *testing.T) {
		// Gerar token válido
		claims := jwt.MapClaims{
			"username": "testuser",
			"exp":      time.Now().Add(1 * time.Hour).Unix(),
			"iat":      time.Now().Unix(),
			"iss":      "test-issuer",
		}

		token, err := GenerateJWT(claims, config)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Esperado status 200, recebeu %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	// Teste 4: Token sem prefixo Bearer - deve retornar 401
	t.Run("Token Sem Bearer", func(t *testing.T) {
		claims := jwt.MapClaims{
			"username": "testuser",
			"exp":      time.Now().Add(1 * time.Hour).Unix(),
		}

		token, _ := GenerateJWT(claims, config)

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", token) // Sem "Bearer "
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 401 {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Esperado status 401, recebeu %d. Body: %s", resp.StatusCode, string(body))
		}
	})
}

func TestJWTGeneration(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key",
		Issuer:    "test-issuer",
		ExpiresIn: 24 * time.Hour,
		Algorithm: "HS256",
	}

	claims := jwt.MapClaims{
		"username": "testuser",
		"email":    "test@example.com",
		"roles":    []string{"user", "admin"},
	}

	// Teste 1: Gerar token
	token, err := GenerateJWT(claims, config)
	if err != nil {
		t.Fatalf("Erro ao gerar token: %v", err)
	}

	if token == "" {
		t.Error("Token gerado está vazio")
	}

	// Teste 2: Validar token gerado
	validatedClaims, err := ValidateJWT(token, config)
	if err != nil {
		t.Fatalf("Erro ao validar token: %v", err)
	}

	if validatedClaims["username"] != "testuser" {
		t.Errorf("Username esperado 'testuser', recebeu '%v'", validatedClaims["username"])
	}

	if validatedClaims["email"] != "test@example.com" {
		t.Errorf("Email esperado 'test@example.com', recebeu '%v'", validatedClaims["email"])
	}
}

func TestJWTRefreshToken(t *testing.T) {
	config := &JWTConfig{
		SecretKey: "test-secret-key",
		Issuer:    "test-issuer",
		ExpiresIn: 24 * time.Hour,
		RefreshIn: 168 * time.Hour,
		Algorithm: "HS256",
	}

	claims := jwt.MapClaims{
		"username": "testuser",
		"email":    "test@example.com",
	}

	// Gerar refresh token
	refreshToken, err := GenerateRefreshToken(claims, config)
	if err != nil {
		t.Fatalf("Erro ao gerar refresh token: %v", err)
	}

	if refreshToken == "" {
		t.Error("Refresh token gerado está vazio")
	}

	// Validar refresh token
	validatedClaims, err := ValidateJWT(refreshToken, config)
	if err != nil {
		t.Fatalf("Erro ao validar refresh token: %v", err)
	}

	if validatedClaims["username"] != "testuser" {
		t.Errorf("Username esperado 'testuser', recebeu '%v'", validatedClaims["username"])
	}
}
