package auth

import (
	"github.com/gofiber/fiber/v3"
)

// AuthMiddleware cria um middleware de autenticação obrigatória usando AuthProvider
func AuthMiddleware(auth AuthProvider) fiber.Handler {
	return func(c fiber.Ctx) error {
		if auth == nil {
			return fiber.NewError(fiber.StatusInternalServerError, "Auth provider não configurado")
		}

		token := auth.ExtractToken(c)
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Token de acesso requerido")
		}

		user, err := auth.ValidateToken(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Token inválido: "+err.Error())
		}

		// Armazenar usuário no contexto
		c.Locals(UserContextKey, user)
		return c.Next()
	}
}

// OptionalAuthMiddleware cria um middleware de autenticação opcional usando AuthProvider
func OptionalAuthMiddleware(auth AuthProvider) fiber.Handler {
	return func(c fiber.Ctx) error {
		if auth == nil {
			return c.Next()
		}

		token := auth.ExtractToken(c)
		if token == "" {
			return c.Next()
		}

		user, err := auth.ValidateToken(token)
		if err != nil {
			// Token inválido, mas não bloqueia a requisição
			return c.Next()
		}

		// Armazenar usuário no contexto
		c.Locals(UserContextKey, user)
		return c.Next()
	}
}

// RequireAuth middleware que requer autenticação
func RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida")
		}
		return c.Next()
	}
}

// RequireRole middleware que requer uma role específica
func RequireRole(role string) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida")
		}

		if !user.HasRole(role) {
			return fiber.NewError(fiber.StatusForbidden, "Acesso negado: role '"+role+"' requerida")
		}

		return c.Next()
	}
}

// RequireAnyRole middleware que requer pelo menos uma das roles
func RequireAnyRole(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida")
		}

		if !user.HasAnyRole(roles...) {
			return fiber.NewError(fiber.StatusForbidden, "Acesso negado: uma das roles requeridas")
		}

		return c.Next()
	}
}

// RequireScope middleware que requer um scope específico
func RequireScope(scope string) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida")
		}

		if !user.HasScope(scope) {
			return fiber.NewError(fiber.StatusForbidden, "Acesso negado: scope '"+scope+"' requerido")
		}

		return c.Next()
	}
}

// RequireAnyScope middleware que requer pelo menos um dos scopes
func RequireAnyScope(scopes ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida")
		}

		if !user.HasAnyScope(scopes...) {
			return fiber.NewError(fiber.StatusForbidden, "Acesso negado: um dos scopes requeridos")
		}

		return c.Next()
	}
}

// RequireAdmin middleware que requer privilégios de administrador
func RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		user := GetCurrentUser(c)
		if user == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Autenticação requerida")
		}

		if !user.IsAdmin() {
			return fiber.NewError(fiber.StatusForbidden, "Acesso negado: privilégios de administrador requeridos")
		}

		return c.Next()
	}
}

// IsAuthenticated verifica se o usuário está autenticado
func IsAuthenticated(c fiber.Ctx) bool {
	return GetCurrentUser(c) != nil
}

// HasRole verifica se o usuário atual possui uma role específica
func HasRole(c fiber.Ctx, role string) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}
	return user.HasRole(role)
}

// HasScope verifica se o usuário atual possui um scope específico
func HasScope(c fiber.Ctx, scope string) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}
	return user.HasScope(scope)
}

// IsAdmin verifica se o usuário atual é administrador
func IsAdmin(c fiber.Ctx) bool {
	user := GetCurrentUser(c)
	if user == nil {
		return false
	}
	return user.IsAdmin()
}
