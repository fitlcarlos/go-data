package odata

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationConfig configurações de validação de inputs
type ValidationConfig struct {
	// Limites de tamanho
	MaxFilterLength  int // Tamanho máximo de string $filter
	MaxSearchLength  int // Tamanho máximo de string $search
	MaxSelectLength  int // Tamanho máximo de string $select
	MaxOrderByLength int // Tamanho máximo de string $orderby
	MaxExpandDepth   int // Profundidade máxima de $expand
	MaxTopValue      int // Valor máximo de $top

	// Validações de caracteres
	AllowedPropertyChars string // Regex de caracteres permitidos em nomes de propriedades

	// XSS Protection
	EnableXSSProtection bool // Habilita sanitização de XSS

	// Validações customizadas
	CustomPropertyValidator func(name string) error
}

// DefaultValidationConfig retorna configuração padrão de validação
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxFilterLength:      DefaultMaxFilterLength,
		MaxSearchLength:      DefaultMaxSearchLength,
		MaxSelectLength:      DefaultMaxSelectLength,
		MaxOrderByLength:     DefaultMaxOrderByLength,
		MaxExpandDepth:       DefaultMaxExpandDepth,
		MaxTopValue:          DefaultMaxTopValue,
		AllowedPropertyChars: PropertyNamePattern,
		EnableXSSProtection:  true,
	}
}

var (
	// Compiled regex patterns for performance
	propertyNameRegex    *regexp.Regexp
	sqlInjectionPatterns []*regexp.Regexp
	xssPatterns          []*regexp.Regexp
)

