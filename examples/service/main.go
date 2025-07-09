package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fitlcarlos/go-data/pkg/odata"
)

// User representa uma entidade de usuário
type User struct {
	ID    int64  `json:"id" odata:"key"`
	Name  string `json:"name" odata:"required"`
	Email string `json:"email" odata:"required"`
}

// Product representa uma entidade de produto
type Product struct {
	ID    int64   `json:"id" odata:"key"`
	Name  string  `json:"name" odata:"required"`
	Price float64 `json:"price"`
}

func main() {
	// Parse dos argumentos de linha de comando

	var (
		command = flag.String("cmd", "server", "Comando: install, uninstall, start, stop, restart, status")
		help    = flag.Bool("help", false, "Mostra ajuda")
	)
	flag.Parse()

	if *help {
		printUsage()
		return
	}

	// Criar servidor GoData (carrega automaticamente configurações do .env)
	server := odata.NewServer()

	// Registrar entidades
	if err := server.RegisterEntity("Users", User{}); err != nil {
		log.Fatalf("Erro ao registrar entidade Users: %v", err)
	}

	if err := server.RegisterEntity("Products", Product{}); err != nil {
		log.Fatalf("Erro ao registrar entidade Products: %v", err)
	}

	// Executar comando
	switch *command {
	case "install":
		// Instalar como serviço
		fmt.Println("📦 Instalando GoData como serviço...")
		if err := server.Install(); err != nil {
			log.Fatalf("Erro ao instalar serviço: %v", err)
		}
		fmt.Println("✅ Serviço instalado com sucesso!")
		fmt.Printf("   Para iniciar: %s -cmd start\n", os.Args[0])
		fmt.Printf("   Para verificar status: %s -cmd status\n", os.Args[0])

	case "uninstall":
		// Desinstalar serviço
		fmt.Println("🗑️  Removendo serviço GoData...")
		if err := server.Uninstall(); err != nil {
			log.Fatalf("Erro ao desinstalar serviço: %v", err)
		}
		fmt.Println("✅ Serviço removido com sucesso!")

	case "start":
		// Iniciar serviço
		fmt.Println("▶️  Iniciando serviço GoData...")
		if err := server.Start(); err != nil {
			log.Fatalf("Erro ao iniciar serviço: %v", err)
		}

	case "stop":
		// Parar serviço
		fmt.Println("⏹️  Parando serviço GoData...")
		if err := server.Stop(); err != nil {
			log.Fatalf("Erro ao parar serviço: %v", err)
		}

	case "restart":
		// Reiniciar serviço
		fmt.Println("🔄 Reiniciando serviço GoData...")
		if err := server.Restart(); err != nil {
			log.Fatalf("Erro ao reiniciar serviço: %v", err)
		}

	case "status":
		// Verificar status do serviço
		fmt.Println("📊 Verificando status do serviço...")
		if _, err := server.Status(); err != nil {
			log.Fatalf("Erro ao verificar status: %v", err)
		}

	default:
		fmt.Printf("❌ Comando desconhecido: %s\n\n", *command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`GoData Service Example - Exemplo de uso do GoData como serviço

Uso: %s -cmd [comando]

Comandos:
  install    Instala o GoData como serviço no sistema
  uninstall  Remove o serviço do sistema
  start      Inicia o serviço
  stop       Para o serviço
  restart    Reinicia o serviço
  status     Mostra o status do serviço

Opções:
  -help      Mostra esta ajuda

Exemplos:

  # Instalar como serviço
  %s -cmd install

  # Iniciar serviço
  %s -cmd start

  # Verificar status
  %s -cmd status

  # Parar serviço
  %s -cmd stop

  # Remover serviço
  %s -cmd uninstall

Configuração:
  O exemplo usa configuração automática via arquivo .env se disponível.
  
  Exemplo de .env:
    DB_TYPE=postgresql
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=postgres
    DB_PASSWORD=password
    DB_NAME=godata_db
    SERVER_HOST=0.0.0.0
    SERVER_PORT=8080

Compatibilidade:
  - Windows: Executa como Windows Service
  - Linux: Executa como systemd service
  - macOS: Executa como launchd service

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
