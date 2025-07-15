# Go-Data — OData v4 para APIs RESTful em Go (Golang)

**Go-Data** é uma biblioteca leve e extensível para criação de APIs RESTful baseadas no padrão [OData v4](https://www.odata.org/) usando Go (Golang).  
Ela oferece suporte completo ao formato JSON, inclui um servidor embutido com [Fiber v3](https://github.com/gofiber/fiber), e funciona com múltiplos bancos de dados (PostgreSQL, MySQL, Oracle).


## 📋 Índice

- [Características](#-características)
- [Instalação](#-instalação)
- [Configuração com .env](#-configuração-com-env)
- [Exemplo de Uso](#-exemplo-de-uso)
- [Configuração do Servidor](#-configuração-do-servidor)
- [Autenticação JWT](#-autenticação-jwt)
- [Multi-Tenant](#-multi-tenant)
- [Eventos de Entidade](#-eventos-de-entidade)
- [Mapeamento de Entidades](#-mapeamento-de-entidades)
- [Bancos de Dados Suportados](#-bancos-de-dados-suportados)
- [Endpoints OData](#-endpoints-odata)
- [Consultas OData](#-consultas-odata)
- [Operadores Suportados](#-operadores-suportados)
- [Mapeamento de Tipos](#-mapeamento-de-tipos)
- [Contribuindo](#-contribuindo)
- [Exemplos](#-exemplos)
- [Referências](#referências)
- [Licença](#-licença)
- [Suporte](#-suporte)

## ✨ Características

### 🌐 **Protocolo OData v4**
- Suporte ao protocolo OData v4 com resposta JSON
- Geração automática de metadados JSON
- Service Document automático
- Operações CRUD completas

### 🚀 **Servidor Fiber v3**
- Servidor HTTP embutido baseado no Fiber v3
- Suporte a HTTPS/TLS
- Configuração de CORS
- Middleware de logging e recovery
- Shutdown graceful

### 💾 **Múltiplos Bancos de Dados**
- PostgreSQL
- Oracle
- MySQL
- Pool de conexões automático

### 🔧 **Mapeamento Automático**
- Sistema de tags para mapeamento de structs
- Relacionamentos bidirecionais
- Operações em cascata
- Tipos nullable personalizados

### 🔍 **Consultas OData**
- Filtros ($filter)
- Ordenação ($orderby)
- Paginação ($top, $skip)
- Seleção de campos ($select)
- Expansão de relacionamentos ($expand)
- Contagem ($count)
- Campos computados ($compute)
- Busca textual ($search)

### 🔐 **Autenticação JWT**
- Geração de tokens de acesso e refresh
- Validação de tokens JWT
- Middleware de autenticação obrigatória e opcional
- Controle de acesso baseado em roles e scopes
- Privilégios de administrador
- Configuração de autenticação por entidade
- Entidades somente leitura

### 🏢 **Multi-Tenant**
- Suporte completo a multi-tenant com isolamento de dados
- Identificação automática via headers, subdomains, path ou JWT
- Pool de conexões gerenciado automaticamente para cada tenant
- Configuração via .env com múltiplos bancos de dados
- Endpoints específicos para gerenciamento de tenants
- Escalabilidade com adição dinâmica de novos tenants

### ⚙️ **Configuração Automática**
- Carregamento automático de configurações via arquivo `.env`
- Busca automática do arquivo `.env` na árvore de diretórios
- Valores padrão sensatos quando `.env` não encontrado
- Configuração completa de banco de dados, servidor, TLS e JWT

## 🚀 Instalação

```bash
go get github.com/fitlcarlos/go-data
```

## 🛠️ Configuração com .env

O Go-Data suporta configuração automática através de arquivos `.env`, similar ao Spring Boot. O sistema busca automaticamente por arquivos `.env` no diretório atual e diretórios pai.

### Exemplo de arquivo .env

```env
# Configurações do Banco de Dados
DB_TYPE=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=testdb
DB_SCHEMA=public
DB_CONNECTION_STRING=
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=600s

# Configurações do Servidor OData
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_ROUTE_PREFIX=/odata
SERVER_ENABLE_CORS=true
SERVER_ALLOWED_ORIGINS=*
SERVER_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
SERVER_ALLOWED_HEADERS=*
SERVER_EXPOSED_HEADERS=OData-Version,Content-Type
SERVER_ALLOW_CREDENTIALS=false
SERVER_ENABLE_LOGGING=true
SERVER_LOG_LEVEL=INFO
SERVER_LOG_FILE=
SERVER_ENABLE_COMPRESSION=false
SERVER_MAX_REQUEST_SIZE=10485760
SERVER_SHUTDOWN_TIMEOUT=30s

# Configurações de SSL/TLS
SERVER_TLS_CERT_FILE=
SERVER_TLS_KEY_FILE=

# Configurações de JWT
JWT_ENABLED=false
JWT_SECRET_KEY=
JWT_ISSUER=go-data-server
JWT_EXPIRES_IN=1h
JWT_REFRESH_IN=24h
JWT_ALGORITHM=HS256
JWT_REQUIRE_AUTH=false

# Configurações Multi-Tenant
MULTI_TENANT_ENABLED=false
TENANT_IDENTIFICATION_MODE=header
TENANT_HEADER_NAME=X-Tenant-ID
DEFAULT_TENANT=default

# Configurações específicas por tenant (exemplo)
TENANT_EMPRESA_A_DB_TYPE=postgresql
TENANT_EMPRESA_A_DB_HOST=localhost
TENANT_EMPRESA_A_DB_PORT=5432
TENANT_EMPRESA_A_DB_NAME=empresa_a
TENANT_EMPRESA_A_DB_USER=user_a
TENANT_EMPRESA_A_DB_PASSWORD=password_a
```

### Descrição das Variáveis

#### Configurações do Banco de Dados
- **DB_TYPE**: Tipo do banco de dados (postgresql, mysql, oracle)
- **DB_HOST**: Endereço do servidor de banco de dados
- **DB_PORT**: Porta do servidor de banco de dados
- **DB_NAME**: Nome do banco de dados
- **DB_USER**: Usuário do banco de dados
- **DB_PASSWORD**: Senha do banco de dados
- **DB_SCHEMA**: Schema do banco de dados (opcional)
- **DB_CONNECTION_STRING**: String de conexão customizada (opcional)
- **DB_MAX_OPEN_CONNS**: Máximo de conexões abertas (padrão: 25)
- **DB_MAX_IDLE_CONNS**: Máximo de conexões inativas (padrão: 5)
- **DB_CONN_MAX_LIFETIME**: Tempo de vida das conexões (padrão: 10m)

#### Configurações do Servidor
- **SERVER_HOST**: Endereço do servidor OData (padrão: localhost)
- **SERVER_PORT**: Porta do servidor OData (padrão: 9090)
- **SERVER_ROUTE_PREFIX**: Prefixo das rotas OData (padrão: /odata)
- **SERVER_ENABLE_CORS**: Habilita CORS (padrão: true)
- **SERVER_ALLOWED_ORIGINS**: Origins permitidas para CORS (padrão: *)
- **SERVER_ALLOWED_METHODS**: Métodos HTTP permitidos
- **SERVER_ALLOWED_HEADERS**: Headers permitidos
- **SERVER_EXPOSED_HEADERS**: Headers expostos
- **SERVER_ALLOW_CREDENTIALS**: Permite credenciais CORS (padrão: false)
- **SERVER_ENABLE_LOGGING**: Habilita logging (padrão: true)
- **SERVER_LOG_LEVEL**: Nível de logging (padrão: INFO)
- **SERVER_LOG_FILE**: Arquivo de log (opcional)
- **SERVER_ENABLE_COMPRESSION**: Habilita compressão (padrão: false)
- **SERVER_MAX_REQUEST_SIZE**: Tamanho máximo da requisição (padrão: 10MB)
- **SERVER_SHUTDOWN_TIMEOUT**: Timeout para shutdown graceful (padrão: 30s)

#### Configurações TLS
- **SERVER_TLS_CERT_FILE**: Caminho para o arquivo de certificado TLS
- **SERVER_TLS_KEY_FILE**: Caminho para o arquivo de chave TLS

#### Configurações JWT
- **JWT_ENABLED**: Habilita autenticação JWT (padrão: false)
- **JWT_SECRET_KEY**: Chave secreta para assinatura JWT
- **JWT_ISSUER**: Emissor do token JWT (padrão: go-data-server)
- **JWT_EXPIRES_IN**: Tempo de expiração do token de acesso (padrão: 1h)
- **JWT_REFRESH_IN**: Tempo de expiração do token de refresh (padrão: 24h)
- **JWT_ALGORITHM**: Algoritmo de assinatura JWT (padrão: HS256)
- **JWT_REQUIRE_AUTH**: Requer autenticação para todas as rotas (padrão: false)

#### Configurações Multi-Tenant
- **MULTI_TENANT_ENABLED**: Habilita suporte multi-tenant (padrão: false)
- **TENANT_IDENTIFICATION_MODE**: Método de identificação do tenant (header, subdomain, path, jwt)
- **TENANT_HEADER_NAME**: Nome do header para identificação (padrão: X-Tenant-ID)
- **DEFAULT_TENANT**: Nome do tenant padrão (padrão: default)
- **TENANT_[NOME]_DB_TYPE**: Tipo de banco para tenant específico
- **TENANT_[NOME]_DB_HOST**: Host do banco para tenant específico
- **TENANT_[NOME]_DB_PORT**: Porta do banco para tenant específico
- **TENANT_[NOME]_DB_NAME**: Nome do banco para tenant específico
- **TENANT_[NOME]_DB_USER**: Usuário do banco para tenant específico
- **TENANT_[NOME]_DB_PASSWORD**: Senha do banco para tenant específico

### Uso Transparente

O método `NewServer()` é **transparente** e carrega automaticamente as configurações do arquivo `.env` quando disponível:

```go
package main

import (
    "log"
    
    "github.com/fitlcarlos/go-data/pkg/odata"
)

func main() {
    // Cria servidor automaticamente:
    // - Se .env existe: carrega configurações completas (servidor + banco)
    // - Se .env não existe: retorna servidor básico para configuração manual
    server := odata.NewServer()
    
    // Registrar entidades
    server.RegisterEntity("Users", User{})
    
    // Iniciar servidor
    log.Fatal(server.Start())
}
```

### Como Funciona

1. **Busca Automática**: O `NewServer()` busca automaticamente por arquivos `.env` no diretório atual e diretórios pai (até a raiz do sistema)
2. **Configuração Automática**: Se encontrar `.env` com `DB_TYPE` válido, configura automaticamente o provider de banco e servidor
3. **Fallback Gracioso**: Se não encontrar `.env` ou `DB_TYPE` inválido, retorna servidor básico para configuração manual
4. **Zero Configuração**: Não precisa chamar métodos específicos - tudo é automático

### Exemplo com Arquivo .env

1. **Crie um arquivo `.env`** na raiz do projeto:

```env
# Configuração PostgreSQL
DB_TYPE=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=mypassword
DB_NAME=mydatabase

# Configuração do servidor
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_ROUTE_PREFIX=/api/v1

# JWT (opcional)
JWT_ENABLED=true
JWT_SECRET_KEY=minha-chave-secreta-super-segura
JWT_ISSUER=minha-aplicacao

# Multi-Tenant (opcional)
MULTI_TENANT_ENABLED=true
TENANT_IDENTIFICATION_MODE=header
TENANT_HEADER_NAME=X-Tenant-ID
DEFAULT_TENANT=default

# Configurações por tenant
TENANT_EMPRESA_A_DB_TYPE=postgresql
TENANT_EMPRESA_A_DB_HOST=postgres-a.empresa.com
TENANT_EMPRESA_A_DB_PORT=5432
TENANT_EMPRESA_A_DB_NAME=empresa_a
TENANT_EMPRESA_A_DB_USER=user_a
TENANT_EMPRESA_A_DB_PASSWORD=password_a

TENANT_EMPRESA_B_DB_TYPE=mysql
TENANT_EMPRESA_B_DB_HOST=mysql-b.empresa.com
TENANT_EMPRESA_B_DB_PORT=3306
TENANT_EMPRESA_B_DB_NAME=empresa_b
TENANT_EMPRESA_B_DB_USER=user_b
TENANT_EMPRESA_B_DB_PASSWORD=password_b
```

2. **Use o servidor transparente**:

```go
func main() {
    // Carrega automaticamente todas as configurações do .env
    server := odata.NewServer()
    
    // Registra entidades
    server.RegisterEntity("Users", User{})
    server.RegisterEntity("Products", Product{})
    
    // Inicia - todas as configurações já estão aplicadas
    log.Fatal(server.Start())
}
```

### Configuração Manual (Fallback)

Se não usar `.env` ou precisar de configurações específicas, ainda pode configurar manualmente:

```go
// Configuração manual tradicional
provider := providers.NewPostgreSQLProvider(db)
server := odata.NewServerWithProvider(provider, "localhost", 8080, "/api")

// Ou configuração completa
config := odata.DefaultServerConfig()
config.Host = "localhost"
config.Port = 8080
server := odata.NewServerWithConfig(provider, config)
```

## 📝 Exemplo de Uso

### Servidor Automático com .env

```go
package main

import (
    "log"
    
    "github.com/fitlcarlos/go-data/pkg/odata"
)

// Entidade de exemplo
type User struct {
    ID    int    `json:"id" odata:"key"`
    Name  string `json:"name" odata:"required"`
    Email string `json:"email" odata:"required"`
}

func main() {
    // Servidor automático (carrega .env se disponível)
    server := odata.NewServer()
    
    // Registrar entidades
    server.RegisterEntity("Users", User{})
    
    // Iniciar servidor
    log.Fatal(server.Start())
}
```

### Servidor Básico

```go
package main

import (
    "database/sql"
    "log"
    
    "github.com/fitlcarlos/go-data/pkg/odata"
    "github.com/fitlcarlos/go-data/pkg/providers"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // Conecta ao banco
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/database")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Cria provider
    provider := providers.NewMySQLProvider(db)
    
    // Cria servidor com configurações específicas
    server := odata.NewServerWithProvider(provider, "localhost", 8080, "/odata")
    
    // Registra entidades
    server.RegisterEntity("Users", User{})
    
    // Inicia servidor
    log.Fatal(server.Start())
}
```

### Definindo Entidades

```go
type User struct {
    TableName string           `table:"users"`
    ID        int64            `json:"id" primaryKey:"idGenerator:sequence"`
    Nome      string           `json:"nome" prop:"[required]; length:100"`
    Email     string           `json:"email" prop:"[required, Unique]; length:255"`
    Idade     nullable.Int64   `json:"idade"`
    Ativo     bool             `json:"ativo" prop:"[required]; default"`
    DtInc     time.Time        `json:"dt_inc" prop:"[required, NoUpdate]; default"`
    
    // Relacionamentos
    Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id"`
}

type Order struct {
    TableName string    `table:"orders"`
    ID        int64     `json:"id" primaryKey:"idGenerator:sequence"`
    UserID    int64     `json:"user_id" prop:"[required]"`
    Total     float64   `json:"total" prop:"[required]; precision:10; scale:2"`
    DtPedido  time.Time `json:"dt_pedido" prop:"[required]"`
    
    // Relacionamento N:1
    User *User `json:"User" association:"foreignKey:user_id; references:id"`
}
```

## ⚙️ Configuração do Servidor

### Configuração Personalizada

```go
config := &odata.ServerConfig{
    Host:              "0.0.0.0",
    Port:              8080,
    
    // CORS
    EnableCORS:       true,
    AllowedOrigins:   []string{"*"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    
    // Logging
    EnableLogging:     true,
    LogLevel:          "INFO",
    
    // Limites
    MaxRequestSize:    5 * 1024 * 1024, // 5MB
    
    // Prefixo das rotas
    RoutePrefix: "/api/odata",
    
    // Timeout
    ShutdownTimeout: 30 * time.Second,
}

server := odata.NewServerWithConfig(provider, config)
```

### HTTPS/TLS

```go
config := odata.DefaultServerConfig()
config.TLSConfig = &tls.Config{
    MinVersion: tls.VersionTLS12,
}
config.CertFile = "server.crt"
config.CertKeyFile = "server.key"
```

## 🔐 Autenticação JWT

O Go-Data oferece suporte completo à autenticação JWT com controle de acesso granular baseado em roles e scopes.

### Configuração Básica

```go
import "github.com/fitlcarlos/go-data/pkg/odata"

// Configurar JWT
jwtConfig := &odata.JWTConfig{
    SecretKey: "sua-chave-secreta-super-segura",
    Issuer:    "seu-aplicativo",
    ExpiresIn: 1 * time.Hour,
    RefreshIn: 24 * time.Hour,
    Algorithm: "HS256",
}

// Configurar servidor com JWT
config := odata.DefaultServerConfig()
config.EnableJWT = true
config.JWTConfig = jwtConfig
config.RequireAuth = false // Autenticação global opcional

server := odata.NewServerWithConfig(provider, config)
```

### Implementando Autenticador

```go
type UserAuthenticator struct {
    // Sua implementação de banco de dados
}

func (a *UserAuthenticator) Authenticate(username, password string) (*odata.UserIdentity, error) {
    // Validar credenciais no banco de dados
    // Retornar UserIdentity com roles e scopes
    return &odata.UserIdentity{
        Username: username,
        Roles:    []string{"user", "manager"},
        Scopes:   []string{"read", "write"},
        Admin:    false,
        Custom: map[string]interface{}{
            "department": "IT",
            "level":      "senior",
        },
    }, nil
}

func (a *UserAuthenticator) GetUserByUsername(username string) (*odata.UserIdentity, error) {
    // Buscar usuário no banco de dados
    return user, nil
}

// Configurar rotas de autenticação
authenticator := &UserAuthenticator{}
server.SetupAuthRoutes(authenticator)
```

### Controle de Acesso por Entidade

```go
// Apenas administradores podem acessar usuários
server.SetEntityAuth("Users", odata.EntityAuthConfig{
    RequireAuth:  true,
    RequireAdmin: true,
})

// Managers e admins podem escrever produtos
server.SetEntityAuth("Products", odata.EntityAuthConfig{
    RequireAuth:    true,
    RequiredRoles:  []string{"manager", "admin"},
    RequiredScopes: []string{"write"},
})

// Entidade somente leitura
server.SetEntityAuth("Reports", odata.EntityAuthConfig{
    RequireAuth: true,
    ReadOnly:    true,
})
```

### Middlewares de Autorização

```go
// Middleware que requer autenticação
app.Use("/admin", odata.RequireAuth())

// Middleware que requer role específica
app.Use("/management", odata.RequireRole("manager"))

// Middleware que requer múltiplas roles
app.Use("/restricted", odata.RequireAnyRole("admin", "supervisor"))

// Middleware que requer scope específico
app.Use("/api/write", odata.RequireScope("write"))

// Middleware que requer privilégios de admin
app.Use("/admin", odata.RequireAdmin())
```

### Endpoints de Autenticação

#### Login
```bash
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password123"
}
```

Resposta:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "username": "admin",
    "roles": ["admin", "user"],
    "scopes": ["read", "write", "delete"],
    "admin": true
  }
}
```

#### Refresh Token
```bash
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### Informações do Usuário
```bash
GET /auth/me
Authorization: Bearer <access_token>
```

#### Logout
```bash
POST /auth/logout
Authorization: Bearer <access_token>
```

### Usando Tokens JWT

```bash
# Acessar endpoint protegido
curl -X GET http://localhost:8080/odata/Users \
  -H "Authorization: Bearer <access_token>"
```

### Exemplo Completo

Veja o exemplo completo em [`examples/jwt/`](examples/jwt/) que demonstra:

- Configuração completa de JWT
- Usuários de teste com diferentes roles
- Controle de acesso por entidade
- Cenários de teste para diferentes tipos de usuário
- Integração com banco de dados

### Estrutura de UserIdentity

```go
type UserIdentity struct {
    Username string                 `json:"username"`
    Roles    []string               `json:"roles"`
    Scopes   []string               `json:"scopes"`
    Admin    bool                   `json:"admin"`
    Custom   map[string]interface{} `json:"custom"`
}

// Métodos disponíveis
user.HasRole("manager")           // Verifica role específica
user.HasAnyRole("admin", "user")  // Verifica múltiplas roles
user.HasScope("write")            // Verifica scope específico
user.IsAdmin()                    // Verifica se é admin
user.GetCustomClaim("department") // Obtém claim customizado
```

### Configuração de Segurança

```go
type JWTConfig struct {
    SecretKey  string        // Chave secreta para assinatura
    Issuer     string        // Emissor do token
    ExpiresIn  time.Duration // Tempo de expiração do access token
    RefreshIn  time.Duration // Tempo de expiração do refresh token
    Algorithm  string        // Algoritmo de assinatura (HS256)
}

type EntityAuthConfig struct {
    RequireAuth    bool     // Requer autenticação
    RequiredRoles  []string // Roles necessárias
    RequiredScopes []string // Scopes necessários
    RequireAdmin   bool     // Requer privilégios de admin
    ReadOnly       bool     // Entidade somente leitura
}
```

## 🏢 Multi-Tenant

O Go-Data oferece suporte completo a multi-tenant, permitindo que uma única instância do servidor gerencie múltiplos bancos de dados para diferentes tenants (clientes, organizações, etc.). Cada tenant mantém isolamento completo dos dados.

### Características Multi-Tenant

- **Identificação automática de tenant** via headers, subdomains, path ou JWT
- **Pool de conexões** gerenciado automaticamente para cada tenant
- **Configuração via .env** com suporte a múltiplos bancos de dados
- **Isolamento completo** de dados por tenant
- **Compatibilidade** com Oracle, PostgreSQL e MySQL
- **Endpoints específicos** para monitoramento e gerenciamento de tenants
- **Escalabilidade** com adição dinâmica de novos tenants

### Configuração Multi-Tenant

#### Arquivo .env

```env
# Configuração Multi-Tenant
MULTI_TENANT_ENABLED=true
TENANT_IDENTIFICATION_MODE=header
TENANT_HEADER_NAME=X-Tenant-ID
DEFAULT_TENANT=default

# Configuração do servidor
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_ROUTE_PREFIX=/api/odata

# Configuração do banco padrão
DB_TYPE=oracle
DB_HOST=localhost
DB_PORT=1521
DB_NAME=ORCL
DB_USER=system
DB_PASSWORD=password

# Configuração específica por tenant
TENANT_EMPRESA_A_DB_TYPE=oracle
TENANT_EMPRESA_A_DB_HOST=oracle1.empresa.com
TENANT_EMPRESA_A_DB_PORT=1521
TENANT_EMPRESA_A_DB_NAME=EMPRESA_A
TENANT_EMPRESA_A_DB_USER=user_a
TENANT_EMPRESA_A_DB_PASSWORD=password_a

TENANT_EMPRESA_B_DB_TYPE=postgres
TENANT_EMPRESA_B_DB_HOST=postgres1.empresa.com
TENANT_EMPRESA_B_DB_PORT=5432
TENANT_EMPRESA_B_DB_NAME=empresa_b
TENANT_EMPRESA_B_DB_USER=user_b
TENANT_EMPRESA_B_DB_PASSWORD=password_b

TENANT_EMPRESA_C_DB_TYPE=mysql
TENANT_EMPRESA_C_DB_HOST=mysql1.empresa.com
TENANT_EMPRESA_C_DB_PORT=3306
TENANT_EMPRESA_C_DB_NAME=empresa_c
TENANT_EMPRESA_C_DB_USER=user_c
TENANT_EMPRESA_C_DB_PASSWORD=password_c
```

#### Código do Servidor

```go
package main

import (
    "log"
    
    "github.com/fitlcarlos/go-data/pkg/odata"
)

func main() {
    // Cria servidor com carregamento automático de configurações multi-tenant
    server := odata.NewServer()
    
    // Registra as entidades (automaticamente multi-tenant se configurado)
    server.RegisterEntity("Produtos", &Produto{})
    server.RegisterEntity("Clientes", &Cliente{})
    server.RegisterEntity("Pedidos", &Pedido{})
    
    // Eventos globais com informações de tenant
    server.OnEntityListGlobal(func(args odata.EventArgs) error {
        if listArgs, ok := args.(*odata.EntityListArgs); ok {
            tenantID := odata.GetCurrentTenant(listArgs.Context.FiberContext)
            log.Printf("📋 Lista acessada: %s (tenant: %s)", 
                listArgs.EntityName, tenantID)
        }
        return nil
    })
    
    // Inicia o servidor
    log.Fatal(server.Start())
}
```

### Métodos de Identificação de Tenant

#### 1. Header (Padrão)

```bash
# Listar produtos do tenant padrão
curl -X GET "http://localhost:8080/api/odata/Produtos"

# Listar produtos da empresa A
curl -X GET "http://localhost:8080/api/odata/Produtos" \
  -H "X-Tenant-ID: empresa_a"
```

#### 2. Subdomain

Configure `TENANT_IDENTIFICATION_MODE=subdomain`:

```bash
# Acesso via subdomain
curl -X GET "http://empresa_a.localhost:8080/api/odata/Produtos"
```

#### 3. Path

Configure `TENANT_IDENTIFICATION_MODE=path`:

```bash
# Acesso via path
curl -X GET "http://localhost:8080/api/empresa_a/odata/Produtos"
```

#### 4. JWT Token

Configure `TENANT_IDENTIFICATION_MODE=jwt` e inclua claim `tenant_id`:

```bash
# Acesso via JWT (com claim tenant_id)
curl -X GET "http://localhost:8080/api/odata/Produtos" \
  -H "Authorization: Bearer <jwt_token_com_tenant_id>"
```

### Endpoints de Gerenciamento Multi-Tenant

#### Listar Tenants

```bash
GET /tenants
```

Resposta:
```json
{
  "multi_tenant": true,
  "tenants": ["default", "empresa_a", "empresa_b", "empresa_c"],
  "total_count": 4
}
```

#### Estatísticas dos Tenants

```bash
GET /tenants/stats
```

Resposta:
```json
{
  "total_tenants": 3,
  "tenants": {
    "empresa_a": {
      "tenant_id": "empresa_a",
      "exists": true,
      "provider_type": "*oracle.OracleProvider",
      "open_connections": 5,
      "in_use": 2,
      "idle": 3
    }
  }
}
```

#### Health Check por Tenant

```bash
GET /tenants/empresa_a/health
```

Resposta:
```json
{
  "tenant_id": "empresa_a",
  "status": "healthy",
  "connection_stats": {
    "open_connections": 5,
    "in_use": 2,
    "idle": 3
  }
}
```

### Entidades Multi-Tenant

As entidades incluem automaticamente o campo `tenant_id` para isolamento:

```go
type Produto struct {
    ID          int64  `json:"id" db:"id" odata:"key"`
    Nome        string `json:"nome" db:"nome"`
    Descricao   string `json:"descricao" db:"descricao"`
    Preco       float64 `json:"preco" db:"preco"`
    Categoria   string `json:"categoria" db:"categoria"`
    TenantID    string `json:"tenant_id" db:"tenant_id"`
}

type Cliente struct {
    ID       int64  `json:"id" db:"id" odata:"key"`
    Nome     string `json:"nome" db:"nome"`
    Email    string `json:"email" db:"email"`
    Telefone string `json:"telefone" db:"telefone"`
    TenantID string `json:"tenant_id" db:"tenant_id"`
}
```

### Adicionando Novos Tenants

Para adicionar um novo tenant, basta incluir no `.env`:

```env
TENANT_NOVO_CLIENTE_DB_TYPE=mysql
TENANT_NOVO_CLIENTE_DB_HOST=mysql.novocliente.com
TENANT_NOVO_CLIENTE_DB_PORT=3306
TENANT_NOVO_CLIENTE_DB_NAME=novo_cliente
TENANT_NOVO_CLIENTE_DB_USER=user
TENANT_NOVO_CLIENTE_DB_PASSWORD=password
```

E reiniciar o servidor. O tenant será automaticamente detectado e configurado.

### Vantagens do Multi-Tenant

1. **Isolamento de dados**: Cada tenant tem seu próprio banco de dados
2. **Escalabilidade**: Adição dinâmica de novos tenants
3. **Flexibilidade**: Diferentes tipos de banco por tenant
4. **Monitoramento**: Estatísticas individuais por tenant
5. **Segurança**: Isolamento completo entre tenants
6. **Performance**: Pool de conexões otimizado por tenant

### Considerações de Segurança

- **Validação de tenant**: Sempre valide se o tenant existe
- **Autenticação**: Use JWT com claim `tenant_id` para maior segurança
- **Auditoria**: Todos os acessos são logados com tenant ID
- **Isolamento**: Dados são completamente isolados por tenant

### Exemplo Completo

Veja o exemplo completo em [`examples/multi_tenant/`](examples/multi_tenant/) que demonstra:

- Configuração completa multi-tenant
- Entidades com isolamento por tenant
- Múltiplos métodos de identificação
- Endpoints de gerenciamento
- Monitoramento e health checks
- Diferentes tipos de banco por tenant

## 🎯 Eventos de Entidade

O Go-Data oferece um sistema completo de eventos de entidade, permitindo interceptar e customizar operações CRUD através de handlers de eventos. Este sistema é ideal para implementar validações customizadas, auditoria, log de atividades e regras de negócio complexas.

### Tipos de Eventos Disponíveis

#### Eventos de Recuperação
- **`OnEntityGet`**: Disparado após uma entidade ser recuperada, antes de ser enviada ao cliente
- **`OnEntityList`**: Disparado quando o cliente consulta uma coleção de entidades

#### Eventos de Inserção
- **`OnEntityInserting`**: Disparado antes de uma entidade ser inserida (cancelável)
- **`OnEntityInserted`**: Disparado após uma entidade ser inserida

#### Eventos de Atualização
- **`OnEntityModifying`**: Disparado antes de uma entidade ser atualizada (cancelável)
- **`OnEntityModified`**: Disparado após uma entidade ser atualizada

#### Eventos de Exclusão
- **`OnEntityDeleting`**: Disparado antes de uma entidade ser excluída (cancelável)
- **`OnEntityDeleted`**: Disparado após uma entidade ser excluída

#### Eventos de Erro
- **`OnEntityError`**: Disparado quando ocorre um erro durante operações da entidade

### Registro de Eventos

#### Eventos Específicos por Entidade

Os eventos específicos por entidade se aplicam apenas à entidade nomeada. Estão disponíveis os seguintes métodos:

**Métodos de Eventos Específicos por Entidade:**
- `OnEntityGet("EntityName", handler)` - Disparado após uma entidade específica ser consultada
- `OnEntityList("EntityName", handler)` - Disparado após uma coleção de entidades específica ser consultada
- `OnEntityInserting("EntityName", handler)` - Disparado antes de uma entidade específica ser inserida
- `OnEntityInserted("EntityName", handler)` - Disparado após uma entidade específica ser inserida
- `OnEntityModifying("EntityName", handler)` - Disparado antes de uma entidade específica ser atualizada
- `OnEntityModified("EntityName", handler)` - Disparado após uma entidade específica ser atualizada
- `OnEntityDeleting("EntityName", handler)` - Disparado antes de uma entidade específica ser excluída
- `OnEntityDeleted("EntityName", handler)` - Disparado após uma entidade específica ser excluída
- `OnEntityError("EntityName", handler)` - Disparado quando ocorre erro em uma entidade específica

**Exemplos de uso:**

```go
// Validação antes da inserção
server.OnEntityInserting("Users", func(args odata.EventArgs) error {
    insertArgs := args.(*odata.EntityInsertingArgs)
    
    // Validação customizada
    if name, ok := insertArgs.Data["name"].(string); ok && len(name) < 2 {
        args.Cancel("Nome deve ter pelo menos 2 caracteres")
        return nil
    }
    
    // Adicionar timestamps automaticamente
    insertArgs.Data["created"] = time.Now()
    insertArgs.Data["updated"] = time.Now()
    
    return nil
})

// Ação após inserção
server.OnEntityInserted("Users", func(args odata.EventArgs) error {
    insertedArgs := args.(*odata.EntityInsertedArgs)
    
    // Enviar email de boas-vindas
    // sendWelcomeEmail(insertedArgs.CreatedEntity)
    
    log.Printf("Usuário criado: %+v", insertedArgs.CreatedEntity)
    return nil
})

// Validação antes da atualização
server.OnEntityModifying("Users", func(args odata.EventArgs) error {
    modifyArgs := args.(*odata.EntityModifyingArgs)
    
    // Impedir alteração de email por usuários não-admin
    if _, emailChanged := modifyArgs.Data["email"]; emailChanged {
        if !isCurrentUserAdmin(modifyArgs.GetContext()) {
            args.Cancel("Apenas administradores podem alterar email")
            return nil
        }
    }
    
    // Atualizar timestamp
    modifyArgs.Data["updated"] = time.Now()
    
    return nil
})

// Controle de acesso para exclusão
server.OnEntityDeleting("Users", func(args odata.EventArgs) error {
    deleteArgs := args.(*odata.EntityDeletingArgs)
    
    // Impedir exclusão se usuário tem dependências
    if hasUserDependencies(deleteArgs.Keys) {
        args.Cancel("Não é possível excluir usuário com dependências")
        return nil
    }
    
    return nil
})

// Ação após exclusão
server.OnEntityDeleted("Users", func(args odata.EventArgs) error {
    deletedArgs := args.(*odata.EntityDeletedArgs)
    
    // Limpar dados relacionados
    // cleanupRelatedData(deletedArgs.Keys)
    
    log.Printf("Usuário excluído: %+v", deletedArgs.Keys)
    return nil
})

// Ação após atualização
server.OnEntityModified("Users", func(args odata.EventArgs) error {
    modifiedArgs := args.(*odata.EntityModifiedArgs)
    
    // Invalidar cache
    // invalidateUserCache(modifiedArgs.Keys)
    
    log.Printf("Usuário atualizado: %+v", modifiedArgs.UpdatedEntity)
    return nil
})

// Auditoria de consultas específicas
server.OnEntityGet("Users", func(args odata.EventArgs) error {
    getArgs := args.(*odata.EntityGetArgs)
    
    // Log de acesso
    log.Printf("Usuário consultado: %+v", getArgs.Keys)
    
    // Contabilizar acesso
    // trackUserAccess(getArgs.Keys)
    
    return nil
})

// Auditoria de listagens específicas
server.OnEntityList("Users", func(args odata.EventArgs) error {
    listArgs := args.(*odata.EntityListArgs)
    
    // Log de listagem
    log.Printf("Lista de usuários consultada: %d resultados", len(listArgs.Results))
    
    // Aplicar filtros adicionais baseados no usuário
    // applyUserFilters(listArgs)
    
    return nil
})

// Tratamento de erros específicos
server.OnEntityError("Users", func(args odata.EventArgs) error {
    errorArgs := args.(*odata.EntityErrorArgs)
    
    // Log específico para erros de usuário
    log.Printf("Erro na entidade Users: %v", errorArgs.Error)
    
    // Enviar notificação específica
    // sendUserErrorNotification(errorArgs.Error)
    
    return nil
})
```

#### Eventos Globais

Os eventos globais se aplicam a todas as entidades registradas no servidor. Estão disponíveis os seguintes métodos:

**Métodos de Eventos Globais:**
- `OnEntityGetGlobal()` - Disparado após qualquer entidade ser consultada
- `OnEntityListGlobal()` - Disparado após qualquer coleção de entidades ser consultada
- `OnEntityInsertingGlobal()` - Disparado antes de qualquer entidade ser inserida
- `OnEntityInsertedGlobal()` - Disparado após qualquer entidade ser inserida
- `OnEntityModifyingGlobal()` - Disparado antes de qualquer entidade ser atualizada
- `OnEntityModifiedGlobal()` - Disparado após qualquer entidade ser atualizada
- `OnEntityDeletingGlobal()` - Disparado antes de qualquer entidade ser excluída
- `OnEntityDeletedGlobal()` - Disparado após qualquer entidade ser excluída
- `OnEntityErrorGlobal()` - Disparado quando ocorre erro em qualquer entidade

**Exemplos de uso:**

```go
// Auditoria global para todas as inserções
server.OnEntityInsertingGlobal(func(args odata.EventArgs) error {
    log.Printf("Inserindo entidade: %s por usuário: %s", 
        args.GetEntityName(), 
        args.GetContext().UserID)
    
    // Registrar auditoria
    // auditLog.Record("INSERT", args.GetEntityName(), args.GetContext().UserID)
    
    return nil
})

// Log de todas as modificações
server.OnEntityModifyingGlobal(func(args odata.EventArgs) error {
    log.Printf("Modificando entidade: %s", args.GetEntityName())
    return nil
})

// Tratamento global de erros
server.OnEntityErrorGlobal(func(args odata.EventArgs) error {
    errorArgs := args.(*odata.EntityErrorArgs)
    
    log.Printf("Erro na entidade %s: %v", 
        args.GetEntityName(), 
        errorArgs.Error)
    
    // Enviar notificação ou alerta
    // errorNotification.Send(errorArgs.Error, errorArgs.Operation)
    
    return nil
})

// Auditoria global para todas as consultas
server.OnEntityGetGlobal(func(args odata.EventArgs) error {
    log.Printf("Entidade acessada: %s", args.GetEntityName())
    return nil
})

// Auditoria global para todas as listagens
server.OnEntityListGlobal(func(args odata.EventArgs) error {
    log.Printf("Lista de entidades acessada: %s", args.GetEntityName())
    return nil
})

// Auditoria global para todas as exclusões (antes)
server.OnEntityDeletingGlobal(func(args odata.EventArgs) error {
    log.Printf("Excluindo entidade: %s", args.GetEntityName())
    return nil
})
```

### Argumentos dos Eventos

#### EntityInsertingArgs
```go
type EntityInsertingArgs struct {
    Data             map[string]interface{} // Dados sendo inseridos
    ValidationErrors []string               // Erros de validação
    // Cancelável: true
}
```

#### EntityInsertedArgs
```go
type EntityInsertedArgs struct {
    CreatedEntity interface{} // Entidade criada
    NewID         interface{} // ID da nova entidade
    // Cancelável: false
}
```

#### EntityModifyingArgs
```go
type EntityModifyingArgs struct {
    Keys             map[string]interface{} // Chaves da entidade
    Data             map[string]interface{} // Dados sendo atualizados
    OriginalEntity   interface{}            // Entidade original
    ValidationErrors []string               // Erros de validação
    // Cancelável: true
}
```

#### EntityGetArgs
```go
type EntityGetArgs struct {
    Keys        map[string]interface{} // Chaves da entidade
    QueryParams map[string]interface{} // Parâmetros da consulta
    // Cancelável: false
}
```

#### EntityListArgs
```go
type EntityListArgs struct {
    QueryOptions  QueryOptions    // Opções da consulta OData
    Results       []interface{}   // Resultados da consulta
    TotalCount    int64          // Total de registros
    CustomFilters map[string]interface{} // Filtros customizados
    // Cancelável: true
}
```

#### EntityModifiedArgs
```go
type EntityModifiedArgs struct {
    Keys          map[string]interface{} // Chaves da entidade
    UpdatedEntity interface{}            // Entidade atualizada
    OriginalEntity interface{}           // Entidade original
    // Cancelável: false
}
```

#### EntityDeletingArgs
```go
type EntityDeletingArgs struct {
    Keys             map[string]interface{} // Chaves da entidade
    EntityToDelete   interface{}            // Entidade a ser excluída
    ValidationErrors []string               // Erros de validação
    // Cancelável: true
}
```

#### EntityDeletedArgs
```go
type EntityDeletedArgs struct {
    Keys           map[string]interface{} // Chaves da entidade excluída
    DeletedEntity  interface{}            // Entidade excluída
    // Cancelável: false
}
```

#### EntityErrorArgs
```go
type EntityErrorArgs struct {
    Error      error       // Erro ocorrido
    Operation  string      // Operação que causou o erro
    Keys       map[string]interface{} // Chaves da entidade (se disponível)
    Data       interface{} // Dados relacionados ao erro
    // Cancelável: false
}
```

### Contexto dos Eventos

Todos os eventos recebem um contexto rico com informações sobre a requisição:

```go
type EventContext struct {
    Context      context.Context // Contexto da requisição
    FiberContext fiber.Ctx       // Contexto do Fiber
    EntityName   string          // Nome da entidade
    EntityType   string          // Tipo da entidade
    UserID       string          // ID do usuário atual
    UserRoles    []string        // Roles do usuário
    UserScopes   []string        // Scopes do usuário
    RequestID    string          // ID da requisição
    Timestamp    int64           // Timestamp do evento
    Extra        map[string]interface{} // Dados extras
}
```

### Cancelamento de Eventos

Alguns eventos podem ser cancelados para impedir a operação:

```go
server.OnEntityInserting("Products", func(args odata.EventArgs) error {
    insertArgs := args.(*odata.EntityInsertingArgs)
    
    // Verificar se pode cancelar
    if args.CanCancel() {
        if price, ok := insertArgs.Data["price"].(float64); ok && price < 0 {
            args.Cancel("Preço não pode ser negativo")
            return nil
        }
    }
    
    return nil
})
```

### Exemplo Prático: Sistema de Auditoria

```go
type AuditLog struct {
    ID        int64     `json:"id"`
    Entity    string    `json:"entity"`
    Operation string    `json:"operation"`
    UserID    string    `json:"user_id"`
    Data      string    `json:"data"`
    Timestamp time.Time `json:"timestamp"`
}

func setupAuditEvents(server *odata.Server) {
    // Registrar todas as inserções
    server.OnEntityInsertedGlobal(func(args odata.EventArgs) error {
        return recordAudit("INSERT", args)
    })
    
    // Registrar todas as atualizações
    server.OnEntityModifiedGlobal(func(args odata.EventArgs) error {
        return recordAudit("UPDATE", args)
    })
    
    // Registrar todas as exclusões
    server.OnEntityDeletedGlobal(func(args odata.EventArgs) error {
        return recordAudit("DELETE", args)
    })
}

func recordAudit(operation string, args odata.EventArgs) error {
    audit := AuditLog{
        Entity:    args.GetEntityName(),
        Operation: operation,
        UserID:    args.GetContext().UserID,
        Data:      fmt.Sprintf("%+v", args.GetEntity()),
        Timestamp: time.Now(),
    }
    
    // Salvar no banco de dados
    // auditService.Save(audit)
    
    return nil
}
```

### Exemplo Prático: Validação Avançada

```go
func setupValidationEvents(server *odata.Server) {
    // Validação de usuários
    server.OnEntityInserting("Users", func(args odata.EventArgs) error {
        insertArgs := args.(*odata.EntityInsertingArgs)
        
        // Validações customizadas
        if err := validateUser(insertArgs.Data); err != nil {
            args.Cancel(err.Error())
            return nil
        }
        
        return nil
    })
    
    // Validação de produtos
    server.OnEntityInserting("Products", func(args odata.EventArgs) error {
        insertArgs := args.(*odata.EntityInsertingArgs)
        
        // Verificar se categoria existe
        if categoryID, ok := insertArgs.Data["category_id"].(int64); ok {
            if !categoryExists(categoryID) {
                args.Cancel("Categoria não encontrada")
                return nil
            }
        }
        
        return nil
    })
}

func validateUser(data map[string]interface{}) error {
    // Validar email único
    if email, ok := data["email"].(string); ok {
        if emailExists(email) {
            return fmt.Errorf("Email já está em uso")
        }
    }
    
    // Validar idade
    if age, ok := data["age"].(int64); ok && age < 18 {
        return fmt.Errorf("Idade deve ser maior que 18 anos")
    }
    
    return nil
}
```

### Gerenciamento de Eventos

```go
// Obter número de handlers registrados
count := server.GetEventManager().GetHandlerCount(odata.EventEntityInserting, "Users")

// Listar todas as assinaturas
subscriptions := server.GetEventManager().ListSubscriptions()

// Limpar handlers de uma entidade específica
server.GetEventManager().ClearEntity("Users")

// Limpar todos os handlers
server.GetEventManager().Clear()
```

### Resumo dos Métodos de Eventos

**Eventos Específicos por Entidade:**
```go
server.OnEntityGet("EntityName", handler)        // Após consulta individual
server.OnEntityList("EntityName", handler)       // Após consulta de coleção
server.OnEntityInserting("EntityName", handler)  // Antes de inserção (cancelável)
server.OnEntityInserted("EntityName", handler)   // Após inserção
server.OnEntityModifying("EntityName", handler)  // Antes de atualização (cancelável)
server.OnEntityModified("EntityName", handler)   // Após atualização
server.OnEntityDeleting("EntityName", handler)   // Antes de exclusão (cancelável)
server.OnEntityDeleted("EntityName", handler)    // Após exclusão
server.OnEntityError("EntityName", handler)      // Quando ocorre erro
```

**Eventos Globais:**
```go
server.OnEntityGetGlobal(handler)        // Após qualquer consulta individual
server.OnEntityListGlobal(handler)       // Após qualquer consulta de coleção
server.OnEntityInsertingGlobal(handler)  // Antes de qualquer inserção (cancelável)
server.OnEntityInsertedGlobal(handler)   // Após qualquer inserção
server.OnEntityModifyingGlobal(handler)  // Antes de qualquer atualização (cancelável)
server.OnEntityModifiedGlobal(handler)   // Após qualquer atualização
server.OnEntityDeletingGlobal(handler)   // Antes de qualquer exclusão (cancelável)
server.OnEntityDeletedGlobal(handler)    // Após qualquer exclusão
server.OnEntityErrorGlobal(handler)      // Quando ocorre qualquer erro
```

### Exemplo Completo

Veja o exemplo completo em [`examples/events/`](examples/events/) que demonstra:

- Configuração completa de eventos
- Validações customizadas
- Sistema de auditoria
- Controle de acesso baseado em contexto
- Tratamento de erros
- Cancelamento de operações

## 🗂️ Mapeamento de Entidades

### Tags Disponíveis

#### Tag `table`
```go
TableName string `table:"users;schema=public"`
```

#### Tag `prop`
```go
Nome  string `prop:"[required]; length:100"`
Email string `prop:"[required, Unique]; length:255"`
DtInc time.Time `prop:"[required, NoUpdate]; default"`
```

#### Tag `primaryKey`
```go
ID int64 `primaryKey:"idGenerator:sequence;name=seq_user_id"`
```

#### Tag `association` (N:1)
```go
User *User `association:"foreignKey:user_id; references:id"`
```

#### Tag `manyAssociation` (1:N)
```go
Orders []Order `manyAssociation:"foreignKey:user_id; references:id"`
```

#### Tag `cascade`
```go
Orders []Order `cascade:"[SaveUpdate, Remove, Refresh]"`
```

### Tipos Nullable

```go
import "github.com/fitlcarlos/go-data/pkg/nullable"

type User struct {
    ID      int64           `json:"id"`
    Nome    string          `json:"nome"`
    Idade   nullable.Int64  `json:"idade"`    // Pode ser null
    Salario nullable.Float64 `json:"salario"` // Pode ser null
    DtAlt   nullable.Time   `json:"dt_alt"`   // Pode ser null
}
```

## 💾 Bancos de Dados Suportados

### PostgreSQL
```go
import (
    "github.com/fitlcarlos/go-data/pkg/providers"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, err := sql.Open("pgx", "postgres://user:password@localhost/database")
provider := providers.NewPostgreSQLProvider(db)
```

### Oracle
```go
import (
    "github.com/fitlcarlos/go-data/pkg/providers"
    _ "github.com/sijms/go-ora/v2"
)

db, err := sql.Open("oracle", "oracle://user:password@localhost:1521/xe")
provider := providers.NewOracleProvider(db)
```

### MySQL
```go
import (
    "github.com/fitlcarlos/go-data/pkg/providers"
    _ "github.com/go-sql-driver/mysql"
)

db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/database")
provider := providers.NewMySQLProvider(db)
```

## 🌐 Endpoints OData

### Service Document
```
GET /odata/
```

### Metadados
```
GET /odata/$metadata
```

### Operações CRUD

#### Listar Entidades
```
GET /odata/Users
```

#### Buscar por ID
```
GET /odata/Users(1)
```

#### Listar Entidades com Multi-Tenant
```
GET /odata/Users
X-Tenant-ID: empresa_a
```

#### Criar Entidade
```
POST /odata/Users
Content-Type: application/json

{
  "nome": "João Silva",
  "email": "joao@email.com",
  "idade": 30
}
```

#### Atualizar Entidade
```
PUT /odata/Users(1)
Content-Type: application/json

{
  "nome": "João Santos",
  "email": "joao.santos@email.com"
}
```

#### Atualizar Parcialmente
```
PATCH /odata/Users(1)
Content-Type: application/json

{
  "idade": 32
}
```

#### Excluir Entidade
```
DELETE /odata/Users(1)
```

## 🔍 Consultas OData

### Filtros ($filter)
```
GET /odata/Users?$filter=idade gt 25
GET /odata/Users?$filter=nome eq 'João'
GET /odata/Users?$filter=contains(nome, 'Silva')
```

### Filtros com Multi-Tenant
```
GET /odata/Users?$filter=idade gt 25
X-Tenant-ID: empresa_a
```

### Ordenação ($orderby)
```
GET /odata/Users?$orderby=nome asc
GET /odata/Users?$orderby=idade desc
GET /odata/Users?$orderby=nome asc,idade desc
```

### Paginação ($top, $skip)
```
GET /odata/Users?$top=10
GET /odata/Users?$skip=20
GET /odata/Users?$top=10&$skip=20
```

### Seleção de Campos ($select)
```
GET /odata/Users?$select=nome,email
```

### Expansão de Relacionamentos ($expand)
```
GET /odata/Users?$expand=Orders
GET /odata/Users?$expand=Orders($filter=total gt 100)
```

### Contagem ($count)
```
GET /odata/Users?$count=true
GET /odata/Users/$count
```

### Campos Computados ($compute)
```
GET /odata/Orders?$compute=total mul 0.1 as tax
```

### Busca Textual ($search)
```
GET /odata/Users?$search=João
```

## 🔧 Operadores Suportados

### Comparação
- `eq` - Igual
- `ne` - Diferente  
- `gt` - Maior que
- `ge` - Maior ou igual
- `lt` - Menor que
- `le` - Menor ou igual

### Funções de String
- `contains(field, 'value')` - Contém
- `startswith(field, 'value')` - Inicia com
- `endswith(field, 'value')` - Termina com
- `tolower(field)` - Converte para minúsculas
- `toupper(field)` - Converte para maiúsculas

### Funções Matemáticas
- `round(field)` - Arredonda
- `floor(field)` - Arredonda para baixo
- `ceiling(field)` - Arredonda para cima

### Lógicos
- `and` - E lógico
- `or` - Ou lógico
- `not` - Negação

## 📊 Mapeamento de Tipos

| Tipo Go | Tipo OData | Tipo SQL |
|---------|------------|----------|
| `string` | `Edm.String` | `VARCHAR` |
| `int`, `int32` | `Edm.Int32` | `INT` |
| `int64` | `Edm.Int64` | `BIGINT` |
| `float32` | `Edm.Single` | `FLOAT` |
| `float64` | `Edm.Double` | `DOUBLE` |
| `bool` | `Edm.Boolean` | `BOOLEAN` |
| `time.Time` | `Edm.DateTimeOffset` | `TIMESTAMP` |
| `nullable.Int64` | `Edm.Int64` | `BIGINT NULL` |
| `nullable.String` | `Edm.String` | `VARCHAR NULL`
| `nullable.Time` | `Edm.DateTimeOffset` | `TIMESTAMP NULL` |

## 🤝 Contribuindo

Contribuições são bem-vindas! Por favor:

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

### Executando Testes
```bash
go test ./...
```

## 📁 Exemplos

O Go-Data inclui diversos exemplos práticos para demonstrar suas funcionalidades:

### 🏢 [Multi-Tenant](examples/multi_tenant/)
Exemplo completo demonstrando:
- Configuração multi-tenant via .env
- Entidades com isolamento por tenant
- Múltiplos métodos de identificação de tenant
- Endpoints de gerenciamento e monitoramento
- Diferentes tipos de banco por tenant

### 🔐 [JWT Authentication](examples/jwt/)
Demonstra sistema completo de autenticação:
- Configuração JWT com roles e scopes
- Endpoints de login, refresh e logout
- Controle de acesso por entidade
- Middleware de autenticação

### 🎯 [Events](examples/events/)
Sistema completo de eventos:
- Validações customizadas
- Auditoria e logging
- Cancelamento de operações
- Controle de acesso baseado em contexto

### 📊 [Básico](examples/basic/)
Exemplo básico de uso:
- Configuração simples
- Entidades e relacionamentos
- Operações CRUD

### 🚀 [Avançado](examples/advanced/)
Funcionalidades avançadas:
- Configurações personalizadas
- Mapeamento complexo
- Relacionamentos N:N

## 📚 Referências
[![Go Reference](https://pkg.go.dev/badge/github.com/fitlcarlos/go-data.svg)](https://pkg.go.dev/github.com/fitlcarlos/go-data)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 📄 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 📞 Suporte

- [GitHub Issues](https://github.com/fitlcarlos/go-data/issues) - Para bugs e feature requests
- [GitHub Discussions](https://github.com/fitlcarlos/go-data/discussions) - Para perguntas e discussões

---