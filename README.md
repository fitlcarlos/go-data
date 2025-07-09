# GoData - Biblioteca OData para Go

Uma biblioteca Go completa para implementar APIs OData v4 com resposta JSON e suporte a múltiplos bancos de dados.

## Características

- **Padrão OData v4**: Implementação completa do protocolo OData
- **Resposta JSON**: Dados retornados exclusivamente em formato JSON
- **Múltiplos Bancos**: Suporte para PostgreSQL, Oracle e MySQL
- **Mapeamento Automático**: Sistema de mapeamento baseado em tags de struct (inspirado no TMS Aurelius)
- **Tipos Nullable**: Suporte completo a valores null
- **Relacionamentos**: Suporte a foreign keys e navegação entre entidades
- **Consultas Avançadas**: Filtros, ordenação, paginação e seleção de campos
- **Geração de Metadados**: Metadados XML automáticos
- **Testes Unitários**: Cobertura completa de testes

## Instalação

```bash
go get github.com/godata/odata
```

## Mapeamento Automático de Entidades

### Sistema de Tags

O GoData utiliza um sistema de tags de struct similar ao TMS Aurelius para definir metadados automaticamente:

```go
type User struct {
    ID      int64           `json:"id" column:"id" odata:"not null" primaryKey:"idGenerator:sequence;name=seq_user_id"`
    Nome    string          `json:"nome" column:"nome" odata:"not null; length:100"`
    Email   string          `json:"email" column:"email" odata:"not null; length:255"`
    Idade   nullable.Int64  `json:"idade" column:"idade"`
    Ativo   bool            `json:"ativo" column:"ativo" odata:"not null; default"`
    DtInc   time.Time       `json:"dt_inc" column:"dt_inc" odata:"not null; default"`
    DtAlt   nullable.Time   `json:"dt_alt" column:"dt_alt"`
    Salario nullable.Float64 `json:"salario" column:"salario" odata:"precision:10; scale:2"`
}
```

### Tags Disponíveis

#### Tag `json`
Define o nome do campo na serialização JSON:
```go
Nome string `json:"nome"`
```

#### Tag `column`
Define o nome da coluna no banco de dados:
```go
Nome string `column:"nome"`
```

#### Tag `odata`
Define atributos OData específicos:
```go
// Exemplos de atributos odata:
`odata:"not null"`           // Campo obrigatório
`odata:"null"`               // Campo opcional
`odata:"default"`            // Possui valor padrão
`odata:"length:100"`         // Tamanho máximo
`odata:"precision:10; scale:2"` // Precisão e escala para decimais
```

#### Tag `primaryKey`
Define chaves primárias e geradores de ID:
```go
// Chave primária sem gerador automático
`primaryKey:"idGenerator:none"`

// Chave primária com sequência
`primaryKey:"idGenerator:sequence;name=seq_user_id"`

// Chave primária com auto incremento
`primaryKey:"idGenerator:identity"`
```

#### Tag `foreignKey`
Define relacionamentos entre entidades:
```go
// Relacionamento simples
MenuPai Menu `foreignKey:"id_parent;references:id"`

// Relacionamento com ações em cascade
MenuPermissao []MenuPermissao `foreignKey:"id_menu;references:id;OnDelete:CASCADE"`
```

### Tipos Nullable

O GoData fornece tipos nullable personalizados para campos opcionais:

```go
import "github.com/godata/odata/pkg/nullable"

type User struct {
    ID      int64           `json:"id"`
    Nome    string          `json:"nome"`
    Idade   nullable.Int64  `json:"idade"`    // Pode ser null
    Salario nullable.Float64 `json:"salario"` // Pode ser null
    DtAlt   nullable.Time   `json:"dt_alt"`   // Pode ser null
}
```

#### Tipos Nullable Disponíveis

- `nullable.Int64` - Inteiro de 64 bits
- `nullable.String` - String
- `nullable.Bool` - Booleano
- `nullable.Time` - Data/hora
- `nullable.Float64` - Número decimal

#### Usando Tipos Nullable

```go
// Criar valor válido
idade := nullable.NewInt64(25)

// Criar valor null
idade := nullable.NullInt64()

// Verificar se é válido
if idade.Valid {
    fmt.Println("Idade:", idade.Val)
}

// Serialização JSON automática
// Valor válido: {"idade": 25}
// Valor null: {"idade": null}
```

### Relacionamentos

O GoData suporte relacionamentos entre entidades usando foreign keys:

```go
type Menu struct {
    ID            int64                 `json:"id" primaryKey:"idGenerator:none"`
    IdParent      nullable.Int64        `json:"id_parent" column:"id_parent"`
    Descricao     string                `json:"descricao" odata:"not null"`
    
    // Relacionamentos
    MenuPermissao []MenuPermissao       `json:"MenuPermissao" foreignKey:"id_menu;references:id;OnDelete:CASCADE"`
    MenuFilho     []Menu                `json:"Menu" foreignKey:"id_parent;references:id;OnDelete:CASCADE"`
}

type MenuPermissao struct {
    ID          int64         `json:"id" primaryKey:"idGenerator:sequence"`
    IdPermissao int64         `json:"id_permissao" odata:"not null"`
    IdMenu      int64         `json:"id_menu" odata:"not null"`
    
    // Relacionamentos
    Permissao   Permissao     `json:"Permissao" foreignKey:"id;references:id_permissao"`
    Menu        Menu          `json:"Menu" foreignKey:"id;references:id_menu"`
}
```

### Exemplo Completo