func init() {
	// Compila regex patterns na inicialização
	propertyNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\.]+$`)

	// Padrões comuns de SQL injection
	sqlInjectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(\s|^)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|<script)(\s|$)`),
		regexp.MustCompile(`(?i)(--|#|\/\*|\*\/)`),                         // Comentários SQL
		regexp.MustCompile(`(?i)(\bor\b|\band\b)\s+\d+\s*=\s*\d+`),         // or 1=1, and 1=1
		regexp.MustCompile(`(?i)(\bor\b|\band\b)\s+['"].*['"]\s*=\s*['"]`), // or 'x'='x'
	}

	// Padrões comuns de XSS
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`), // onclick=, onload=, etc
		regexp.MustCompile(`(?i)<img[^>]*>`),
		regexp.MustCompile(`(?i)<object[^>]*>`),
		regexp.MustCompile(`(?i)<embed[^>]*>`),
	}
}

// ValidateFilterQuery valida query string de $filter
func ValidateFilterQuery(filter string, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	// Valida tamanho
	if len(filter) > config.MaxFilterLength {
		return fmt.Errorf("filter query too long: %d bytes (max: %d)", len(filter), config.MaxFilterLength)
	}

	// Valida UTF-8 válido
	if !utf8.ValidString(filter) {
		return fmt.Errorf("filter contains invalid UTF-8 characters")
	}

	// Detecta padrões de SQL injection
	if err := detectSQLInjection(filter); err != nil {
		return fmt.Errorf("filter contains suspicious SQL patterns: %w", err)
	}

	// Sanitiza XSS se habilitado
	if config.EnableXSSProtection {
		if err := detectXSS(filter); err != nil {
			return fmt.Errorf("filter contains suspicious XSS patterns: %w", err)
		}
	}

	return nil
}

// ValidateSearchQuery valida query string de $search
func ValidateSearchQuery(search string, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	// Valida tamanho
	if len(search) > config.MaxSearchLength {
		return fmt.Errorf("search query too long: %d bytes (max: %d)", len(search), config.MaxSearchLength)
	}

	// Valida UTF-8 válido
	if !utf8.ValidString(search) {
		return fmt.Errorf("search contains invalid UTF-8 characters")
	}

	// Detecta padrões de SQL injection
	if err := detectSQLInjection(search); err != nil {
		return fmt.Errorf("search contains suspicious SQL patterns: %w", err)
	}

	// Sanitiza XSS se habilitado
	if config.EnableXSSProtection {
		if err := detectXSS(search); err != nil {
			return fmt.Errorf("search contains suspicious XSS patterns: %w", err)
		}
	}

	return nil
}

// ValidateSelectString valida query string de $select
func ValidateSelectString(selectStr string, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	// Valida tamanho
	if len(selectStr) > config.MaxSelectLength {
		return fmt.Errorf("select query too long: %d bytes (max: %d)", len(selectStr), config.MaxSelectLength)
	}

	// Valida UTF-8 válido
	if !utf8.ValidString(selectStr) {
		return fmt.Errorf("select contains invalid UTF-8 characters")
	}

	// Valida cada propriedade individual
	properties := strings.Split(selectStr, ",")
	for _, prop := range properties {
		prop = strings.TrimSpace(prop)
		if err := ValidatePropertyName(prop, config); err != nil {
			return fmt.Errorf("invalid property in select: %w", err)
		}
	}

	return nil
}

// ValidateOrderByQuery valida query string de $orderby
func ValidateOrderByQuery(orderBy string, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	// Valida tamanho
	if len(orderBy) > config.MaxOrderByLength {
		return fmt.Errorf("orderby query too long: %d bytes (max: %d)", len(orderBy), config.MaxOrderByLength)
	}

	// Valida UTF-8 válido
	if !utf8.ValidString(orderBy) {
		return fmt.Errorf("orderby contains invalid UTF-8 characters")
	}

	// Valida cada expressão de ordenação
	expressions := strings.Split(orderBy, ",")
	for _, expr := range expressions {
		expr = strings.TrimSpace(expr)

		// Remove direção (asc/desc)
		parts := strings.Fields(expr)
		if len(parts) == 0 {
			continue
		}

		propertyName := parts[0]
		if err := ValidatePropertyName(propertyName, config); err != nil {
			return fmt.Errorf("invalid property in orderby: %w", err)
		}

		// Valida direção se presente
		if len(parts) > 1 {
			direction := strings.ToLower(parts[1])
			if direction != "asc" && direction != "desc" {
				return fmt.Errorf("invalid orderby direction: %s (must be 'asc' or 'desc')", parts[1])
			}
		}
	}

	return nil
}

// ValidatePropertyName valida nome de propriedade
func ValidatePropertyName(name string, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	if name == "" {
		return fmt.Errorf("property name cannot be empty")
	}

	// Valida comprimento razoável
	if len(name) > 100 {
		return fmt.Errorf("property name too long: %s", name)
	}

	// Valida caracteres permitidos
	pattern := config.AllowedPropertyChars
	if pattern == "" {
		pattern = `^[a-zA-Z0-9_\.]+$`
	}

	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return fmt.Errorf("error validating property name: %w", err)
	}

	if !matched {
		return fmt.Errorf("property name contains invalid characters: %s", name)
	}

	// Validação customizada se definida
	if config.CustomPropertyValidator != nil {
		if err := config.CustomPropertyValidator(name); err != nil {
			return err
		}
	}

	return nil
}

// ValidateExpandDepth valida profundidade de $expand
func ValidateExpandDepth(expand []ExpandOption, maxDepth int, currentDepth int) error {
	if currentDepth > maxDepth {
		return fmt.Errorf("expand depth exceeds maximum: %d (max: %d)", currentDepth, maxDepth)
	}

	for _, exp := range expand {
		if len(exp.Expand) > 0 {
			if err := ValidateExpandDepth(exp.Expand, maxDepth, currentDepth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// ValidateTopValue valida valor de $top
func ValidateTopValue(top int, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	if top < 0 {
		return fmt.Errorf("top value cannot be negative: %d", top)
	}

	if top > config.MaxTopValue {
		return fmt.Errorf("top value exceeds maximum: %d (max: %d)", top, config.MaxTopValue)
	}

	return nil
}

// ValidateSkipValue valida valor de $skip
func ValidateSkipValue(skip int) error {
	if skip < 0 {
		return fmt.Errorf("skip value cannot be negative: %d", skip)
	}

	// Limite razoável para prevenir ataques de paginação excessiva
	maxSkip := 100000 // 100k registros
	if skip > maxSkip {
		return fmt.Errorf("skip value too large: %d (max: %d)", skip, maxSkip)
	}

	return nil
}

// SanitizeInput sanitiza input removendo padrões perigosos
func SanitizeInput(input string, config *ValidationConfig) string {
	if config == nil {
		config = DefaultValidationConfig()
	}

	if !config.EnableXSSProtection {
		return input
	}

	sanitized := input

	// Remove padrões de XSS
	for _, pattern := range xssPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, "")
	}

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Limita caracteres de controle perigosos
	sanitized = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F]`).ReplaceAllString(sanitized, "")

	return sanitized
}

