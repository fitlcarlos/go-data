# GoData - Biblioteca OData para Go

Uma biblioteca Go completa para implementar APIs OData v4 com resposta JSON e suporte a múltiplos bancos de dados.

## Características

- **Padrão OData v4**: Implementação completa do protocolo OData
- **Resposta JSON**: Dados retornados exclusivamente em formato JSON
- **Múltiplos Bancos**: Suporte para PostgreSQL, Oracle e MySQL
- **Mapeamento Automático**: Sistema de mapeamento baseado em tags de struct
- **Tipos Nullable**: Suporte completo a valores null
- **Relacionamentos Bidirecionais**: Suporte a association e manyAssociation
- **Consultas Avançadas**: Filtros, ordenação, paginação e seleção de campos
- **Campos Computados**: Suporte a $compute para cálculos em tempo real
- **Busca Textual**: Suporte a $search para busca em texto
- **Geração de Metadados**: Metadados JSON automáticos
- **Testes Unitários**: Cobertura completa de testes

## Instalação

```bash
go get github.com/godata/odata
```

## Mapeamento Automático de Entidades

### Sistema de Tags

O GoData utiliza um sistema avançado de tags de struct para definir metadados automaticamente:

```go
type User struct {
    TableName string           `table:"users;schema=public"`
    ID        int64            `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_user_id"`
    Nome      string           `json:"nome" column:"nome" prop:"[required, Unique]; length:100"`
    Email     string           `json:"email" column:"email" prop:"[required, Unique]; length:255"`
    Idade     nullable.Int64   `json:"idade" column:"idade"`
    Ativo     bool             `json:"ativo" column:"ativo" prop:"[required]; default"`
    DtInc     time.Time        `json:"dt_inc" column:"dt_inc" prop:"[required, NoUpdate]; default"`
    DtAlt     nullable.Time    `json:"dt_alt" column:"dt_alt"`
    Salario   nullable.Float64 `json:"salario" column:"salario" prop:"precision:10; scale:2"`
    
    // Relacionamentos
    Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
}
```

### Tags Disponíveis

#### Tag `table`
Define o nome da tabela e schema no banco de dados:
```go
TableName string `table:"users;schema=public"`
TableName string `table:"fab_operacao;schema=nbs"`
```

#### Tag `json`
Define o nome do campo na serialização JSON:
```go
Nome string `json:"nome"`
Email string `json:"email"`
```

#### Tag `column`
Define o nome da coluna no banco de dados:
```go
Nome string `column:"nome"`
Email string `column:"email"`
```

#### Tag `prop`
Define propriedades avançadas do campo:
```go
// Flags disponíveis: required, NoInsert, NoUpdate, Unique, Lazy
ID    int64  `prop:"[required]"`                    // Campo obrigatório
Nome  string `prop:"[required, Unique]; length:100"` // Obrigatório e único
DtInc time.Time `prop:"[required, NoUpdate]; default"` // Não pode ser alterado
Email string `prop:"[required, Unique]; length:255"` // Obrigatório e único
```

#### Tag `primaryKey`
Define chaves primárias e geradores de ID:
```go
// Tipos de geradores disponíveis: none, sequence, identity, guid, uuid38, uuid36, uuid32, smartGuid
`primaryKey:"idGenerator:none"`                    // Sem gerador automático
`primaryKey:"idGenerator:sequence;name=seq_user_id"` // Sequência Oracle/PostgreSQL
`primaryKey:"idGenerator:identity"`                // Auto incremento MySQL
`primaryKey:"idGenerator:guid"`                    // GUID
`primaryKey:"idGenerator:uuid36"`                  // UUID 36 caracteres
`primaryKey:"idGenerator:uuid32"`                  // UUID 32 caracteres
`primaryKey:"idGenerator:smartGuid"`               // Smart GUID otimizado
```

#### Tag `association`
Define relacionamentos simples (1:1 ou N:1):
```go
// Relacionamento N:1 (muitos para um)
User *User `json:"User" association:"foreignKey:user_id; references:id"`
```

#### Tag `manyAssociation`
Define relacionamentos múltiplos (1:N ou N:N):
```go
// Relacionamento 1:N (um para muitos)
Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id"`

// Relacionamento N:N com tabela de junção
Tags []Tag `json:"Tags" manyAssociation:"foreignKey:post_id; references:id; joinTable:post_tags; joinColumn:post_id; inverseJoinColumn:tag_id"`
```

#### Tag `cascade`
Define ações em cascata para relacionamentos:
```go
// Flags disponíveis: SaveUpdate, Remove, Refresh, RemoveOrphan
`cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
`cascade:"[SaveUpdate, Refresh]"`
```

