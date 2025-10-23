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
- [Autenticação Basic](#-autenticação-basic)
- [Segurança](#-segurança)
- [Performance](#-performance)
- [Rate Limiting](#-rate-limiting)
- [Multi-Tenant](#-multi-tenant)
- [Eventos de Entidade](#-eventos-de-entidade)
- [Service Operations](#-service-operations)
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
- Expansão de relacionamentos ($expand) com otimização N+1
- Contagem ($count)
- Campos computados ($compute)
- Busca textual ($search)
- **Batch requests ($batch)**: Múltiplas operações em uma requisição com suporte a transações

### 🔐 **Autenticação**
- **JWT**: Tokens de acesso e refresh, roles, scopes e configuração flexível
- **Basic Auth**: HTTP Basic Authentication com validação customizável
- Interface `AuthProvider` permite implementar qualquer estratégia de autenticação
- Middleware de autenticação obrigatória e opcional
- Controle de acesso baseado em roles e scopes
- Privilégios de administrador
- Configuração de autenticação por entidade
- Entidades somente leitura
    
### ⚡ **Performance**
- **Otimização N+1 para $expand**: Usa batching automático para evitar múltiplas queries
- **String Builder**: Concatenação otimizada em query building
- **Benchmarks completos**: Suite de testes de performance com profiling

### 🛡️ **Rate Limiting**
- Controle de taxa de requisições por IP, usuário ou API key
- Configuração flexível de limites e janelas de tempo
- Headers informativos de rate limit nas respostas
- Estratégias customizáveis de geração de chaves
- Suporte a burst de requisições simultâneas
- Limpeza automática de clientes inativos
- Integração transparente com middleware do servidor

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

### 🔧 **Execução como Serviço (Kardianos)**
- Integração transparente usando biblioteca [kardianos/service](https://github.com/kardianos/service)
- Suporte completo a Windows Service, systemd (Linux) e launchd (macOS)
- Métodos unificados: `Install()`, `Start()`, `Stop()`, `Restart()`, `Status()`, `Uninstall()`
- Detecção automática de contexto de execução (serviço vs. modo normal)
- Shutdown graceful e auto-restart em caso de falha
- Logging integrado com Event Log/journalctl/Console nativo
- Configuração automática por plataforma com dependências específicas

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

# Configurações de Rate Limit
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST_SIZE=20
RATE_LIMIT_WINDOW_SIZE=1m
RATE_LIMIT_HEADERS=true

# Configurações do Serviço
SERVICE_NAME=godata-service
SERVICE_DISPLAY_NAME=GoData OData Service
SERVICE_DESCRIPTION=Serviço GoData OData v4 para APIs RESTful

# Configurações Multi-Tenant
MULTI_TENANT_ENABLED=false
TENANT_IDENTIFICATION_MODE=header
TENANT_HEADER_NAME=X-Tenant-ID
DEFAULT_TENANT=default

# Configurações específicas por tenant (exemplo)
TENANT_EMPRESA_A_DB_DRIVER=postgresql
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

#### Configurações do Serviço
- **SERVICE_NAME**: Nome do serviço (padrão: godata-service)
- **SERVICE_DISPLAY_NAME**: Nome de exibição do serviço (padrão: GoData OData Service)
- **SERVICE_DESCRIPTION**: Descrição do serviço (padrão: Serviço GoData OData v4 para APIs RESTful)

#### Configurações Multi-Tenant
- **MULTI_TENANT_ENABLED**: Habilita suporte multi-tenant (padrão: false)
- **TENANT_IDENTIFICATION_MODE**: Método de identificação do tenant (header, subdomain, path, jwt)
- **TENANT_HEADER_NAME**: Nome do header para identificação (padrão: X-Tenant-ID)
- **DEFAULT_TENANT**: Nome do tenant padrão (padrão: default)
- **TENANT_[NOME]_DB_DRIVER**: Tipo de banco para tenant específico
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
TENANT_EMPRESA_A_DB_DRIVER=postgresql
TENANT_EMPRESA_A_DB_HOST=postgres-a.empresa.com
TENANT_EMPRESA_A_DB_PORT=5432
TENANT_EMPRESA_A_DB_NAME=empresa_a
TENANT_EMPRESA_A_DB_USER=user_a
TENANT_EMPRESA_A_DB_PASSWORD=password_a

TENANT_EMPRESA_B_DB_DRIVER=mysql
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

O Go-Data oferece suporte à autenticação JWT através de um modelo **desacoplado e flexível**. O JWT não está embutido no servidor - você define sua própria lógica de autenticação e configura por entidade usando o padrão **Functional Options**.

### Características

- ✅ **Desacoplado**: JWT como plugin opcional, não embutido
- ✅ **Flexível**: Controle total sobre geração e validação de tokens
- ✅ **Customizável**: Claims, algoritmos e lógica completamente personalizáveis
- ✅ **Por Entidade**: Configure autenticação diferente para cada entidade
- ✅ **Múltiplos JWTs**: Use diferentes JWTs no mesmo servidor

### Interface AuthProvider

O Go-Data define uma interface `AuthProvider` que permite implementar qualquer estratégia de autenticação:

```go
type AuthProvider interface {
    ValidateToken(token string) (*UserIdentity, error)
    GenerateToken(user *UserIdentity) (string, error)
    ExtractToken(c fiber.Ctx) string
}
```

### Uso Básico com JwtAuth

A implementação padrão `JwtAuth` oferece autenticação JWT completa com **configuração automática via .env**:

#### Opção 1: Configuração via .env (Recomendado)

```env
# .env
JWT_SECRET=your-super-secret-key-with-at-least-32-characters
JWT_ISSUER=my-app
JWT_EXPIRATION=3600
JWT_REFRESH_EXPIRATION=86400
JWT_ALGORITHM=HS256
```

```go
import "github.com/fitlcarlos/go-data/pkg/odata"

func main() {
    server := odata.NewServer()
    
    // 1. Criar JwtAuth (lê automaticamente do .env)
    jwtAuth := odata.NewJwtAuth(nil)
    
    // 2. Registrar entidades com WithAuth()
    server.RegisterEntity("Users", User{}, 
        odata.WithAuth(jwtAuth),
    )
    
    server.Start()
}
```

#### Opção 2: Override Parcial

```go
// Usa JWT_SECRET do .env, mas override expiration
jwtAuth := odata.NewJwtAuth(&odata.JWTConfig{
    ExpiresIn: 2 * time.Hour, // Override apenas isso
})
```

#### Opção 3: Configuração Manual Completa

```go
// Configuração completamente manual (ignora .env)
jwtAuth := odata.NewJwtAuth(&odata.JWTConfig{
    SecretKey: "manual-secret-key-min-32-chars",
    Issuer:    "my-app",
    ExpiresIn: 1 * time.Hour,
    RefreshIn: 24 * time.Hour,
    Algorithm: "HS256",
})

server.RegisterEntity("Products", Product{}, 
        odata.WithAuth(jwtAuth),
        odata.WithReadOnly(false),
    )
    
    // 3. Criar suas próprias rotas de autenticação
    router := server.GetRouter()
    
    router.Post("/auth/login", handleLogin(jwtAuth))
    router.Post("/auth/refresh", handleRefresh(jwtAuth))
    router.Get("/auth/me", odata.AuthMiddleware(jwtAuth), handleMe())
    
    server.Start()
}
```

### Interface ContextAuthenticator

A partir da versão mais recente, o Go-Data oferece a interface `ContextAuthenticator` que fornece acesso ao **contexto enriquecido** durante a autenticação, incluindo ObjectManager, Connection, Provider, Pool e informações da requisição (IP, Headers, etc).

#### Benefícios do ContextAuthenticator

- 🔐 **Login com banco de dados**: Validar credenciais diretamente no banco
- 🔄 **Refresh token inteligente**: Recarregar roles/permissions atualizadas
- 📝 **Audit logging**: Registrar IP, device, tentativas de login
- 🚫 **Validação em tempo real**: Verificar se usuário está ativo durante refresh
- 🏢 **Multi-tenant**: Acesso ao pool de conexões e tenant ID

#### Definição da Interface

```go
type ContextAuthenticator interface {
    // AuthenticateWithContext autentica usuário durante login
    // ctx fornece acesso ao banco de dados, IP do cliente, headers, etc
    AuthenticateWithContext(ctx *AuthContext, username, password string) (*UserIdentity, error)
    
    // RefreshToken recarrega/valida dados do usuário durante refresh token
    // Permite validar se usuário ainda está ativo e atualizar roles/permissions
    // O contexto está disponível caso você queira validar no banco de dados
    RefreshToken(ctx *AuthContext, username string) (*UserIdentity, error)
}
```

#### Exemplo Completo

```go
type DatabaseAuthenticator struct{}

// AuthenticateWithContext - Login com validação no banco
func (a *DatabaseAuthenticator) AuthenticateWithContext(ctx *odata.AuthContext, username, password string) (*odata.UserIdentity, error) {
    conn := ctx.GetConnection()
    
    // Buscar usuário no banco
    var dbPassword string
    var userID int64
    var isActive bool
    
    query := "SELECT id, password, is_active FROM users WHERE email = ?"
    err := conn.QueryRow(query, username).Scan(&userID, &dbPassword, &isActive)
    if err != nil {
        log.Printf("❌ Login failed: user not found - %s from IP %s", username, ctx.IP())
        return nil, errors.New("credenciais inválidas")
    }
    
    // Validar senha (use bcrypt em produção!)
    if dbPassword != password {
        log.Printf("❌ Login failed: invalid password - %s from IP %s", username, ctx.IP())
        return nil, errors.New("credenciais inválidas")
    }
    
    if !isActive {
        return nil, errors.New("usuário inativo")
    }
    
    // Audit log
    conn.Exec("INSERT INTO audit_log (user_id, action, ip) VALUES (?, 'login', ?)", userID, ctx.IP())
    
    return &odata.UserIdentity{
        Username: username,
        Roles:    []string{"user"},
        Custom: map[string]interface{}{
            "user_id":  userID,
            "login_ip": ctx.IP(),
        },
    }, nil
}

// RefreshToken - Recarregar dados atualizados do usuário
func (a *DatabaseAuthenticator) RefreshToken(ctx *odata.AuthContext, username string) (*odata.UserIdentity, error) {
    conn := ctx.GetConnection()
    
    // Buscar dados ATUALIZADOS do usuário (roles podem ter mudado!)
    var userID int64
    var isActive bool
    var isAdmin bool
    
    query := "SELECT id, is_active, is_admin FROM users WHERE email = ?"
    err := conn.QueryRow(query, username).Scan(&userID, &isActive, &isAdmin)
    if err != nil || !isActive {
        log.Printf("❌ Refresh failed: user not found or inactive - %s", username)
        return nil, errors.New("usuário não encontrado ou inativo")
    }
    
    // Audit log
    conn.Exec("INSERT INTO audit_log (user_id, action, ip) VALUES (?, 'refresh', ?)", userID, ctx.IP())
    
    roles := []string{"user"}
    if isAdmin {
        roles = append(roles, "admin")
    }
    
    return &odata.UserIdentity{
        Username: username,
        Roles:    roles,
        Admin:    isAdmin,
        Custom: map[string]interface{}{
            "user_id":     userID,
            "refreshed_ip": ctx.IP(),
        },
    }, nil
}

// Configurar no servidor
func main() {
    server := odata.NewServer()
    server.RegisterEntity("Users", User{})
    
    // SetupAuthRoutes usa automaticamente ContextAuthenticator
    authenticator := &DatabaseAuthenticator{}
    server.SetupAuthRoutes(authenticator)
    
    server.Start()
}
```

#### Endpoints Criados Automaticamente

O método `SetupAuthRoutes()` cria automaticamente:

- `POST /auth/login` - Login com AuthenticateWithContext
- `POST /auth/refresh` - Refresh usando RefreshToken
- `POST /auth/logout` - Logout (invalidação de token)
- `GET /auth/me` - Informações do usuário autenticado

### Criando Rotas de Autenticação Manualmente

Se preferir não usar `SetupAuthRoutes()`, você pode criar suas próprias rotas de autenticação com total controle:

```go
func handleLogin(jwtAuth *odata.JwtAuth) fiber.Handler {
    return func(c fiber.Ctx) error {
        var req LoginRequest
        if err := c.Bind().JSON(&req); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Dados inválidos"})
        }
        
        // Validar credenciais (seu código)
        user, err := authenticateUser(req.Username, req.Password)
        if err != nil {
            return c.Status(401).JSON(fiber.Map{"error": "Credenciais inválidas"})
        }
        
        // Gerar tokens
        accessToken, _ := jwtAuth.GenerateToken(user)
        refreshToken, _ := jwtAuth.GenerateRefreshToken(user)
        
        return c.JSON(fiber.Map{
            "access_token":  accessToken,
            "refresh_token": refreshToken,
            "token_type":    "Bearer",
            "expires_in":    int64(jwtAuth.GetConfig().ExpiresIn.Seconds()),
            "user":          user,
        })
    }
}
```

### Customização Avançada

#### Customizar Geração de Tokens

```go
jwtAuth := odata.NewJwtAuth(config)

// Opção 1: Adicionar claims extras e chamar o método padrão
jwtAuth.TokenGenerator = func(user *odata.UserIdentity) (string, error) {
    // Adicionar claims extras
    if user.Custom == nil {
        user.Custom = make(map[string]interface{})
    }
    user.Custom["ip"] = getCurrentIP()
    user.Custom["device"] = getDeviceInfo()
    user.Custom["generated_at"] = time.Now().Unix()
    
    // ✅ Chamar o método padrão (PÚBLICO)
    return jwtAuth.DefaultGenerateToken(user)
}

// Opção 2: Implementação completamente customizada
jwtAuth.TokenGenerator = func(user *odata.UserIdentity) (string, error) {
    // Sua lógica JWT customizada do zero
    token := jwt.NewWithClaims(jwt.SigningMethodHS512, customClaims)
    return token.SignedString([]byte("custom-secret"))
}
```

#### Customizar Validação de Tokens

```go
// Opção 1: Adicionar validações extras e chamar o método padrão
jwtAuth.TokenValidator = func(tokenString string) (*odata.UserIdentity, error) {
    // Verificações extras ANTES da validação padrão
    if isTokenBlacklisted(tokenString) {
        return nil, errors.New("token revogado")
    }
    
    // ✅ Chamar validação padrão (PÚBLICO)
    user, err := jwtAuth.DefaultValidateToken(tokenString)
    if err != nil {
        return nil, err
    }
    
    // Verificações extras DEPOIS da validação
    if !isUserActive(user.Username) {
        return nil, errors.New("usuário inativo")
    }
    
    return user, nil
}

// Opção 2: Implementação completamente customizada
jwtAuth.TokenValidator = func(tokenString string) (*odata.UserIdentity, error) {
    // Parser JWT customizado
    claims, err := parseCustomToken(tokenString)
    if err != nil {
        return nil, err
    }
    
    return &odata.UserIdentity{
        Username: claims.Username,
        Roles:    claims.Roles,
        // ...
    }, nil
}
```

#### Customizar Extração de Tokens

```go
// Opção 1: Tentar múltiplas fontes com fallback para o padrão
jwtAuth.TokenExtractor = func(c fiber.Ctx) string {
    // 1. Tentar cookie primeiro
    if token := c.Cookies("auth_token"); token != "" {
        return token
    }
    
    // 2. Tentar query parameter (não recomendado em produção)
    if token := c.Query("token"); token != "" {
        return token
    }
    
    // 3. ✅ Fallback para extração padrão (Header Authorization: Bearer)
    return jwtAuth.DefaultExtractToken(c)
}

// Opção 2: Implementação completamente customizada
jwtAuth.TokenExtractor = func(c fiber.Ctx) string {
    // Extração customizada (ex: de um header customizado)
    token := c.Get("X-Custom-Auth-Token")
    return strings.TrimPrefix(token, "Token ")
}
```

### Diferentes JWTs para Diferentes Entidades

```go
// JWT para usuários admin
adminAuth := odata.NewJwtAuth(&odata.JWTConfig{
    SecretKey: "admin-secret",
    ExpiresIn: 30 * time.Minute, // Tokens admin expiram mais rápido
})

// JWT para usuários normais
userAuth := odata.NewJwtAuth(&odata.JWTConfig{
    SecretKey: "user-secret",
    ExpiresIn: 2 * time.Hour,
})

// JWT para API keys
apiKeyAuth := odata.NewJwtAuth(&odata.JWTConfig{
    SecretKey: "api-secret",
    ExpiresIn: 365 * 24 * time.Hour, // 1 ano
})

// Aplicar diferentes auths
server.RegisterEntity("Users", User{}, odata.WithAuth(adminAuth))
server.RegisterEntity("Products", Product{}, odata.WithAuth(userAuth))
server.RegisterEntity("Reports", Report{}, odata.WithAuth(apiKeyAuth), odata.WithReadOnly(true))
```

### Implementar AuthProvider Customizado

Você pode implementar sua própria autenticação (OAuth, SAML, etc):

```go
type OAuth2Provider struct {
    clientID     string
    clientSecret string
}

func (o *OAuth2Provider) ValidateToken(token string) (*odata.UserIdentity, error) {
    // Validar com servidor OAuth2
    claims, err := validateOAuth2Token(token, o.clientID, o.clientSecret)
    if err != nil {
        return nil, err
    }
    
    return &odata.UserIdentity{
        Username: claims.Email,
        Roles:    claims.Roles,
        // ...
    }, nil
}

func (o *OAuth2Provider) GenerateToken(user *odata.UserIdentity) (string, error) {
    // OAuth2 não gera tokens diretamente
    return "", errors.New("use OAuth2 authorization flow")
}

func (o *OAuth2Provider) ExtractToken(c fiber.Ctx) string {
    return c.Get("Authorization")
}

// Usar
oauth := &OAuth2Provider{clientID: "...", clientSecret: "..."}
server.RegisterEntity("Users", User{}, odata.WithAuth(oauth))
```

### Estrutura de UserIdentity

```go
type UserIdentity struct {
    Username string                 `json:"username"`
    Roles    []string               `json:"roles"`
    Scopes   []string               `json:"scopes"`
    Admin    bool                   `json:"admin"`
    Custom   map[string]interface{} `json:"custom"` // Claims customizados
}

// Métodos disponíveis
user.HasRole("manager")           // Verifica role específica
user.HasAnyRole("admin", "user")  // Verifica múltiplas roles
user.HasScope("write")            // Verifica scope específico
user.IsAdmin()                    // Verifica se é admin
user.GetCustomClaim("department") // Obtém claim customizado
```

### Middleware de Autenticação

```go
// Middleware obrigatório
router.Get("/protected", odata.AuthMiddleware(jwtAuth), handler)

// Middleware opcional
router.Get("/public", odata.OptionalAuthMiddleware(jwtAuth), handler)

// Verificar usuário no handler
func handler(c fiber.Ctx) error {
    user := odata.GetCurrentUser(c)
    if user == nil {
        return c.Status(401).JSON(fiber.Map{"error": "Não autenticado"})
    }
    
    if !user.HasRole("admin") {
        return c.Status(403).JSON(fiber.Map{"error": "Sem permissão"})
    }
    
    return c.JSON(fiber.Map{"message": "Acesso permitido"})
}
```

### Entity Options

```go
// WithAuth - Configura autenticação
server.RegisterEntity("Users", User{}, odata.WithAuth(jwtAuth))

// WithReadOnly - Entidade somente leitura
server.RegisterEntity("Reports", Report{}, 
    odata.WithAuth(jwtAuth),
    odata.WithReadOnly(true),
)

// Sem autenticação (público)
server.RegisterEntity("PublicData", PublicData{})
```

### Exemplo de Login Completo

```bash
# 1. Fazer login
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password123"
}

# Resposta:
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "username": "admin",
    "roles": ["admin"],
    "admin": true
  }
}

# 2. Acessar endpoint protegido
GET /odata/Users
Authorization: Bearer eyJhbGc...

# 3. Renovar token
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGc..."
}
```

### Exemplos Completos

Veja exemplos completos de autenticação:

- [`examples/jwt/`](examples/jwt/) - JWT desacoplado com múltiplos usuários
- [`examples/jwt_banco/`](examples/jwt_banco/) - JWT com integração de banco de dados
- [`examples/basic_auth/`](examples/basic_auth/) - Basic Auth com validação em banco de dados

### Configuração de Segurança

```go
type JWTConfig struct {
    SecretKey  string        // Chave secreta para assinatura
    Issuer     string        // Emissor do token
    ExpiresIn  time.Duration // Tempo de expiração do access token
    RefreshIn  time.Duration // Tempo de expiração do refresh token
    Algorithm  string        // Algoritmo de assinatura (HS256)
}
```

### Migração do Modelo Antigo

Se você usava o modelo antigo embutido, veja como migrar:

```go
// ANTES (modelo antigo - embutido)
server.SetupAuthRoutes(authenticator)
server.SetEntityAuth("Users", odata.EntityAuthConfig{...})

// DEPOIS (modelo novo - desacoplado)
jwtAuth := odata.NewJwtAuth(config)
server.RegisterEntity("Users", User{}, odata.WithAuth(jwtAuth))
router.Post("/auth/login", handleLogin(jwtAuth))
```

## 🔓 Autenticação Basic

O Go-Data oferece suporte à autenticação Basic (HTTP Basic Authentication) através do mesmo modelo **desacoplado e flexível** do JWT. A autenticação Basic é ideal para APIs internas, scripts, integração entre servidores e ambientes onde simplicidade é preferível.

### Características

- ✅ **Desacoplado**: Implementa a interface `AuthProvider`
- ✅ **Stateless**: Sem necessidade de armazenamento de sessão
- ✅ **Simples**: Credenciais em Base64 no header Authorization
- ✅ **Customizável**: Validação de usuário completamente personalizável
- ✅ **Por Entidade**: Configure autenticação diferente para cada entidade
- ✅ **WWW-Authenticate**: Suporte ao header padrão RFC 7617

### Uso Básico com BasicAuth

A implementação `BasicAuth` oferece autenticação HTTP Basic completa:

```go
import (
    "github.com/fitlcarlos/go-data/pkg/odata"
)

func main() {
    server := odata.NewServer()
    
    // 1. Criar BasicAuth com função de validação
    basicAuth := odata.NewBasicAuth(
        &odata.BasicAuthConfig{
            Realm: "My API", // Nome do realm para o WWW-Authenticate header
        },
        validateUser, // Função que valida username/password
    )
    
    // 2. Registrar entidades com WithAuth()
    server.RegisterEntity("Users", User{}, 
        odata.WithAuth(basicAuth),
    )
    
    server.RegisterEntity("Products", Product{}, 
        odata.WithAuth(basicAuth),
        odata.WithReadOnly(false),
    )
    
    server.Start()
}

// validateUser valida credenciais e retorna UserIdentity
func validateUser(username, password string) (*odata.UserIdentity, error) {
    // Validar contra banco de dados, cache, etc
    user, err := db.GetUserByCredentials(username, password)
    if err != nil {
        return nil, errors.New("credenciais inválidas")
    }
    
    return &odata.UserIdentity{
        ID:       user.ID,
        Username: user.Username,
        Email:    user.Email,
        Role:     user.Role,
        Claims: map[string]interface{}{
            "department": user.Department,
        },
    }, nil
}
```

### Middleware Específico para Basic Auth

O Basic Auth possui um middleware específico que envia o header `WWW-Authenticate`:

```go
router := server.GetRouter()

// Rota protegida com Basic Auth
router.Get("/api/me", odata.BasicAuthMiddleware(basicAuth), func(c fiber.Ctx) error {
    user := odata.GetUserFromContext(c)
    return c.JSON(user)
})

// Também funciona com o middleware genérico
router.Get("/api/info", odata.AuthMiddleware(basicAuth), handler)
```

### Customização da Validação

```go
basicAuth := odata.NewBasicAuth(config, validateUser)

// Adicionar logging e métricas
originalValidator := basicAuth.UserValidator
basicAuth.UserValidator = func(username, password string) (*odata.UserIdentity, error) {
    log.Printf("Tentativa de login: %s", username)
    
    user, err := originalValidator(username, password)
    
    if err != nil {
        log.Printf("Login falhou: %s - %v", username, err)
        metrics.IncrementFailedLogins()
        return nil, err
    }
    
    log.Printf("Login bem-sucedido: %s", username)
    metrics.IncrementSuccessfulLogins()
    return user, nil
}
```

### Customizar Extração de Credenciais

```go
basicAuth := odata.NewBasicAuth(config, validateUser)

// Suportar múltiplas fontes de credenciais
basicAuth.TokenExtractor = func(c fiber.Ctx) string {
    // 1. Tentar header padrão primeiro
    if token := basicAuth.DefaultExtractToken(c); token != "" {
        return token
    }
    
    // 2. Tentar header customizado
    if customAuth := c.Get("X-Custom-Auth"); customAuth != "" {
        // Processar formato customizado
        return extractFromCustomHeader(customAuth)
    }
    
    return ""
}
```

### Usar Basic Auth com Banco de Dados

```go
func validateUser(username, password string) (*odata.UserIdentity, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    var user User
    query := `SELECT id, username, email, role, active 
              FROM users 
              WHERE username = ? AND password = ? AND active = 1`
    
    err := db.QueryRowContext(ctx, query, username, password).Scan(
        &user.ID, &user.Username, &user.Email, &user.Role, &user.Active,
    )
    
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, errors.New("credenciais inválidas")
        }
        return nil, fmt.Errorf("erro ao consultar usuário: %w", err)
    }
    
    return &odata.UserIdentity{
        ID:       fmt.Sprintf("%d", user.ID),
        Username: user.Username,
        Email:    user.Email,
        Role:     user.Role,
    }, nil
}
```

### Diferentes Auths para Diferentes Entidades

```go
// Basic Auth para API interna
internalAuth := odata.NewBasicAuth(
    &odata.BasicAuthConfig{Realm: "Internal API"},
    validateInternalUser,
)

// JWT para API pública
publicAuth := odata.NewJwtAuth(&odata.JWTConfig{
    SecretKey: "public-secret",
})

// Aplicar diferentes auths
server.RegisterEntity("InternalReports", Report{}, odata.WithAuth(internalAuth))
server.RegisterEntity("PublicProducts", Product{}, odata.WithAuth(publicAuth))
```

### Exemplo de Requisição

```bash
# 1. Usando curl com -u (recomendado)
curl -u admin:admin123 http://localhost:3000/api/v1/Users

# 2. Usando header Authorization manual
curl -H "Authorization: Basic YWRtaW46YWRtaW4xMjM=" http://localhost:3000/api/v1/Users

# 3. Gerar Base64 manualmente
echo -n "admin:admin123" | base64
# Resultado: YWRtaW46YWRtaW4xMjM=
```

### Resposta 401 com WWW-Authenticate

Quando credenciais são inválidas ou ausentes, o servidor responde com:

```http
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Basic realm="My API"
Content-Type: application/json

{
  "error": "Autenticação requerida"
}
```

Isso faz com que navegadores modernos exibam um prompt de login automaticamente.

### Exemplo Completo

Veja um exemplo completo com banco de dados em [`examples/basic_auth/`](examples/basic_auth/).

### Quando Usar Basic Auth

✅ **Recomendado para:**
- APIs internas entre servidores
- Scripts e automações
- Ambientes com HTTPS garantido
- Integrações simples
- Prototipagem rápida

⚠️ **Não recomendado para:**
- APIs públicas expostas na internet
- Aplicações web frontend (use JWT)
- Ambientes sem HTTPS (credenciais são enviadas em Base64)
- Quando precisa de logout/expiração (use JWT)

### Segurança

**IMPORTANTE:** Basic Auth **DEVE** ser usado **APENAS com HTTPS/TLS**. As credenciais são enviadas em Base64 (não criptografadas) e podem ser facilmente decodificadas.

```go
// Configure TLS para produção
server := odata.NewServer(&odata.Config{
    TLS: &odata.TLSConfig{
        Enabled:  true,
        CertFile: "/path/to/cert.pem",
        KeyFile:  "/path/to/key.pem",
    },
})
```

### Comparação: Basic Auth vs JWT

| Característica | Basic Auth | JWT |
|---------------|------------|-----|
| Complexidade | Simples | Moderada |
| Stateless | ✅ Sim | ✅ Sim |
| Expiração | ❌ Não | ✅ Sim |
| Revogação | ❌ Difícil | ✅ Possível |
| Performance | ⚡ Rápida | ⚡ Rápida |
| Logout | ❌ Não | ✅ Sim |
| Refresh Token | ❌ Não | ✅ Sim |
| Casos de Uso | APIs internas | APIs públicas |

## 🔒 Segurança

O Go-Data implementa múltiplas camadas de segurança para proteger suas APIs contra ataques e vazamentos de dados.

### Proteção contra SQL Injection

✅ **Implementado automaticamente** - Todas as queries usam **Prepared Statements** com parametrização via `sql.Named`.

```go
// ✅ Seguro - Uso automático de prepared statements
server.RegisterEntity("Users", User{})
// Queries como: $filter=name eq 'value' são automaticamente parametrizadas
```

**Validação de Inputs:**
- Tamanho máximo de queries ($filter, $search, etc)
- Detecção de padrões de SQL injection
- Validação de nomes de propriedades
- Limites de profundidade em $expand

```go
config := &odata.ValidationConfig{
    MaxFilterLength:  5000,  // 5KB
    MaxSearchLength:  1000,  // 1KB
    MaxTopValue:      1000,  // máximo 1000 registros
    MaxExpandDepth:   5,     // máximo 5 níveis
    EnableXSSProtection: true,
}
server.GetConfig().ValidationConfig = config
```

### Security Headers

Headers de segurança são **habilitados por padrão**:

```http
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; ...
Strict-Transport-Security: max-age=31536000
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), ...
```

**Configurações disponíveis:**

```go
// Padrão (Balanceado)
config := odata.DefaultSecurityHeadersConfig()

// Estrito (Máxima Segurança)
config := odata.StrictSecurityHeadersConfig()

// Relaxado (Desenvolvimento)
config := odata.RelaxedSecurityHeadersConfig()

// Desabilitar (não recomendado)
config := odata.DisableSecurityHeaders()
```

### Audit Logging

Sistema completo de auditoria para rastrear todas operações críticas:

```go
config := &odata.AuditLogConfig{
    Enabled:  true,
    LogType:  "file",       // "file", "stdout", "stderr"
    FilePath: "audit.log",
    Format:   "json",       // "json" ou "text"
}
server.GetConfig().AuditLogConfig = config
```

**Operações Auditadas:**
- ✅ CREATE, UPDATE, DELETE (operações de escrita)
- ✅ AUTH_SUCCESS, AUTH_FAILURE (autenticação)
- ✅ UNAUTHORIZED (tentativas de acesso negadas)

**Exemplo de Log Entry:**

```json
{
  "timestamp": "2025-10-18T10:30:45Z",
  "username": "john.doe",
  "ip": "192.168.1.100",
  "method": "POST",
  "path": "/odata/Users",
  "entity_name": "Users",
  "operation": "CREATE",
  "success": true,
  "duration_ms": 45
}
```

**Usando com Autenticação:**

```go
jwtAuth := odata.NewJwtAuth(config)

// Com audit logging automático
router.Get("/protected", 
    odata.AuthMiddlewareWithAudit(jwtAuth, server.GetAuditLogger()),
    handler)
```

### Input Validation

Validação automática de todos os inputs OData:

```go
// Validar filter
err := odata.ValidateFilterQuery("name eq 'john'", config)

// Validar propriedades
err := odata.ValidatePropertyName("username", config)

// Validar $top
err := odata.ValidateTopValue(100, config)

// Validar profundidade de $expand
err := odata.ValidateExpandDepth(expandOptions, 5, 1)

// Sanitizar input (remove XSS)
safe := odata.SanitizeInput(userInput, config)
```

**Padrões Detectados Automaticamente:**
- SQL Injection: `UNION`, `DROP`, `--`, `1=1`, etc
- XSS: `<script>`, `<iframe>`, `javascript:`, `onclick=`, etc
- Caracteres inválidos em nomes de propriedades
- Queries muito longas (DoS prevention)

### Rate Limiting (Habilitado por Padrão)

⚠️ **IMPORTANTE**: Rate limiting está **HABILITADO por padrão** desde a versão atual.

```go
// Configuração padrão (100 req/min)
config := odata.DefaultRateLimitConfig()
// config.Enabled = true (já habilitado)
// config.RequestsPerMinute = 100
// config.BurstSize = 20

// Para desabilitar (não recomendado)
server.GetConfig().RateLimitConfig.Enabled = false
```

### Checklist de Segurança

- [x] **SQL Injection**: Protegido com prepared statements
- [x] **XSS**: Sanitização e CSP headers
- [x] **CSRF**: Headers configuráveis
- [x] **Clickjacking**: X-Frame-Options
- [x] **Rate Limiting**: Habilitado por padrão
- [x] **Audit Logging**: Sistema completo disponível
- [x] **Input Validation**: Múltiplas validações automáticas
- [x] **Security Headers**: 8+ headers implementados
- [ ] **HTTPS/TLS**: Configure manualmente para produção
- [ ] **Secrets Management**: Use variáveis de ambiente

### Documentação de Segurança

Para guia completo de segurança, incluindo melhores práticas e como reportar vulnerabilidades, veja:

📄 **[docs/SECURITY.md](docs/SECURITY.md)**

## ⚡ Performance

O Go-Data implementa múltiplas otimizações de performance para garantir baixa latência e alto throughput.

### Otimização N+1 (Expand Batching)

O problema N+1 ocorre quando expandimos relacionamentos e executamos uma query para cada entidade relacionada. Go-Data resolve isso automaticamente usando **batching**.

**Antes (N+1 Problem)**:
```
GET /odata/Products?$expand=Category

Queries executadas:
1. SELECT * FROM products              -- 1 query inicial
2. SELECT * FROM categories WHERE id=1 -- Para produto 1
3. SELECT * FROM categories WHERE id=1 -- Para produto 2
4. SELECT * FROM categories WHERE id=2 -- Para produto 3
... (N queries, uma por produto)

Total: 1 + N queries = O(N) ❌ LENTO
```

**Depois (Batching)**:
```
GET /odata/Products?$expand=Category

Queries executadas:
1. SELECT * FROM products                     -- 1 query inicial
2. SELECT * FROM categories WHERE id IN (1,2) -- 1 query em batch

Total: 2 queries = O(1) ✅ RÁPIDO (50x mais rápido!)
```

#### Exemplo de Uso

A otimização é **automática e transparente**:

```go
// Registrar entidades normalmente
server.RegisterEntity("Products", Product{})
server.RegisterEntity("Categories", Category{})

// Cliente faz: GET /odata/Products?$expand=Category
// Sistema automaticamente:
// - Detecta expand
// - Coleta todos os CategoryIDs
// - Executa query em batch: WHERE CategoryID IN (1,2,3,...)
// - Associa resultados em memória
// Performance: 2 queries ao invés de N+1! 🚀
```

#### Configuração

Por padrão, batching está **habilitado**. Para debugging ou casos especiais:

```go
config := odata.DefaultServerConfig()
config.DisableJoinForExpand = true  // Força comportamento legado (não recomendado)
server := odata.NewServerWithConfig(config, db)
```

**⚠️ Não recomendado desabilitar**: Pode causar problemas sérios de performance em produção.

#### Logs de Performance

Habilite logs para monitorar otimizações:

```go
config := odata.DefaultServerConfig()
config.LogLevel = "DEBUG"
```

Você verá logs como:
```
🔍 EXPAND: Using BATCHING for Category (evitando N+1)
🔍 EXPAND BATCH: Filter = CategoryID in (1,2,3) (querying 3 related entities)
✅ EXPAND BATCH: Retrieved 3 related entities in 1 query
✅ EXPAND BATCH: Associated related entities to 100 parent entities
```

#### Comparação de Performance

| Cenário | Antes (N+1) | Depois (Batching) | Ganho |
|---------|-------------|-------------------|-------|
| 100 Products + Category | 101 queries (~1010ms) | 2 queries (~20ms) | **50x mais rápido** |
| 1000 Products + Category | 1001 queries (~10s) | 2 queries (~20ms) | **500x mais rápido** |
| Nested expand (2 níveis) | N×M queries | 3 queries | **Drasticamente melhor** |

### String Builder Optimization

Construção otimizada de queries SQL usando `strings.Builder` ao invés de concatenação `+`:

- **12% menos alocações de memória**
- **3-5% mais rápido** em query building
- Especialmente eficiente em queries complexas com múltiplos filtros

### Benchmarks

Execute benchmarks para medir performance:

```bash
# Todos os benchmarks
go test -bench=. -benchmem ./pkg/odata

# Benchmarks específicos
go test -bench=BenchmarkParse -benchmem ./pkg/odata     # Parsers
go test -bench=BenchmarkExpand -benchmem ./pkg/odata    # Expand operations
go test -bench=BenchmarkBuild -benchmem ./pkg/odata     # Query building

# Com profiling (CPU + memória)
PROFILE=1 go test -bench=BenchmarkProfile -cpuprofile=cpu.prof -memprofile=mem.prof ./pkg/odata

# Visualizar profile no navegador
go tool pprof -http=:8080 cpu.prof
```

### Metas de Performance

- ✅ **Parsers**: < 50µs para queries simples
- ✅ **Query Building**: < 100µs para queries completas  
- ✅ **Expand Operations**: < 10ms com batching
- ✅ **N+1 Elimination**: 2 queries ao invés de N+1
- ✅ **Memory**: 10-15% menos alocações

📄 **[pkg/odata/PERFORMANCE.md](pkg/odata/PERFORMANCE.md)** - Documentação completa de performance  
📄 **[pkg/odata/BENCHMARKS.md](pkg/odata/BENCHMARKS.md)** - Guia de benchmarks

## 🛡️ Rate Limiting

O Go-Data implementa um sistema robusto de rate limiting para proteger suas APIs contra abuso e garantir disponibilidade. O sistema oferece controle granular de taxa de requisições com múltiplas estratégias de identificação de clientes.

### Características do Rate Limiting

- **Controle de taxa** por IP, usuário autenticado, API key ou tenant
- **Configuração flexível** de limites e janelas de tempo
- **Headers informativos** nas respostas HTTP
- **Estratégias customizáveis** de geração de chaves
- **Suporte a burst** de requisições simultâneas
- **Limpeza automática** de clientes inativos
- **Integração transparente** com middleware do servidor

### Configuração via .env

```env
# Habilitar rate limiting
RATE_LIMIT_ENABLED=true

# 100 requisições por minuto por cliente
RATE_LIMIT_REQUESTS_PER_MINUTE=100

# Permite burst de até 20 requisições simultâneas
RATE_LIMIT_BURST_SIZE=20

# Janela de tempo para contagem (1 minuto)
RATE_LIMIT_WINDOW_SIZE=1m

# Incluir headers de rate limit na resposta
RATE_LIMIT_HEADERS=true
```

### Configuração Programática

```go
import "github.com/fitlcarlos/go-data/pkg/odata"

// Configuração básica de rate limit
rateLimitConfig := &odata.RateLimitConfig{
    Enabled:           true,
    RequestsPerMinute: 100,
    BurstSize:         20,
    WindowSize:        time.Minute,
    KeyGenerator:      odata.defaultKeyGenerator, // Por IP
    Headers:           true,
}

// Configurar servidor com rate limit
config := odata.DefaultServerConfig()
config.RateLimitConfig = rateLimitConfig

server := odata.NewServerWithConfig(provider, config)
```

### Estratégias de Rate Limiting

#### 1. Por IP (Padrão)

```go
// Limita por endereço IP do cliente
rateLimitConfig.KeyGenerator = odata.defaultKeyGenerator
```

#### 2. Por Usuário Autenticado

```go
// Limita por usuário autenticado (JWT)
rateLimitConfig.KeyGenerator = odata.UserBasedKeyGenerator
```

#### 3. Por API Key

```go
// Limita por chave de API
rateLimitConfig.KeyGenerator = odata.APIKeyBasedKeyGenerator
```

#### 4. Por Tenant (Multi-Tenant)

```go
// Limita por tenant em ambiente multi-tenant
rateLimitConfig.KeyGenerator = odata.TenantBasedKeyGenerator
```

#### 5. Estratégia Customizada

```go
// Implementar estratégia personalizada
rateLimitConfig.KeyGenerator = func(c fiber.Ctx) string {
    // Sua lógica customizada
    userID := c.Locals("user_id")
    ip := c.IP()
    return fmt.Sprintf("custom:%v:%s", userID, ip)
}
```

### Headers de Resposta

Quando habilitado, o sistema inclui headers informativos:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642678800
X-RateLimit-Retry-After: 30 (apenas quando bloqueado)
```

### Resposta de Rate Limit Excedido

Quando o limite é excedido, o servidor retorna HTTP 429:

```json
{
  "error": {
    "code": "RateLimitExceeded",
    "message": "Rate limit exceeded. Try again in 30 seconds.",
    "target": "rate_limit"
  }
}
```

### Configuração Avançada

```go
// Configuração avançada com múltiplas estratégias
rateLimitConfig := &odata.RateLimitConfig{
    Enabled:           true,
    RequestsPerMinute: 200,
    BurstSize:         50,
    WindowSize:        2 * time.Minute,
    KeyGenerator:      odata.UserBasedKeyGenerator,
    SkipSuccessful:    false, // Contar requisições bem-sucedidas
    SkipFailed:        false, // Contar requisições com falha
    Headers:           true,
}

// Aplicar configuração em runtime
server.SetRateLimitConfig(rateLimitConfig)
```

### Monitoramento e Métricas

```go
// Obter configuração atual
currentConfig := server.GetRateLimitConfig()
if currentConfig != nil {
    log.Printf("Rate limit ativo: %d req/min", 
        currentConfig.RequestsPerMinute)
}
```

### Exemplo Prático

```go
package main

import (
    "log"
    "time"
    
    "github.com/fitlcarlos/go-data/pkg/odata"
    _ "github.com/fitlcarlos/go-data/pkg/providers"
)

func main() {
    // Configurar rate limit
    rateLimitConfig := &odata.RateLimitConfig{
        Enabled:           true,
        RequestsPerMinute: 60,  // 1 requisição por segundo
        BurstSize:         10,  // Permite 10 requisições simultâneas
        WindowSize:        time.Minute,
        KeyGenerator:      odata.defaultKeyGenerator,
        Headers:           true,
    }
    
    // Configurar servidor
    config := odata.DefaultServerConfig()
    config.RateLimitConfig = rateLimitConfig
    
    server := odata.NewServerWithConfig(nil, config)
    
    // Registrar entidades
    server.RegisterEntity("Users", User{})
    
    // Iniciar servidor
    if err := server.Start(); err != nil {
        log.Fatalf("Erro ao iniciar servidor: %v", err)
    }
}
```

### Boas Práticas

1. **Configure limites apropriados** baseados na capacidade do seu sistema
2. **Use burst size** para permitir picos de tráfego legítimos
3. **Monitore headers** para ajustar limites conforme necessário
4. **Implemente estratégias diferentes** para diferentes tipos de clientes
5. **Teste em ambiente de produção** para validar configurações

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
TENANT_EMPRESA_A_DB_DRIVER=oracle
TENANT_EMPRESA_A_DB_HOST=oracle1.empresa.com
TENANT_EMPRESA_A_DB_PORT=1521
TENANT_EMPRESA_A_DB_NAME=EMPRESA_A
TENANT_EMPRESA_A_DB_USER=user_a
TENANT_EMPRESA_A_DB_PASSWORD=password_a

TENANT_EMPRESA_B_DB_DRIVER=postgres
TENANT_EMPRESA_B_DB_HOST=postgres1.empresa.com
TENANT_EMPRESA_B_DB_PORT=5432
TENANT_EMPRESA_B_DB_NAME=empresa_b
TENANT_EMPRESA_B_DB_USER=user_b
TENANT_EMPRESA_B_DB_PASSWORD=password_b

TENANT_EMPRESA_C_DB_DRIVER=mysql
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
TENANT_NOVO_CLIENTE_DB_DRIVER=mysql
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

## 🎯 Service Operations

O Go-Data implementa Service Operations similares ao XData, mas usando padrões idiomáticos do Go. O sistema oferece um `ServiceContext` otimizado que equivale funcionalmente ao `TXDataOperationContext` do XData.

### Características do Service Operations

- ✅ **ServiceContext Otimizado**: Equivale ao `TXDataOperationContext.Current.GetManager()` do XData
- ✅ **Sintaxe Simples**: Similar ao Fiber para registro de handlers
- ✅ **Autenticação Flexível**: Controle automático baseado na configuração JWT
- ✅ **Multi-Tenant**: Suporte automático a multi-tenant
- ✅ **ObjectManager Integrado**: Acesso direto ao ObjectManager do contexto
- ✅ **Menos Boilerplate**: 95% menos código que implementações tradicionais

### ServiceContext

```go
type ServiceContext struct {
    Manager      *ObjectManager  // Equivale ao TXDataOperationContext.Current.GetManager()
    FiberContext fiber.Ctx       // Contexto do Fiber (já tem TenantID via GetCurrentTenant())
    User         *UserIdentity   // Usuário autenticado (só se JWT habilitado)
}
```

### Registro de Services

#### Service Sem Autenticação

```go
server.Service("GET", "/Service/GetTopSelling", func(ctx *odata.ServiceContext) error {
    products, err := ctx.GetManager().Query("Products").
        Where("sales_count gt 100").
        OrderBy("sales_count desc").
        Top(10).
        List()
    
    if err != nil {
        return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
    }
    
    return ctx.JSON(map[string]interface{}{
        "products": products,
        "tenant": ctx.GetTenantID(),
    })
})
```

#### Service Com Autenticação

```go
server.ServiceWithAuth("POST", "/Service/CalculateTotal", func(ctx *odata.ServiceContext) error {
    // ctx.User garantidamente não será nil se JWT habilitado
    productIDs := ctx.Query("product_ids")
    
    manager := ctx.GetManager()
    // ... lógica do service
    
    return ctx.JSON(result)
}, true)
```

#### Service Com Roles

```go
server.ServiceWithRoles("GET", "/Service/AdminData", func(ctx *odata.ServiceContext) error {
    // ctx.User garantidamente tem role "admin"
    manager := ctx.GetManager()
    // ... lógica administrativa
    
    return ctx.JSON(data)
}, "admin")
```

#### Service Groups

```go
products := server.ServiceGroup("Products")

products.ServiceWithAuth("GET", "GetTopSelling", func(ctx *odata.ServiceContext) error {
    // Handler implementation
    return ctx.JSON(result)
}, true)

products.ServiceWithRoles("GET", "AdminStats", func(ctx *odata.ServiceContext) error {
    // Handler implementation
    return ctx.JSON(result)
}, "admin")
```

### Métodos do ServiceContext

```go
// Acesso ao ObjectManager (equivale ao XData)
manager := ctx.GetManager()

// Informações do usuário
user := ctx.GetUser()
tenantID := ctx.GetTenantID()

// Verificações de autenticação
isAuth := ctx.IsAuthenticated()
isAdmin := ctx.IsAdmin()
hasRole := ctx.HasRole("manager")

// Acesso aos dados da requisição
params := ctx.QueryParams()
productID := ctx.Query("product_id")
body := ctx.Body()

// Resposta
ctx.JSON(data)
ctx.Status(200).JSON(data)
ctx.SetHeader("Content-Type", "application/json")
```

### Comparação com XData

| Funcionalidade XData | Go-Data ServiceContext |
|---------------------|----------------------|
| `TXDataOperationContext.Current.GetManager()` | `ctx.GetManager()` |
| `TXDataOperationContext.Current.Request` | `ctx.FiberContext` |
| `TXDataOperationContext.Current.Response` | `ctx.FiberContext` |
| Service Contract Interface | `ServiceHandler` function |
| Service Implementation | Handler function direta |
| Routing automático | `server.Service(method, endpoint, handler)` |
| Memory management | `ObjectManager` automático |
| ~20 linhas de setup | ~3 linhas de setup |

### Exemplo Completo

Veja o exemplo completo em [`examples/service_operations/`](examples/service_operations/) que demonstra:

- ServiceContext otimizado com ObjectManager integrado
- Acesso direto a Connection, Provider e Pool
- Criação de múltiplos ObjectManagers isolados
- Sintaxe simples similar ao Fiber para registro
- Controle automático de autenticação baseado em JWT
- Suporte completo a multi-tenant
- Service Groups para organização
- Equivalência funcional ao TXDataOperationContext do XData

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

### Batch ($batch) - OData v4
O OData v4 suporta **batch requests**, permitindo executar múltiplas operações em uma única requisição HTTP. Isso reduz latência, suporta transações e melhora a performance em operações bulk.

**Características:**
- Múltiplas operações GET/POST/PUT/PATCH/DELETE em uma requisição
- Changesets transacionais (tudo ou nada)
- Reduz overhead de conexões HTTP
- Suporte a Content-ID para referenciar operações

**Exemplo: Múltiplas leituras**
```bash
POST /odata/$batch
Content-Type: multipart/mixed; boundary=batch_boundary

--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products?$top=5 HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Categories HTTP/1.1
Host: localhost:3000


--batch_boundary--
```

**Exemplo: Changeset transacional**
```bash
POST /odata/$batch
Content-Type: multipart/mixed; boundary=batch_boundary

--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Products HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{"name":"Produto Novo","price":99.90}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /api/v1/Orders HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{"product_id": 1, "quantity": 5}

--changeset_boundary--

--batch_boundary--
```

**Exemplo: Batch misto (leitura + changeset)**
```bash
POST /odata/$batch
Content-Type: multipart/mixed; boundary=batch_boundary

--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products?$top=3 HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Categories HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{"name":"Nova Categoria"}

--changeset_boundary--

--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Orders HTTP/1.1
Host: localhost:3000


--batch_boundary--
```

**Configuração do Batch:**
```go
// Usar configuração padrão (automática)
server := odata.NewServer()

// Ou customizar
config := &odata.BatchConfig{
    MaxOperations:      100,          // Máximo de operações por batch
    MaxChangesets:      10,           // Máximo de changesets
    Timeout:            30 * time.Second,
    EnableTransactions: true,         // Habilitar transações para changesets
}
```

**Benefícios:**
- ⚡ **Performance**: Reduz latência ao combinar múltiplas requisições
- 🔄 **Transações**: Changesets garantem atomicidade (tudo ou nada)
- 🌐 **Rede**: Menos overhead de conexões HTTP
- 📊 **Bulk**: Ideal para operações em lote

**Limitações Conhecidas:**

⚠️ **Importante**: A implementação atual do $batch possui as seguintes limitações:

1. **Transações por Changeset**:
   - Cada changeset é executado em uma transação separada
   - Não há transação global para múltiplos changesets em um único batch
   - Se você precisa de atomicidade entre changesets, use apenas um changeset

2. **Content-ID**:
   - Content-IDs são resolvidos apenas dentro do mesmo changeset
   - Referências entre changesets diferentes não são suportadas
   - Recomendação: Use Content-IDs sequenciais (1, 2, 3...) para melhor compatibilidade

3. **Autenticação**:
   - A autenticação é aplicada uma vez no batch request
   - Todas as operações no batch usam as mesmas credenciais
   - Não é possível usar credenciais diferentes para operações individuais

4. **Limites de Performance**:
   - `MaxOperations`: Máximo de 100 operações por batch (configurável)
   - `MaxChangesets`: Máximo de 10 changesets por batch (configurável)
   - `Timeout`: 30 segundos por padrão (configurável)
   - Batches muito grandes podem causar timeouts

5. **Tipos de Operações**:
   - ✅ GET, POST, PUT, PATCH, DELETE suportados
   - ❌ $batch aninhado não suportado (batch dentro de batch)
   - ❌ Operações assíncronas não implementadas

6. **Tratamento de Erros**:
   - Em changesets: um erro cancela todas as operações do changeset (rollback)
   - Fora de changesets: cada operação é independente (erros não afetam outras operações)
   - Erros são retornados com status HTTP apropriado na resposta multipart

7. **Formato de Resposta**:
   - Sempre retorna `multipart/mixed` conforme OData v4
   - A ordem das respostas corresponde à ordem das requisições
   - Cada resposta inclui status HTTP e corpo (se aplicável)

8. **Compatibilidade**:
   - Implementado conforme OData v4 specification
   - Testado com: Postman, curl, e clientes HTTP padrão
   - Algumas ferramentas podem ter dificuldade com multipart/mixed complexo

**Recomendações de Uso:**

```go
// ✅ BOM: Um changeset transacional
Changeset 1: [POST Product, POST Order, PUT Inventory]

// ✅ BOM: Múltiplas leituras independentes
Request 1: GET /Products
Request 2: GET /Categories
Request 3: GET /Orders

// ⚠️ CUIDADO: Múltiplos changesets (não há transação global)
Changeset 1: [POST Product]
Changeset 2: [POST Order]  // Se falhar, Changeset 1 já foi commitado

// ❌ EVITAR: Batch muito grande
100+ operações em um único batch // Pode causar timeout
```

**Roadmap Futuro:**
- [ ] Transações globais entre changesets
- [ ] Content-ID cross-changeset
- [ ] Operações assíncronas
- [ ] Streaming de respostas
- [ ] Batch aninhado

Veja o exemplo completo em [`examples/batch/main.go`](examples/batch/main.go).

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

## 🔧 Execução como Serviço

O GoData possui funcionalidade de serviço **integrada transparentemente** usando a biblioteca [kardianos/service](https://github.com/kardianos/service), permitindo execução como serviço nativo no Windows, Linux e macOS sem necessidade de executáveis separados.

### 🎯 Biblioteca Kardianos Service

O GoData utiliza a biblioteca `github.com/kardianos/service` que oferece:

- **Multi-plataforma**: Windows Service, systemd (Linux), launchd (macOS)
- **Interface unificada**: Mesma API para todas as plataformas
- **Logging integrado**: Logs direcionados para Event Log/journalctl/Console
- **Configuração automática**: Dependências e configurações específicas por plataforma
- **Controle de ciclo de vida**: Install, start, stop, restart, uninstall

### 🚀 Como Usar

A funcionalidade de serviço está disponível através de métodos do próprio servidor GoData:

```go
package main

import (
    "log"
    "github.com/fitlcarlos/go-data/pkg/odata"
)

func main() {
    // Criar servidor (carrega automaticamente configurações do .env)
    server := odata.NewServer()
    
    // Registrar entidades
    server.RegisterEntity("Users", User{})
    
    // Instalar como serviço
    if err := server.Install(); err != nil {
        log.Fatal("Erro ao instalar:", err)
    }
    
    // Iniciar serviço  
    if err := server.Start(); err != nil {
        log.Fatal("Erro ao iniciar:", err)
    }
}
```

### 📋 Métodos Disponíveis

```go
// Gerenciamento de serviço (kardianos/service)
server.Install() error           // Instala como serviço do sistema
server.Uninstall() error         // Remove o serviço
server.Start() error             // Inicia (detecta automaticamente se é serviço ou normal)
server.Stop() error              // Para o serviço gracefully
server.Restart() error           // Reinicia o serviço
server.Status() (service.Status, error) // Verifica status do serviço

// Métodos auxiliares
server.IsRunningAsService() bool  // Detecta se está executando como serviço
server.Shutdown() error          // Para apenas o servidor HTTP
```

### 🔍 Detecção Automática de Serviço

O método `Start()` detecta automaticamente se deve executar como serviço através de:

1. **Argumentos de linha de comando**:
   ```bash
   ./app run          # Força execução como serviço
   ./app --service    # Força execução como serviço  
   ./app -service     # Força execução como serviço
   ```

2. **Variável de ambiente**:
   ```bash
   export GODATA_RUN_AS_SERVICE=true
   ./app
   ```

3. **Contexto do sistema**:
   - **Windows**: Detecta execução pelo SCM (Service Control Manager)
   - **Linux**: Detecta `INVOCATION_ID` (systemd) ou `PPID=1`
   - **macOS**: Detecta contexto de execução do launchd

### ⚙️ Configuração do Serviço

```go
// Configuração automática via .env
server := odata.NewServer()

// As configurações do serviço são carregadas automaticamente do .env:
// SERVICE_NAME=godata-prod
// SERVICE_DISPLAY_NAME=GoData Production  
// SERVICE_DESCRIPTION=Servidor GoData OData
// SERVER_HOST=0.0.0.0
// SERVER_PORT=8080

// Instalar e iniciar
server.Install()
server.Start()
```

### 🔧 Sobrescrevendo Configurações (Opcional)

Se necessário, ainda é possível sobrescrever as configurações carregadas do .env:

```go
server := odata.NewServer()

// Sobrescrever apenas se necessário
config := server.GetConfig()
config.Name = "godata-customizado"
config.DisplayName = "GoData Personalizado"
config.Description = "Configuração personalizada"

server.Install()
server.Start()
```

### 🏗️ Configurações Automáticas por Plataforma (Kardianos)

O GoData configura automaticamente o serviço com otimizações específicas para cada plataforma:

#### Windows Service
```
StartType: Automatic
Dependencies: Tcpip, Dhcp
OnFailure: Restart
OnFailureDelayDuration: 5s
OnFailureResetPeriod: 10
```

#### Linux systemd
```
[Unit]
Requires=network.target
After=network-online.target syslog.target

[Service]
Type=notify
Restart=always
RestartSec=5
User=godata
Group=godata
LimitNOFILE=65536
KillMode=mixed
TimeoutStopSec=30
```

#### macOS launchd
Configuração automática com propriedades adequadas para execução em background.

### 🎯 Exemplo Prático

Veja o exemplo completo em [`examples/service/`](examples/service/) que demonstra:

- Como usar os métodos de serviço integrados
- Configuração personalizada de serviço
- Gerenciamento via linha de comando
- Entidades de exemplo (Users e Products)

### 📊 Monitoramento e Logs (Kardianos)

O kardianos/service integra automaticamente com os sistemas de log nativos:

#### Linux (systemd + journalctl)
```bash
# Status detalhado (use o nome configurado no server.config.Name)
sudo systemctl status meu-godata-service

# Logs em tempo real (integrados via kardianos)
sudo journalctl -u meu-godata-service -f

# Logs específicos do GoData
sudo journalctl -u meu-godata-service --since "1 hour ago"
```

#### Windows (Event Log)
```cmd
# Gerenciador de Serviços (procurar pelo DisplayName)
services.msc

# PowerShell (usar o Name configurado)
Get-Service meu-godata-service

# Event Viewer - logs integrados via kardianos
eventvwr.msc
# Navegar: Windows Logs > Application > Source = "meu-godata-service"
```

#### macOS (Console)
```bash
# Console.app para logs do sistema
# ou via linha de comando:
log stream --predicate 'subsystem == "meu-godata-service"'
```

### 🔒 Configuração de Produção

```env
# Arquivo .env para produção
SERVICE_NAME=godata-prod
SERVICE_DISPLAY_NAME=GoData Production Service
SERVICE_DESCRIPTION=Servidor GoData OData v4 - Produção

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_ENABLE_CORS=true
SERVER_ALLOWED_ORIGINS=https://meuapp.com
SERVER_TLS_CERT_FILE=/etc/ssl/certs/server.crt
SERVER_TLS_KEY_FILE=/etc/ssl/private/server.key

JWT_ENABLED=true
JWT_REQUIRE_AUTH=true
JWT_SECRET_KEY=minha-chave-super-secreta-de-producao
```

```go
// Configuração para produção com kardianos/service
server := odata.NewServer()  // Carrega automaticamente do .env

// Instalar e configurar o serviço
log.Fatal(server.Install())  // Instala via kardianos
log.Fatal(server.Start())    // Inicia com detecção automática
```

### 📚 Integração com CI/CD

#### Script de Deploy Automatizado

```bash
#!/bin/bash
# deploy-godata.sh

set -e

# Configurações
SERVICE_NAME="godata"
INSTALL_DIR="/opt/godata"

echo "🚀 Iniciando deploy do GoData Service..."

# Parar serviço se estiver rodando
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "⏹️ Parando serviço..."
    sudo systemctl stop $SERVICE_NAME
fi

# Fazer backup do executável atual
if [ -f "$INSTALL_DIR/godata" ]; then
    sudo cp "$INSTALL_DIR/godata" "$INSTALL_DIR/godata.backup"
fi

# Copiar novo executável
sudo cp ./godata $INSTALL_DIR/
sudo chown godata:godata $INSTALL_DIR/godata
sudo chmod +x $INSTALL_DIR/godata

# Instalar/atualizar serviço
sudo $INSTALL_DIR/godata install

# Iniciar serviço
sudo systemctl start $SERVICE_NAME
sudo systemctl enable $SERVICE_NAME

# Verificar status
sleep 2
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "✅ Deploy concluído com sucesso!"
    sudo systemctl status $SERVICE_NAME
else
    echo "❌ Erro no deploy!"
    exit 1
fi
```

#### GitHub Actions Workflow

```yaml
name: Deploy GoData Service

on:
  push:
    tags: ['v*']

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Build Service
      run: make build-all
    
    - name: Deploy to Production
      run: |
        # Copiar binário para servidor
        scp build/godata-linux-amd64 user@server:/tmp/godata
        
        # Executar deploy no servidor
        ssh user@server 'sudo /tmp/deploy-godata.sh'
```

Para um exemplo completo de uso, consulte: [`examples/service/`](examples/service/)

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
- Arquivo .env completo com configurações multi-tenant

### 🔐 [JWT Authentication](examples/jwt/)
Demonstra sistema completo de autenticação JWT:
- Configuração JWT com roles e scopes
- Endpoints de login, refresh e logout
- Controle de acesso por entidade
- Middleware de autenticação
- Arquivo .env com JWT habilitado

### 🔓 [Basic Authentication](examples/basic_auth/)
Demonstra autenticação HTTP Basic:
- Configuração Basic Auth com validação em banco de dados
- Customização de UserValidator com logging
- Entidades protegidas por autenticação
- WWW-Authenticate header automático
- Múltiplos usuários de teste com roles

### 🎯 [Events](examples/events/)
Sistema completo de eventos:
- Validações customizadas
- Auditoria e logging
- Cancelamento de operações
- Controle de acesso baseado em contexto
- Arquivo .env com configurações para eventos

### 🔧 [Service](examples/service/)
Execução como serviço do sistema:
- Funcionalidade kardianos/service integrada
- Gerenciamento multi-plataforma (Windows/Linux/macOS)
- Detecção automática de contexto de execução
- Configuração de serviço personalizada
- Logging integrado com sistemas nativos
- Arquivo .env completo com configurações de serviço

### 🎯 [Service Operations](examples/service_operations/)
Sistema de Service Operations equivalente ao XData:
- ServiceContext otimizado com ObjectManager integrado
- Sintaxe simples similar ao Fiber para registro
- Controle automático de autenticação baseado em JWT
- Suporte completo a multi-tenant
- Service Groups para organização
- Equivalência funcional ao TXDataOperationContext do XData
- Arquivo .env com configurações JWT e multi-tenant

### 📊 [Básico](examples/basic/)
Exemplo básico de uso:
- Configuração simples
- Entidades e relacionamentos
- Operações CRUD
- Arquivo .env com configurações básicas

### 🚀 [Avançado](examples/advanced/)
Funcionalidades avançadas:
- Configurações personalizadas
- Mapeamento complexo
- Relacionamentos N:N
- Arquivo .env com configurações de produção

## 📚 Referências
[![Go Reference](https://pkg.go.dev/badge/github.com/fitlcarlos/go-data.svg)](https://pkg.go.dev/github.com/fitlcarlos/go-data)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 📄 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 📞 Suporte

- [GitHub Issues](https://github.com/fitlcarlos/go-data/issues) - Para bugs e feature requests
- [GitHub Discussions](https://github.com/fitlcarlos/go-data/discussions) - Para perguntas e discussões

---