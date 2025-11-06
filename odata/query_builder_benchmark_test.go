package odata

import (
	"context"
	"testing"
)

func BenchmarkBuildWhereClause(b *testing.B) {
	qb := NewQueryBuilder("mysql")
	metadata := createBenchmarkMetadata()

	testCases := []struct {
		name   string
		filter string
	}{
		{"Simple", "Name eq 'John'"},
		{"Complex", "Name eq 'John' and Age gt 18"},
		{"VeryComplex", "(Name eq 'John' or Email eq 'john@example.com') and (Age gt 18 and Age lt 65) and Active eq true"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ctx := context.Background()
			filterQuery, _ := ParseFilterString(ctx, tc.filter)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := qb.BuildWhereClause(ctx, filterQuery.Tree, metadata)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkBuildCompleteQuery(b *testing.B) {
	qb := NewQueryBuilder("mysql")
	metadata := createBenchmarkMetadata()
	ctx := context.Background()

	// Setup query options
	filterQuery, _ := ParseFilterString(ctx, "Active eq true")
	selectQuery, _ := ParseSelectString(ctx, "Name,Email,Age")

	options := QueryOptions{
		Filter:  filterQuery,
		Select:  selectQuery,
		OrderBy: "Name",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := qb.BuildCompleteQuery(ctx, metadata, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func createBenchmarkMetadata() EntityMetadata {
	return EntityMetadata{
		Name:      "Users",
		TableName: "users",
		Properties: []PropertyMetadata{
			{Name: "ID", ColumnName: "id", Type: "int64", IsKey: true},
			{Name: "Name", ColumnName: "name", Type: "string"},
			{Name: "Email", ColumnName: "email", Type: "string"},
			{Name: "Age", ColumnName: "age", Type: "int32"},
			{Name: "Active", ColumnName: "active", Type: "bool"},
			{Name: "Address", ColumnName: "address", Type: "string"},
			{Name: "Phone", ColumnName: "phone", Type: "string"},
			{Name: "CreatedAt", ColumnName: "created_at", Type: "time.Time"},
		},
	}
}
