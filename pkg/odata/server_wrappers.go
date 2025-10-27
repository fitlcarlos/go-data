package odata

import (
	"github.com/gofiber/fiber/v3"
)

// Get registra rota GET com prefixo automático
// O path fornecido será prefixado automaticamente com RoutePrefix
// Exemplo: server.Post("/auth/login", handler) -> /api/v1/auth/login
func (s *Server) Get(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Get requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Get(fullPath, handlers[0])
	}

	// Último handler é o handler final, os anteriores são middlewares
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Get(fullPath, finalHandler, middlewares...)
}

// Post registra rota POST com prefixo automático
// O path fornecido será prefixado automaticamente com RoutePrefix
// Exemplo: server.Post("/auth/login", handler) -> /api/v1/auth/login
func (s *Server) Post(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Post requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Post(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Post(fullPath, finalHandler, middlewares...)
}

// Put registra rota PUT com prefixo automático
// O path fornecido será prefixado automaticamente com RoutePrefix
// Exemplo: server.Put("/auth/update", handler) -> /api/v1/auth/update
func (s *Server) Put(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Put requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Put(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Put(fullPath, finalHandler, middlewares...)
}

// Delete registra rota DELETE com prefixo automático
// O path fornecido será prefixado automaticamente com RoutePrefix
// Exemplo: server.Delete("/auth/revoke", handler) -> /api/v1/auth/revoke
func (s *Server) Delete(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Delete requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Delete(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Delete(fullPath, finalHandler, middlewares...)
}

// Patch registra rota PATCH com prefixo automático
// O path fornecido será prefixado automaticamente com RoutePrefix
// Exemplo: server.Patch("/auth/refresh", handler) -> /api/v1/auth/refresh
func (s *Server) Patch(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Patch requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Patch(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Patch(fullPath, finalHandler, middlewares...)
}

// Options registra rota OPTIONS com prefixo automático
func (s *Server) Options(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Options requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Options(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Options(fullPath, finalHandler, middlewares...)
}

// Head registra rota HEAD com prefixo automático
func (s *Server) Head(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Head requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Head(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Head(fullPath, finalHandler, middlewares...)
}

// All registra rota para todos os métodos HTTP com prefixo automático
func (s *Server) All(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("All requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.All(fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.All(fullPath, finalHandler, middlewares...)
}

// Group cria grupo de rotas
func (s *Server) Group(prefix string, handlers ...fiber.Handler) fiber.Router {
	return s.router.Group(prefix, handlers...)
}

// Use adiciona middleware global
func (s *Server) Use(args ...interface{}) fiber.Router {
	return s.router.Use(args...)
}

// Add adiciona uma rota com método customizado com prefixo automático
func (s *Server) Add(methods []string, path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Add requer pelo menos um handler")
	}

	// Aplicar prefixo automaticamente
	fullPath := s.config.RoutePrefix + path

	if len(handlers) == 1 {
		return s.router.Add(methods, fullPath, handlers[0])
	}

	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Add(methods, fullPath, finalHandler, middlewares...)
}
