package odata

import (
	"context"
	"fmt"
	"strings"
)

// ComputeTokenType representa tipos de tokens para $compute
type ComputeTokenType int

func (c ComputeTokenType) Value() int {
	return int(c)
}

const (
	ComputeTokenProperty ComputeTokenType = iota
	ComputeTokenFunction
	ComputeTokenOperator
	ComputeTokenString
	ComputeTokenNumber
	ComputeTokenOpenParen
	ComputeTokenCloseParen
	ComputeTokenComma
	ComputeTokenAs
	ComputeTokenAlias
	ComputeTokenWhitespace
)

// ComputeExpression representa uma expressão de compute
type ComputeExpression struct {
	Expression string     // A expressão a ser computada (ex: "Price mul Quantity")
	Alias      string     // O alias para o resultado (ex: "Total")
	ParseTree  *ParseNode // Árvore de parse da expressão
}

// ComputeOption representa todas as expressões de compute
type ComputeOption struct {
	Expressions []ComputeExpression
}

// CreateComputeTokenizer cria um tokenizer específico para $compute
func CreateComputeTokenizer() *Tokenizer {
	t := &Tokenizer{}

	// Palavra-chave 'as' (case insensitive)
	t.Add(`^(?i)as\b`, int(ComputeTokenAs))

	// Parênteses
	t.Add(`^\(`, int(ComputeTokenOpenParen))
	t.Add(`^\)`, int(ComputeTokenCloseParen))

	// Vírgula (separador de expressões)
	t.Add(`^,`, int(ComputeTokenComma))

	// Operadores aritméticos
	t.Add(`^(?i)(add|sub|mul|div|mod)\b`, int(ComputeTokenOperator))

	// Funções matemáticas e de string
	t.Add(`^(?i)(round|floor|ceiling|abs|sqrt|length|tolower|toupper|trim|concat|substring|indexof|year|month|day|hour|minute|second|now|date|time)\b`, int(ComputeTokenFunction))

	// Strings (single quotes)
	t.Add(`^'([^'\\]|\\.)*'`, int(ComputeTokenString))

	// Números (int, float, decimal)
	t.Add(`^-?\d+(\.\d+)?([eE][+-]?\d+)?[dDfFmM]?`, int(ComputeTokenNumber))

	// Propriedades/Identificadores (deve vir por último)
	t.Add(`^[a-zA-Z_][a-zA-Z0-9_]*`, int(ComputeTokenProperty))

	return t
}

// ComputeParser parser para expressões $compute
type ComputeParser struct {
	tokenizer        *Tokenizer
	expressionParser *ExpressionParser
}

// NewComputeParser cria um novo parser de compute
func NewComputeParser() *ComputeParser {
	return &ComputeParser{
		tokenizer:        CreateComputeTokenizer(),
		expressionParser: NewExpressionParser(),
	}
}

// ParseCompute analisa uma string $compute
func (p *ComputeParser) ParseCompute(ctx context.Context, computeStr string) (*ComputeOption, error) {
	if computeStr == "" {
		return &ComputeOption{Expressions: []ComputeExpression{}}, nil
	}

	// Tokenize a string completa
	tokens, err := p.tokenizer.Tokenize(ctx, computeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize compute expression: %w", err)
	}

	// Parse as expressões separadas por vírgula
	expressions, err := p.parseComputeExpressions(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compute expressions: %w", err)
	}

	return &ComputeOption{Expressions: expressions}, nil
}

// parseComputeExpressions analisa múltiplas expressões separadas por vírgula
func (p *ComputeParser) parseComputeExpressions(ctx context.Context, tokens []*Token) ([]ComputeExpression, error) {
	var expressions []ComputeExpression
	var currentExpr []string
	var currentTokens []*Token

	i := 0
	for i < len(tokens) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		token := tokens[i]

		if token.Type == int(ComputeTokenComma) {
			// Finaliza a expressão atual
			if len(currentTokens) > 0 {
				expr, err := p.parseSingleComputeExpression(ctx, currentTokens, strings.Join(currentExpr, " "))
				if err != nil {
					return nil, err
				}
				expressions = append(expressions, expr)

				// Reset para próxima expressão
				currentExpr = []string{}
				currentTokens = []*Token{}
			}
		} else {
			currentExpr = append(currentExpr, token.Value)
			currentTokens = append(currentTokens, token)
		}

		i++
	}

	// Processa a última expressão
	if len(currentTokens) > 0 {
		expr, err := p.parseSingleComputeExpression(ctx, currentTokens, strings.Join(currentExpr, " "))
		if err != nil {
			return nil, err
		}
		expressions = append(expressions, expr)
	}

	return expressions, nil
}

