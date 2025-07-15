# Exemplo Multi-Tenant - Go-Data OData Server

Este exemplo demonstra como configurar e usar o servidor OData Go-Data com suporte multi-tenant, permitindo que uma única instância do servidor gerencie múltiplos bancos de dados para diferentes tenants.

## Características

- **Identificação automática de tenant** via headers, subdomains, path ou JWT
- **Pool de conexões** gerenciado automaticamente para cada tenant
- **Configuração via .env** com suporte a múltiplos bancos de dados
- **Isolamento completo** de dados por tenant
- **Compatibilidade** com Oracle, PostgreSQL e MySQL
- **Endpoints específicos** para monitoramento e gerenciamento de tenants

## Configuração

### 1. Arquivo .env

O exemplo cria automaticamente um arquivo `.env` com configuração multi-tenant:

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
```

### 2. Executar o exemplo

```bash
cd examples/multi_tenant
go run main.go
```

## Métodos de Identificação de Tenant

### 1. Header (Padrão)

```bash
# Listar produtos do tenant padrão
curl -X GET "http://localhost:8080/api/odata/Produtos"

# Listar produtos da empresa A
curl -X GET "http://localhost:8080/api/odata/Produtos" \
  -H "X-Tenant-ID: empresa_a"
```

### 2. Subdomain

Configure `TENANT_IDENTIFICATION_MODE=subdomain`:

```bash
# Acesso via subdomain
curl -X GET "http://empresa_a.localhost:8080/api/odata/Produtos"
```

### 3. Path

Configure `TENANT_IDENTIFICATION_MODE=path`:

```bash
# Acesso via path
curl -X GET "http://localhost:8080/api/empresa_a/odata/Produtos"
```

### 4. JWT Token

Configure `TENANT_IDENTIFICATION_MODE=jwt` e inclua claim `tenant_id`:

```bash
# Acesso via JWT (com claim tenant_id)
curl -X GET "http://localhost:8080/api/odata/Produtos" \
  -H "Authorization: Bearer <jwt_token_com_tenant_id>"
```

## Endpoints Específicos Multi-Tenant

### Listar Tenants

```bash
curl -X GET "http://localhost:8080/tenants"
```

Resposta:
```json
{
  "multi_tenant": true,
  "tenants": ["default", "empresa_a", "empresa_b", "empresa_c"],
  "total_count": 4
}
```

### Estatísticas dos Tenants

```bash
curl -X GET "http://localhost:8080/tenants/stats"
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

### Health Check por Tenant

```bash
curl -X GET "http://localhost:8080/tenants/empresa_a/health"
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

## Exemplos de Uso das Entidades

### Produtos

```bash
# Listar todos os produtos
curl -X GET "http://localhost:8080/api/odata/Produtos" \
  -H "X-Tenant-ID: empresa_a"

# Obter produto específico
curl -X GET "http://localhost:8080/api/odata/Produtos(1)" \
  -H "X-Tenant-ID: empresa_a"

# Filtrar produtos por categoria
curl -X GET "http://localhost:8080/api/odata/Produtos?\$filter=categoria eq 'Eletrônicos'" \
  -H "X-Tenant-ID: empresa_a"

# Criar novo produto
curl -X POST "http://localhost:8080/api/odata/Produtos" \
  -H "X-Tenant-ID: empresa_a" \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "Smartphone",
    "descricao": "Smartphone Android",
    "preco": 899.99,
    "categoria": "Eletrônicos"
  }'
```

### Clientes

```bash
# Listar clientes
curl -X GET "http://localhost:8080/api/odata/Clientes" \
  -H "X-Tenant-ID: empresa_b"

# Criar novo cliente
curl -X POST "http://localhost:8080/api/odata/Clientes" \
  -H "X-Tenant-ID: empresa_b" \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "João Silva",
    "email": "joao@empresa.com",
    "telefone": "(11) 99999-9999"
  }'
```

### Pedidos

```bash
# Listar pedidos
curl -X GET "http://localhost:8080/api/odata/Pedidos" \
  -H "X-Tenant-ID: empresa_c"

# Criar novo pedido
curl -X POST "http://localhost:8080/api/odata/Pedidos" \
  -H "X-Tenant-ID: empresa_c" \
  -H "Content-Type: application/json" \
  -d '{
    "cliente_id": 1,
    "produto_id": 1,
    "quantidade": 2,
    "valor_total": 1799.98,
    "data_pedido": "2024-01-15"
  }'
```

## Estrutura do Código

### Entidades

Cada entidade inclui um campo `tenant_id` para isolamento:

```go
type Produto struct {
    ID          int64  `json:"id" db:"id" odata:"key"`
    Nome        string `json:"nome" db:"nome"`
    Descricao   string `json:"descricao" db:"descricao"`
    Preco       float64 `json:"preco" db:"preco"`
    Categoria   string `json:"categoria" db:"categoria"`
    TenantID    string `json:"tenant_id" db:"tenant_id"`
}
```

### Registro de Entidades

```go
// Registra as entidades (automaticamente multi-tenant se configurado)
server.RegisterEntity("Produtos", &Produto{})
server.RegisterEntity("Clientes", &Cliente{})
server.RegisterEntity("Pedidos", &Pedido{})
```

### Eventos

```go
// Registra eventos globais com informações de tenant
server.OnEntityListGlobal(func(args odata.EventArgs) error {
    if listArgs, ok := args.(*odata.EntityListArgs); ok {
        tenantID := odata.GetCurrentTenant(listArgs.Context.FiberContext)
        log.Printf("📋 Lista acessada: %s (tenant: %s)", 
            listArgs.EntityName, tenantID)
    }
    return nil
})
```

## Logs e Monitoramento

O servidor produz logs detalhados para cada tenant:

```
[OData-MultiTenant] 2024/01/15 10:30:00 🏢 Tenant identificado: empresa_a
[OData-MultiTenant] 2024/01/15 10:30:00 🏢 [empresa_a] Produtos - Query: Success
[OData-MultiTenant] 2024/01/15 10:30:00 📋 Lista de entidades acessada: Produtos (tenant: empresa_a)
```

## Vantagens do Multi-Tenant

1. **Isolamento de dados**: Cada tenant tem seu próprio banco de dados
2. **Escalabilidade**: Adição dinâmica de novos tenants
3. **Flexibilidade**: Diferentes tipos de banco por tenant
4. **Monitoramento**: Estatísticas individuais por tenant
5. **Segurança**: Isolamento completo entre tenants
6. **Performance**: Pool de conexões otimizado por tenant

## Adicionando Novos Tenants

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

## Considerações de Segurança

- **Validação de tenant**: Sempre valide se o tenant existe
- **Autenticação**: Use JWT com claim `tenant_id` para maior segurança
- **Auditoria**: Todos os acessos são logados com tenant ID
- **Isolamento**: Dados são completamente isolados por tenant

## Troubleshooting

### Tenant não encontrado

```json
{
  "error": {
    "code": "BadRequest",
    "message": "Tenant 'inexistente' não encontrado"
  }
}
```

### Conexão de banco indisponível

```json
{
  "tenant_id": "empresa_a",
  "status": "unhealthy",
  "error": "dial tcp: connection refused"
}
```

### Pool de conexões esgotado

Ajuste as configurações no `.env`:

```env
TENANT_EMPRESA_A_DB_MAX_OPEN_CONNS=50
TENANT_EMPRESA_A_DB_MAX_IDLE_CONNS=10
TENANT_EMPRESA_A_DB_CONN_MAX_LIFETIME=30m
``` 