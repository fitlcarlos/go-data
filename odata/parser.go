package odata

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Singleton global para otimização de performance
var (
	globalParser *ODataParser
	parserOnce   sync.Once
)

// GetGlobalParser retorna uma instância singleton do parser para melhor performance
func GetGlobalParser() *ODataParser {
	parserOnce.Do(func() {
		globalParser = &ODataParser{
			supportedParams: map[string]bool{
				"$filter":      true,
				"$orderby":     true,
				"$select":      true,
				"$expand":      true,
				"$skip":        true,
				"$top":         true,
				"$count":       true,
				"$compute":     true,
				"$search":      true,
				"$format":      true,
				"$apply":       true,
				"$inlinecount": true,
			},
		}
	})
	return globalParser
}

// ODataParser é responsável por fazer o parsing das consultas OData
type ODataParser struct {
	supportedParams map[string]bool
}

// NewODataParser cria uma nova instância do parser
func NewODataParser() *ODataParser {
	return GetGlobalParser()
}

// ComplianceConfig define configurações de compliance OData
type ComplianceConfig int

const (
	ComplianceStrict ComplianceConfig = iota
	ComplianceIgnoreUnknownKeywords
	ComplianceIgnoreDuplicateKeywords
)

// getCaseInsensitiveValue busca um valor nos parâmetros de consulta de forma case insensitive
func (p *ODataParser) getCaseInsensitiveValue(values url.Values, key string) string {
	// Primeiro tenta buscar exatamente como foi passado
	if value := values.Get(key); value != "" {
		return value
	}

	// Se não encontrar, busca de forma case insensitive
	for k, v := range values {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}

	return ""
}

// validateQueryParameters valida os parâmetros da query seguindo compliance OData
func (p *ODataParser) validateQueryParameters(values url.Values, config ComplianceConfig) error {
	for key, vals := range values {
		// Verifica se o parâmetro é suportado
		if !p.supportedParams[strings.ToLower(key)] && config == ComplianceStrict {
			return fmt.Errorf("query parameter '%s' is not supported", key)
		}

		// Verifica duplicatas
		if len(vals) > 1 && config != ComplianceIgnoreDuplicateKeywords {
			return fmt.Errorf("query parameter '%s' cannot be specified more than once", key)
		}
	}
	return nil
}

// ParseQueryOptions faz o parsing das opções de consulta OData da URL
func (p *ODataParser) ParseQueryOptions(values url.Values) (QueryOptions, error) {
	return p.ParseQueryOptionsWithConfig(values, ComplianceStrict)
}