```go
package main

import (
    "log"
    "time"
    "github.com/godata/odata/pkg/odata"
    "github.com/godata/odata/pkg/nullable"
)

type User struct {
    ID      int64           `json:"id" column:"id" odata:"not null" primaryKey:"idGenerator:sequence;name=seq_user_id"`
    Nome    string          `json:"nome" column:"nome" odata:"not null; length:100"`
    Email   string          `json:"email" column:"email" odata:"not null; length:255"`
    Idade   nullable.Int64  `json:"idade" column:"idade"`
    Ativo   bool            `json:"ativo" column:"ativo" odata:"not null; default"`
    DtInc   time.Time       `json:"dt_inc" column:"dt_inc" odata:"not null; default"`
    DtAlt   nullable.Time   `json:"dt_alt" column:"dt_alt"`
    Salario nullable.Float64 `json:"salario" column:"salario" odata:"precision:10; scale:2"`
}

func main() {
    // Mapeamento automático
    metadata, err := odata.MapEntityFromStruct(User{})
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Entidade: %s", metadata.Name)
    log.Printf("Tabela: %s", metadata.TableName)
    log.Printf("Chaves: %v", metadata.Keys)
    
    // Registro automático no servidor
    server := odata.NewServer(provider)
    if err := server.RegisterEntity("Users", User{}); err != nil {
        log.Fatal(err)
    }
    
    // Registro em lote
    entities := map[string]interface{}{
        "Users":    User{},
        "Products": Product{},
    }
    
    if err := server.AutoRegisterEntities(entities); err != nil {
        log.Fatal(err)
    }
}
```

## Exemplo de Uso Básico

```go
package main

import (
    "database/sql"
    "log"
    "net/http"
    
    "github.com/godata/odata/pkg/odata"
    "github.com/godata/odata/pkg/providers"
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
    server := odata.NewServer(provider)
    
    // Registra entidades usando mapeamento automático
    entities := map[string]interface{}{
        "Users":    User{},
        "Products": Product{},
    }
    
    if err := server.AutoRegisterEntities(entities); err != nil {
        log.Fatal(err)
    }
    
    // Inicia servidor
    log.Println("Servidor iniciado em http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", server.GetHandler()))
}
```

## Bancos de Dados Suportados

### PostgreSQL
```go
import (
    "github.com/godata/odata/pkg/providers"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, err := sql.Open("pgx", "postgres://user:password@localhost/database")
provider := providers.NewPostgreSQLProvider(db)
```

### Oracle
```go
import (
    "github.com/godata/odata/pkg/providers"
    _ "github.com/sijms/go-ora/v2"
)

db, err := sql.Open("oracle", "oracle://user:password@localhost:1521/xe")
provider := providers.NewOracleProvider(db)
```

### MySQL
```go
import (
    "github.com/godata/odata/pkg/providers"
    _ "github.com/go-sql-driver/mysql"
)

db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/database")
provider := providers.NewMySQLProvider(db)
```

## Endpoints OData

### Service Document
```
GET /odata/
```
Retorna o documento de serviço em formato JSON.

### Metadados
```
GET /odata/$metadata
```
Retorna os metadados da API em formato JSON (anteriormente XML).

Exemplo de resposta:
```json
{
  "@odata.context": "$metadata",
  "@odata.version": "4.0",
  "entities": [
    {
      "name": "Users",
      "namespace": "Default",
      "keys": ["ID"],
      "properties": [
        {
          "name": "ID",
          "type": "Edm.Int64",
          "nullable": false,
          "isKey": true,
          "hasDefault": false
        },
        {
          "name": "Name",
          "type": "Edm.String",
          "nullable": false,
          "maxLength": 100,
          "isKey": false,
          "hasDefault": false
        }
      ]
    }
  ],
  "entitySets": [
    {
      "name": "Users",
      "entityType": "Default.Users",
      "kind": "EntitySet",
      "url": "Users"
    }
  ]
}
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
  "email": "joao.santos@email.com",
  "idade": 31
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

## Consultas OData

### Filtros ($filter)
```
GET /odata/Users?$filter=idade gt 25
GET /odata/Users?$filter=nome eq 'João'
GET /odata/Users?$filter=contains(nome, 'Silva')
GET /odata/Users?$filter=startswith(email, 'joao')
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
GET /odata/Users?$select=*
```

### Contagem ($count)
```
GET /odata/Users?$count=true
GET /odata/Users/$count
```

## Operadores Suportados

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

### Lógicos
- `and` - E lógico
- `or` - Ou lógico

## Mapeamento de Tipos

| Tipo Go | Tipo OData | Tipo SQL |
|---------|------------|----------|
| `string` | `Edm.String` | `VARCHAR` |
| `int`, `int32` | `Edm.Int32` | `INT` |
| `int64` | `Edm.Int64` | `BIGINT` |
| `float32` | `Edm.Single` | `FLOAT` |
| `float64` | `Edm.Double` | `DOUBLE` |
| `bool` | `Edm.Boolean` | `BOOLEAN` |
| `time.Time` | `Edm.DateTimeOffset` | `TIMESTAMP` |
| `[]byte` | `Edm.Binary` | `BLOB` |
| `nullable.Int64` | `Edm.Int64` | `BIGINT NULL` |
| `nullable.String` | `Edm.String` | `VARCHAR NULL` |
| `nullable.Bool` | `Edm.Boolean` | `BOOLEAN NULL` |
| `nullable.Time` | `Edm.DateTimeOffset` | `TIMESTAMP NULL` |
| `nullable.Float64` | `Edm.Double` | `DOUBLE NULL` |

## Referências

- [Especificação OData v4](https://docs.oasis-open.org/odata/odata/v4.0/)