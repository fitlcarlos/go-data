package odata

import (
	"context"
	"testing"
)

func BenchmarkParseFilterString(b *testing.B) {
	testCases := []struct {
		name   string
		filter string
	}{
		{"Simple", "Name eq 'John'"},
		{"Complex", "Name eq 'John' and Age gt 18 and Active eq true"},
		{"Nested", "(Name eq 'John' or Name eq 'Jane') and Age gt 18"},
		{"Functions", "contains(Name, 'John') and startswith(Email, 'john')"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ParseFilterString(ctx, tc.filter)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseExpandString(b *testing.B) {
	testCases := []struct {
		name   string
		expand string
	}{
		{"Simple", "Category"},
		{"Nested", "Category($expand=Parent)"},
		{"Multiple", "Category,Tags,Author"},
		{"WithOptions", "Products($filter=Active eq true;$top=10)"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ParseExpandString(ctx, tc.expand)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseSelectString(b *testing.B) {
	selects := []string{
		"Name",
		"Name,Email,Age",
		"Name,Email,Age,Address,Phone,Active,CreatedAt",
	}

	for _, sel := range selects {
		b.Run(sel, func(b *testing.B) {
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ParseSelectString(ctx, sel)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