// ParseQueryOptionsWithConfig faz o parsing com configuração de compliance
func (p *ODataParser) ParseQueryOptionsWithConfig(values url.Values, config ComplianceConfig) (QueryOptions, error) {
	options := QueryOptions{}

	// Validação rigorosa dos parâmetros
	if err := p.validateQueryParameters(values, config); err != nil {
		return options, err
	}

	// Parse $filter (case insensitive)
	if filter := p.getCaseInsensitiveValue(values, "$filter"); filter != "" {
		filterQuery, err := ParseFilterString(context.Background(), filter)
		if err != nil {
			return options, fmt.Errorf("invalid $filter: %w", err)
		}
		options.Filter = filterQuery
	}

	// Parse $orderby (case insensitive)
	if orderBy := p.getCaseInsensitiveValue(values, "$orderby"); orderBy != "" {
		options.OrderBy = orderBy
	}

	// Parse $select (case insensitive)
	if selectStr := p.getCaseInsensitiveValue(values, "$select"); selectStr != "" {
		selectQuery, err := ParseSelectString(context.Background(), selectStr)
		if err != nil {
			return options, fmt.Errorf("failed to parse $select: %w", err)
		}
		options.Select = selectQuery
	}

	// Parse $expand (case insensitive)
	if expandStr := p.getCaseInsensitiveValue(values, "$expand"); expandStr != "" {
		expandQuery, err := ParseExpandString(context.Background(), expandStr)
		if err != nil {
			return options, fmt.Errorf("invalid $expand value: %w", err)
		}
		options.Expand = expandQuery
	}

	// Parse $skip (case insensitive)
	if skipStr := p.getCaseInsensitiveValue(values, "$skip"); skipStr != "" {
		skipQuery, err := ParseSkipString(context.Background(), skipStr)
		if err != nil {
			return options, fmt.Errorf("invalid $skip value: %w", err)
		}
		options.Skip = skipQuery
	}

	// Parse $top (case insensitive)
	if topStr := p.getCaseInsensitiveValue(values, "$top"); topStr != "" {
		topQuery, err := ParseTopString(context.Background(), topStr)
		if err != nil {
			return options, fmt.Errorf("invalid $top value: %w", err)
		}
		options.Top = topQuery
	}

	// Parse $count (case insensitive) - Otimizado como bool simples
	if countStr := p.getCaseInsensitiveValue(values, "$count"); countStr != "" {
		countQuery, err := ParseCountString(countStr)
		if err != nil {
			return options, fmt.Errorf("invalid $count: %w", err)
		}
		options.Count = countQuery
	}

	// Parse $compute (case insensitive)
	if computeStr := p.getCaseInsensitiveValue(values, "$compute"); computeStr != "" {
		// O parsing real será feito pelo ComputeParser no EntityService
		// Aqui apenas armazenamos a string para parsing posterior
		options.Compute = &ComputeOption{Expressions: []ComputeExpression{}}
		// Temporariamente armazenamos a string raw no primeiro expression
		options.Compute.Expressions = append(options.Compute.Expressions, ComputeExpression{Expression: computeStr})
	}

	// Parse $search (case insensitive)
	if searchStr := p.getCaseInsensitiveValue(values, "$search"); searchStr != "" {
		// O parsing real será feito pelo SearchParser no EntityService
		// Aqui apenas armazenamos a string para parsing posterior
		options.Search = &SearchOption{RawQuery: searchStr}
	}

	return options, nil
}

// ParseFilter faz o parsing de uma expressão de filtro OData
func (p *ODataParser) ParseFilter(filter string) ([]FilterExpression, error) {
	if filter == "" {
		return nil, nil
	}

	// Implementação básica para filtros simples
	// Suporta operadores como: eq, ne, gt, ge, lt, le, contains, startswith, endswith
	expressions := []FilterExpression{}

	// Remove espaços extras
	filter = strings.TrimSpace(filter)

	// Split por 'and' primeiro (implementação básica, case insensitive)
	parts := p.splitCaseInsensitive(filter, " and ")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		expr, err := p.parseFilterExpression(part)
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}

	return expressions, nil
}

// splitCaseInsensitive divide uma string por um separador de forma case insensitive
func (p *ODataParser) splitCaseInsensitive(text, separator string) []string {
	if text == "" {
		return []string{}
	}

	lowerText := strings.ToLower(text)
	lowerSeparator := strings.ToLower(separator)

	var parts []string
	start := 0

	for {
		index := strings.Index(lowerText[start:], lowerSeparator)
		if index == -1 {
			// Não encontrou mais separadores, adiciona o resto
			parts = append(parts, text[start:])
			break
		}

		// Adiciona a parte antes do separador
		parts = append(parts, text[start:start+index])
		start = start + index + len(separator)
	}

	return parts
}

