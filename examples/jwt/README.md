# Exemplo JWT - Autenticação e Autorização

Este exemplo demonstra como implementar autenticação e autorização JWT no GoData OData server.

## Recursos Implementados

### 🔐 Autenticação JWT
- Geração de tokens de acesso e refresh
- Validação de tokens
- Middleware de autenticação obrigatória e opcional
- Rotas de autenticação (/auth/login, /auth/refresh, /auth/logout, /auth/me)

### 🛡️ Autorização Granular
- Controle de acesso baseado em roles
- Controle de acesso baseado em scopes
- Privilégios de administrador
- Configuração de autenticação por entidade
- Entidades somente leitura

### 👥 Usuários de Teste
- **admin/password123**: Administrador com acesso total
- **manager/password123**: Gerente com acesso de escrita
- **user/password123**: Usuário com acesso de leitura

## Configuração de Segurança

### Entidades e Permissões

| Entidade | Permissão | Descrição |
|----------|-----------|-----------|
| Users | Admin apenas | Somente administradores podem acessar |
| Products | Manager/Admin | Managers e admins podem escrever, usuários podem ler |
| Orders | Usuários autenticados | Todos os usuários autenticados podem acessar |

### Estrutura de Roles e Scopes

```go
// Administrador
{
    Username: "admin",
    Roles:    []string{"admin", "user"},
    Scopes:   []string{"read", "write", "delete"},
    Admin:    true
}

// Gerente
{
    Username: "manager",
    Roles:    []string{"manager", "user"},
    Scopes:   []string{"read", "write"},
    Admin:    false
}

// Usuário
{
    Username: "user",
    Roles:    []string{"user"},
    Scopes:   []string{"read"},
    Admin:    false
}
```

## Executando o Exemplo

### 1. Instalar Dependências

```bash
go mod tidy
```

### 2. Executar o Servidor

```bash
go run examples/jwt/main.go
```

### 3. Testar a Autenticação

#### Fazer Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password123"}'
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
    "admin": true,
    "custom": {
      "department": "IT",
      "level": "senior"
    }
  }
}
```

#### Acessar Endpoint Protegido
```bash
curl -X GET http://localhost:8080/api/v1/Users \
  -H "Authorization: Bearer <access_token>"
```

#### Renovar Token
```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```

#### Obter Informações do Usuário
```bash
curl -X GET http://localhost:8080/auth/me \
  -H "Authorization: Bearer <access_token>"
```

## Cenários de Teste

### 1. Acesso de Administrador
```bash
# Login como admin
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password123"}'

# Acessar usuários (permitido)
curl -X GET http://localhost:8080/api/v1/Users \
  -H "Authorization: Bearer <admin_token>"

# Criar produto (permitido)
curl -X POST http://localhost:8080/api/v1/Products \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Produto Teste","price":99.99,"category":"Teste"}'
```

### 2. Acesso de Gerente
```bash
# Login como manager
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"manager","password":"password123"}'

# Acessar usuários (negado - 403)
curl -X GET http://localhost:8080/api/v1/Users \
  -H "Authorization: Bearer <manager_token>"

# Criar produto (permitido)
curl -X POST http://localhost:8080/api/v1/Products \
  -H "Authorization: Bearer <manager_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Produto Manager","price":49.99,"category":"Manager"}'
```

### 3. Acesso de Usuário
```bash
# Login como user
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user","password":"password123"}'

# Acessar usuários (negado - 403)
curl -X GET http://localhost:8080/api/v1/Users \
  -H "Authorization: Bearer <user_token>"

# Criar produto (negado - 403)
curl -X POST http://localhost:8080/api/v1/Products \
  -H "Authorization: Bearer <user_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Produto User","price":19.99,"category":"User"}'

# Listar produtos (permitido)
curl -X GET http://localhost:8080/api/v1/Products \
  -H "Authorization: Bearer <user_token>"
```

## Configuração Personalizada

### Configurar JWT
```go
jwtConfig := &odata.JWTConfig{
    SecretKey: "sua-chave-secreta-aqui",
    Issuer:    "seu-aplicativo",
    ExpiresIn: 1 * time.Hour,
    RefreshIn: 24 * time.Hour,
    Algorithm: "HS256",
}
```

### Configurar Autenticação por Entidade
```go
// Apenas administradores
server.SetEntityAuth("Users", odata.EntityAuthConfig{
    RequireAuth:  true,
    RequireAdmin: true,
})

// Roles específicas
server.SetEntityAuth("Products", odata.EntityAuthConfig{
    RequireAuth:    true,
    RequiredRoles:  []string{"manager", "admin"},
    RequiredScopes: []string{"write"},
})

// Somente leitura
server.SetEntityAuth("Reports", odata.EntityAuthConfig{
    RequireAuth: true,
    ReadOnly:    true,
})
```

### Implementar Autenticador Personalizado
```go
type CustomAuthenticator struct {
    // Sua implementação
}

func (a *CustomAuthenticator) Authenticate(username, password string) (*odata.UserIdentity, error) {
    // Validar credenciais no banco de dados
    // Retornar UserIdentity com roles e scopes
}

func (a *CustomAuthenticator) GetUserByUsername(username string) (*odata.UserIdentity, error) {
    // Buscar usuário no banco de dados
}
```

## Endpoints Disponíveis

### Autenticação
- `POST /auth/login` - Fazer login
- `POST /auth/refresh` - Renovar token
- `POST /auth/logout` - Fazer logout
- `GET /auth/me` - Informações do usuário atual

### OData
- `GET /api/v1/$metadata` - Metadados do serviço
- `GET /api/v1/` - Documento do serviço
- `GET /api/v1/Users` - Lista usuários (Admin)
- `GET /api/v1/Products` - Lista produtos (Autenticado)
- `POST /api/v1/Products` - Criar produto (Manager/Admin)
- `GET /api/v1/Orders` - Lista pedidos (Autenticado)

### Utilitários
- `GET /health` - Health check
- `GET /info` - Informações do servidor

## Segurança

### Boas Práticas Implementadas
- Tokens JWT com expiração
- Refresh tokens para renovação
- Validação de assinatura HMAC-SHA256
- Controle de acesso granular
- Separação de roles e scopes
- Claims customizados

### Configurações Recomendadas
- Use chaves secretas fortes (256 bits)
- Configure tempos de expiração apropriados
- Implemente blacklist de tokens para logout
- Use HTTPS em produção
- Valide entrada de dados
- Implemente rate limiting

## Próximos Passos

1. Integrar com banco de dados real para usuários
2. Implementar blacklist de tokens
3. Adicionar rate limiting
4. Configurar HTTPS
5. Adicionar logs de auditoria
6. Implementar recuperação de senha
7. Adicionar autenticação de dois fatores 