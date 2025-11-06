package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTopString_Valid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		top      string
		expected int
	}{
		{"Zero", "0", 0},
		{"Small number", "10", 10},
		{"Medium number", "100", 100},
		{"Large number", "1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTopString(ctx, tt.top)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, int(*result))
		})
	}
}

func TestParseTopString_Empty(t *testing.T) {
	ctx := context.Background()

	result, err := ParseTopString(ctx, "")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseTopString_Invalid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		top  string
	}{
		{"Not a number", "abc"},
		{"Negative", "-10"},
		{"Exceeds limit", "10000000"},
		{"Decimal", "10.5"},
		{"With spaces", "10 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTopString(ctx, tt.top)
			assert.Error(t, err)
		})
	}
}

func TestParseSkipString_Valid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		skip     string
		expected int
	}{
		{"Zero", "0", 0},
		{"Small number", "10", 10},
		{"Medium number", "100", 100},
		{"Large number", "1000", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSkipString(ctx, tt.skip)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, int(*result))
		})
	}
}

func TestParseSkipString_Empty(t *testing.T) {
	ctx := context.Background()

	result, err := ParseSkipString(ctx, "")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestParseSkipString_Invalid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		skip string
	}{
		{"Not a number", "xyz"},
		{"Negative", "-5"},
		{"Exceeds limit", "10000000"},
		{"Decimal", "5.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSkipString(ctx, tt.skip)
			assert.Error(t, err)
		})
	}
}

func TestValidateTopQuery(t *testing.T) {
	t.Run("Valid top", func(t *testing.T) {
		top := GoDataTopQuery(10)
		err := ValidateTopQuery(&top)
		assert.NoError(t, err)
	})

	t.Run("Nil top", func(t *testing.T) {
		err := ValidateTopQuery(nil)
		assert.NoError(t, err)
	})

	t.Run("Zero top", func(t *testing.T) {
		top := GoDataTopQuery(0)
		err := ValidateTopQuery(&top)
		assert.NoError(t, err)
	})

	t.Run("Negative top", func(t *testing.T) {
		top := GoDataTopQuery(-1)
		err := ValidateTopQuery(&top)
		assert.Error(t, err)
	})

	t.Run("Exceeds limit", func(t *testing.T) {
		top := GoDataTopQuery(1000000)
		err := ValidateTopQuery(&top)
		assert.Error(t, err)
	})
}

func TestValidateSkipQuery(t *testing.T) {
	t.Run("Valid skip", func(t *testing.T) {
		skip := GoDataSkipQuery(10)
		err := ValidateSkipQuery(&skip)
		assert.NoError(t, err)
	})

	t.Run("Nil skip", func(t *testing.T) {
		err := ValidateSkipQuery(nil)
		assert.NoError(t, err)
	})

	t.Run("Zero skip", func(t *testing.T) {
		skip := GoDataSkipQuery(0)
		err := ValidateSkipQuery(&skip)
		assert.NoError(t, err)
	})

	t.Run("Negative skip", func(t *testing.T) {
		skip := GoDataSkipQuery(-1)
		err := ValidateSkipQuery(&skip)
		assert.Error(t, err)
	})

	t.Run("Exceeds limit", func(t *testing.T) {
		skip := GoDataSkipQuery(10000000)
		err := ValidateSkipQuery(&skip)
		assert.Error(t, err)
	})
}

func TestGetTopValue(t *testing.T) {
	t.Run("Returns value", func(t *testing.T) {
		top := GoDataTopQuery(42)
		value := GetTopValue(&top)
		assert.Equal(t, 42, value)
	})

	t.Run("Nil returns zero", func(t *testing.T) {
		value := GetTopValue(nil)
		assert.Equal(t, 0, value)
	})
}

func TestGetSkipValue(t *testing.T) {
	t.Run("Returns value", func(t *testing.T) {
		skip := GoDataSkipQuery(42)
		value := GetSkipValue(&skip)
		assert.Equal(t, 42, value)
	})

	t.Run("Nil returns zero", func(t *testing.T) {
		value := GetSkipValue(nil)
		assert.Equal(t, 0, value)
	})
}

func TestSetTopValue(t *testing.T) {
	t.Run("Set valid value", func(t *testing.T) {
		top := GoDataTopQuery(0)
		err := SetTopValue(&top, 10)

		require.NoError(t, err)
		assert.Equal(t, 10, int(top))
	})

	t.Run("Set negative value", func(t *testing.T) {
		top := GoDataTopQuery(0)
		err := SetTopValue(&top, -1)

		assert.Error(t, err)
	})

	t.Run("Set value exceeding limit", func(t *testing.T) {
		top := GoDataTopQuery(0)
		err := SetTopValue(&top, 1000000)

		assert.Error(t, err)
	})
}