// parseFilterExpression faz o parsing de uma expressão de filtro individual
func (p *ODataParser) parseFilterExpression(expression string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Converte a expressão para lowercase para comparação case insensitive
	lowerExpression := strings.ToLower(expression)

	// Primeiro, verifica se é uma função de string ou matemática seguida de um operador
	stringFunctions := []string{"length(", "tolower(", "toupper(", "trim(", "concat(", "indexof(", "substring("}
	for _, fn := range stringFunctions {
		if strings.Contains(lowerExpression, fn) {
			return p.parseStringFunctionExpression(expression, fn)
		}
	}

	// Verifica operadores matemáticos
	mathFunctions := []string{"add(", "sub(", "mul(", "div(", "mod("}
	for _, fn := range mathFunctions {
		if strings.Contains(lowerExpression, fn) {
			return p.parseMathFunctionExpression(expression, fn)
		}
	}

	// Verifica funções de data/hora
	datetimeFunctions := []string{"year(", "month(", "day(", "hour(", "minute(", "second(", "now("}
	for _, fn := range datetimeFunctions {
		if strings.Contains(lowerExpression, fn) {
			return p.parseDateTimeFunctionExpression(expression, fn)
		}
	}

	// Depois verifica funções tradicionais (contains, startswith, endswith)
	traditionalFunctions := []string{"contains(", "startswith(", "endswith("}
	for _, fn := range traditionalFunctions {
		if strings.Contains(lowerExpression, fn) {
			return p.parseOperatorExpression(expression, fn)
		}
	}

	// Por último, verifica operadores simples
	simpleOperators := []string{" eq ", " ne ", " gt ", " ge ", " lt ", " le "}
	for _, op := range simpleOperators {
		if strings.Contains(lowerExpression, op) {
			return p.parseOperatorExpression(expression, op)
		}
	}

	return expr, fmt.Errorf("unsupported filter expression: %s", expression)
}

// parseOperatorExpression faz o parsing de operadores específicos
func (p *ODataParser) parseOperatorExpression(expression, operator string) (FilterExpression, error) {
	expr := FilterExpression{}

	switch operator {
	case "contains(":
		return p.parseFunctionExpression(expression, "contains")
	case "startswith(":
		return p.parseFunctionExpression(expression, "startswith")
	case "endswith(":
		return p.parseFunctionExpression(expression, "endswith")
	case "length(":
		return p.parseFunctionExpression(expression, "length")
	case "tolower(":
		return p.parseFunctionExpression(expression, "tolower")
	case "toupper(":
		return p.parseFunctionExpression(expression, "toupper")
	case "trim(":
		return p.parseFunctionExpression(expression, "trim")
	case "concat(":
		return p.parseFunctionExpression(expression, "concat")
	case "indexof(":
		return p.parseFunctionExpression(expression, "indexof")
	case "substring(":
		return p.parseFunctionExpression(expression, "substring")
	default:
		// Operadores como eq, ne, gt, etc. (case insensitive)
		actualOperator := p.findCaseInsensitiveOperator(expression, operator)
		if actualOperator == "" {
			return expr, fmt.Errorf("operator %s not found in expression: %s", operator, expression)
		}

		parts := strings.Split(expression, actualOperator)
		if len(parts) != 2 {
			return expr, fmt.Errorf("invalid expression: %s", expression)
		}

		expr.Property = strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove aspas da string
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = value[1 : len(value)-1]
		}

		expr.Value = value
		expr.Operator = FilterOperator(strings.TrimSpace(strings.ToLower(actualOperator)))
	}

	return expr, nil
}

// findCaseInsensitiveOperator encontra o operador real na expressão de forma case insensitive
func (p *ODataParser) findCaseInsensitiveOperator(expression, targetOperator string) string {
	lowerExpression := strings.ToLower(expression)
	lowerOperator := strings.ToLower(targetOperator)

	index := strings.Index(lowerExpression, lowerOperator)
	if index == -1 {
		return ""
	}

	// Retorna o operador real da expressão original
	return expression[index : index+len(targetOperator)]
}

