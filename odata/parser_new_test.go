package odata

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGlobalParser(t *testing.T) {
	t.Run("Returns singleton instance", func(t *testing.T) {
		parser1 := GetGlobalParser()
		parser2 := GetGlobalParser()

		assert.NotNil(t, parser1)
		assert.NotNil(t, parser2)
		assert.Equal(t, parser1, parser2, "Should return same instance (singleton)")
	})

	t.Run("Has supported params configured", func(t *testing.T) {
		parser := GetGlobalParser()

		assert.True(t, parser.supportedParams["$filter"])
		assert.True(t, parser.supportedParams["$orderby"])
		assert.True(t, parser.supportedParams["$select"])
		assert.True(t, parser.supportedParams["$expand"])
		assert.True(t, parser.supportedParams["$skip"])
		assert.True(t, parser.supportedParams["$top"])
		assert.True(t, parser.supportedParams["$count"])
	})
}

func TestNewODataParser(t *testing.T) {
	t.Run("Returns parser instance", func(t *testing.T) {
		parser := NewODataParser()

		assert.NotNil(t, parser)
		assert.NotNil(t, parser.supportedParams)
	})
}

func TestParseQueryOptions_Empty(t *testing.T) {
	parser := NewODataParser()
	values := url.Values{}

	options, err := parser.ParseQueryOptions(values)

	require.NoError(t, err)
	assert.Nil(t, options.Filter)
	assert.Empty(t, options.OrderBy)
	assert.Nil(t, options.Select)
	assert.Nil(t, options.Expand)
	assert.Nil(t, options.Skip)
	assert.Nil(t, options.Top)
	assert.Nil(t, options.Count)
}

func TestParseQueryOptions_Filter(t *testing.T) {
	parser := NewODataParser()

	t.Run("Simple filter", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{"Name eq 'John'"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
	})

	t.Run("Complex filter", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{"Name eq 'John' and Age gt 25"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
	})
}