### Relacionamentos Bidirecionais

O GoData suporta relacionamentos bidirecionais usando `association` e `manyAssociation`:

```go
// Entidade User (lado 1)
type User struct {
    TableName string    `table:"users;schema=public"`
    ID        int64     `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_user_id"`
    Nome      string    `json:"nome" column:"nome" prop:"[required]; length:100"`
    Email     string    `json:"email" column:"email" prop:"[required, Unique]; length:255"`
    Ativo     bool      `json:"ativo" column:"ativo" prop:"[required]; default"`
    DtInc     time.Time `json:"dt_inc" column:"dt_inc" prop:"[required, NoUpdate]; default"`
    
    // Relacionamento 1:N - Um usuário tem muitos pedidos
    Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
}

// Entidade Order (lado N)
type Order struct {
    TableName string    `table:"orders;schema=public"`
    ID        int64     `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_order_id"`
    UserID    int64     `json:"user_id" column:"user_id" prop:"[required]"`
    Total     float64   `json:"total" column:"total" prop:"[required]; precision:10; scale:2"`
    DtPedido  time.Time `json:"dt_pedido" column:"dt_pedido" prop:"[required]"`
    
    // Relacionamento N:1 - Muitos pedidos pertencem a um usuário
    User  *User       `json:"User" association:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Refresh]"`
    // Relacionamento 1:N - Um pedido tem muitos itens
    Items []OrderItem `json:"Items" manyAssociation:"foreignKey:order_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
}

// Entidade OrderItem (lado N)
type OrderItem struct {
    TableName string `table:"order_items;schema=public"`
    ID        int64  `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_order_item_id"`
    OrderID   int64  `json:"order_id" column:"order_id" prop:"[required, NoUpdate]"`
    ProductID int64  `json:"product_id" column:"product_id" prop:"[required]"`
    Quantity  int32  `json:"quantity" column:"quantity" prop:"[required]"`
    Price     float64 `json:"price" column:"price" prop:"[required]; precision:8; scale:2"`
    
    // Relacionamentos N:1 
    Order   *Order   `json:"Order" association:"foreignKey:order_id; references:id" cascade:"[SaveUpdate, Refresh]"`
    Product *Product `json:"Product" association:"foreignKey:product_id; references:id" cascade:"[SaveUpdate, Refresh]"`
}

// Entidade Product
type Product struct {
    TableName string           `table:"products;schema=public"`
    ID        int64            `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_product_id"`
    Nome      string           `json:"nome" column:"nome" prop:"[required]; length:100"`
    Descricao nullable.String  `json:"descricao" column:"descricao" prop:"length:500"`
    Preco     float64          `json:"preco" column:"preco" prop:"[required]; precision:8; scale:2"`
    Ativo     bool             `json:"ativo" column:"ativo" prop:"[required]; default"`
    
    // Relacionamento 1:N - Um produto pode estar em muitos itens de pedido
    OrderItems []OrderItem `json:"OrderItems" manyAssociation:"foreignKey:product_id; references:id" cascade:"[SaveUpdate, Refresh]"`
}
```

### Relacionamentos N:N com Tabela de Junção

Para relacionamentos muitos-para-muitos, use a tabela de junção:

```go
// Entidade Post
type Post struct {
    TableName string `table:"posts;schema=public"`
    ID        int64  `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_post_id"`
    Title     string `json:"title" column:"title" prop:"[required]; length:200"`
    Content   string `json:"content" column:"content" prop:"[required]"`
    
    // Relacionamento N:N - Posts podem ter muitas tags
    Tags []Tag `json:"Tags" manyAssociation:"foreignKey:post_id; references:id; joinTable:post_tags; joinColumn:post_id; inverseJoinColumn:tag_id" cascade:"[SaveUpdate, Refresh]"`
}

// Entidade Tag
type Tag struct {
    TableName string `table:"tags;schema=public"`
    ID        int64  `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_tag_id"`
    Name      string `json:"name" column:"name" prop:"[required, Unique]; length:50"`
    Color     string `json:"color" column:"color" prop:"length:7"` // Hex color
    
    // Relacionamento N:N - Tags podem estar em muitos posts
    Posts []Post `json:"Posts" manyAssociation:"foreignKey:tag_id; references:id; joinTable:post_tags; joinColumn:tag_id; inverseJoinColumn:post_id" cascade:"[SaveUpdate, Refresh]"`
}
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

### Exemplo Completo com Relacionamentos