// parseStringFunctionExpression faz o parsing de funções de string seguidas de operadores
func (p *ODataParser) parseStringFunctionExpression(expression, function string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Remove "(" do final da função para obter apenas o nome
	functionName := strings.TrimSuffix(function, "(")

	// Exemplo: length(Name) eq 5
	// Primeiro, encontra onde a função termina
	lowerExpression := strings.ToLower(expression)
	lowerFunction := strings.ToLower(function)

	start := strings.Index(lowerExpression, lowerFunction)
	if start == -1 {
		return expr, fmt.Errorf("function %s not found in expression: %s", function, expression)
	}

	// Encontra o final da função
	openParen := 1
	i := start + len(function)
	for i < len(expression) && openParen > 0 {
		if expression[i] == '(' {
			openParen++
		} else if expression[i] == ')' {
			openParen--
		}
		i++
	}

	if openParen > 0 {
		return expr, fmt.Errorf("unmatched parentheses in function: %s", expression)
	}

	// Extrai a parte da função: length(Name)
	functionPart := expression[start:i]

	// Extrai a parte do operador: eq 5
	operatorPart := strings.TrimSpace(expression[i:])

	// Faz o parsing da função
	functionExpr, err := p.parseFunctionExpression(functionPart, functionName)
	if err != nil {
		return expr, err
	}

	// Faz o parsing do operador e valor
	operatorExpression, err := p.parseOperatorPart(operatorPart)
	if err != nil {
		return expr, err
	}

	// Combina os resultados
	expr.Property = functionExpr.Property
	expr.Operator = functionExpr.Operator
	expr.Arguments = functionExpr.Arguments
	expr.Value = operatorExpression.Value

	return expr, nil
}

// parseOperatorPart faz o parsing da parte do operador (ex: "eq 5")
func (p *ODataParser) parseOperatorPart(operatorPart string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Adiciona espaços se não existirem para padronizar
	operatorPart = " " + strings.TrimSpace(operatorPart) + " "

	operators := []string{" eq ", " ne ", " gt ", " ge ", " lt ", " le "}
	lowerOperatorPart := strings.ToLower(operatorPart)

	for _, op := range operators {
		if strings.Contains(lowerOperatorPart, op) {
			// Encontra o operador real na string original
			actualOperator := p.findCaseInsensitiveOperator(operatorPart, op)
			if actualOperator != "" {
				parts := strings.Split(operatorPart, actualOperator)
				if len(parts) == 2 {
					value := strings.TrimSpace(parts[1])
					// Remove aspas se for string
					if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
						value = value[1 : len(value)-1]
					}
					expr.Value = value
					return expr, nil
				}
			}
		}
	}

	return expr, fmt.Errorf("invalid operator part: %s", strings.TrimSpace(operatorPart))
}

// parseMathFunctionExpression faz o parsing de funções matemáticas seguidas de operadores
func (p *ODataParser) parseMathFunctionExpression(expression, function string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Remove "(" do final da função para obter apenas o nome
	functionName := strings.TrimSuffix(function, "(")

	// Exemplo: add(Price, 10) eq 100
	// Primeiro, encontra onde a função termina
	lowerExpression := strings.ToLower(expression)
	lowerFunction := strings.ToLower(function)

	start := strings.Index(lowerExpression, lowerFunction)
	if start == -1 {
		return expr, fmt.Errorf("math function %s not found in expression: %s", function, expression)
	}

	// Encontra o final da função
	openParen := 1
	i := start + len(function)
	for i < len(expression) && openParen > 0 {
		if expression[i] == '(' {
			openParen++
		} else if expression[i] == ')' {
			openParen--
		}
		i++
	}

	if openParen > 0 {
		return expr, fmt.Errorf("unmatched parentheses in math function: %s", expression)
	}

	// Extrai a parte da função: add(Price, 10)
	functionPart := expression[start:i]

	// Extrai a parte do operador: eq 100
	operatorPart := strings.TrimSpace(expression[i:])

	// Faz o parsing da função matemática
	functionExpr, err := p.parseMathFunction(functionPart, functionName)
	if err != nil {
		return expr, err
	}

	// Faz o parsing do operador e valor
	operatorExpression, err := p.parseOperatorPart(operatorPart)
	if err != nil {
		return expr, err
	}

	// Combina os resultados
	expr.Property = functionExpr.Property
	expr.Operator = functionExpr.Operator
	expr.Arguments = functionExpr.Arguments
	expr.Value = operatorExpression.Value

	return expr, nil
}

