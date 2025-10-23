# Exemplo: Basic Authentication

Este exemplo demonstra como usar autenticação HTTP Basic com o Go-Data.

## 📋 Características

- ✅ Autenticação Basic com validação em banco de dados
- ✅ Customização de UserValidator com logging
- ✅ Entidades protegidas por autenticação
- ✅ Entidade somente leitura (Users)
- ✅ Múltiplos usuários de teste com roles diferentes
- ✅ WWW-Authenticate header automático

## 🗄️ Pré-requisitos

- Go 1.24+
- MySQL rodando localmente
- Banco de dados `odata_test` criado

## 🚀 Como Executar

1. **Criar banco de dados:**

```sql
CREATE DATABASE odata_test;
```

2. **Configurar variáveis de ambiente:**

Copie o arquivo `env.example` para `.env` e configure suas credenciais:

```bash
cp env.example .env
```

Edite o `.env`:
```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_secure_password_here
DB_NAME=odata_test
```

**⚠️ IMPORTANTE:** NUNCA comite o arquivo `.env` no Git!

3. **Instalar dependências:**

```bash
go mod download
```

4. **Executar:**

```bash
cd examples/basic_auth
go run main.go
```

O servidor iniciará em `http://localhost:3000`.

## 📊 Estrutura

### Tabelas

**users:**
- id (INT, PK, AUTO_INCREMENT)
- username (VARCHAR)
- password (VARCHAR) - senha hasheada com bcrypt
- email (VARCHAR)
- role (VARCHAR) - "admin", "user", "manager"
- active (BOOLEAN)
- created_at (TIMESTAMP)

**products:**
- id (INT, PK, AUTO_INCREMENT)
- name (VARCHAR)
- description (TEXT)
- price (DECIMAL)
- stock (INT)
- created_at (TIMESTAMP)

### Usuários de Teste

O exemplo cria automaticamente 3 usuários:

| Username | Password   | Role    | Descrição           |
|----------|------------|---------|---------------------|
| admin    | admin123   | admin   | Administrador       |
| user     | user123    | user    | Usuário comum       |
| manager  | manager123 | manager | Gerente             |

## 🔐 Endpoints

### Público

```bash
# Informações da API
GET /api/v1/info
```

### Protegidos (requer autenticação)

```bash
# Listar usuários (somente leitura)
GET /api/v1/Users

# Obter usuário autenticado
GET /api/v1/me

# Listar produtos
GET /api/v1/Products

# Criar produto
POST /api/v1/Products

# Atualizar produto
PUT /api/v1/Products(1)

# Deletar produto
DELETE /api/v1/Products(1)
```

## 💡 Exemplos de Uso

### 1. Usando curl com -u (recomendado)

```bash
# Listar usuários
curl -u admin:admin123 http://localhost:3000/api/v1/Users

# Ver dados do usuário autenticado
curl -u admin:admin123 http://localhost:3000/api/v1/me

# Criar produto
curl -u admin:admin123 -X POST \
  http://localhost:3000/api/v1/Products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Novo Produto",
    "description": "Descrição do produto",
    "price": 99.90,
    "stock": 10
  }'
```

### 2. Usando header Authorization manual

Primeiro, gere o Base64:
```bash
echo -n "admin:admin123" | base64
# Resultado: YWRtaW46YWRtaW4xMjM=
```

Depois use no header:
```bash
curl -H "Authorization: Basic YWRtaW46YWRtaW4xMjM=" \
  http://localhost:3000/api/v1/Users
```

### 3. Consultas OData

```bash
# Filtrar usuários por role
curl -u admin:admin123 \
  "http://localhost:3000/api/v1/Users?\$filter=role eq 'admin'"

# Ordenar produtos por preço
curl -u admin:admin123 \
  "http://localhost:3000/api/v1/Products?\$orderby=price desc"

# Paginação
curl -u admin:admin123 \
  "http://localhost:3000/api/v1/Products?\$top=5&\$skip=0"

# Selecionar campos específicos
curl -u admin:admin123 \
  "http://localhost:3000/api/v1/Products?\$select=name,price"
```

### 4. Testando autenticação inválida

```bash
# Sem credenciais (401)
curl -v http://localhost:3000/api/v1/Users

# Resposta:
# HTTP/1.1 401 Unauthorized
# WWW-Authenticate: Basic realm="OData API"

# Credenciais inválidas (401)
curl -u admin:senhaerrada http://localhost:3000/api/v1/Users
```

## 🔧 Customização

### Adicionar logging

O exemplo já demonstra como adicionar logging ao validator:

```go
originalValidator := basicAuth.UserValidator
basicAuth.UserValidator = func(username, password string) (*odata.UserIdentity, error) {
    log.Printf("Tentativa de login: %s", username)
    user, err := originalValidator(username, password)
    if err != nil {
        log.Printf("Login falhou: %s", username)
    }
    return user, err
}
```

### Senhas Hasheadas com Bcrypt

✅ **Este exemplo já usa bcrypt para hash de senhas!**

O código inclui funções helper prontas para uso:

```go
// Criar hash de senha
hash, err := HashPassword("minha_senha_segura")

// Verificar senha
if CheckPasswordHash("minha_senha_segura", hash) {
    // Senha correta
}
```

**Implementação no código:**

```go
// Função helper para hash
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

// Função helper para verificação
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}

// Uso na validação
func validateUser(username, password string) (*odata.UserIdentity, error) {
    // 1. Buscar usuário e hash da senha no banco
    var passwordHash string
    query := `SELECT password FROM users WHERE username = ?`
    db.QueryRow(query, username).Scan(&passwordHash)
    
    // 2. Verificar senha com bcrypt
    if !CheckPasswordHash(password, passwordHash) {
        return nil, errors.New("credenciais inválidas")
    }
    
    // 3. Retornar identidade do usuário
    return userIdentity, nil
}
```

### Customizar Extração de Credenciais

```go
basicAuth.TokenExtractor = func(c fiber.Ctx) string {
    // Tentar header customizado primeiro
    if token := c.Get("X-API-Key"); token != "" {
        return token
    }
    
    // Fallback para Basic padrão
    return basicAuth.DefaultExtractToken(c)
}
```

## 🔒 Segurança

**⚠️ IMPORTANTE:**

1. **Use HTTPS em produção**: Basic Auth envia credenciais em Base64 (não criptografado)
2. **✅ Senhas hasheadas**: Este exemplo usa bcrypt (cost=10) para hash seguro
3. **✅ Variáveis de ambiente**: Credenciais sensíveis via `.env` (nunca hardcoded)
4. **Implemente rate limiting**: Previna ataques de força bruta
5. **Use logs de auditoria**: Monitore tentativas de login

### Práticas de Segurança Implementadas

✅ **Bcrypt para senhas**: Todas as senhas são hasheadas com bcrypt antes de serem armazenadas
✅ **Variáveis de ambiente**: Credenciais do banco via `.env` (não comitado no Git)
✅ **Validação de secrets**: Código verifica se `DB_PASSWORD` está configurado
✅ **Logging de auth**: Tentativas de login são registradas para auditoria
✅ **Senha nunca exposta**: Campo `password` tem tag `json:"-"` para não ser serializado

### Checklist de Segurança para Produção

- [ ] Usar HTTPS (TLS/SSL) obrigatório
- [ ] Configurar rate limiting (ex: 5 tentativas por minuto)
- [ ] Implementar bloqueio temporário após N falhas
- [ ] Adicionar MFA (Multi-Factor Authentication) se possível
- [ ] Rotacionar credenciais regularmente
- [ ] Monitorar logs de acesso suspeito
- [ ] Usar senhas fortes (mínimo 12 caracteres)
- [ ] Implementar timeout de sessão
- [ ] Adicionar CORS apropriado para produção
- [ ] Usar secrets manager (AWS Secrets Manager, Azure Key Vault, etc)

## 🧪 Testes

```bash
# Teste 1: Endpoint público
curl http://localhost:3000/api/v1/info

# Teste 2: Endpoint protegido sem auth (deve falhar)
curl http://localhost:3000/api/v1/Users

# Teste 3: Endpoint protegido com auth (deve funcionar)
curl -u admin:admin123 http://localhost:3000/api/v1/Users

# Teste 4: Entidade somente leitura (POST deve falhar)
curl -u admin:admin123 -X POST http://localhost:3000/api/v1/Users \
  -H "Content-Type: application/json" \
  -d '{"username":"novo","password":"123"}'
```

## 📚 Recursos

- [RFC 7617 - HTTP Basic Authentication](https://tools.ietf.org/html/rfc7617)
- [OData v4 Specification](https://www.odata.org/documentation/)
- [Go-Data Documentation](../../README.md)

## 💼 Casos de Uso

Basic Auth é ideal para:

- ✅ APIs internas entre servidores
- ✅ Scripts e automações
- ✅ Integrações simples
- ✅ Ambientes com HTTPS garantido
- ✅ Prototipagem rápida

Para APIs públicas com frontend, considere usar [JWT](../jwt/).