// parseSingleComputeExpression analisa uma única expressão compute
func (p *ComputeParser) parseSingleComputeExpression(ctx context.Context, tokens []*Token, rawExpression string) (ComputeExpression, error) {
	// Encontra a palavra-chave 'as' para separar expressão do alias
	asIndex := -1
	for i, token := range tokens {
		if token.Type == int(ComputeTokenAs) {
			asIndex = i
			break
		}
	}

	if asIndex == -1 {
		return ComputeExpression{}, fmt.Errorf("compute expression must have an alias using 'as' keyword: %s", rawExpression)
	}

	if asIndex == 0 {
		return ComputeExpression{}, fmt.Errorf("compute expression cannot start with 'as' keyword: %s", rawExpression)
	}

	if asIndex == len(tokens)-1 {
		return ComputeExpression{}, fmt.Errorf("compute expression must have an alias after 'as' keyword: %s", rawExpression)
	}

	// Separa tokens da expressão e do alias
	exprTokens := tokens[:asIndex]
	aliasTokens := tokens[asIndex+1:]

	// Valida que o alias é um único identificador
	if len(aliasTokens) != 1 || aliasTokens[0].Type != int(ComputeTokenProperty) {
		return ComputeExpression{}, fmt.Errorf("compute alias must be a single identifier: %s", rawExpression)
	}

	alias := aliasTokens[0].Value

	// Constrói a expressão como string
	var exprParts []string
	for _, token := range exprTokens {
		exprParts = append(exprParts, token.Value)
	}
	expression := strings.Join(exprParts, " ")

	// Converte tokens de compute para tokens de expressão para reuso do parser
	filterTokens := p.convertComputeTokensToFilterTokens(exprTokens)

	// Usa o expression parser para construir a árvore
	postfix, err := p.expressionParser.InfixToPostfix(ctx, filterTokens)
	if err != nil {
		return ComputeExpression{}, fmt.Errorf("failed to convert compute expression to postfix: %w", err)
	}

	parseTree, err := p.expressionParser.PostfixToTree(ctx, postfix)
	if err != nil {
		return ComputeExpression{}, fmt.Errorf("failed to build parse tree for compute expression: %w", err)
	}

	return ComputeExpression{
		Expression: expression,
		Alias:      alias,
		ParseTree:  parseTree,
	}, nil
}

// convertComputeTokensToFilterTokens converte tokens de compute para tokens de filtro
func (p *ComputeParser) convertComputeTokensToFilterTokens(computeTokens []*Token) []*Token {
	var filterTokens []*Token

	for _, token := range computeTokens {
		var filterType int

		switch token.Type {
		case int(ComputeTokenProperty):
			filterType = 1 // FilterTokenProperty
		case int(ComputeTokenFunction):
			filterType = 2 // FilterTokenFunction
		case int(ComputeTokenOperator):
			filterType = 3 // FilterTokenArithmetic
		case int(ComputeTokenString):
			filterType = 4 // FilterTokenString
		case int(ComputeTokenNumber):
			filterType = 5 // FilterTokenNumber
		case int(ComputeTokenOpenParen):
			filterType = 6 // FilterTokenOpenParen
		case int(ComputeTokenCloseParen):
			filterType = 7 // FilterTokenCloseParen
		case int(ComputeTokenComma):
			filterType = 8 // FilterTokenComma
		default:
			continue // Skip unknown tokens
		}

		filterToken := &Token{
			Type:  filterType,
			Value: token.Value,
		}

		filterTokens = append(filterTokens, filterToken)
	}

	return filterTokens
}