// parseMathFunction faz o parsing de funções matemáticas
func (p *ODataParser) parseMathFunction(expression, function string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Encontra os argumentos da função
	lowerExpression := strings.ToLower(expression)
	lowerFunction := strings.ToLower(function)

	start := strings.Index(lowerExpression, lowerFunction+"(")
	if start == -1 {
		return expr, fmt.Errorf("invalid math function expression: %s", expression)
	}

	// Encontra o final da função
	openParen := 1
	i := start + len(function) + 1
	for i < len(expression) && openParen > 0 {
		if expression[i] == '(' {
			openParen++
		} else if expression[i] == ')' {
			openParen--
		}
		i++
	}

	if openParen > 0 {
		return expr, fmt.Errorf("unmatched parentheses in: %s", expression)
	}

	// Extrai os argumentos da expressão original
	args := expression[start+len(function)+1 : i-1]
	argParts := strings.Split(args, ",")

	// Operadores matemáticos precisam de exatamente 2 argumentos
	if len(argParts) != 2 {
		return expr, fmt.Errorf("math function %s requires exactly 2 arguments, got %d", function, len(argParts))
	}

	expr.Property = strings.TrimSpace(argParts[0])
	expr.Operator = FilterOperator(strings.ToLower(function))

	// Processa o segundo argumento
	secondArg := strings.TrimSpace(argParts[1])

	// Remove aspas se for string
	if strings.HasPrefix(secondArg, "'") && strings.HasSuffix(secondArg, "'") {
		secondArg = secondArg[1 : len(secondArg)-1]
	}

	// Adiciona o segundo argumento
	expr.Arguments = []interface{}{secondArg}

	return expr, nil
}

// parseDateTimeFunctionExpression faz o parsing de funções de data/hora seguidas de operadores
func (p *ODataParser) parseDateTimeFunctionExpression(expression, function string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Remove "(" do final da função para obter apenas o nome
	functionName := strings.TrimSuffix(function, "(")

	// Exemplo: year(DateField) eq 2023
	// Primeiro, encontra onde a função termina
	lowerExpression := strings.ToLower(expression)
	lowerFunction := strings.ToLower(function)

	start := strings.Index(lowerExpression, lowerFunction)
	if start == -1 {
		return expr, fmt.Errorf("datetime function %s not found in expression: %s", function, expression)
	}

	// Encontra o final da função
	openParen := 1
	i := start + len(function)
	for i < len(expression) && openParen > 0 {
		if expression[i] == '(' {
			openParen++
		} else if expression[i] == ')' {
			openParen--
		}
		i++
	}

	if openParen > 0 {
		return expr, fmt.Errorf("unmatched parentheses in datetime function: %s", expression)
	}

	// Extrai a parte da função: year(DateField)
	functionPart := expression[start:i]

	// Extrai a parte do operador: eq 2023
	operatorPart := strings.TrimSpace(expression[i:])

	// Faz o parsing da função de data/hora
	functionExpr, err := p.parseDateTimeFunction(functionPart, functionName)
	if err != nil {
		return expr, err
	}

	// Faz o parsing do operador e valor
	operatorExpression, err := p.parseOperatorPart(operatorPart)
	if err != nil {
		return expr, err
	}

	// Combina os resultados
	expr.Property = functionExpr.Property
	expr.Operator = functionExpr.Operator
	expr.Arguments = functionExpr.Arguments
	expr.Value = operatorExpression.Value

	return expr, nil
}

