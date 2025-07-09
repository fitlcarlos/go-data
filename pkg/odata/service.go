package odata

import (
	"context"
	"time"

	"github.com/kardianos/service"
)

// =================================================================================================
// IMPLEMENTAÇÃO DE SERVIÇO (Windows Service, systemd, launchd)
// =================================================================================================

// ServiceWrapper implementa a interface service.Interface para o servidor GoData
type ServiceWrapper struct {
	server *Server
}

// Start é chamado pelo gerenciador de serviços para iniciar o serviço
// Implementa a interface service.Interface
func (sw *ServiceWrapper) Start(svc service.Service) error {
	if sw.server.serviceLogger != nil {
		sw.server.serviceLogger.Info("🚀 Iniciando serviço GoData...")
	}

	// Cria contexto para controle do serviço
	sw.server.serviceCtx, sw.server.serviceCancel = context.WithCancel(context.Background())

	// Inicia o serviço em goroutine separada
	go sw.runAsService()

	return nil
}

func (sw *ServiceWrapper) runAsService() {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			if sw.server.serviceLogger != nil {
				sw.server.serviceLogger.Errorf("Erro crítico no serviço: %v", panicValue)
			}
		}
	}()

	if sw.server.serviceLogger != nil {
		sw.server.serviceLogger.Info("📊 Servidor GoData configurado e iniciado como serviço")
	}

	// Inicia o servidor com contexto de serviço
	if err := sw.server.startWithContext(sw.server.serviceCtx); err != nil {
		if sw.server.serviceLogger != nil {
			sw.server.serviceLogger.Errorf("Erro ao iniciar servidor: %v", err)
		}
	}
}

// Stop é chamado pelo gerenciador de serviços para parar o serviço
// Implementa a interface service.Interface
func (sw *ServiceWrapper) Stop(svc service.Service) error {
	if sw.server.serviceLogger != nil {
		sw.server.serviceLogger.Info("⏹️ Parando serviço GoData...")
	}

	// Cancela o contexto para sinalizar shutdown
	if sw.server.serviceCancel != nil {
		sw.server.serviceCancel()
	}

	// Aguarda um tempo para shutdown graceful
	time.Sleep(2 * time.Second)

	// Para o servidor se estiver rodando
	if err := sw.server.Shutdown(); err != nil {
		if sw.server.serviceLogger != nil {
			sw.server.serviceLogger.Errorf("Erro ao parar servidor: %v", err)
		}
		return err
	}

	if sw.server.serviceLogger != nil {
		sw.server.serviceLogger.Info("✅ Serviço GoData parado com sucesso")
	}

	return nil
}
