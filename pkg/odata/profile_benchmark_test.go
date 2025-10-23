package odata

import (
	"context"
	"os"
	"runtime/pprof"
	"testing"
)

// Executar com: go test -run=^$ -bench=BenchmarkProfile -cpuprofile=cpu.prof -memprofile=mem.prof
// Analisar com: go tool pprof cpu.prof

func BenchmarkProfileQueryBuilding(b *testing.B) {
	if os.Getenv("PROFILE") != "" {
		f, _ := os.Create("query_building_cpu.prof")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	qb := NewQueryBuilder("mysql")
	metadata := createBenchmarkMetadata()
	ctx := context.Background()

	filterQuery, _ := ParseFilterString(ctx, "Name eq 'John' and Age gt 18")
	options := QueryOptions{Filter: filterQuery}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb.BuildCompleteQuery(ctx, metadata, options)
	}
}

func BenchmarkProfileParsing(b *testing.B) {
	if os.Getenv("PROFILE") != "" {
		f, _ := os.Create("parsing_cpu.prof")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	ctx := context.Background()
	filter := "Name eq 'John' and Age gt 18 and Active eq true"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFilterString(ctx, filter)
	}
}
