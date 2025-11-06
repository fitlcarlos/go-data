# 🔌 Pool de Conexões - Guia Completo

## 📋 Índice

- [Introdução](#introdução)
- [Parâmetros de Configuração](#parâmetros-de-configuração)
- [Como o Pool Funciona](#como-o-pool-funciona)
- [Configurações Recomendadas](#configurações-recomendadas)
- [Exemplos Práticos](#exemplos-práticos)
- [Troubleshooting](#troubleshooting)
- [Boas Práticas](#boas-práticas)

---

## 🎯 Introdução

O **Go-Data** utiliza o sistema de pool de conexões nativo do Go (`database/sql`) para gerenciar conexões com o banco de dados de forma eficiente e escalável.

Um pool de conexões mantém um conjunto reutilizável de conexões abertas com o banco, eliminando o overhead de criar e fechar conexões para cada requisição.

---

## ⚙️ Parâmetros de Configuração

### **1️⃣ `DB_MAX_OPEN_CONNS`**

**O que controla:** Número **MÁXIMO** de conexões abertas simultaneamente (em uso + ociosas).

| Valor | Comportamento |
|-------|---------------|
| `0` | **SEM LIMITE** ⚠️ (perigoso - pode esgotar recursos do sistema) |
| `> 0` | Limita o número total de conexões abertas |

**Padrão:** `10`

**Recomendado:** 
- Aplicações pequenas: `10-25`
- Aplicações médias: `25-50`
- Aplicações grandes: `50-100`

**Importante:** Se todas as conexões estiverem em uso, novas requisições **aguardarão** até que uma fique disponível.

---

### **2️⃣ `DB_MAX_IDLE_CONNS`**

**O que controla:** Número **MÁXIMO** de conexões **ociosas** (não em uso) mantidas no pool.

| Valor | Comportamento |
|-------|---------------|
| `0` | Nenhuma conexão fica ociosa - **TODAS são fechadas após o uso** |
| `> 0` | Mantém até N conexões ociosas no pool para reutilização rápida |

**Padrão:** `2`

**Recomendado:** 
- Deve ser **menor ou igual** a `DB_MAX_OPEN_CONNS`
- Valor típico: `20-40%` do `MaxOpenConns`
- Exemplo: Se `MaxOpenConns=25`, use `MaxIdleConns=5-10`

**Importante:** Conexões acima deste limite são **fechadas automaticamente** após o uso.

---

### **3️⃣ `DB_CONN_MAX_LIFETIME`**

**O que controla:** Tempo **MÁXIMO DE VIDA** de uma conexão desde que foi **CRIADA**.

| Valor | Comportamento |
|-------|---------------|
| `0` | Conexão **NUNCA expira** por idade ✅ |
| `> 0` | Conexão é **fechada e recriada** após o tempo especificado |

**Padrão:** `10m` (10 minutos)

**Recomendado:**
- **Desenvolvimento:** `0` (nunca expira)
- **Produção:** `0` ou `1h-24h`

**Quando usar valores maiores que 0:**
- Load balancers com timeout de conexão
- Bancos de dados que requerem rotação de conexões
- Problemas de memory leak no driver

**NÃO é:** Tempo limite de execução de query!

---

### **4️⃣ `DB_CONN_MAX_IDLE_TIME`**

**O que controla:** Tempo **MÁXIMO DE OCIOSIDADE** (sem uso) de uma conexão.

| Valor | Comportamento |
|-------|---------------|
| `0` | Usa o padrão do sistema operacional (geralmente `90s`) |
| `> 0` | Conexão ociosa é fechada após N segundos sem uso |

**Padrão:** `10m` (10 minutos)

**Recomendado:**
- **Desenvolvimento:** `0` (usa padrão do SO)
- **Produção:** `0` ou igual a `MaxLifetime`

**⚠️ IMPORTANTE:** Este era o parâmetro que causava o erro "sql: database is closed" quando não configurado!

---

## 🔄 Como o Pool Funciona

### **Ciclo de Vida de uma Conexão**

```
┌─────────────────────────────────────────────────────────────────┐
│                    POOL DE CONEXÕES                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  MaxOpenConns = 25  ←─ Máximo total (em uso + ociosas)         │
│  MaxIdleConns = 5   ←─ Máximo ociosas mantidas no pool         │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Conn #1    │  │   Conn #2    │  │   Conn #3    │         │
│  │  (EM USO)    │  │  (EM USO)    │  │  (OCIOSA)    │         │
│  │  Lifetime:   │  │  Lifetime:   │  │  IdleTime:   │         │
│  │  5m/1h       │  │  30m/1h      │  │  2m/10m      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                  │
│  REGRAS DE GERENCIAMENTO:                                       │
│                                                                  │
│  📥 Quando uma requisição precisa de conexão:                   │
│     1. Busca uma conexão OCIOSA no pool                        │
│     2. Se não tiver, CRIA nova (até MaxOpenConns)              │
│     3. Se exceder MaxOpenConns, AGUARDA uma ficar livre        │
│                                                                  │
│  📤 Quando uma requisição termina de usar a conexão:            │
│     1. Se pool < MaxIdleConns: MANTÉM no pool (ociosa)         │
│     2. Se pool >= MaxIdleConns: FECHA a conexão                │
│                                                                  │
│  ⏰ Expiração Automática:                                        │
│     • MaxLifetime: Fecha após tempo desde CRIAÇÃO              │
│     • MaxIdleTime: Fecha após tempo sem USO                     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### **Exemplo de Fluxo**

```
Requisição 1: Precisa de conexão
├─ Pool está vazio
├─ CRIA nova conexão (Conn #1)
├─ USA conexão
└─ Devolve ao pool (Conn #1 → OCIOSA)

Requisição 2: Precisa de conexão
├─ Pool tem Conn #1 ociosa
├─ REUSA Conn #1
├─ USA conexão
└─ Devolve ao pool (Conn #1 → OCIOSA)

Requisição 3 e 4 simultâneas:
├─ Requisição 3: REUSA Conn #1
├─ Requisição 4: Pool vazio, CRIA Conn #2
└─ Após uso, ambas voltam ao pool (2 ociosas)

Após 65 segundos sem uso (e MaxIdleTime=60s):
├─ Conn #1: FECHADA (excedeu MaxIdleTime)
└─ Conn #2: FECHADA (excedeu MaxIdleTime)
```

---

## 🎯 Configurações Recomendadas

### **Cenário 1: Desenvolvimento Local**

```env
# Conexões sempre abertas, sem expiração
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=0      # Nunca expira
DB_CONN_MAX_IDLE_TIME=0     # Usa padrão do SO
```

**Vantagens:**
- ✅ Simplicidade
- ✅ Performance consistente
- ✅ Sem surpresas

---

### **Cenário 2: Produção com Tráfego Baixo**

```env
# Pool pequeno, mantém conexões abertas
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=3600s   # 1 hora
DB_CONN_MAX_IDLE_TIME=600s   # 10 minutos
```

**Vantagens:**
- ✅ Economiza recursos
- ✅ Fecha conexões não utilizadas
- ✅ Previne memory leaks

---

### **Cenário 3: Produção com Tráfego Alto**

```env
# Pool grande, mantém muitas conexões prontas
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME=3600s   # 1 hora
DB_CONN_MAX_IDLE_TIME=1800s  # 30 minutos
```

**Vantagens:**
- ✅ Alta concorrência
- ✅ Baixa latência
- ✅ Reuso eficiente

---

### **Cenário 4: Microserviços com Load Balancer**

```env
# Rotação frequente de conexões
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=300s    # 5 minutos
DB_CONN_MAX_IDLE_TIME=60s    # 1 minuto
```

**Vantagens:**
- ✅ Compatível com LB timeouts
- ✅ Distribuição de carga
- ✅ Recuperação rápida de falhas

---

## 💻 Exemplos Práticos

### **Exemplo 1: Configuração Básica no `.env`**

```env
# Database
DB_DRIVER=postgresql
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp
DB_USER=postgres
DB_PASSWORD=secret

# Connection Pool
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=0
DB_CONN_MAX_IDLE_TIME=0
```

### **Exemplo 2: Configuração Programática**

```go
package main

import (
    "github.com/fitlcarlos/go-data/odata"
    "time"
)

func main() {
    server := odata.NewServer()
    
    // Carrega configurações do .env
    config, _ := odata.LoadEnvOrDefault()
    
    // Sobrescreve configurações de pool
    config.DBMaxOpenConns = 50
    config.DBMaxIdleConns = 10
    config.DBConnMaxLifetime = time.Hour
    config.DBConnMaxIdleTime = 30 * time.Minute
    
    // Cria provider com configurações customizadas
    provider := config.CreateProviderFromConfig()
    
    server.SetProvider(provider)
    server.Start()
}
```

### **Exemplo 3: Monitorando o Pool**

```go
// Após configurar o provider, você pode monitorar suas estatísticas
db := provider.GetConnection()

stats := db.Stats()
log.Printf("📊 Pool Stats:")
log.Printf("   MaxOpenConnections: %d", stats.MaxOpenConnections)
log.Printf("   OpenConnections: %d", stats.OpenConnections)
log.Printf("   InUse: %d", stats.InUse)
log.Printf("   Idle: %d", stats.Idle)
log.Printf("   WaitCount: %d", stats.WaitCount)
log.Printf("   WaitDuration: %s", stats.WaitDuration)
```

---

## 🐛 Troubleshooting

### **Problema: "sql: database is closed"**

**Causa:** Conexão foi fechada por `MaxIdleTime` ou `MaxLifetime`.

**Solução:**
```env
# Configure valores maiores ou 0
DB_CONN_MAX_LIFETIME=0
DB_CONN_MAX_IDLE_TIME=0
```

---

### **Problema: "too many connections"**

**Causa:** `MaxOpenConns` muito alto ou sem limite (`0`).

**Solução:**
```env
# Limite o número de conexões
DB_MAX_OPEN_CONNS=25  # Ajuste conforme capacidade do banco
```

---

### **Problema: Alta latência em picos de tráfego**

**Causa:** `MaxIdleConns` muito baixo, criando muitas conexões novas.

**Solução:**
```env
# Aumente o número de conexões ociosas
DB_MAX_IDLE_CONNS=10  # 20-40% do MaxOpenConns
```

---

### **Problema: Memory leaks ou conexões "travadas"**

**Causa:** Conexões antigas acumulando recursos.

**Solução:**
```env
# Force rotação periódica
DB_CONN_MAX_LIFETIME=3600s   # 1 hora
DB_CONN_MAX_IDLE_TIME=600s   # 10 minutos
```

---

## ✅ Boas Práticas

### **1. Não Use Valores Muito Baixos**

```env
# ❌ RUIM: Pool muito pequeno
DB_MAX_OPEN_CONNS=2
DB_MAX_IDLE_CONNS=1

# ✅ BOM: Pool adequado
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
```

### **2. MaxIdleConns ≤ MaxOpenConns**

```env
# ❌ RUIM: Idle maior que Open
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=20

# ✅ BOM: Idle menor ou igual
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
```

### **3. Use 0 para Conexões Permanentes**

```env
# ✅ BOM: Conexões nunca expiram (ideal para dev)
DB_CONN_MAX_LIFETIME=0
DB_CONN_MAX_IDLE_TIME=0
```

### **4. Monitore as Estatísticas**

```go
// Adicione logging periódico
ticker := time.NewTicker(1 * time.Minute)
go func() {
    for range ticker.C {
        stats := db.Stats()
        log.Printf("Pool: Open=%d, InUse=%d, Idle=%d, Wait=%d",
            stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
    }
}()
```

### **5. Ajuste Baseado em Métricas**

- **Alta WaitCount**: Aumente `MaxOpenConns`
- **Muitas conexões Idle**: Reduza `MaxIdleConns`
- **Timeouts frequentes**: Aumente `MaxLifetime` e `MaxIdleTime`

---

## 📊 Tabela de Referência Rápida

| Parâmetro | Padrão | Dev | Produção Baixa | Produção Alta |
|-----------|--------|-----|----------------|---------------|
| `DB_MAX_OPEN_CONNS` | `10` | `10` | `25` | `100` |
| `DB_MAX_IDLE_CONNS` | `2` | `5` | `5` | `25` |
| `DB_CONN_MAX_LIFETIME` | `10m` | `0` | `3600s` | `3600s` |
| `DB_CONN_MAX_IDLE_TIME` | `10m` | `0` | `600s` | `1800s` |

---

## 🔗 Referências

- [Go database/sql Documentation](https://pkg.go.dev/database/sql)
- [Configuring sql.DB for Better Performance](https://www.alexedwards.net/blog/configuring-sqldb)
- [Go-Data README](../README.md)

---

## 📝 Changelog

| Versão | Data | Descrição |
|--------|------|-----------|
| `1.0.0` | 2025-10-28 | Documentação inicial do pool de conexões |

---

**💡 Dica Final:** Sempre comece com valores conservadores e ajuste baseado em métricas reais da sua aplicação!

