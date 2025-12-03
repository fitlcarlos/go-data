package odata

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/basicauth"
)

// BasicAuthConfig configurações para Basic Auth
type BasicAuthConfig struct {
	Users map[string]string // username -> password
	Realm string
}

// NewRouterBasicAuth retorna middleware Basic Auth do Fiber v3
// Aceita um validator customizado ou usa config com map de usuários
func (s *Server) NewRouterBasicAuth(validator func(string, string) bool, config ...*BasicAuthConfig) fiber.Handler {
	var cfg basicauth.Config

	// Se validator foi fornecido, usa ele
	if validator != nil {
		cfg = basicauth.Config{
			Authorizer: func(username, password string, c fiber.Ctx) bool {
				return validator(username, password)
			},
			Unauthorized: func(c fiber.Ctx) error {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "Unauthorized",
					"message": "Credenciais inválidas",
				})
			},
		}
	} else if len(config) > 0 && config[0] != nil {
		// Usa map de usuários da config
		users := config[0].Users
		realm := config[0].Realm
		if realm == "" {
			realm = "Restricted"
		}

		cfg = basicauth.Config{
			Users: users,
			Realm: realm,
			Unauthorized: func(c fiber.Ctx) error {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "Unauthorized",
					"message": "Credenciais inválidas",
				})
			},
		}
	} else {
		s.logger.Fatal("BasicAuth requer validator ou config com Users")
	}

	return basicauth.New(cfg)
}

// GetBasicAuthUsername retorna o username do Basic Auth do contexto
func GetBasicAuthUsername(c fiber.Ctx) string {
	if username := c.Locals("username"); username != nil {
		if str, ok := username.(string); ok {
			return str
		}
	}
	return ""
}
