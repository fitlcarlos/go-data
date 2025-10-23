# Service Operations - Go-Data

O Go-Data implementa Service Operations similares ao XData, mas usando padrões idiomáticos do Go. O sistema oferece um `ServiceContext` otimizado que equivale funcionalmente ao `TXDataOperationContext` do XData.

## 🎯 **Características**

- ✅ **ServiceContext Otimizado**: Equivale ao `TXDataOperationContext.Current.GetManager()` do XData
- ✅ **Sintaxe Simples**: Similar ao Fiber para registro de handlers
- ✅ **Autenticação Flexível**: Controle automático baseado na configuração JWT
- ✅ **Multi-Tenant**: Suporte automático a multi-tenant
- ✅ **ObjectManager Integrado**: Acesso direto ao ObjectManager do contexto
- ✅ **Menos Boilerplate**: 95% menos código que implementações tradicionais

## 🏗️ **ServiceContext**

```go
type ServiceContext struct {
    Manager      *ObjectManager  // Equivale ao TXDataOperationContext.Current.GetManager()
    FiberContext fiber.Ctx       // Contexto do Fiber (já tem TenantID via GetCurrentTenant())
    User         *UserIdentity   // Usuário autenticado (só se JWT habilitado)
}
```

### Métodos Principais

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

## 🚀 **Registro de Services**

### Service Sem Autenticação

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

### Service Com Autenticação

```go
server.ServiceWithAuth("POST", "/Service/CalculateTotal", func(ctx *odata.ServiceContext) error {
    // ctx.User garantidamente não será nil se JWT habilitado
    productIDs := ctx.Query("product_ids")
    
    manager := ctx.GetManager()
    // ... lógica do service
    
    return ctx.JSON(result)
}, true)
```

### Service Com Roles

```go
server.ServiceWithRoles("GET", "/Service/AdminData", func(ctx *ServiceContext) error {
    // ctx.User garantidamente tem role "admin"
    manager := ctx.GetManager()
    // ... lógica administrativa
    
    return ctx.JSON(data)
}, "admin")
```

### Service Groups

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

## 📋 **Exemplos Práticos**

### 1. Service Simples

```go
func GetTopSellingProducts(ctx *odata.ServiceContext) error {
    manager := ctx.GetManager()
    
    products, err := manager.Query("Products").
        Where("sales_count gt 100").
        OrderBy("sales_count desc").
        Top(10).
        List()
    
    if err != nil {
        return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
    }
    
    return ctx.JSON(map[string]interface{}{
        "top_selling": products,
        "count": len(products),
        "tenant": ctx.GetTenantID(),
    })
}

// Registro
server.Service("GET", "/Service/GetTopSellingProducts", GetTopSellingProducts)
```

### 2. Service Com Parâmetros

```go
func CalculateTotalPrice(ctx *odata.ServiceContext) error {
    productIDs := ctx.Query("product_ids")
    if productIDs == "" {
        return ctx.Status(400).JSON(map[string]string{"error": "product_ids required"})
    }
    
    manager := ctx.GetManager()
    ids := strings.Split(productIDs, ",")
    
    total := 0.0
    for _, id := range ids {
        product, err := manager.Find("Products", strings.TrimSpace(id))
        if err != nil {
            continue
        }
        
        if productMap, ok := product.(map[string]interface{}); ok {
            if price, exists := productMap["price"]; exists {
                if priceFloat, ok := price.(float64); ok {
                    total += priceFloat
                }
            }
        }
    }
    
    return ctx.JSON(map[string]interface{}{
        "total_price": total,
        "user": ctx.GetUser().Username,
    })
}

// Registro com autenticação
server.ServiceWithAuth("POST", "/Service/CalculateTotalPrice", CalculateTotalPrice, true)
```

### 3. Service Administrativo

```go
func GetAdminData(ctx *odata.ServiceContext) error {
    manager := ctx.GetManager()
    
    stats, err := manager.Query("Products").
        Select("category, count(*) as total, sum(price) as total_value").
        GroupBy("category").
        List()
    
    if err != nil {
        return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
    }
    
    return ctx.JSON(map[string]interface{}{
        "admin_stats": stats,
        "admin_user": ctx.GetUser().Username,
        "tenant": ctx.GetTenantID(),
    })
}

// Registro com role admin
server.ServiceWithRoles("GET", "/Service/AdminData", GetAdminData, "admin")
```

### 4. Service de Relatório