func TestSetSkipValue(t *testing.T) {
	t.Run("Set valid value", func(t *testing.T) {
		skip := GoDataSkipQuery(0)
		err := SetSkipValue(&skip, 10)

		require.NoError(t, err)
		assert.Equal(t, 10, int(skip))
	})

	t.Run("Set negative value", func(t *testing.T) {
		skip := GoDataSkipQuery(0)
		err := SetSkipValue(&skip, -1)

		assert.Error(t, err)
	})

	t.Run("Set value exceeding limit", func(t *testing.T) {
		skip := GoDataSkipQuery(0)
		err := SetSkipValue(&skip, 10000000)

		assert.Error(t, err)
	})
}

func TestTopQuery_String(t *testing.T) {
	t.Run("Returns string representation", func(t *testing.T) {
		top := GoDataTopQuery(42)
		assert.Equal(t, "42", top.String())
	})

	t.Run("Nil returns empty", func(t *testing.T) {
		var top *GoDataTopQuery
		assert.Empty(t, top.String())
	})
}

func TestSkipQuery_String(t *testing.T) {
	t.Run("Returns string representation", func(t *testing.T) {
		skip := GoDataSkipQuery(42)
		assert.Equal(t, "42", skip.String())
	})

	t.Run("Nil returns empty", func(t *testing.T) {
		var skip *GoDataSkipQuery
		assert.Empty(t, skip.String())
	})
}

func TestConvertTopSkipToSQL(t *testing.T) {
	t.Run("Both top and skip", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(5)

		sql, args := ConvertTopSkipToSQL(&top, &skip)

		assert.NotEmpty(t, sql)
		assert.Contains(t, sql, "LIMIT")
		assert.Contains(t, sql, "OFFSET")
		assert.Len(t, args, 2)
		assert.Equal(t, 10, args[0])
		assert.Equal(t, 5, args[1])
	})

	t.Run("Only top", func(t *testing.T) {
		top := GoDataTopQuery(10)

		sql, args := ConvertTopSkipToSQL(&top, nil)

		assert.NotEmpty(t, sql)
		assert.Contains(t, sql, "LIMIT")
		assert.NotContains(t, sql, "OFFSET")
		assert.Len(t, args, 1)
		assert.Equal(t, 10, args[0])
	})

	t.Run("Only skip", func(t *testing.T) {
		skip := GoDataSkipQuery(5)

		sql, args := ConvertTopSkipToSQL(nil, &skip)

		assert.NotEmpty(t, sql)
		assert.Contains(t, sql, "OFFSET")
		assert.NotContains(t, sql, "LIMIT")
		assert.Len(t, args, 1)
		assert.Equal(t, 5, args[0])
	})

	t.Run("Neither top nor skip", func(t *testing.T) {
		sql, args := ConvertTopSkipToSQL(nil, nil)

		assert.Empty(t, sql)
		assert.Empty(t, args)
	})

	t.Run("Zero values", func(t *testing.T) {
		top := GoDataTopQuery(0)
		skip := GoDataSkipQuery(0)

		sql, args := ConvertTopSkipToSQL(&top, &skip)

		// Zero values are treated as not set
		assert.Empty(t, sql)
		assert.Empty(t, args)
	})
}

func TestValidateTopSkipCombination(t *testing.T) {
	t.Run("Valid combination", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(5)

		err := ValidateTopSkipCombination(&top, &skip)
		assert.NoError(t, err)
	})

	t.Run("Nil values", func(t *testing.T) {
		err := ValidateTopSkipCombination(nil, nil)
		assert.NoError(t, err)
	})

	t.Run("Skip without top", func(t *testing.T) {
		skip := GoDataSkipQuery(5)

		err := ValidateTopSkipCombination(nil, &skip)
		assert.Error(t, err, "Should require top when skip is specified")
	})

	t.Run("Top without skip", func(t *testing.T) {
		top := GoDataTopQuery(10)

		err := ValidateTopSkipCombination(&top, nil)
		assert.NoError(t, err, "Top without skip is valid")
	})

	t.Run("Invalid top", func(t *testing.T) {
		top := GoDataTopQuery(-1)
		skip := GoDataSkipQuery(5)

		err := ValidateTopSkipCombination(&top, &skip)
		assert.Error(t, err)
	})

	t.Run("Invalid skip", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(-1)

		err := ValidateTopSkipCombination(&top, &skip)
		assert.Error(t, err)
	})
}