// parseDateTimeFunction faz o parsing de funções de data/hora
func (p *ODataParser) parseDateTimeFunction(expression, function string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Encontra os argumentos da função
	lowerExpression := strings.ToLower(expression)
	lowerFunction := strings.ToLower(function)

	start := strings.Index(lowerExpression, lowerFunction+"(")
	if start == -1 {
		return expr, fmt.Errorf("invalid datetime function expression: %s", expression)
	}

	// Encontra o final da função
	openParen := 1
	i := start + len(function) + 1
	for i < len(expression) && openParen > 0 {
		if expression[i] == '(' {
			openParen++
		} else if expression[i] == ')' {
			openParen--
		}
		i++
	}

	if openParen > 0 {
		return expr, fmt.Errorf("unmatched parentheses in: %s", expression)
	}

	// Extrai os argumentos da expressão original
	args := expression[start+len(function)+1 : i-1]
	argParts := strings.Split(args, ",")

	expr.Operator = FilterOperator(strings.ToLower(function))

	switch strings.ToLower(function) {
	case "year", "month", "day", "hour", "minute", "second":
		// Funções que precisam de exatamente 1 argumento (campo de data/hora)
		if len(argParts) != 1 {
			return expr, fmt.Errorf("datetime function %s requires exactly 1 argument, got %d", function, len(argParts))
		}
		expr.Property = strings.TrimSpace(argParts[0])

	case "now":
		// Função now() não precisa de argumentos
		if len(argParts) > 0 && strings.TrimSpace(argParts[0]) != "" {
			return expr, fmt.Errorf("datetime function now requires no arguments, got %d", len(argParts))
		}
		expr.Property = "now()"

	default:
		return expr, fmt.Errorf("unsupported datetime function: %s", function)
	}

	return expr, nil
}

// parseFunctionExpression faz o parsing de funções OData
func (p *ODataParser) parseFunctionExpression(expression, function string) (FilterExpression, error) {
	expr := FilterExpression{}

	// Exemplo: contains(Name, 'John') ou CONTAINS(Name, 'John')
	lowerExpression := strings.ToLower(expression)
	lowerFunction := strings.ToLower(function)

	start := strings.Index(lowerExpression, lowerFunction+"(")
	if start == -1 {
		return expr, fmt.Errorf("invalid function expression: %s", expression)
	}

	// Encontra o final da função
	openParen := 1
	i := start + len(function) + 1
	for i < len(expression) && openParen > 0 {
		if expression[i] == '(' {
			openParen++
		} else if expression[i] == ')' {
			openParen--
		}
		i++
	}

	if openParen > 0 {
		return expr, fmt.Errorf("unmatched parentheses in: %s", expression)
	}

	// Extrai os argumentos da expressão original
	args := expression[start+len(function)+1 : i-1]
	argParts := strings.Split(args, ",")

	// Processa argumentos baseado no tipo de função
	expr.Operator = FilterOperator(strings.ToLower(function))

	switch strings.ToLower(function) {
	case "length", "tolower", "toupper", "trim":
		// Funções que precisam de 1 argumento
		if len(argParts) != 1 {
			return expr, fmt.Errorf("function %s requires 1 argument, got %d", function, len(argParts))
		}
		expr.Property = strings.TrimSpace(argParts[0])

	case "contains", "startswith", "endswith", "indexof":
		// Funções que precisam de 2 argumentos
		if len(argParts) != 2 {
			return expr, fmt.Errorf("function %s requires 2 arguments, got %d", function, len(argParts))
		}
		expr.Property = strings.TrimSpace(argParts[0])
		value := strings.TrimSpace(argParts[1])
		// Remove aspas da string
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			value = value[1 : len(value)-1]
		}
		expr.Value = value

	case "concat":
		// Concat pode ter 2 ou mais argumentos
		if len(argParts) < 2 {
			return expr, fmt.Errorf("function concat requires at least 2 arguments, got %d", len(argParts))
		}
		expr.Property = strings.TrimSpace(argParts[0])
		var arguments []interface{}
		for i := 1; i < len(argParts); i++ {
			value := strings.TrimSpace(argParts[i])
			// Remove aspas da string
			if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
				value = value[1 : len(value)-1]
			}
			arguments = append(arguments, value)
		}
		expr.Arguments = arguments

	case "substring":
		// Substring pode ter 2 ou 3 argumentos: substring(string, start) ou substring(string, start, length)
		if len(argParts) < 2 || len(argParts) > 3 {
			return expr, fmt.Errorf("function substring requires 2 or 3 arguments, got %d", len(argParts))
		}
		expr.Property = strings.TrimSpace(argParts[0])
		var arguments []interface{}
		for i := 1; i < len(argParts); i++ {
			value := strings.TrimSpace(argParts[i])
			// Converte para número se for um índice
			if num, err := strconv.Atoi(value); err == nil {
				arguments = append(arguments, num)
			} else {
				// Remove aspas se for string
				if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
					value = value[1 : len(value)-1]
				}
				arguments = append(arguments, value)
			}
		}
		expr.Arguments = arguments

	default:
		return expr, fmt.Errorf("unsupported function: %s", function)
	}

	return expr, nil
}

