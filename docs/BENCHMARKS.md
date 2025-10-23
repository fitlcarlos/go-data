# Benchmarks

## Executar Benchmarks

```bash
# Todos os benchmarks
go test -bench=. -benchmem

# Benchmarks específicos
go test -bench=BenchmarkParseFilter -benchmem
go test -bench=BenchmarkExpand -benchmem
go test -bench=BenchmarkQueryBuilder -benchmem

# Com profiling
PROFILE=1 go test -bench=BenchmarkProfile -cpuprofile=cpu.prof -memprofile=mem.prof
```

## Analisar Profiles

```bash
# CPU profile (top 10)
go tool pprof -top cpu.prof

# Memory profile
go tool pprof -top mem.prof

# Interface web interativa
go tool pprof -http=:8080 cpu.prof
```

## Comparar Antes/Depois

```bash
# Benchmark antes
go test -bench=. -benchmem > before.txt

# Fazer mudanças...

# Benchmark depois
go test -bench=. -benchmem > after.txt

# Comparar (necessita benchstat)
# Instalar: go install golang.org/x/perf/cmd/benchstat@latest
benchstat before.txt after.txt
```

## Exemplos de Uso

### Benchmark de Parsers

```bash
# Parse de filtros
go test -bench=BenchmarkParseFilter -benchmem

# Parse de expand
go test -bench=BenchmarkParseExpand -benchmem

# Parse de select
go test -bench=BenchmarkParseSelect -benchmem
```

### Benchmark de Query Building

```bash
# WHERE clause
go test -bench=BenchmarkBuildWhereClause -benchmem

# Complete query
go test -bench=BenchmarkBuildCompleteQuery -benchmem
```

### Profiling Detalhado

```bash
# Com variável de ambiente PROFILE
PROFILE=1 go test -bench=BenchmarkProfileQueryBuilding -cpuprofile=cpu.prof

# Visualizar no navegador
go tool pprof -http=:8080 cpu.prof

# Ver top 20 funções
go tool pprof -top20 cpu.prof

# Ver call graph
go tool pprof -pdf cpu.prof > profile.pdf
```

## Interpretar Resultados

```
BenchmarkParseFilterString/Simple-8    100000    10450 ns/op    4096 B/op    45 allocs/op
                                          │         │            │            │
                                          │         │            │            └─ Alocações por operação
                                          │         │            └─ Bytes alocados por operação
                                          │         └─ Nanosegundos por operação
                                          └─ Número de iterações
```

### O que procurar:

- **ns/op**: Tempo de execução (menor é melhor)
- **B/op**: Memória alocada (menor é melhor)
- **allocs/op**: Número de alocações (menor é melhor)

### Metas de Performance:

- **Parsers**: < 50µs (50000 ns) para queries simples
- **Query Building**: < 100µs (100000 ns) para queries completas
- **Expand Operations**: < 10ms (10000000 ns) com batching/JOIN

