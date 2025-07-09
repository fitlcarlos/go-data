package main

import (
	"log"
	"os"
	"time"

	"github.com/fitlcarlos/go-data/pkg/odata"
	_ "github.com/fitlcarlos/go-data/pkg/providers" // Importa providers para registrar factories
)

// Exemplo de uso avançado do servidor HTTP embutido
// Para executar: go run advanced_server.go

// Entidades de exemplo
type User struct {
	TableName string    `table:"users;schema=public"`
	ID        int64     `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_user_id"`
	Name      string    `json:"name" column:"name" prop:"[required]; length:100"`
	Email     string    `json:"email" column:"email" prop:"[required, Unique]; length:255"`
	Active    bool      `json:"active" column:"active" prop:"[required]; default"`
	CreatedAt time.Time `json:"created_at" column:"created_at" prop:"[required, NoUpdate]; default"`

	// Relacionamentos
	Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
}

type Order struct {
	TableName string    `table:"orders;schema=public"`
	ID        int64     `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_order_id"`
	UserID    int64     `json:"user_id" column:"user_id" prop:"[required]"`
	Total     float64   `json:"total" column:"total" prop:"[required]; precision:10; scale:2"`
	OrderDate time.Time `json:"order_date" column:"order_date" prop:"[required]"`

	// Relacionamentos
	User *User `json:"User" association:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Refresh]"`
}

func main() {
	log.Println("=== Go-Data OData Server - Configurações Avançadas ===")
	log.Println()

	// Cria o servidor (carrega automaticamente configurações do .env se disponível)
	server := odata.NewServer()

	// Registrar entidades
	if err := registerEntities(server); err != nil {
		log.Fatal("Erro ao registrar entidades:", err)
	}

	// Iniciar servidor
	startServer(server)
}

// registerEntities registra as entidades no servidor
func registerEntities(server *odata.Server) error {
	entities := map[string]interface{}{
		"Users":  User{},
		"Orders": Order{},
	}

	log.Println("📝 Registrando entidades...")
	return server.AutoRegisterEntities(entities)
}

// startServer inicia o servidor com monitoramento
func startServer(server *odata.Server) {
	log.Println()
	log.Println("🚀 Iniciando servidor OData...")

	// Monitora o status do servidor
	go monitorServer(server)

	// Inicia o servidor (bloqueante)
	if err := server.Start(); err != nil {
		log.Fatalf("❌ Erro ao iniciar servidor: %v", err)
	}
}

// monitorServer monitora o status do servidor periodicamente
func monitorServer(server *odata.Server) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if server.IsRunning() {
			log.Printf("📊 Servidor ativo em %s | Entidades: %d",
				server.GetAddress(),
				len(server.GetEntities()))
		}
	}
}

// isDevelopment verifica se está em ambiente de desenvolvimento
func isDevelopment() bool {
	env := os.Getenv("GO_ENV")
	return env == "" || env == "development" || env == "dev"
}

// Função para criar certificados SSL auto-assinados (para desenvolvimento)
func createSelfSignedCertificate() error {
	// Esta é uma implementação simplificada
	// Em produção, use certificados de uma CA confiável
	log.Println("⚠️  Para HTTPS, você precisa de certificados SSL válidos")
	log.Println("   Para desenvolvimento, você pode criar certificados auto-assinados:")
	log.Println("   openssl req -x509 -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -nodes")
	return nil
}

/*
Exemplo de configurações de ambiente para produção:

Environment Variables:
- GO_ENV=production
- DB_HOST=localhost
- DB_PORT=3306
- DB_USER=odata_user
- DB_PASS=secure_password
- DB_NAME=odata_db
- TLS_CERT_FILE=/etc/ssl/certs/server.crt
- TLS_KEY_FILE=/etc/ssl/private/server.key
- ALLOWED_ORIGINS=https://app.mycompany.com,https://admin.mycompany.com
- LOG_LEVEL=INFO
- LOG_FILE=/var/log/odata_server.log

Docker Compose Example:
version: '3.8'
services:
  odata-server:
    build: .
    ports:
      - "8443:8443"
    environment:
      - GO_ENV=production
      - DB_HOST=mysql
      - DB_USER=odata_user
      - DB_PASS=secure_password
    volumes:
      - ./certs:/etc/ssl/certs
      - ./logs:/var/log
    depends_on:
      - mysql

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=root_password
      - MYSQL_DATABASE=odata_db
      - MYSQL_USER=odata_user
      - MYSQL_PASSWORD=secure_password
    volumes:
      - mysql_data:/var/lib/mysql

volumes:
  mysql_data:

Systemd Service Example:
[Unit]
Description=Go-Data OData Server
After=network.target mysql.service

[Service]
Type=simple
User=odata
WorkingDirectory=/opt/odata
ExecStart=/opt/odata/server
Restart=always
RestartSec=5
Environment=GO_ENV=production

[Install]
WantedBy=multi-user.target

Nginx Reverse Proxy Example:
server {
    listen 80;
    server_name api.mycompany.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.mycompany.com;

    ssl_certificate /etc/ssl/certs/api.mycompany.com.crt;
    ssl_certificate_key /etc/ssl/private/api.mycompany.com.key;

    location / {
        proxy_pass https://localhost:8443;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
*/