// ValidateComputeExpression valida uma expressão de compute contra metadados
func (p *ComputeParser) ValidateComputeExpression(expr ComputeExpression, metadata EntityMetadata) error {
	if expr.ParseTree == nil {
		return fmt.Errorf("compute expression has no parse tree")
	}

	// Valida que todas as propriedades existem
	err := p.validateComputeNode(expr.ParseTree, metadata)
	if err != nil {
		return fmt.Errorf("invalid compute expression '%s': %w", expr.Expression, err)
	}

	// Valida que o alias não conflita com propriedades existentes
	for _, prop := range metadata.Properties {
		if prop.Name == expr.Alias {
			return fmt.Errorf("compute alias '%s' conflicts with existing property", expr.Alias)
		}
	}

	return nil
}

// validateComputeNode valida recursivamente um nó da árvore de parse
func (p *ComputeParser) validateComputeNode(node *ParseNode, metadata EntityMetadata) error {
	if node == nil {
		return nil
	}

	switch node.Token.Type {
	case 1: // FilterTokenProperty
		// Verifica se a propriedade existe
		propertyName := node.Token.Value
		found := false
		for _, prop := range metadata.Properties {
			if strings.EqualFold(prop.Name, propertyName) && !prop.IsNavigation {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("property '%s' not found or is a navigation property", propertyName)
		}

	case 2: // FilterTokenFunction
		// Valida que a função é suportada em compute
		functionName := node.Token.Value
		if !p.isSupportedComputeFunction(functionName) {
			return fmt.Errorf("function '%s' is not supported in compute expressions", functionName)
		}
	}

	// Valida filhos recursivamente
	for _, child := range node.Children {
		if err := p.validateComputeNode(child, metadata); err != nil {
			return err
		}
	}

	return nil
}

// isSupportedComputeFunction verifica se uma função é suportada em expressões compute
func (p *ComputeParser) isSupportedComputeFunction(functionName string) bool {
	supportedFunctions := []string{
		// Funções matemáticas
		"round", "floor", "ceiling", "abs", "sqrt",
		// Funções de string
		"length", "tolower", "toupper", "trim", "concat", "substring", "indexof",
		// Funções de data/hora
		"year", "month", "day", "hour", "minute", "second", "now", "date", "time",
		// Operadores aritméticos
		"add", "sub", "mul", "div", "mod",
	}

	for _, supported := range supportedFunctions {
		if functionName == supported {
			return true
		}
	}

	return false
}

// GetComputeFields retorna os campos computados como metadados de propriedade
func (p *ComputeParser) GetComputeFields(computeOption *ComputeOption) []PropertyMetadata {
	if computeOption == nil {
		return []PropertyMetadata{}
	}

	var fields []PropertyMetadata

	for _, expr := range computeOption.Expressions {
		// Determina o tipo baseado na expressão
		fieldType := p.inferComputeFieldType(expr.ParseTree)

		field := PropertyMetadata{
			Name:         expr.Alias,
			Type:         fieldType,
			ColumnName:   "", // Será gerado dinamicamente
			IsKey:        false,
			IsNullable:   true,
			IsNavigation: false,
			HasDefault:   false,
		}

		fields = append(fields, field)
	}

	return fields
}

// inferComputeFieldType infere o tipo de dados de uma expressão compute
func (p *ComputeParser) inferComputeFieldType(node *ParseNode) string {
	if node == nil {
		return "string"
	}

	switch node.Token.Type {
	case 5: // FilterTokenNumber
		return "number"
	case 4: // FilterTokenString
		return "string"
	case 2: // FilterTokenFunction
		// Mapeia funções para tipos
		switch node.Token.Value {
		case "round", "floor", "ceiling", "abs", "sqrt", "add", "sub", "mul", "div", "mod":
			return "number"
		case "length", "indexof", "year", "month", "day", "hour", "minute", "second":
			return "number"
		case "tolower", "toupper", "trim", "concat", "substring":
			return "string"
		case "now", "date", "time":
			return "datetime"
		default:
			return "string"
		}
	case 3: // FilterTokenArithmetic
		// Operações aritméticas retornam números
		return "number"
	default:
		// Para propriedades, seria necessário consultar os metadados
		// Por simplicidade, assumimos string
		return "string"
	}
}
