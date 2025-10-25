package odata

import (
	"github.com/gofiber/fiber/v3"
)

// Get registra rota GET
// Fiber v3: primeiro handler é o handler final, os seguintes são middlewares
func (s *Server) Get(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Get requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Get(path, handlers[0])
	}
	// Último handler é o handler final, os anteriores são middlewares
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Get(path, finalHandler, middlewares...)
}

// Post registra rota POST
func (s *Server) Post(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Post requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Post(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Post(path, finalHandler, middlewares...)
}

// Put registra rota PUT
func (s *Server) Put(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Put requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Put(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Put(path, finalHandler, middlewares...)
}

// Delete registra rota DELETE
func (s *Server) Delete(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Delete requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Delete(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Delete(path, finalHandler, middlewares...)
}

// Patch registra rota PATCH
func (s *Server) Patch(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Patch requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Patch(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Patch(path, finalHandler, middlewares...)
}

// Options registra rota OPTIONS
func (s *Server) Options(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Options requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Options(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Options(path, finalHandler, middlewares...)
}

// Head registra rota HEAD
func (s *Server) Head(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Head requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Head(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Head(path, finalHandler, middlewares...)
}

// All registra rota para todos os métodos HTTP
func (s *Server) All(path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("All requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.All(path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.All(path, finalHandler, middlewares...)
}

// Group cria grupo de rotas
func (s *Server) Group(prefix string, handlers ...fiber.Handler) fiber.Router {
	return s.router.Group(prefix, handlers...)
}

// Use adiciona middleware global
func (s *Server) Use(args ...interface{}) fiber.Router {
	return s.router.Use(args...)
}

// Add adiciona uma rota com método customizado
func (s *Server) Add(methods []string, path string, handlers ...fiber.Handler) fiber.Router {
	if len(handlers) == 0 {
		s.logger.Fatal("Add requer pelo menos um handler")
	}
	if len(handlers) == 1 {
		return s.router.Add(methods, path, handlers[0])
	}
	finalHandler := handlers[len(handlers)-1]
	middlewares := handlers[:len(handlers)-1]
	return s.router.Add(methods, path, finalHandler, middlewares...)
}