// ParseOrderBy faz o parsing de uma expressão de ordenação OData
func (p *ODataParser) ParseOrderBy(orderBy string) ([]OrderByExpression, error) {
	if orderBy == "" {
		return nil, nil
	}

	expressions := []OrderByExpression{}
	parts := strings.Split(orderBy, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		expr := OrderByExpression{
			Direction: OrderAsc, // Padrão
		}

		// Verifica se tem direção especificada (case insensitive)
		lowerPart := strings.ToLower(part)
		if strings.HasSuffix(lowerPart, " desc") {
			expr.Property = strings.TrimSpace(part[:len(part)-5])
			expr.Direction = OrderDesc
		} else if strings.HasSuffix(lowerPart, " asc") {
			expr.Property = strings.TrimSpace(part[:len(part)-4])
			expr.Direction = OrderAsc
		} else {
			expr.Property = part
		}

		expressions = append(expressions, expr)
	}

	return expressions, nil
}

// ParseExpand faz o parsing de uma expressão de expansão OData
func (p *ODataParser) ParseExpand(expand string) ([]ExpandOption, error) {
	if expand == "" {
		return nil, nil
	}

	var expandOptions []ExpandOption

	// Divide as opções de expand por vírgula no nível superior
	expandParts := p.splitExpandParts(expand)

	for _, part := range expandParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		expandOption, err := p.parseExpandOption(part)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expand option '%s': %w", part, err)
		}

		expandOptions = append(expandOptions, expandOption)
	}

	return expandOptions, nil
}

