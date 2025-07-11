# GoData - Biblioteca OData para Go

Uma biblioteca Go para implementar APIs OData v4 com resposta JSON, servidor Fiber v3 embutido e suporte a múltiplos bancos de dados.

## 📋 Índice

- [Características](#-características)
- [Instalação](#-instalação)
- [Exemplo de Uso](#-exemplo-de-uso)
- [Configuração do Servidor](#-configuração-do-servidor)
- [Autenticação JWT](#-autenticação-jwt)
- [Mapeamento de Entidades](#-mapeamento-de-entidades)
- [Bancos de Dados Suportados](#-bancos-de-dados-suportados)
- [Endpoints OData](#-endpoints-odata)
- [Consultas OData](#-consultas-odata)
- [Operadores Suportados](#-operadores-suportados)
- [Mapeamento de Tipos](#-mapeamento-de-tipos)
- [Contribuindo](#-contribuindo)
- [Licença](#-licença)

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

## 🚀 Instalação

```bash
go get github.com/fitlcarlos/godata
```

## 📝 Exemplo de Uso

### Servidor Básico

```go
package main

import (
    "database/sql"
    "log"
    
    "github.com/fitlcarlos/godata/pkg/odata"
    "github.com/fitlcarlos/godata/pkg/providers"
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
    
    // Cria servidor
    server := odata.NewServer(provider, "localhost", 8080, "/odata")
    
    // Registra entidades
    entities := map[string]interface{}{
        "Users":    User{},
        "Products": Product{},
    }
    
    if err := server.AutoRegisterEntities(entities); err != nil {
        log.Fatal(err)
    }
    
    // Inicia servidor
    log.Println("Servidor iniciado em http://localhost:8080/odata")
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
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

O GoData oferece suporte completo à autenticação JWT com controle de acesso granular baseado em roles e scopes.

### Configuração Básica

```go
import "github.com/fitlcarlos/godata/pkg/odata"

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
import "github.com/fitlcarlos/godata/pkg/nullable"

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
    "github.com/fitlcarlos/godata/pkg/providers"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, err := sql.Open("pgx", "postgres://user:password@localhost/database")
provider := providers.NewPostgreSQLProvider(db)
```

### Oracle
```go
import (
    "github.com/fitlcarlos/godata/pkg/providers"
    _ "github.com/sijms/go-ora/v2"
)

db, err := sql.Open("oracle", "oracle://user:password@localhost:1521/xe")
provider := providers.NewOracleProvider(db)
```

### MySQL
```go
import (
    "github.com/fitlcarlos/godata/pkg/providers"
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
| `nullable.String` | `Edm.String` | `VARCHAR NULL` |
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

## 📄 Licença

Este projeto está licenciado sob a Licença MIT - veja o arquivo [LICENSE](LICENSE) para detalhes.

## 📞 Suporte

- [GitHub Issues](https://github.com/fitlcarlos/godata/issues) - Para bugs e feature requests
- [GitHub Discussions](https://github.com/fitlcarlos/godata/discussions) - Para perguntas e discussões

---

<div align="center">
  <strong>GoData - Biblioteca OData para Go</strong>
</div> 