func TestGetPaginationInfo(t *testing.T) {
	t.Run("With both top and skip", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(20)

		info := GetPaginationInfo(&top, &skip)

		assert.Equal(t, 10, info.PageSize)
		assert.Equal(t, 20, info.Offset)
		assert.True(t, info.HasTop)
		assert.True(t, info.HasSkip)
		assert.Equal(t, 3, info.PageNumber) // 20/10 + 1 = 3
	})

	t.Run("With only top", func(t *testing.T) {
		top := GoDataTopQuery(10)

		info := GetPaginationInfo(&top, nil)

		assert.Equal(t, 10, info.PageSize)
		assert.Equal(t, 0, info.Offset)
		assert.True(t, info.HasTop)
		assert.False(t, info.HasSkip)
		assert.Equal(t, 1, info.PageNumber)
	})

	t.Run("With neither", func(t *testing.T) {
		info := GetPaginationInfo(nil, nil)

		assert.Equal(t, 0, info.PageSize)
		assert.Equal(t, 0, info.Offset)
		assert.False(t, info.HasTop)
		assert.False(t, info.HasSkip)
		assert.Equal(t, 1, info.PageNumber)
	})
}

func TestPaginationInfo_IsFirstPage(t *testing.T) {
	t.Run("First page", func(t *testing.T) {
		info := PaginationInfo{PageNumber: 1}
		assert.True(t, info.IsFirstPage())
	})

	t.Run("Second page", func(t *testing.T) {
		info := PaginationInfo{PageNumber: 2}
		assert.False(t, info.IsFirstPage())
	})

	t.Run("Zero page", func(t *testing.T) {
		info := PaginationInfo{PageNumber: 0}
		assert.True(t, info.IsFirstPage())
	})
}

func TestPaginationInfo_HasPagination(t *testing.T) {
	t.Run("With top", func(t *testing.T) {
		info := PaginationInfo{HasTop: true, HasSkip: false}
		assert.True(t, info.HasPagination())
	})

	t.Run("With skip", func(t *testing.T) {
		info := PaginationInfo{HasTop: false, HasSkip: true}
		assert.True(t, info.HasPagination())
	})

	t.Run("With both", func(t *testing.T) {
		info := PaginationInfo{HasTop: true, HasSkip: true}
		assert.True(t, info.HasPagination())
	})

	t.Run("With neither", func(t *testing.T) {
		info := PaginationInfo{HasTop: false, HasSkip: false}
		assert.False(t, info.HasPagination())
	})
}

func TestPaginationInfo_GetNextOffset(t *testing.T) {
	t.Run("Calculate next offset", func(t *testing.T) {
		info := PaginationInfo{PageSize: 10, Offset: 20}
		assert.Equal(t, 30, info.GetNextOffset())
	})

	t.Run("Zero page size", func(t *testing.T) {
		info := PaginationInfo{PageSize: 0, Offset: 20}
		assert.Equal(t, 20, info.GetNextOffset())
	})
}

func TestPaginationInfo_GetPreviousOffset(t *testing.T) {
	t.Run("Calculate previous offset", func(t *testing.T) {
		info := PaginationInfo{PageSize: 10, Offset: 20}
		assert.Equal(t, 10, info.GetPreviousOffset())
	})

	t.Run("First page", func(t *testing.T) {
		info := PaginationInfo{PageSize: 10, Offset: 5}
		assert.Equal(t, 0, info.GetPreviousOffset())
	})

	t.Run("Zero offset", func(t *testing.T) {
		info := PaginationInfo{PageSize: 10, Offset: 0}
		assert.Equal(t, 0, info.GetPreviousOffset())
	})

	t.Run("Zero page size", func(t *testing.T) {
		info := PaginationInfo{PageSize: 0, Offset: 20}
		assert.Equal(t, 0, info.GetPreviousOffset())
	})
}

