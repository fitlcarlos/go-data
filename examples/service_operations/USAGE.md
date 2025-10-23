# Service Operations - Instruções de Uso

## 🚀 **Execução Rápida**

### 1. Configurar Banco de Dados

```bash
# Copiar arquivo de configuração
cp config.env .env

# Editar configurações do banco
# DB_TYPE=mysql
# DB_HOST=localhost
# DB_PORT=3306
# DB_NAME=service_operations_db
# DB_USER=root
# DB_PASSWORD=password
```

### 2. Executar o Servidor

```bash
# Instalar dependências
go mod tidy

# Executar servidor
go run main.go
```

### 3. Testar Service Operations

```bash
# Service sem autenticação
curl -X GET "http://localhost:8080/api/odata/Service/GetTopSellingProducts"

# Service com autenticação (após fazer login)
curl -X POST "http://localhost:8080/api/odata/Service/CalculateTotalPrice?product_ids=1,2,3" \
  -H "Authorization: Bearer <token>"

# Service com parâmetros
curl -X GET "http://localhost:8080/api/odata/Service/GetProductStats?category=electronics&min_price=100"

# Service administrativo
curl -X GET "http://localhost:8080/api/odata/Service/AdminData" \
  -H "Authorization: Bearer <admin_token>"
```

## 📋 **Endpoints Disponíveis**

### Services Sem Autenticação
- `GET /api/odata/Service/GetTopSellingProducts` - Lista produtos mais vendidos
- `GET /api/odata/Service/GetProductStats` - Estatísticas de produtos com filtros

### Services Com Autenticação
- `POST /api/odata/Service/CalculateTotalPrice` - Calcula preço total de produtos
- `GET /api/odata/Service/GenerateReport` - Gera relatório de produtos
- `GET /api/odata/Service/GetUserProfile` - Obtém perfil do usuário

### Services Com Roles
- `GET /api/odata/Service/AdminData` - Dados administrativos (role: admin)
- `GET /api/odata/Service/ManagerData` - Dados de gerência (roles: manager, admin)

### Service Groups
- `GET /api/odata/Service/Products/GetTopSelling` - Produtos mais vendidos (com auth)
- `GET /api/odata/Service/Products/AdminStats` - Estatísticas admin de produtos

## 🔐 **Autenticação JWT**

### Login
```bash
curl -X POST "http://localhost:8080/api/odata/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@engage.com",
    "password": "password"
  }'
```

### Usar Token
```bash
# Substituir <token> pelo token retornado no login
curl -X GET "http://localhost:8080/api/odata/Service/GetUserProfile" \
  -H "Authorization: Bearer <token>"
```

## 🏢 **Multi-Tenant**

### Usar Tenant Específico
```bash
# Adicionar header X-Tenant-ID
curl -X GET "http://localhost:8080/api/odata/Service/GetTopSellingProducts" \
  -H "X-Tenant-ID: company_a"
```

## 📊 **Exemplos de Resposta**

### GetTopSellingProducts
```json
{
  "top_selling": [
    {
      "id": 1,
      "name": "Product A",
      "price": 99.99,
      "sales_count": 150
    }
  ],
  "count": 1,
  "tenant": "default",
  "authenticated": false
}
```

### CalculateTotalPrice
```json
{
  "total_price": 299.97,
  "product_count": 3,
  "user": "admin@engage.com",
  "tenant": "default"
}
```

### AdminData
```json
{
  "admin_stats": [
    {
      "category": "electronics",
      "total": 5,
      "total_value": 499.95
    }
  ],
  "admin_user": "admin@engage.com",
  "tenant": "default"
}
```

## 🛠️ **Desenvolvimento**

### Adicionar Novo Service

```go
// 1. Definir handler
func MyCustomService(ctx *odata.ServiceContext) error {
    manager := ctx.GetManager()
    
    // Sua lógica aqui
    data, err := manager.Query("MyEntity").List()
    if err != nil {
        return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
    }
    
    return ctx.JSON(map[string]interface{}{
        "data": data,
        "tenant": ctx.GetTenantID(),
    })
}

// 2. Registrar service
server.Service("GET", "/Service/MyCustom", MyCustomService)

// 3. Ou com autenticação
server.ServiceWithAuth("POST", "/Service/MyCustom", MyCustomService, true)

// 4. Ou com roles
server.ServiceWithRoles("GET", "/Service/MyCustom", MyCustomService, "admin")
```

### Service Groups

```go
// Criar grupo
myGroup := server.ServiceGroup("MyGroup")

// Registrar services no grupo
myGroup.Service("GET", "List", MyListHandler)
myGroup.ServiceWithAuth("POST", "Create", MyCreateHandler, true)
myGroup.ServiceWithRoles("DELETE", "Delete", MyDeleteHandler, "admin")
```

## 🔧 **Configurações Avançadas**

### Personalizar ServiceContext

```go
// O ServiceContext é criado automaticamente, mas você pode acessar:
func MyService(ctx *odata.ServiceContext) error {
    // ObjectManager (equivale ao XData GetManager())
    manager := ctx.GetManager()
    
    // Contexto do Fiber (acesso completo à requisição)
    fiberCtx := ctx.FiberContext
    
    // Usuário autenticado (se JWT habilitado)
    user := ctx.GetUser()
    
    // Tenant atual
    tenantID := ctx.GetTenantID()
    
    // Métodos utilitários
    params := ctx.QueryParams()
    productID := ctx.Query("product_id")
    body := ctx.Body()
    
    // Resposta
    return ctx.JSON(result)
}
```

### Tratamento de Erros

```go
func MyService(ctx *odata.ServiceContext) error {
    // Validação de parâmetros
    productID := ctx.Query("product_id")
    if productID == "" {
        return ctx.Status(400).JSON(map[string]string{
            "error": "product_id parameter required"
        })
    }
    
    // Validação de autenticação
    if !ctx.IsAuthenticated() {
        return ctx.Status(401).JSON(map[string]string{
            "error": "Authentication required"
        })
    }
    
    // Validação de roles
    if !ctx.HasRole("admin") {
        return ctx.Status(403).JSON(map[string]string{
            "error": "Admin access required"
        })
    }
    
    // Sua lógica aqui
    manager := ctx.GetManager()
    // ...
    
    return ctx.JSON(result)
}
```

## 📝 **Logs e Debug**

O servidor exibe logs detalhados:

```
=== Go-Data Service Operations Example ===
📝 Registrando Service Operations...
✅ Service Operations registrados com sucesso!
🚀 Servidor iniciado com Service Operations!
📋 Endpoints disponíveis:
   - GET  /api/odata/Service/GetTopSellingProducts
   - POST /api/odata/Service/CalculateTotalPrice
   - GET  /api/odata/Service/GetProductStats
   - GET  /api/odata/Service/GenerateReport
   - GET  /api/odata/Service/GetUserProfile
   - GET  /api/odata/Service/AdminData (requer role 'admin')
   - GET  /api/odata/Service/ManagerData (requer role 'manager')
```

## 🎯 **Próximos Passos**

1. **Personalizar**: Modifique os handlers para suas necessidades
2. **Adicionar Entidades**: Registre suas próprias entidades
3. **Configurar Auth**: Ajuste as configurações JWT
4. **Multi-Tenant**: Configure tenants específicos
5. **Deploy**: Use o exemplo como base para sua aplicação

O ServiceContext otimizado mantém toda a **potência** do XData Service Operations, mas com **simplicidade** e **padrões Go idiomáticos**! 🎉
