package odata

import (
	"testing"
)

func TestValidatePropertyName(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name      string
		propName  string
		wantError bool
	}{
		{"valid simple", "Username", false},
		{"valid with underscore", "user_name", false},
		{"valid with number", "user123", false},
		{"valid with dot", "user.name", false},
		{"empty name", "", true},
		{"with space", "user name", true},
		{"with special char", "user-name", true},
		{"with sql injection", "id; DROP TABLE users--", true},
		{"too long", string(make([]byte, 150)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePropertyName(tt.propName, config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePropertyName() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateFilterQuery(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name      string
		filter    string
		wantError bool
	}{
		{"valid simple", "name eq 'John'", false},
		{"valid complex", "age gt 18 and status eq 'active'", false},
		{"too long", string(make([]byte, 6000)), true},
		{"sql injection union", "name eq 'x' UNION SELECT * FROM users", true},
		{"sql injection comment", "name eq 'x'--", true},
		{"sql injection or 1=1", "name eq 'x' OR 1=1", true},
		{"xss script tag", "name eq '<script>alert(1)</script>'", true},
		{"valid unicode", "name eq 'José'", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilterQuery(tt.filter, config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateFilterQuery() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateTopValue(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name      string
		top       int
		wantError bool
	}{
		{"valid small", 10, false},
		{"valid max", 1000, false},
		{"negative", -1, true},
		{"exceeds max", 1001, true},
		{"zero", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTopValue(tt.top, config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTopValue() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateExpandDepth(t *testing.T) {
	tests := []struct {
		name      string
		expand    []ExpandOption
		maxDepth  int
		wantError bool
	}{
		{
			name: "valid shallow",
			expand: []ExpandOption{
				{Property: "orders"},
			},
			maxDepth:  5,
			wantError: false,
		},
		{
			name: "valid nested",
			expand: []ExpandOption{
				{
					Property: "orders",
					Expand: []ExpandOption{
						{Property: "items"},
					},
				},
			},
			maxDepth:  5,
			wantError: false,
		},
		{
			name: "exceeds depth",
			expand: []ExpandOption{
				{
					Property: "level1",
					Expand: []ExpandOption{
						{
							Property: "level2",
							Expand: []ExpandOption{
								{Property: "level3"},
							},
						},
					},
				},
			},
			maxDepth:  2,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpandDepth(tt.expand, tt.maxDepth, 1)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateExpandDepth() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no xss", "hello world", "hello world"},
		{"script tag", "<script>alert(1)</script>", ""}, // XSS completo é removido
		{"with null byte", "hello\x00world", "helloworld"},
		{"multiple patterns", "<script>bad</script><iframe></iframe>", ""}, // XSS completo é removido
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input, config)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectSQLInjection(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"clean input", "name eq 'John'", false},
		{"union attack", "UNION SELECT", true},
		{"comment attack", "name--", true},
		{"or 1=1 attack", "OR 1=1", true},
		{"case insensitive", "union select", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detectSQLInjection(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("detectSQLInjection() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateSelectString(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name      string
		selectStr string
		wantError bool
	}{
		{"single property", "name", false},
		{"multiple properties", "name,age,email", false},
		{"with spaces", "name, age, email", false},
		{"with dots", "user.name,user.email", false},
		{"invalid chars", "name,age;DROP TABLE", true},
		{"empty property", "name,,age", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSelectString(tt.selectStr, config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateSelectString() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateOrderByQuery(t *testing.T) {
	config := DefaultValidationConfig()

	tests := []struct {
		name      string
		orderBy   string
		wantError bool
	}{
		{"single asc", "name asc", false},
		{"single desc", "name desc", false},
		{"no direction", "name", false},
		{"multiple", "name asc, age desc", false},
		{"invalid direction", "name invalid", true},
		{"sql injection", "name; DROP TABLE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrderByQuery(tt.orderBy, config)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateOrderByQuery() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateQueryOptions(t *testing.T) {
	config := DefaultValidationConfig()

	t.Run("nil options", func(t *testing.T) {
		err := ValidateQueryOptions(nil, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions(nil) should not error, got: %v", err)
		}
	})

	t.Run("empty options", func(t *testing.T) {
		options := &QueryOptions{}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions(empty) should not error, got: %v", err)
		}
	})

	t.Run("valid filter", func(t *testing.T) {
		options := &QueryOptions{
			Filter: &GoDataFilterQuery{
				RawValue: "name eq 'John'",
			},
		}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions with valid filter should not error, got: %v", err)
		}
	})

	t.Run("invalid filter (SQL injection)", func(t *testing.T) {
		options := &QueryOptions{
			Filter: &GoDataFilterQuery{
				RawValue: "name eq 'x' UNION SELECT * FROM users",
			},
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with SQL injection should error")
		}
	})

	t.Run("valid select", func(t *testing.T) {
		options := &QueryOptions{
			Select: &GoDataSelectQuery{
				RawValue: "name,email",
				SelectItems: []*SelectItem{
					{Segments: []*Token{{Value: "name"}}},
					{Segments: []*Token{{Value: "email"}}},
				},
			},
		}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions with valid select should not error, got: %v", err)
		}
	})

	t.Run("select too long", func(t *testing.T) {
		longSelect := string(make([]byte, 3000))
		options := &QueryOptions{
			Select: &GoDataSelectQuery{
				RawValue:    longSelect,
				SelectItems: []*SelectItem{},
			},
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with long select should error")
		}
	})

	t.Run("valid top", func(t *testing.T) {
		top := GoDataTopQuery(10)
		options := &QueryOptions{
			Top: &top,
		}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions with valid top should not error, got: %v", err)
		}
	})

	t.Run("negative top", func(t *testing.T) {
		top := GoDataTopQuery(-1)
		options := &QueryOptions{
			Top: &top,
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with negative top should error")
		}
	})

	t.Run("top exceeds max", func(t *testing.T) {
		top := GoDataTopQuery(2000)
		options := &QueryOptions{
			Top: &top,
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with excessive top should error")
		}
	})

	t.Run("valid skip", func(t *testing.T) {
		skip := GoDataSkipQuery(10)
		options := &QueryOptions{
			Skip: &skip,
		}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions with valid skip should not error, got: %v", err)
		}
	})

	t.Run("negative skip", func(t *testing.T) {
		skip := GoDataSkipQuery(-1)
		options := &QueryOptions{
			Skip: &skip,
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with negative skip should error")
		}
	})

	t.Run("valid orderby", func(t *testing.T) {
		options := &QueryOptions{
			OrderBy: "name asc",
		}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions with valid orderby should not error, got: %v", err)
		}
	})

	t.Run("orderby with SQL injection", func(t *testing.T) {
		options := &QueryOptions{
			OrderBy: "name; DROP TABLE users",
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with SQL injection in orderby should error")
		}
	})

	t.Run("nil config uses default", func(t *testing.T) {
		options := &QueryOptions{
			Filter: &GoDataFilterQuery{
				RawValue: "name eq 'test'",
			},
		}
		err := ValidateQueryOptions(options, nil)
		if err != nil {
			t.Errorf("ValidateQueryOptions with nil config should use default, got: %v", err)
		}
	})
}

func TestCalculateExpandDepth(t *testing.T) {
	t.Run("nil expand", func(t *testing.T) {
		depth := calculateExpandDepth(nil)
		if depth != 0 {
			t.Errorf("calculateExpandDepth(nil) = %d, want 0", depth)
		}
	})

	t.Run("empty expand", func(t *testing.T) {
		expand := &GoDataExpandQuery{
			ExpandItems: []*ExpandItem{},
		}
		depth := calculateExpandDepth(expand)
		if depth != 0 {
			t.Errorf("calculateExpandDepth(empty) = %d, want 0", depth)
		}
	})

	t.Run("single level expand", func(t *testing.T) {
		expand := &GoDataExpandQuery{
			ExpandItems: []*ExpandItem{
				{Path: []*Token{{Value: "orders"}}},
			},
		}
		depth := calculateExpandDepth(expand)
		if depth != 1 {
			t.Errorf("calculateExpandDepth(single) = %d, want 1", depth)
		}
	})

	t.Run("two level expand", func(t *testing.T) {
		expand := &GoDataExpandQuery{
			ExpandItems: []*ExpandItem{
				{
					Path: []*Token{{Value: "orders"}},
					Expand: &GoDataExpandQuery{
						ExpandItems: []*ExpandItem{
							{Path: []*Token{{Value: "items"}}},
						},
					},
				},
			},
		}
		depth := calculateExpandDepth(expand)
		if depth != 2 {
			t.Errorf("calculateExpandDepth(two level) = %d, want 2", depth)
		}
	})

	t.Run("three level expand", func(t *testing.T) {
		expand := &GoDataExpandQuery{
			ExpandItems: []*ExpandItem{
				{
					Path: []*Token{{Value: "orders"}},
					Expand: &GoDataExpandQuery{
						ExpandItems: []*ExpandItem{
							{
								Path: []*Token{{Value: "items"}},
								Expand: &GoDataExpandQuery{
									ExpandItems: []*ExpandItem{
										{Path: []*Token{{Value: "product"}}},
									},
								},
							},
						},
					},
				},
			},
		}
		depth := calculateExpandDepth(expand)
		if depth != 3 {
			t.Errorf("calculateExpandDepth(three level) = %d, want 3", depth)
		}
	})

	t.Run("multiple branches, different depths", func(t *testing.T) {
		expand := &GoDataExpandQuery{
			ExpandItems: []*ExpandItem{
				{
					Path: []*Token{{Value: "orders"}},
					Expand: &GoDataExpandQuery{
						ExpandItems: []*ExpandItem{
							{Path: []*Token{{Value: "items"}}},
						},
					},
				},
				{
					Path: []*Token{{Value: "profile"}},
					// No nested expand
				},
			},
		}
		depth := calculateExpandDepth(expand)
		if depth != 2 {
			t.Errorf("calculateExpandDepth(multiple branches) = %d, want 2 (max depth)", depth)
		}
	})
}

func TestValidateQueryOptions_WithExpand(t *testing.T) {
	config := DefaultValidationConfig()
	config.MaxExpandDepth = 2

	t.Run("valid expand depth", func(t *testing.T) {
		options := &QueryOptions{
			Expand: &GoDataExpandQuery{
				ExpandItems: []*ExpandItem{
					{Path: []*Token{{Value: "orders"}}},
				},
			},
		}
		err := ValidateQueryOptions(options, config)
		if err != nil {
			t.Errorf("ValidateQueryOptions with valid expand should not error, got: %v", err)
		}
	})

	t.Run("expand depth exceeds max", func(t *testing.T) {
		options := &QueryOptions{
			Expand: &GoDataExpandQuery{
				ExpandItems: []*ExpandItem{
					{
						Path: []*Token{{Value: "orders"}},
						Expand: &GoDataExpandQuery{
							ExpandItems: []*ExpandItem{
								{
									Path: []*Token{{Value: "items"}},
									Expand: &GoDataExpandQuery{
										ExpandItems: []*ExpandItem{
											{Path: []*Token{{Value: "product"}}},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		err := ValidateQueryOptions(options, config)
		if err == nil {
			t.Error("ValidateQueryOptions with excessive expand depth should error")
		}
	})
}