func TestParseTopSkipFromURL(t *testing.T) {
	t.Run("Both values", func(t *testing.T) {
		top, skip, err := ParseTopSkipFromURL("10", "5")

		require.NoError(t, err)
		assert.NotNil(t, top)
		assert.NotNil(t, skip)
		assert.Equal(t, 10, int(*top))
		assert.Equal(t, 5, int(*skip))
	})

	t.Run("Only top", func(t *testing.T) {
		top, skip, err := ParseTopSkipFromURL("10", "")

		require.NoError(t, err)
		assert.NotNil(t, top)
		assert.Nil(t, skip)
		assert.Equal(t, 10, int(*top))
	})

	t.Run("Only skip", func(t *testing.T) {
		top, skip, err := ParseTopSkipFromURL("", "5")

		require.NoError(t, err)
		assert.Nil(t, top)
		assert.NotNil(t, skip)
		assert.Equal(t, 5, int(*skip))
	})

	t.Run("Neither value", func(t *testing.T) {
		top, skip, err := ParseTopSkipFromURL("", "")

		require.NoError(t, err)
		assert.Nil(t, top)
		assert.Nil(t, skip)
	})

	t.Run("Invalid top", func(t *testing.T) {
		_, _, err := ParseTopSkipFromURL("abc", "5")
		assert.Error(t, err)
	})

	t.Run("Invalid skip", func(t *testing.T) {
		_, _, err := ParseTopSkipFromURL("10", "xyz")
		assert.Error(t, err)
	})
}

func TestApplyDefaultLimits(t *testing.T) {
	t.Run("Apply default top", func(t *testing.T) {
		top, skip := ApplyDefaultLimits(nil, nil, 50)

		assert.NotNil(t, top)
		assert.Nil(t, skip)
		assert.Equal(t, 50, int(*top))
	})

	t.Run("Keep existing top", func(t *testing.T) {
		existingTop := GoDataTopQuery(10)
		top, skip := ApplyDefaultLimits(&existingTop, nil, 50)

		assert.NotNil(t, top)
		assert.Nil(t, skip)
		assert.Equal(t, 10, int(*top))
	})

	t.Run("Zero default does not apply", func(t *testing.T) {
		top, skip := ApplyDefaultLimits(nil, nil, 0)

		assert.Nil(t, top)
		assert.Nil(t, skip)
	})

	t.Run("Skip is not affected", func(t *testing.T) {
		existingSkip := GoDataSkipQuery(5)
		top, skip := ApplyDefaultLimits(nil, &existingSkip, 50)

		assert.NotNil(t, top)
		assert.NotNil(t, skip)
		assert.Equal(t, 50, int(*top))
		assert.Equal(t, 5, int(*skip))
	})
}

func TestFormatTopSkipForURL(t *testing.T) {
	t.Run("Both values", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(5)

		result := FormatTopSkipForURL(&top, &skip)

		assert.Contains(t, result, "$top=10")
		assert.Contains(t, result, "$skip=5")
		assert.Contains(t, result, "&")
	})

	t.Run("Only top", func(t *testing.T) {
		top := GoDataTopQuery(10)

		result := FormatTopSkipForURL(&top, nil)

		assert.Contains(t, result, "$top=10")
		assert.NotContains(t, result, "$skip")
	})

	t.Run("Only skip", func(t *testing.T) {
		skip := GoDataSkipQuery(5)

		result := FormatTopSkipForURL(nil, &skip)

		assert.Contains(t, result, "$skip=5")
		assert.NotContains(t, result, "$top")
	})

	t.Run("Neither value", func(t *testing.T) {
		result := FormatTopSkipForURL(nil, nil)
		assert.Empty(t, result)
	})
}

func TestGetTopSkipComplexity(t *testing.T) {
	t.Run("No pagination", func(t *testing.T) {
		complexity := GetTopSkipComplexity(nil, nil)
		assert.Equal(t, 0, complexity)
	})

	t.Run("Simple top", func(t *testing.T) {
		top := GoDataTopQuery(10)
		complexity := GetTopSkipComplexity(&top, nil)
		assert.Equal(t, 1, complexity)
	})

	t.Run("Large top increases complexity", func(t *testing.T) {
		top := GoDataTopQuery(2000)
		complexity := GetTopSkipComplexity(&top, nil)
		assert.Greater(t, complexity, 1)
	})

	t.Run("Simple skip", func(t *testing.T) {
		skip := GoDataSkipQuery(10)
		complexity := GetTopSkipComplexity(nil, &skip)
		assert.Equal(t, 1, complexity)
	})

	t.Run("Large skip increases complexity more", func(t *testing.T) {
		skip := GoDataSkipQuery(2000)
		complexity := GetTopSkipComplexity(nil, &skip)
		assert.Greater(t, complexity, 2)
	})

	t.Run("Both top and skip", func(t *testing.T) {
		top := GoDataTopQuery(10)
		skip := GoDataSkipQuery(5)
		complexity := GetTopSkipComplexity(&top, &skip)
		assert.Equal(t, 2, complexity)
	})
}