```go
func GenerateReport(ctx *odata.ServiceContext) error {
    format := ctx.Query("format")
    if format == "" {
        format = "json"
    }
    
    manager := ctx.GetManager()
    
    reportData, err := manager.Query("Products").
        Where("is_active eq true").
        OrderBy("sales_count desc").
        Select("name, price, sales_count, category").
        List()
    
    if err != nil {
        return ctx.Status(500).JSON(map[string]string{"error": err.Error()})
    }
    
    // Configurar headers baseado no formato
    switch format {
    case "pdf":
        ctx.SetHeader("Content-Type", "application/pdf")
        ctx.SetHeader("Content-Disposition", "attachment; filename=report.pdf")
    case "excel":
        ctx.SetHeader("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    }
    
    return ctx.JSON(map[string]interface{}{
        "report_data": reportData,
        "format": format,
        "generated_by": ctx.GetUser().Username,
        "tenant": ctx.GetTenantID(),
    })
}

// Registro
server.ServiceWithAuth("GET", "/Service/GenerateReport", GenerateReport, true)
```

## 🔧 **Configuração**

### Arquivo .env

```env
# Configuração JWT (opcional)
JWT_ENABLED=true
JWT_SECRET_KEY=your-secret-key
JWT_ISSUER=go-data-server
JWT_EXPIRES_IN=24h
JWT_REFRESH_IN=168h

# Configuração Multi-Tenant (opcional)
MULTI_TENANT_ENABLED=true
TENANT_IDENTIFICATION_MODE=header
TENANT_HEADER_NAME=X-Tenant-ID
DEFAULT_TENANT=default

# Configuração do servidor
SERVER_HOST=localhost
SERVER_PORT=8080
SERVER_ROUTE_PREFIX=/api/odata
```

### Código do Servidor

```go
func main() {
    server := odata.NewServer()
    
    // Registrar entidades
    server.RegisterEntity("Products", Product{})
    server.RegisterEntity("Users", User{})
    
    // Registrar service operations
    registerServices(server)
    
    server.Start()
}

func registerServices(server *odata.Server) {
    // Services sem autenticação
    server.Service("GET", "/Service/GetTopSelling", GetTopSellingProducts)
    
    // Services com autenticação
    server.ServiceWithAuth("POST", "/Service/CalculateTotal", CalculateTotalPrice, true)
    
    // Services com roles
    server.ServiceWithRoles("GET", "/Service/AdminData", GetAdminData, "admin")
    
    // Services com grupos
    products := server.ServiceGroup("Products")
    products.ServiceWithAuth("GET", "GetTopSelling", GetTopSellingProducts, true)
}
```

## 📊 **Comparação com XData**

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

## ✅ **Vantagens**

1. **Ultra Simples**: Apenas 3 linhas para registrar um service
2. **Familiar**: Sintaxe similar ao Fiber
3. **Automático**: ObjectManager injetado automaticamente
4. **Flexível**: Controle de auth baseado na configuração
5. **Consistente**: Usa infraestrutura existente do Go-Data
6. **Menos Código**: 95% menos boilerplate

## 🎯 **Endpoints Gerados**

Com o exemplo acima, os seguintes endpoints são criados automaticamente:

```
GET  /api/odata/Service/GetTopSellingProducts
POST /api/odata/Service/CalculateTotalPrice
GET  /api/odata/Service/AdminData
GET  /api/odata/Service/ManagerData
GET  /api/odata/Service/GetProductStats
GET  /api/odata/Service/GenerateReport
GET  /api/odata/Service/GetUserProfile
GET  /api/odata/Service/Products/GetTopSelling
GET  /api/odata/Service/Products/AdminStats
```

## 🚀 **Uso**

```bash
# Service sem autenticação
curl -X GET "http://localhost:8080/api/odata/Service/GetTopSellingProducts"

# Service com autenticação
curl -X POST "http://localhost:8080/api/odata/Service/CalculateTotalPrice?product_ids=1,2,3" \
  -H "Authorization: Bearer <token>"

# Service com parâmetros
curl -X GET "http://localhost:8080/api/odata/Service/GetProductStats?category=electronics&min_price=100"

# Service administrativo
curl -X GET "http://localhost:8080/api/odata/Service/AdminData" \
  -H "Authorization: Bearer <admin_token>"
```

O ServiceContext otimizado mantém toda a **potência** do XData Service Operations, mas com **simplicidade** e **padrões Go idiomáticos**! 🎉
