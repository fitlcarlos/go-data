package odata

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// IsRunning retorna se o servidor está em execução
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// IsRunningAsService verifica se o processo está rodando como serviço do sistema
func (s *Server) IsRunningAsService() bool {
	// Esta função pode ser refinada conforme a plataforma
	// Por enquanto, consideramos que se o contexto de serviço está definido, está rodando como serviço
	return s.serviceCtx != nil && s.serviceCancel != nil
}

// GetConfig retorna a configuração do servidor
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// GetRouter retorna o router do servidor
func (s *Server) GetRouter() *fiber.App {
	return s.router
}

// GetHandler retorna o handler HTTP do servidor (para compatibilidade)
func (s *Server) GetHandler() *fiber.App {
	return s.router
}

// GetAddress retorna o endereço do servidor
func (s *Server) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// GetEntities retorna a lista de entidades registradas
func (s *Server) GetEntities() map[string]EntityService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entities := make(map[string]EntityService)
	for name, service := range s.entities {
		entities[name] = service
	}
	return entities
}

// GetEventManager retorna o gerenciador de eventos
func (s *Server) GetEventManager() *EntityEventManager {
	return s.eventManager
}

// GetEntityService retorna o EntityService registrado para o nome especificado
func (s *Server) GetEntityService(name string) EntityService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entities[name]
}

// GetEntityAuth retorna a configuração de autenticação de uma entidade
func (s *Server) GetEntityAuth(name string) (EntityAuthConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	auth, ok := s.entityAuth[name]
	return auth, ok
}