```go
package main

import (
    "log"
    "time"
    "github.com/godata/odata/pkg/odata"
    "github.com/godata/odata/pkg/nullable"
)

// Exemplo de estrutura completa com relacionamentos bidirecionais
type User struct {
    TableName string           `table:"users;schema=public"`
    ID        int64            `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_user_id"`
    Nome      string           `json:"nome" column:"nome" prop:"[required]; length:100"`
    Email     string           `json:"email" column:"email" prop:"[required, Unique]; length:255"`
    Idade     nullable.Int64   `json:"idade" column:"idade"`
    Ativo     bool             `json:"ativo" column:"ativo" prop:"[required]; default"`
    DtInc     time.Time        `json:"dt_inc" column:"dt_inc" prop:"[required, NoUpdate]; default"`
    DtAlt     nullable.Time    `json:"dt_alt" column:"dt_alt"`
    Salario   nullable.Float64 `json:"salario" column:"salario" prop:"precision:10; scale:2"`
    
    // Relacionamentos
    Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
    Profile *UserProfile `json:"Profile" association:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Remove, Refresh]"`
}

type UserProfile struct {
    TableName string         `table:"user_profiles;schema=public"`
    ID        int64          `json:"id" column:"id" prop:"[required]" primaryKey:"idGenerator:sequence;name=seq_profile_id"`
    UserID    int64          `json:"user_id" column:"user_id" prop:"[required, Unique]"`
    Bio       nullable.String `json:"bio" column:"bio" prop:"length:1000"`
    Avatar    nullable.String `json:"avatar" column:"avatar" prop:"length:255"`
    
    // Relacionamento 1:1 bidirecional
    User *User `json:"User" association:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Refresh]"`
}

func main() {
    // Registro automático no servidor
    server := odata.NewServer(provider)
    server.RegisterEntity("Users", User{})
    server.RegisterEntity("UserProfiles", UserProfile{})
    server.RegisterEntity("Orders", Order{})
    server.RegisterEntity("OrderItems", OrderItem{})
    server.RegisterEntity("Products", Product{})
    
    // Registro em lote
    entities := map[string]interface{}{
        "Users":        User{},
        "UserProfiles": UserProfile{},
        "Orders":       Order{},
        "OrderItems":   OrderItem{},
        "Products":     Product{},
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
Retorna os metadados da API em formato JSON.

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

### Expansão de Relacionamentos ($expand)
```
GET /odata/Users?$expand=Orders
GET /odata/Users?$expand=Orders,Profile
GET /odata/Users?$expand=Orders($select=id,total;$filter=total gt 100)
GET /odata/Orders?$expand=User,Items($expand=Product)
```

### Contagem ($count)
```
GET /odata/Users?$count=true
GET /odata/Users/$count
```

### Campos Computados ($compute)
```
GET /odata/Orders?$compute=total mul 0.1 as tax
GET /odata/Users?$compute=tolower(nome) as nome_lower
GET /odata/Products?$compute=preco mul 1.2 as preco_com_taxa
```

### Busca Textual ($search)
```
GET /odata/Users?$search=João
GET /odata/Products?$search="produto especial"
GET /odata/Users?$search=João AND Silva
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
- `tolower(field)` - Converte para minúsculas
- `toupper(field)` - Converte para maiúsculas
- `trim(field)` - Remove espaços
- `length(field)` - Comprimento da string

### Funções Matemáticas
- `round(field)` - Arredonda
- `floor(field)` - Arredonda para baixo
- `ceiling(field)` - Arredonda para cima
- `abs(field)` - Valor absoluto

### Operadores Aritméticos (para $compute)
- `add` - Adição (+)
- `sub` - Subtração (-)
- `mul` - Multiplicação (*)
- `div` - Divisão (/)
- `mod` - Módulo (%)

### Lógicos
- `and` - E lógico
- `or` - Ou lógico
- `not` - Negação

## Cascata de Operações

### Flags de Cascata Disponíveis

- `SaveUpdate` - Salva/atualiza entidades relacionadas automaticamente
- `Remove` - Remove entidades relacionadas quando a entidade principal é removida
- `Refresh` - Atualiza entidades relacionadas quando a entidade principal é atualizada
- `RemoveOrphan` - Remove entidades órfãs (apenas para manyAssociation)

### Exemplo de Uso

```go
type User struct {
    Orders []Order `json:"Orders" manyAssociation:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Remove, Refresh, RemoveOrphan]"`
}

type Order struct {
    User *User `json:"User" association:"foreignKey:user_id; references:id" cascade:"[SaveUpdate, Refresh]"`
}
```

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
- [Go Database/SQL Tutorial](https://golang.org/doc/tutorial/database-access) 