// splitExpandParts divide as partes do expand respeitando parênteses aninhados
func (p *ODataParser) splitExpandParts(expand string) []string {
	var parts []string
	var current strings.Builder
	var parenCount int

	for _, char := range expand {
		switch char {
		case '(':
			parenCount++
			current.WriteRune(char)
		case ')':
			parenCount--
			current.WriteRune(char)
		case ',':
			if parenCount == 0 {
				// Vírgula no nível superior, divide aqui
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Adiciona a última parte
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseExpandOption faz o parsing de uma única opção de expand
func (p *ODataParser) parseExpandOption(expandStr string) (ExpandOption, error) {
	option := ExpandOption{}

	// Verifica se tem parênteses (opções)
	openParenIndex := strings.Index(expandStr, "(")
	if openParenIndex == -1 {
		// Expand simples sem opções
		option.Property = strings.TrimSpace(expandStr)
		return option, nil
	}

	// Extrai o nome da propriedade
	option.Property = strings.TrimSpace(expandStr[:openParenIndex])

	// Encontra o parêntese de fechamento correspondente
	closeParenIndex := p.findMatchingCloseParen(expandStr, openParenIndex)
	if closeParenIndex == -1 {
		return option, fmt.Errorf("unmatched parentheses in expand option: %s", expandStr)
	}

	// Extrai as opções dentro dos parênteses
	optionsStr := expandStr[openParenIndex+1 : closeParenIndex]
	if optionsStr == "" {
		return option, nil
	}

	// Parse das opções dentro dos parênteses
	err := p.parseExpandOptions(optionsStr, &option)
	if err != nil {
		return option, fmt.Errorf("failed to parse expand options for '%s': %w", option.Property, err)
	}

	return option, nil
}

// findMatchingCloseParen encontra o parêntese de fechamento correspondente
func (p *ODataParser) findMatchingCloseParen(str string, openIndex int) int {
	parenCount := 1
	for i := openIndex + 1; i < len(str); i++ {
		switch str[i] {
		case '(':
			parenCount++
		case ')':
			parenCount--
			if parenCount == 0 {
				return i
			}
		}
	}
	return -1
}

// parseExpandOptions faz o parsing das opções dentro dos parênteses do expand
func (p *ODataParser) parseExpandOptions(optionsStr string, option *ExpandOption) error {
	// Divide as opções por & ou ;
	optionParts := p.splitExpandOptionParts(optionsStr)

	for _, part := range optionParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		err := p.parseExpandOptionPart(part, option)
		if err != nil {
			return fmt.Errorf("failed to parse expand option part '%s': %w", part, err)
		}
	}

	return nil
}

// splitExpandOptionParts divide as opções respeitando parênteses aninhados
func (p *ODataParser) splitExpandOptionParts(optionsStr string) []string {
	var parts []string
	var current strings.Builder
	var parenCount int

	for _, char := range optionsStr {
		switch char {
		case '(':
			parenCount++
			current.WriteRune(char)
		case ')':
			parenCount--
			current.WriteRune(char)
		case '&', ';':
			if parenCount == 0 {
				// Separador no nível superior
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Adiciona a última parte
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseExpandOptionPart faz o parsing de uma parte individual da opção
func (p *ODataParser) parseExpandOptionPart(part string, option *ExpandOption) error {
	// Remove $ do início se existir
	part = strings.TrimPrefix(part, "$")

	// Procura por = para dividir chave=valor
	equalIndex := strings.Index(part, "=")
	if equalIndex == -1 {
		return fmt.Errorf("invalid expand option format: %s", part)
	}

	key := strings.TrimSpace(part[:equalIndex])
	value := strings.TrimSpace(part[equalIndex+1:])

	// Remove aspas se existirem
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		value = value[1 : len(value)-1]
	}

	// Parse baseado na chave (case insensitive)
	switch strings.ToLower(key) {
	case "filter":
		option.Filter = value
	case "orderby":
		option.OrderBy = value
	case "select":
		option.Select = strings.Split(value, ",")
		for i := range option.Select {
			option.Select[i] = strings.TrimSpace(option.Select[i])
		}
	case "expand":
		expandOptions, err := p.ParseExpand(value)
		if err != nil {
			return fmt.Errorf("failed to parse nested expand: %w", err)
		}
		option.Expand = expandOptions
	case "skip":
		skip, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid skip value: %s", value)
		}
		option.Skip = skip
	case "top":
		top, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid top value: %s", value)
		}
		option.Top = top
	case "count":
		count, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid count value: %s", value)
		}
		option.Count = count
	default:
		return fmt.Errorf("unsupported expand option: %s", key)
	}

	return nil
}

// ValidateQueryOptions valida as opções de consulta
func (p *ODataParser) ValidateQueryOptions(options QueryOptions) error {
	if err := ValidateTopQuery(options.Top); err != nil {
		return err
	}

	if err := ValidateSkipQuery(options.Skip); err != nil {
		return err
	}

	return nil
}
