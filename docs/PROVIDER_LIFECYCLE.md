# 🔄 Ciclo de Vida do Provider - Do Início ao EntityService

## 📋 Índice

- [Visão Geral](#visão-geral)
- [Fluxo Completo de Inicialização](#fluxo-completo-de-inicialização)
- [Detalhamento por Etapa](#detalhamento-por-etapa)
- [Exemplo Prático](#exemplo-prático)
- [Modo Multi-Tenant](#modo-multi-tenant)
- [Diagrama Visual](#diagrama-visual)

---

## 🎯 Visão Geral

O `DatabaseProvider` é criado **UMA VEZ** durante a inicialização da aplicação e é **compartilhado** por todos os `EntityService` registrados no servidor.

**Princípio fundamental:** 
- **1 Provider** = **1 Pool de Conexões** = **Compartilhado por TODAS as entidades**

---

## 🔄 Fluxo Completo de Inicialização

### **📍 Passo a Passo**

```
┌─────────────────────────────────────────────────────────────────┐
│                  INICIALIZAÇÃO DA APLICAÇÃO                      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 1️⃣  main.go: server := odata.NewServer()                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 2️⃣  server.go:51 - NewServer()                                  │
│    • Carrega configurações do .env                              │
│    • Chama: LoadMultiTenantConfig()                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 3️⃣  server.go:62 - CreateProviderFromConfig()                   │
│    • Cria chave de cache baseada na connection string           │
│    • Verifica se provider já existe no cache                    │
│    • Se NÃO existe: cria novo provider                          │
│    • Se existe: reusa do cache (singleton)                      │
│                                                                  │
│    📦 RESULTADO: provider := NewPostgreSQLProvider()            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 4️⃣  provider_postgresql.go:19 - NewPostgreSQLProvider()         │
│    • Carrega configurações do .env                              │
│    • Cria conexão: sql.Open("pgx", connectionString)            │
│    • Configura pool:                                            │
│      - db.SetMaxOpenConns()                                     │
│      - db.SetMaxIdleConns()                                     │
│      - db.SetConnMaxLifetime()                                  │
│      - db.SetConnMaxIdleTime()                                  │
│    • Testa conexão: db.Ping()                                   │
│                                                                  │
│    📦 RESULTADO: provider com *sql.DB configurado               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 5️⃣  server.go:66 - newServerWithConfig(provider, config)        │
│    • Cria estrutura do Server                                   │
│    • Atribui provider ao servidor (linha 148):                  │
│      server := &Server{                                         │
│          provider: provider,  ← AQUI O PROVIDER É ATRIBUÍDO     │
│          ...                                                     │
│      }                                                           │
│                                                                  │
│    📦 RESULTADO: Server com provider configurado                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 6️⃣  main.go: server.RegisterEntity("Users", User{})             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 7️⃣  server.go:253 - NewBaseEntityService(s.provider, ...)       │
│    • Recebe o provider do servidor                              │
│    • Cria BaseEntityService:                                    │
│      service := &BaseEntityService{                             │
│          provider: provider,  ← PROVIDER DO SERVIDOR            │
│          metadata: metadata,                                     │
│          server: server,                                         │
│      }                                                           │
│                                                                  │
│    📦 RESULTADO: EntityService com provider compartilhado       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ 8️⃣  Durante REQUISIÇÕES HTTP                                    │
│    • Cliente faz GET /api/v1/Users                              │
│    • EntityService.Query() usa s.provider.GetConnection()       │
│    • Obtém conexão do pool compartilhado                        │
│    • Executa query                                              │
│    • Devolve conexão ao pool                                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📚 Detalhamento por Etapa

### **1️⃣ Inicialização: `NewServer()`**

**Arquivo:** `odata/server.go` (linha 51)

```go
func NewServer() *Server {
    // Carrega configurações multi-tenant automaticamente
    multiTenantConfig := LoadMultiTenantConfig()
    
    // Se não está em modo multi-tenant, usa o comportamento original
    if multiTenantConfig.EnvConfig != nil {
        // AQUI É ONDE O PROVIDER É CRIADO!
        provider := multiTenantConfig.EnvConfig.CreateProviderFromConfig()
        if provider == nil {
            return newServerWithConfig(nil, multiTenantConfig.EnvConfig.ToServerConfig())
        }
        return newServerWithConfig(provider, multiTenantConfig.EnvConfig.ToServerConfig())
    }
    
    return newServerWithConfig(nil, DefaultServerConfig())
}
```

**O que acontece:**
- ✅ Carrega `.env` automaticamente
- ✅ Cria **UM** provider baseado nas configurações
- ✅ Passa o provider para `newServerWithConfig()`

---

### **2️⃣ Criação do Provider: `CreateProviderFromConfig()`**

**Arquivo:** `odata/config.go` (linha 21)

```go
func (c *EnvConfig) CreateProviderFromConfig() DatabaseProvider {
    // Cria chave única baseada na connection string
    cacheKey := fmt.Sprintf("%s:%s@%s:%s/%s", c.DBDriver, c.DBUser, c.DBHost, c.DBPort, c.DBName)
    
    log.Printf("🔍 [CACHE] CreateProviderFromConfig chamado - cacheKey: %s", cacheKey)
    
    // Verifica se já existe no cache (SINGLETON!)
    providerCacheMu.RLock()
    if cached, exists := providerCache[cacheKey]; exists {
        providerCacheMu.RUnlock()
        log.Printf("📦 [CACHE] Reusando provider do cache")
        return cached
    }
    providerCacheMu.RUnlock()
    
    log.Printf("🆕 [CACHE] Provider NÃO encontrado no cache, criando novo...")
    
    // Cria novo provider baseado no driver
    var provider DatabaseProvider
    switch c.DBDriver {
    case "postgresql", "postgres", "pgx":
        provider = NewPostgreSQLProvider()
    case "mysql":
        provider = NewMySQLProvider()
    case "oracle":
        provider = NewOracleProvider()
    default:
        return nil
    }
    
    if provider != nil {
        log.Printf("✅ [CACHE] Adicionando provider ao cache: %p", provider)
        providerCache[cacheKey] = provider
    }
    
    return provider
}
```

**O que acontece:**
- ✅ Implementa padrão **Singleton** (1 provider por config)
- ✅ Usa cache global para evitar múltiplas conexões
- ✅ Cria pool de conexões no provider

---

### **3️⃣ Criação do Pool: `NewPostgreSQLProvider()`**

**Arquivo:** `odata/provider_postgresql.go` (linha 19)

```go
func NewPostgreSQLProvider(connection ...*sql.DB) *PostgreSQLProvider {
    log.Printf("🔍 [PROVIDER] NewPostgreSQLProvider chamado!")
    var db *sql.DB
    
    // Se não recebeu conexão, tenta carregar do .env
    if len(connection) == 0 || connection[0] == nil {
        log.Printf("🔍 [PROVIDER] Nenhuma conexão passada, carregando do .env...")
        config, err := LoadEnvOrDefault()
        if err != nil {
            // ...
        }
        
        // Cria conexão com base no .env
        connectionString := config.BuildConnectionString()
        
        // Tenta conectar
        if config.DBUser != "" && config.DBPassword != "" {
            var err error
            db, err = sql.Open("pgx", connectionString)
            if err != nil {
                // ...
            }
            
            // CONFIGURA POOL DE CONEXÕES
            db.SetMaxOpenConns(config.DBMaxOpenConns)
            db.SetMaxIdleConns(config.DBMaxIdleConns)
            db.SetConnMaxLifetime(config.DBConnMaxLifetime)
            db.SetConnMaxIdleTime(config.DBConnMaxIdleTime)
            
            log.Printf("✅ [POOL] Configurações: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%s, MaxIdleTime=%s",
                config.DBMaxOpenConns, config.DBMaxIdleConns, config.DBConnMaxLifetime, config.DBConnMaxIdleTime)
            
            // Testa conexão
            if err := db.Ping(); err != nil {
                // ...
            }
            
            log.Printf("✅ Conexão PostgreSQL estabelecida e testada com sucesso")
        }
    }
    
    provider := &PostgreSQLProvider{
        BaseProvider: BaseProvider{
            driverName: "pgx",
            db:         db,  // ← POOL DE CONEXÕES ARMAZENADO AQUI
        },
    }
    
    // Inicializa query builder e parsers
    if db != nil {
        provider.InitQueryBuilder()
        provider.InitParsers()
        log.Printf("✅ [PROVIDER] PostgreSQL Provider criado com sucesso. DB: %p, Provider: %p", db, provider)
    }
    
    return provider
}
```

**O que acontece:**
- ✅ Cria `*sql.DB` (pool de conexões do Go)
- ✅ Configura parâmetros do pool
- ✅ Testa conexão com `Ping()`
- ✅ Retorna provider com pool configurado

---

### **4️⃣ Atribuição ao Servidor: `newServerWithConfig()`**

**Arquivo:** `odata/server.go` (linha 140)

```go
func newServerWithConfig(provider DatabaseProvider, config *ServerConfig) *Server {
    logger := log.New(os.Stdout, "[OData] ", log.LstdFlags|log.Lshortfile)
    
    server := &Server{
        entities:     make(map[string]EntityService),
        router:       fiber.New(),
        parser:       NewODataParser(),
        urlParser:    NewURLParser(),
        provider:     provider,  // ← PROVIDER ATRIBUÍDO AQUI!
        config:       config,
        logger:       logger,
        entityAuth:   make(map[string]EntityAuthConfig),
        eventManager: NewEntityEventManager(logger),
    }
    
    // Configura middlewares, rotas, etc...
    
    return server
}
```

**O que acontece:**
- ✅ Cria estrutura do `Server`
- ✅ Atribui o provider ao campo `server.provider`
- ✅ Provider fica disponível para todas as entidades

---

### **5️⃣ Registro de Entidade: `RegisterEntity()`**

**Arquivo:** `odata/server.go` (linha 230)

```go
func (s *Server) RegisterEntity(name string, entity interface{}, opts ...EntityOption) error {
    // ... código de configuração ...
    
    metadata, err := MapEntityFromStruct(entity)
    if err != nil {
        return fmt.Errorf("erro ao registrar entidade %s: %w", name, err)
    }
    
    var service EntityService
    
    // Se multi-tenant estiver habilitado, usa MultiTenantEntityService
    if s.multiTenantConfig != nil && s.multiTenantConfig.Enabled {
        service = NewMultiTenantEntityService(metadata, s)
    } else {
        // AQUI O PROVIDER DO SERVIDOR É PASSADO PARA O ENTITYSERVICE!
        service = NewBaseEntityService(s.provider, metadata, s)
    }
    
    // Armazena o service no mapa de entidades
    s.mu.Lock()
    s.entities[name] = service
    s.mu.Unlock()
    
    // Configura rotas
    s.setupEntityRoutes(name)
    
    return nil
}
```

**O que acontece:**
- ✅ Recebe a struct da entidade (ex: `User{}`)
- ✅ Cria metadados da entidade
- ✅ Cria `BaseEntityService` passando **`s.provider`** (provider do servidor)
- ✅ Registra o service no servidor

---

### **6️⃣ Criação do EntityService: `NewBaseEntityService()`**

**Arquivo:** `odata/entity_service.go` (linha 20)

```go
func NewBaseEntityService(provider DatabaseProvider, metadata EntityMetadata, server *Server) *BaseEntityService {
    return &BaseEntityService{
        provider:      provider,  // ← PROVIDER ARMAZENADO NO ENTITYSERVICE
        metadata:      metadata,
        server:        server,
        computeParser: nil,
        searchParser:  nil,
    }
}
```

**O que acontece:**
- ✅ Recebe o provider do servidor
- ✅ Armazena no campo `service.provider`
- ✅ Service agora tem acesso ao pool de conexões

---

### **7️⃣ Durante Requisições: Uso do Provider**

**Arquivo:** `odata/query_executor.go` (linha 15)

```go
func (s *BaseEntityService) executeQuery(ctx context.Context, query string, args []any) (*sql.Rows, error) {
    log.Printf("🔍 [REQ] executeQuery chamado - provider: %T", s.provider)
    
    // Verifica se a conexão está disponível
    conn := s.provider.GetConnection()  // ← USA O PROVIDER ARMAZENADO!
    
    log.Printf("🔍 [REQ] conn obtida: %p, nil? %v", conn, conn == nil)
    
    if conn == nil {
        return nil, fmt.Errorf("database connection is nil")
    }
    
    // Testa a conexão antes de usar
    log.Printf("🔍 [REQ] Testando conexão com Ping()...")
    if err := conn.Ping(); err != nil {
        log.Printf("❌ [REQ] Ping FALHOU: %v", err)
        return nil, fmt.Errorf("database connection is closed: %w", err)
    }
    log.Printf("✅ [REQ] Ping OK! Executando query...")
    
    rows, err := conn.QueryContext(ctx, query, args...)
    if err != nil {
        log.Printf("❌ [REQ] QueryContext FALHOU: %v", err)
        return nil, err
    }
    
    log.Printf("✅ [REQ] Query executada com sucesso!")
    return rows, nil
}
```

**O que acontece:**
- ✅ Service usa `s.provider.GetConnection()` para obter conexão do pool
- ✅ Pool do Go gerencia automaticamente as conexões
- ✅ Após uso, conexão volta ao pool

---

## 💻 Exemplo Prático

### **Código Completo**

```go
package main

import (
    "github.com/fitlcarlos/go-data/odata"
)

// 1️⃣ Define as entidades
type User struct {
    ID    int64  `json:"id" odata:"key,table=users"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

type Product struct {
    ID    int64   `json:"id" odata:"key,table=products"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

func main() {
    // 2️⃣ Cria o servidor (provider é criado automaticamente aqui!)
    server := odata.NewServer()
    
    // 3️⃣ Registra entidades (todas usam o MESMO provider!)
    server.RegisterEntity("Users", User{})
    server.RegisterEntity("Products", Product{})
    
    // 4️⃣ Inicia o servidor
    server.Start()
}
```

### **O que acontece nos bastidores:**

```
main.go: NewServer()
    ↓
server.go: Carrega .env
    ↓
config.go: CreateProviderFromConfig()
    ↓
provider_postgresql.go: NewPostgreSQLProvider()
    ↓
    Cria *sql.DB com pool configurado
    ↓
server.go: newServerWithConfig(provider, config)
    ↓
    server.provider = provider (ÚNICO PROVIDER!)
    ↓
main.go: RegisterEntity("Users", User{})
    ↓
server.go: NewBaseEntityService(s.provider, metadata, s)
    ↓
    userService.provider = s.provider (MESMO PROVIDER!)
    ↓
main.go: RegisterEntity("Products", Product{})
    ↓
server.go: NewBaseEntityService(s.provider, metadata, s)
    ↓
    productService.provider = s.provider (MESMO PROVIDER!)
    ↓
Resultado: Users e Products compartilham o MESMO POOL de conexões!
```

---

## 🏢 Modo Multi-Tenant

Em modo multi-tenant, o fluxo é ligeiramente diferente:

```
┌─────────────────────────────────────────────────────────────────┐
│ MODO MULTI-TENANT                                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│ NewServer()                                                      │
│   ↓                                                              │
│ newMultiTenantServer()                                           │
│   ↓                                                              │
│ server.multiTenantPool = NewMultiTenantProviderPool()           │
│   ↓                                                              │
│ RegisterEntity("Users", User{})                                  │
│   ↓                                                              │
│ NewMultiTenantEntityService(metadata, server)                    │
│   ↓                                                              │
│ Durante requisição:                                              │
│   • Extrai Tenant-ID do header/query                            │
│   • Obtém provider específico do tenant:                        │
│     provider = server.multiTenantPool.GetProvider(tenantID)     │
│   • Cada tenant tem SEU PRÓPRIO pool de conexões                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Diferença chave:**
- **Modo normal:** 1 provider compartilhado por TODAS as entidades
- **Modo multi-tenant:** 1 provider POR TENANT

---

## 📊 Diagrama Visual

```
┌────────────────────────────────────────────────────────────────────┐
│                           APLICAÇÃO                                 │
│                                                                     │
│  main.go                                                            │
│  ┌──────────────────┐                                              │
│  │ NewServer()      │                                              │
│  └────────┬─────────┘                                              │
│           │                                                         │
│           ↓                                                         │
│  ┌──────────────────────────────────────────────────────┐         │
│  │ Server                                                │         │
│  │ ┌──────────────────────────────────────────────────┐ │         │
│  │ │ provider: *PostgreSQLProvider (ÚNICO!)           │ │         │
│  │ │   ↓                                               │ │         │
│  │ │   db: *sql.DB (POOL DE CONEXÕES)                 │ │         │
│  │ │       ├─ MaxOpenConns: 25                        │ │         │
│  │ │       ├─ MaxIdleConns: 5                         │ │         │
│  │ │       ├─ MaxLifetime: 0                          │ │         │
│  │ │       └─ MaxIdleTime: 0                          │ │         │
│  │ └──────────────────────────────────────────────────┘ │         │
│  │                                                        │         │
│  │ entities:                                              │         │
│  │ ┌────────────────────────────────────────────────┐   │         │
│  │ │ "Users" → UserEntityService                    │   │         │
│  │ │           provider: ↑ (aponta para o mesmo!)   │   │         │
│  │ └────────────────────────────────────────────────┘   │         │
│  │ ┌────────────────────────────────────────────────┐   │         │
│  │ │ "Products" → ProductEntityService              │   │         │
│  │ │              provider: ↑ (aponta para o mesmo!)│   │         │
│  │ └────────────────────────────────────────────────┘   │         │
│  │ ┌────────────────────────────────────────────────┐   │         │
│  │ │ "Orders" → OrderEntityService                  │   │         │
│  │ │            provider: ↑ (aponta para o mesmo!)  │   │         │
│  │ └────────────────────────────────────────────────┘   │         │
│  └────────────────────────────────────────────────────────┘         │
│                                                                     │
│  RESULTADO: Todas as entidades compartilham o MESMO pool!          │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘

                              ↓↓↓

┌────────────────────────────────────────────────────────────────────┐
│                    BANCO DE DADOS PostgreSQL                        │
│                                                                     │
│  Pool de 25 conexões compartilhadas:                               │
│  [Conn1] [Conn2] [Conn3] [Conn4] [Conn5] ... [Conn25]             │
│                                                                     │
│  • Users, Products e Orders usam as MESMAS conexões                │
│  • Pool gerencia automaticamente a distribuição                    │
│  • Conexões são reutilizadas entre requisições                     │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

---

## ✅ Pontos Importantes

### **1. Um Provider, Um Pool**
- ✅ O provider é criado **UMA VEZ** na inicialização
- ✅ Todas as entidades **compartilham** o mesmo provider
- ✅ Todas as entidades **compartilham** o mesmo pool de conexões

### **2. Singleton Pattern**
- ✅ `CreateProviderFromConfig()` implementa cache
- ✅ Mesma configuração = mesmo provider reutilizado
- ✅ Evita múltiplas conexões desnecessárias

### **3. Thread-Safe**
- ✅ `*sql.DB` do Go é thread-safe
- ✅ Pool gerencia automaticamente concorrência
- ✅ Múltiplas requisições simultâneas funcionam perfeitamente

### **4. Lifecycle**
- ✅ Provider vive enquanto o servidor vive
- ✅ Conexões são gerenciadas pelo pool do Go
- ✅ Apenas é fechado durante `server.Shutdown()`

---

## 🔗 Referências

- [Connection Pool Documentation](CONNECTION_POOL.md)
- [Go-Data README](../README.md)
- [Go database/sql Package](https://pkg.go.dev/database/sql)

---

**💡 Resumo:** O provider é criado uma vez no início, atribuído ao servidor, e compartilhado por todos os EntityServices. Simples e eficiente!

