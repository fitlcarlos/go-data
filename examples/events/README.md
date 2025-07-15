# Exemplo de Eventos de Entidade

Este exemplo demonstra como usar o sistema de eventos de entidade do Go-Data para implementar validações customizadas, auditoria e regras de negócio complexas.

## 📋 Funcionalidades Demonstradas

### 🔍 Eventos de Validação
- **OnEntityInserting**: Validação antes da inserção de usuários e produtos
- **OnEntityModifying**: Validação antes da atualização com controle de acesso
- **OnEntityDeleting**: Validação antes da exclusão com verificação de dependências

### 📊 Sistema de Auditoria
- Log de todas as operações CRUD
- Registro do usuário que executou cada operação
- Timestamps automáticos para criação e atualização

### 🔒 Controle de Acesso
- Verificação de permissões baseada em roles
- Restrições específicas por tipo de operação
- Proteção de campos sensíveis

### 🌐 Eventos Globais
- Auditoria global para todas as entidades
- Tratamento centralizado de erros
- Log de atividades do sistema

## 🚀 Como Executar

### 1. Preparar o Banco de Dados

```sql
-- Criar banco de dados
CREATE DATABASE testdb;

-- Usar o banco
\c testdb;

-- Criar tabela de usuários
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Criar tabela de produtos
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category VARCHAR(50),
    description TEXT,
    created_by INTEGER REFERENCES users(id),
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 2. Executar o Exemplo

```bash
# Instalar dependências
go mod tidy

# Executar o servidor
go run main.go
```

### 3. Testar os Eventos

#### Teste de Validação - Inserção de Usuário

```bash
# Teste com dados válidos
curl -X POST http://localhost:8080/odata/Users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "João Silva",
    "email": "joao@exemplo.com"
  }'

# Teste com nome inválido (muito curto)
curl -X POST http://localhost:8080/odata/Users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "J",
    "email": "j@exemplo.com"
  }'
```

#### Teste de Validação - Produto

```bash
# Teste com preço negativo
curl -X POST http://localhost:8080/odata/Products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Produto Teste",
    "price": -10.50,
    "category": "Eletrônicos"
  }'

# Teste com dados válidos
curl -X POST http://localhost:8080/odata/Products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Notebook",
    "price": 2500.00,
    "category": "Eletrônicos",
    "description": "Notebook para desenvolvimento"
  }'
```

#### Teste de Atualização com Controle de Acesso

```bash
# Teste de atualização sem permissão de admin
curl -X PUT http://localhost:8080/odata/Users(1) \
  -H "Content-Type: application/json" \
  -d '{
    "name": "João Santos",
    "email": "joao.santos@exemplo.com"
  }'
```

#### Teste de Exclusão com Validação

```bash
# Teste de exclusão de usuário
curl -X DELETE http://localhost:8080/odata/Users(1)
```

## 📝 Logs Esperados

Ao executar as operações, você verá logs similares a estes:

```
🔍 [Users] Inserindo usuário: map[email:joao@exemplo.com name:João Silva]
🔍 [GLOBAL] Inserindo entidade: Users por usuário: 
✅ [Users] Usuário inserido com sucesso: map[created:2024-01-20T10:30:00Z email:joao@exemplo.com id:1 is_active:true name:João Silva updated:2024-01-20T10:30:00Z]

🔍 [Products] Inserindo produto: map[category:Eletrônicos name:Produto Teste price:-10.5]
❌ Evento cancelado: Preço não pode ser negativo

🔍 [Users] Modificando usuário: map[email:joao.santos@exemplo.com name:João Santos]
❌ Evento cancelado: Apenas administradores podem alterar email

🗑️ [Users] Deletando usuário: map[id:1]
🔍 [GLOBAL] Deletando entidade: Users
```

## 🔧 Customização

### Adicionando Novos Tipos de Validação

```go
// Adicionar no setupUserEvents
server.OnEntityInserting("Users", func(args odata.EventArgs) error {
    insertArgs := args.(*odata.EntityInsertingArgs)
    
    // Sua validação customizada aqui
    if err := myCustomValidation(insertArgs.Data); err != nil {
        args.Cancel(err.Error())
        return nil
    }
    
    return nil
})
```

### Implementando Auditoria Personalizada

```go
func setupCustomAudit(server *odata.Server) {
    server.OnEntityInsertedGlobal(func(args odata.EventArgs) error {
        // Salvar no seu sistema de auditoria
        return saveToAuditSystem(args)
    })
}
```

### Adicionando Notificações

```go
server.OnEntityInserted("Users", func(args odata.EventArgs) error {
    insertedArgs := args.(*odata.EntityInsertedArgs)
    
    // Enviar notificação
    go sendWelcomeNotification(insertedArgs.CreatedEntity)
    
    return nil
})
```

## 🎯 Casos de Uso Avançados

### 1. Validação Assíncrona

```go
server.OnEntityInserting("Users", func(args odata.EventArgs) error {
    insertArgs := args.(*odata.EntityInsertingArgs)
    
    // Validação assíncrona (cuidado com timeouts)
    if email, ok := insertArgs.Data["email"].(string); ok {
        if !isValidEmailAsync(email) {
            args.Cancel("Email inválido ou já em uso")
            return nil
        }
    }
    
    return nil
})
```

### 2. Enriquecimento de Dados

```go
server.OnEntityInserting("Products", func(args odata.EventArgs) error {
    insertArgs := args.(*odata.EntityInsertingArgs)
    
    // Enriquecer dados automaticamente
    if category, ok := insertArgs.Data["category"].(string); ok {
        // Adicionar informações da categoria
        categoryInfo := getCategoryInfo(category)
        insertArgs.Data["category_info"] = categoryInfo
    }
    
    return nil
})
```

### 3. Integração com Sistemas Externos

```go
server.OnEntityInserted("Users", func(args odata.EventArgs) error {
    insertedArgs := args.(*odata.EntityInsertedArgs)
    
    // Integrar com CRM externo
    go func() {
        err := syncWithExternalCRM(insertedArgs.CreatedEntity)
        if err != nil {
            log.Printf("Erro ao sincronizar com CRM: %v", err)
        }
    }()
    
    return nil
})
```

## 🔍 Depuração

Para depurar os eventos, você pode:

1. **Aumentar o nível de log**:
```go
server.GetEventManager().SetLogLevel("DEBUG")
```

2. **Adicionar logs detalhados**:
```go
server.OnEntityInsertingGlobal(func(args odata.EventArgs) error {
    log.Printf("DEBUG: Dados recebidos: %+v", args.GetEntity())
    log.Printf("DEBUG: Contexto: %+v", args.GetContext())
    return nil
})
```

3. **Verificar handlers registrados**:
```go
subscriptions := server.GetEventManager().ListSubscriptions()
log.Printf("Handlers registrados: %+v", subscriptions)
```

## 📚 Referências

- [Documentação Principal](../../README.md#-eventos-de-entidade)
- [Exemplo JWT](../jwt/README.md)
- [Exemplo Básico](../basic/README.md) 