# Go-Data Security Guide

## Práticas de Segurança Implementadas

### 1. Proteção contra SQL Injection

O Go-Data utiliza **Prepared Statements** em todas as queries SQL, usando `sql.Named` para parametriz

ação segura.

#### ✅ Implementação Segura

```go
// ✅ SEGURO - Usa prepared statements
namedArgs := NewNamedArgs(dialect)
placeholder := namedArgs.AddArg(userInput)
query := fmt.Sprintf("SELECT * FROM users WHERE name = %s", placeholder)
```

#### ❌ Evite

```go
// ❌ VULNERÁVEL - Concatenação direta
query := "SELECT * FROM users WHERE name = '" + userInput + "'"
```

#### Validação de Inputs

Todos os inputs OData passam por validação antes do parsing:

- **$filter**: Valida tamanho máximo e detecta padrões de SQL injection
- **$search**: Valida tamanho e caracteres permitidos
- **$select**: Valida nomes de propriedades
- **$orderby**: Valida propriedades e direções (asc/desc)
- **$top**: Limita valor máximo (padrão: 1000)
- **$skip**: Limita valor máximo (padrão: 100000)
- **$expand**: Limita profundidade (padrão: 5 níveis)

### 2. Input Validation

#### Configuração

```go
config := &odata.ValidationConfig{
    MaxFilterLength:      5000,  // 5KB
    MaxSearchLength:      1000,  // 1KB
    MaxTopValue:          1000,  // máximo 1000 registros
    MaxExpandDepth:       5,     // máximo 5 níveis
    AllowedPropertyChars: `^[a-zA-Z0-9_\.]+$`,
    EnableXSSProtection:  true,
}
```

#### Padrões Detectados

**SQL Injection:**
- `UNION`, `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `DROP`
- Comentários SQL: `--`, `#`, `/*`, `*/`
- `OR 1=1`, `AND 1=1`
- `OR 'x'='x'`

**XSS:**
- `<script>`, `<iframe>`, `<object>`, `<embed>`
- `javascript:`
- Event handlers: `onclick=`, `onload=`, etc

### 3. Security Headers

Headers de segurança são automaticamente adicionados a todas as respostas:

```http
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; ...
Strict-Transport-Security: max-age=31536000; includeSubDomains
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), ...
```

#### Configurações

**Padrão (Balanceada):**
```go
config := odata.DefaultSecurityHeadersConfig()
```

**Estrita (Máxima Segurança):**
```go
config := odata.StrictSecurityHeadersConfig()
```

**Relaxada (Desenvolvimento):**
```go
config := odata.RelaxedSecurityHeadersConfig()
```

**Desabilitar:**
```go
config := odata.DisableSecurityHeaders()
```

### 4. Rate Limiting

Rate limiting está **habilitado por padrão** para proteção contra DDoS e abuso.

#### Configuração Padrão

```go
config := &odata.RateLimitConfig{
    Enabled:           true,  // ⚠️ HABILITADO POR PADRÃO
    RequestsPerMinute: 100,
    BurstSize:         20,
    WindowSize:        time.Minute,
}
```

#### Desabilitar (não recomendado)

```go
server := odata.NewServer()
server.GetConfig().RateLimitConfig.Enabled = false
```

### 5. Audit Logging

Registra todas operações críticas:

- CREATE, UPDATE, DELETE (operações de escrita)
- AUTH_SUCCESS, AUTH_FAILURE, AUTH_LOGOUT
- UNAUTHORIZED (tentativas de acesso não autorizado)

#### Configuração

```go
config := &odata.AuditLogConfig{
    Enabled:  true,
    LogType:  "file",       // "file", "stdout", "stderr"
    FilePath: "audit.log",
    Format:   "json",       // "json" ou "text"
    IncludeSensitiveData: false,  // ⚠️ Não habilite em produção
}
```

#### Exemplo de Log Entry

```json
{
  "timestamp": "2025-10-17T10:30:45Z",
  "user_id": "123",
  "username": "john.doe",
  "ip": "192.168.1.100",
  "method": "POST",
  "path": "/odata/Users",
  "entity_name": "Users",
  "operation": "CREATE",
  "entity_id": "456",
  "success": true,
  "duration_ms": 45,
  "user_agent": "Mozilla/5.0...",
  "request_id": "abc-123"
}
```

### 6. Autenticação e Autorização

#### JWT

- Usa algoritmo HS256 por padrão
- Tokens assinados com secret key
- Expiração configurável
- Refresh tokens para renovação segura

**⚠️ Práticas Recomendadas:**
- Use secrets fortes (mínimo 32 caracteres)
- Armazene secrets em variáveis de ambiente
- Rotacione secrets regularmente
- Use HTTPS em produção

```go
jwtAuth := odata.NewJwtAuth(&odata.JWTConfig{
    SecretKey: os.Getenv("JWT_SECRET"), // ✅ De variável de ambiente
    ExpiresIn: 1 * time.Hour,
})
```

#### Basic Auth

- Credenciais em Base64
- **EXIGE HTTPS em produção**
- Use bcrypt para senhas

```go
// ❌ NÃO use senha plain text
password := "senha123"

// ✅ Use bcrypt
hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
```

### 7. HTTPS/TLS

**OBRIGATÓRIO em produção!**

```go
server := odata.NewServer(&odata.Config{
    TLS: &odata.TLSConfig{
        Enabled:  true,
        CertFile: "/path/to/cert.pem",
        KeyFile:  "/path/to/key.pem",
    },
})
```

### 8. CORS

Configure CORS apropriadamente:

```go
// ❌ Não use em produção
AllowedOrigins: []string{"*"}

// ✅ Especifique domínios permitidos
AllowedOrigins: []string{
    "https://app.example.com",
    "https://admin.example.com",
}
```

## Checklist de Segurança para Produção

- [ ] HTTPS/TLS habilitado
- [ ] JWT secrets fortes e em variáveis de ambiente
- [ ] Rate limiting habilitado e configurado
- [ ] Audit logging habilitado
- [ ] Security headers habilitados
- [ ] CORS configurado com origens específicas
- [ ] Input validation habilitada
- [ ] Senhas usando bcrypt (nunca plain text)
- [ ] Backup regular de audit logs
- [ ] Monitoramento de tentativas de autenticação falhadas
- [ ] Monitoramento de rate limit violations
- [ ] Firewall configurado
- [ ] Banco de dados com autenticação forte
- [ ] Princípio de menor privilégio em permissões de BD

## Relatando Vulnerabilidades

Se você descobrir uma vulnerabilidade de segurança, por favor **NÃO** abra uma issue pública.

Envie um email para: security@example.com (substitua pelo email real)

Incluindo:
- Descrição da vulnerabilidade
- Passos para reproduzir
- Impacto potencial
- Sugestões de correção (se houver)

## Atualizações de Segurança

Mantenha o Go-Data atualizado. Verificamos e atualizamos dependências regularmente.

```bash
go get -u github.com/fitlcarlos/go-data
```

## Recursos Adicionais

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Checklist](https://github.com/Checkmarx/Go-SCP)
- [OData Security Best Practices](https://www.odata.org/documentation/)

## Licença

Este documento está licenciado sob a mesma licença do Go-Data.

