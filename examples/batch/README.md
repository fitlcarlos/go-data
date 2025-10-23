# Exemplo: OData v4 $batch

Este exemplo demonstra o uso de **batch requests** do OData v4, permitindo executar múltiplas operações em uma única requisição HTTP.

## 🎯 O que é $batch?

O `$batch` é uma funcionalidade do OData v4 que permite:
- **Combinar múltiplas requisições** em uma única chamada HTTP
- **Executar transações** (changesets) - tudo ou nada
- **Reduzir latência** e overhead de rede
- **Melhorar performance** em operações bulk

## 🗄️ Estrutura

Este exemplo usa SQLite in-memory com três entidades:
- **Products**: Produtos do catálogo
- **Categories**: Categorias de produtos
- **Orders**: Pedidos de produtos

## 🚀 Como Executar

```bash
cd examples/batch
go run main.go
```

O servidor iniciará em `http://localhost:3000`.

## 📋 Tipos de Batch

### 1. Múltiplas Leituras

Execute várias requisições GET independentes:

```bash
curl -X POST http://localhost:3000/api/v1/$batch \
  -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
  --data-binary @- << 'EOF'
--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products?$top=5 HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Categories HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Orders?$filter=status eq 'pending' HTTP/1.1
Host: localhost:3000


--batch_boundary--
EOF
```

### 2. Changeset Transacional

Execute múltiplas operações de escrita que devem todas ter sucesso ou falhar juntas:

```bash
curl -X POST http://localhost:3000/api/v1/$batch \
  -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
  --data-binary @- << 'EOF'
--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Products HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "name": "Novo Produto",
  "description": "Criado via batch",
  "price": 99.90,
  "stock": 10,
  "category_id": 1
}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /api/v1/Orders HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "product_id": 1,
  "quantity": 5,
  "total_price": 499.50,
  "status": "pending"
}

--changeset_boundary--

--batch_boundary--
EOF
```

### 3. Batch Misto

Combine leituras e changesets em uma única requisição:

```bash
curl -X POST http://localhost:3000/api/v1/$batch \
  -H "Content-Type: multipart/mixed; boundary=batch_boundary" \
  --data-binary @- << 'EOF'
--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products?$top=3 HTTP/1.1
Host: localhost:3000


--batch_boundary
Content-Type: multipart/mixed; boundary=changeset_boundary

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Categories HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "name": "Nova Categoria",
  "description": "Categoria via batch",
  "active": true
}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /api/v1/Products(1) HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{
  "price": 1299.99
}

--changeset_boundary--

--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Orders HTTP/1.1
Host: localhost:3000


--batch_boundary--
EOF
```

## 🔧 Anatomia de um Batch Request

### Estrutura Geral

```
POST /api/v1/$batch
Content-Type: multipart/mixed; boundary=<batch_boundary>

--<batch_boundary>
[Parte 1: Requisição ou Changeset]

--<batch_boundary>
[Parte 2: Requisição ou Changeset]

--<batch_boundary>--
```

### Requisição Individual (GET)

```
--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET /api/v1/Products HTTP/1.1
Host: localhost:3000

[linha vazia marca fim dos headers]

--batch_boundary
```

### Changeset (Transacional)

```
--batch_boundary
Content-Type: multipart/mixed; boundary=<changeset_boundary>

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /api/v1/Products HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{"name":"Product 1"}

--changeset_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /api/v1/Orders HTTP/1.1
Host: localhost:3000
Content-Type: application/json

{"product_id": 1}

--changeset_boundary--

--batch_boundary
```

## 📊 Content-ID

O `Content-ID` permite referenciar operações dentro do batch:

```
Content-ID: 1

POST /api/v1/Products HTTP/1.1
...

# Usar $1 para referenciar o resultado da operação com Content-ID: 1
GET /api/v1/Products($1)/Orders HTTP/1.1
```

## ⚡ Benefícios

### Performance
- **Reduz round-trips**: Uma requisição HTTP vs múltiplas
- **Menor latência**: Menos overhead de rede
- **Bulk operations**: Ideal para importação de dados

### Transações
- **Atomicidade**: Changesets garantem "tudo ou nada"
- **Consistência**: Operações interdependentes
- **Rollback automático**: Se uma operação falha, todas revertem

### Eficiência
- **Menos conexões**: Reduz uso de recursos do servidor
- **Batching inteligente**: Agrupa operações similares
- **Throughput maior**: Mais operações por segundo

## 🔒 Limitações

Por padrão, o servidor tem limites de segurança:

```go
config := &odata.BatchConfig{
    MaxOperations:      100,          // Máximo de operações por batch
    MaxChangesets:      10,           // Máximo de changesets
    Timeout:            30 * time.Second,
    EnableTransactions: true,
}
```

## 🎓 Casos de Uso

### 1. Importação de Dados
Inserir múltiplos registros relacionados em uma transação:
- Criar categorias
- Criar produtos nessas categorias
- Criar pedidos para esses produtos

### 2. Dashboard Loading
Carregar todos os dados necessários para um dashboard em uma requisição:
- Produtos
- Categorias
- Pedidos recentes
- Estatísticas

### 3. Operações Complexas
Executar múltiplas operações que dependem umas das outras:
- Criar produto
- Atualizar estoque
- Criar pedido
- Atualizar inventário

## 📚 Referências

- [OData v4 Batch Specification](http://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_BatchRequests)
- [RFC 2046 - Multipart Media Types](https://www.rfc-editor.org/rfc/rfc2046.html)

## 🐛 Troubleshooting

### Erro: "boundary parameter is required"
- Certifique-se de incluir o boundary no Content-Type
- Formato correto: `Content-Type: multipart/mixed; boundary=batch_boundary`

### Erro: "invalid multipart format"
- Verifique se há duas hífens antes do boundary: `--batch_boundary`
- Verifique se há dois hífens extras no final: `--batch_boundary--`
- Certifique-se de ter linhas vazias entre headers e body

### Changeset não é transacional
- Verifique se `EnableTransactions: true` na configuração
- Certifique-se de que o banco de dados suporta transações
- Verifique logs do servidor para erros

## 💡 Dicas

1. **Use changesets para escrita**: Sempre use changesets para operações POST/PUT/PATCH/DELETE
2. **Content-ID é opcional**: Mas muito útil para referenciar operações
3. **Ordene as operações**: Coloque operações independentes primeiro
4. **Teste com curl**: Use arquivos com `--data-binary @batch_request.txt`
5. **Monitore o tamanho**: Batches muito grandes podem causar timeout

## 🔍 Ver Também

- [`examples/basic/`](../basic/) - Exemplo básico de OData
- [`examples/jwt/`](../jwt/) - Autenticação JWT
- [`pkg/odata/batch.go`](../../pkg/odata/batch.go) - Implementação do $batch