func TestParseQueryOptions_OrderBy(t *testing.T) {
	parser := NewODataParser()

	t.Run("Single field", func(t *testing.T) {
		values := url.Values{
			"$orderby": []string{"Name"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.Equal(t, "Name", options.OrderBy)
	})

	t.Run("Multiple fields", func(t *testing.T) {
		values := url.Values{
			"$orderby": []string{"Name asc, Age desc"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.Equal(t, "Name asc, Age desc", options.OrderBy)
	})
}

func TestParseQueryOptions_Select(t *testing.T) {
	parser := NewODataParser()

	t.Run("Single property", func(t *testing.T) {
		values := url.Values{
			"$select": []string{"Name"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Select)
	})

	t.Run("Multiple properties", func(t *testing.T) {
		values := url.Values{
			"$select": []string{"Name,Email,Age"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Select)
	})
}

func TestParseQueryOptions_Expand(t *testing.T) {
	parser := NewODataParser()

	t.Run("Single navigation", func(t *testing.T) {
		values := url.Values{
			"$expand": []string{"Orders"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Expand)
	})

	t.Run("Multiple navigations", func(t *testing.T) {
		values := url.Values{
			"$expand": []string{"Orders,Products"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Expand)
	})
}

func TestParseQueryOptions_TopSkip(t *testing.T) {
	parser := NewODataParser()

	t.Run("Top only", func(t *testing.T) {
		values := url.Values{
			"$top": []string{"10"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Top)
		assert.Equal(t, 10, int(*options.Top))
	})

	t.Run("Skip only", func(t *testing.T) {
		values := url.Values{
			"$skip": []string{"5"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Skip)
		assert.Equal(t, 5, int(*options.Skip))
	})

	t.Run("Both top and skip", func(t *testing.T) {
		values := url.Values{
			"$top":  []string{"10"},
			"$skip": []string{"5"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Top)
		assert.NotNil(t, options.Skip)
		assert.Equal(t, 10, int(*options.Top))
		assert.Equal(t, 5, int(*options.Skip))
	})

	t.Run("Invalid top", func(t *testing.T) {
		values := url.Values{
			"$top": []string{"invalid"},
		}

		_, err := parser.ParseQueryOptions(values)
		assert.Error(t, err)
	})

	t.Run("Negative top", func(t *testing.T) {
		values := url.Values{
			"$top": []string{"-10"},
		}

		_, err := parser.ParseQueryOptions(values)
		assert.Error(t, err)
	})
}

func TestParseQueryOptions_Count(t *testing.T) {
	parser := NewODataParser()

	t.Run("Count true", func(t *testing.T) {
		values := url.Values{
			"$count": []string{"true"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Count)
		assert.True(t, GetCountValue(options.Count))
	})

	t.Run("Count false", func(t *testing.T) {
		values := url.Values{
			"$count": []string{"false"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Count)
		assert.False(t, GetCountValue(options.Count))
	})
}

func TestParseQueryOptions_MultipleParams(t *testing.T) {
	parser := NewODataParser()

	t.Run("All params combined", func(t *testing.T) {
		values := url.Values{
			"$filter":  []string{"Age gt 25"},
			"$orderby": []string{"Name asc"},
			"$select":  []string{"Name,Email"},
			"$top":     []string{"10"},
			"$skip":    []string{"5"},
			"$count":   []string{"true"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
		assert.NotEmpty(t, options.OrderBy)
		assert.NotNil(t, options.Select)
		assert.NotNil(t, options.Top)
		assert.NotNil(t, options.Skip)
		assert.NotNil(t, options.Count)
		assert.True(t, GetCountValue(options.Count))
	})
}

func TestParseQueryOptions_CaseInsensitive(t *testing.T) {
	parser := NewODataParser()

	t.Run("Uppercase params", func(t *testing.T) {
		values := url.Values{
			"$FILTER": []string{"Name eq 'John'"},
			"$TOP":    []string{"10"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
		assert.NotNil(t, options.Top)
	})

	t.Run("Mixed case params", func(t *testing.T) {
		values := url.Values{
			"$FiLtEr": []string{"Name eq 'John'"},
			"$ToP":    []string{"10"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
		assert.NotNil(t, options.Top)
	})
}

func TestParseQueryOptionsWithConfig_Compliance(t *testing.T) {
	parser := NewODataParser()

	t.Run("Strict compliance - rejects unknown params", func(t *testing.T) {
		values := url.Values{
			"$filter":  []string{"Name eq 'John'"},
			"$unknown": []string{"value"},
		}

		_, err := parser.ParseQueryOptionsWithConfig(values, ComplianceStrict)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("Ignore unknown keywords", func(t *testing.T) {
		values := url.Values{
			"$filter":  []string{"Name eq 'John'"},
			"$unknown": []string{"value"},
		}

		options, err := parser.ParseQueryOptionsWithConfig(values, ComplianceIgnoreUnknownKeywords)
		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
	})

	t.Run("Strict compliance - rejects duplicate params", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{"Name eq 'John'", "Age gt 25"},
		}

		_, err := parser.ParseQueryOptionsWithConfig(values, ComplianceStrict)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "more than once")
	})

	t.Run("Ignore duplicate keywords", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{"Name eq 'John'", "Age gt 25"},
		}

		options, err := parser.ParseQueryOptionsWithConfig(values, ComplianceIgnoreDuplicateKeywords)
		require.NoError(t, err)
		assert.NotNil(t, options.Filter)
	})
}

func TestComplianceConfig_Constants(t *testing.T) {
	t.Run("Compliance constants defined", func(t *testing.T) {
		assert.Equal(t, ComplianceConfig(0), ComplianceStrict)
		assert.NotEqual(t, ComplianceStrict, ComplianceIgnoreUnknownKeywords)
		assert.NotEqual(t, ComplianceStrict, ComplianceIgnoreDuplicateKeywords)
	})
}

func TestParseQueryOptions_Search(t *testing.T) {
	parser := NewODataParser()

	t.Run("Simple search", func(t *testing.T) {
		values := url.Values{
			"$search": []string{"john"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Search)
	})
}

func TestParseQueryOptions_Compute(t *testing.T) {
	parser := NewODataParser()

	t.Run("Simple compute", func(t *testing.T) {
		values := url.Values{
			"$compute": []string{"Price mul Quantity as Total"},
		}

		options, err := parser.ParseQueryOptions(values)

		require.NoError(t, err)
		assert.NotNil(t, options.Compute)
	})
}

func TestParseQueryOptions_EdgeCases(t *testing.T) {
	parser := NewODataParser()

	t.Run("Empty string values", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{""},
			"$top":    []string{""},
		}

		options, err := parser.ParseQueryOptions(values)

		// Empty filter is allowed, empty top might error
		_ = options
		_ = err
	})

	t.Run("Whitespace only values", func(t *testing.T) {
		values := url.Values{
			"$filter": []string{"   "},
		}

		options, err := parser.ParseQueryOptions(values)

		// Should handle gracefully
		_ = options
		_ = err
	})
}