// detectSQLInjection detecta padrões comuns de SQL injection
func detectSQLInjection(input string) error {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("detected potential SQL injection pattern")
		}
	}
	return nil
}

// detectXSS detecta padrões comuns de XSS
func detectXSS(input string) error {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("detected potential XSS pattern")
		}
	}
	return nil
}

// ValidateQueryOptions valida todas as opções de query OData
func ValidateQueryOptions(options *QueryOptions, config *ValidationConfig) error {
	if options == nil {
		return nil
	}

	if config == nil {
		config = DefaultValidationConfig()
	}

	// Validar $filter
	if options.Filter != nil {
		filterStr := options.Filter.RawValue
		if err := ValidateFilterQuery(filterStr, config); err != nil {
			return fmt.Errorf("invalid $filter: %w", err)
		}
	}

	// Validar $select
	if options.Select != nil {
		selectStr := options.Select.RawValue
		if len(selectStr) > config.MaxSelectLength {
			return fmt.Errorf("$select too long: %d bytes (max: %d)", len(selectStr), config.MaxSelectLength)
		}

		// Validar cada item de seleção
		for _, item := range options.Select.SelectItems {
			for _, segment := range item.Segments {
				if segment != nil && segment.Value != "" {
					if err := ValidatePropertyName(segment.Value, config); err != nil {
						return fmt.Errorf("invalid property in $select: %w", err)
					}
				}
			}
		}
	}

	// Validar $expand
	if options.Expand != nil {
		// Validar profundidade de expand
		depth := calculateExpandDepth(options.Expand)
		if depth > config.MaxExpandDepth {
			return fmt.Errorf("$expand depth exceeds maximum: %d > %d", depth, config.MaxExpandDepth)
		}
	}

	// Validar $orderby
	if options.OrderBy != "" {
		if len(options.OrderBy) > config.MaxOrderByLength {
			return fmt.Errorf("$orderby too long: %d bytes (max: %d)", len(options.OrderBy), config.MaxOrderByLength)
		}

		// Validar se não contém SQL injection
		if err := detectSQLInjection(options.OrderBy); err != nil {
			return fmt.Errorf("$orderby contains suspicious patterns: %w", err)
		}
	}

	// Validar $search
	if options.Search != nil {
		searchStr := fmt.Sprintf("%v", options.Search)
		if len(searchStr) > config.MaxSearchLength {
			return fmt.Errorf("$search too long: %d bytes (max: %d)", len(searchStr), config.MaxSearchLength)
		}

		// Validar XSS se habilitado
		if config.EnableXSSProtection {
			if err := detectXSS(searchStr); err != nil {
				return fmt.Errorf("$search contains suspicious patterns: %w", err)
			}
		}
	}

	// Validar $top
	if options.Top != nil {
		topValue := int(*options.Top)
		if topValue < 0 {
			return fmt.Errorf("$top cannot be negative: %d", topValue)
		}
		if topValue > config.MaxTopValue {
			return fmt.Errorf("$top exceeds maximum: %d > %d", topValue, config.MaxTopValue)
		}
	}

	// Validar $skip
	if options.Skip != nil {
		skipValue := int(*options.Skip)
		if skipValue < 0 {
			return fmt.Errorf("$skip cannot be negative: %d", skipValue)
		}
	}

	// Validar $count (sempre válido se presente)
	// A validação de habilitação é feita no parser

	// Validar $compute
	if options.Compute != nil {
		// Compute pode conter expressões complexas
		// Validar contra SQL injection
		computeStr := fmt.Sprintf("%v", options.Compute)
		if err := detectSQLInjection(computeStr); err != nil {
			return fmt.Errorf("$compute contains suspicious patterns: %w", err)
		}
	}

	return nil
}

// calculateExpandDepth calcula a profundidade máxima de um expand
func calculateExpandDepth(expand *GoDataExpandQuery) int {
	if expand == nil || len(expand.ExpandItems) == 0 {
		return 0
	}

	maxDepth := 1
	for _, item := range expand.ExpandItems {
		if item.Expand != nil {
			depth := 1 + calculateExpandDepth(item.Expand)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}
