package odata

import (
	"context"
	"fmt"
	"strings"
)

// SearchTokenType representa tipos de tokens para $search
type SearchTokenType int

func (s SearchTokenType) Value() int {
	return int(s)
}

const (
	SearchTokenTerm SearchTokenType = iota
	SearchTokenPhrase
	SearchTokenAND
	SearchTokenOR
	SearchTokenNOT
	SearchTokenOpenParen
	SearchTokenCloseParen
	SearchTokenWhitespace
)

// SearchExpression representa uma expressão de busca
type SearchExpression struct {
	Type     SearchExpressionType
	Value    string
	Children []*SearchExpression
}

// SearchExpressionType tipos de expressões de busca
type SearchExpressionType int

const (
	SearchExpressionTerm SearchExpressionType = iota
	SearchExpressionPhrase
	SearchExpressionAND
	SearchExpressionOR
	SearchExpressionNOT
	SearchExpressionGroup
)

// SearchOption representa uma opção de busca
type SearchOption struct {
	Expression *SearchExpression
	RawQuery   string
}

// CreateSearchTokenizer cria um tokenizer específico para $search
func CreateSearchTokenizer() *Tokenizer {
	t := &Tokenizer{}

	// Ignore whitespace
	t.Add(`^\s+`, -1) // -1 indica que deve ser ignorado

	// Operadores booleanos (case insensitive)
	t.Add(`^(?i)\bAND\b`, int(SearchTokenAND))
	t.Add(`^(?i)\bOR\b`, int(SearchTokenOR))
	t.Add(`^(?i)\bNOT\b`, int(SearchTokenNOT))

	// Parênteses para agrupamento
	t.Add(`^\(`, int(SearchTokenOpenParen))
	t.Add(`^\)`, int(SearchTokenCloseParen))

	// Frases entre aspas (double quotes)
	t.Add(`^"([^"\\]|\\.)*"`, int(SearchTokenPhrase))

	// Termos simples (palavras, wildcards)
	t.Add(`^[^\s"()]+`, int(SearchTokenTerm))

	return t
}

// SearchParser parser para expressões $search
type SearchParser struct {
	tokenizer *Tokenizer
}

// NewSearchParser cria um novo parser de search
func NewSearchParser() *SearchParser {
	return &SearchParser{
		tokenizer: CreateSearchTokenizer(),
	}
}

// ParseSearch analisa uma string $search
func (p *SearchParser) ParseSearch(ctx context.Context, searchStr string) (*SearchOption, error) {
	if searchStr == "" {
		return &SearchOption{
			Expression: nil,
			RawQuery:   "",
		}, nil
	}

	// Tokenize a string
	tokens, err := p.tokenizer.Tokenize(ctx, searchStr)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize search expression: %w", err)
	}

	if len(tokens) == 0 {
		return &SearchOption{
			Expression: nil,
			RawQuery:   searchStr,
		}, nil
	}

	// Parse a expressão
	expression, err := p.parseSearchExpression(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse search expression: %w", err)
	}

	return &SearchOption{
		Expression: expression,
		RawQuery:   searchStr,
	}, nil
}

// parseSearchExpression analisa tokens em uma expressão de busca
func (p *SearchParser) parseSearchExpression(ctx context.Context, tokens []*Token) (*SearchExpression, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty search expression")
	}

	// Converte tokens para expressões usando precedência de operadores
	postfix, err := p.convertToPostfix(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to convert search to postfix: %w", err)
	}

	// Constrói árvore de expressão a partir da notação postfix
	expression, err := p.buildSearchTree(ctx, postfix)
	if err != nil {
		return nil, fmt.Errorf("failed to build search tree: %w", err)
	}

	return expression, nil
}

// convertToPostfix converte tokens de busca para notação postfix
func (p *SearchParser) convertToPostfix(ctx context.Context, tokens []*Token) ([]*Token, error) {
	var output []*Token
	var operators []*Token

	for i, token := range tokens {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		switch token.Type {
		case int(SearchTokenTerm), int(SearchTokenPhrase):
			output = append(output, token)

		case int(SearchTokenNOT):
			operators = append(operators, token)

		case int(SearchTokenAND), int(SearchTokenOR):
			// Pop operators com precedência maior ou igual
			for len(operators) > 0 {
				top := operators[len(operators)-1]
				if top.Type == int(SearchTokenOpenParen) {
					break
				}

				// NOT tem precedência maior que AND e OR
				// AND tem precedência maior que OR
				if (token.Type == int(SearchTokenOR) && (top.Type == int(SearchTokenAND) || top.Type == int(SearchTokenNOT))) ||
					(token.Type == int(SearchTokenAND) && top.Type == int(SearchTokenNOT)) {
					output = append(output, top)
					operators = operators[:len(operators)-1]
				} else {
					break
				}
			}
			operators = append(operators, token)

		case int(SearchTokenOpenParen):
			operators = append(operators, token)

		case int(SearchTokenCloseParen):
			// Pop até encontrar o parêntese de abertura
			found := false
			for len(operators) > 0 {
				top := operators[len(operators)-1]
				operators = operators[:len(operators)-1]

				if top.Type == int(SearchTokenOpenParen) {
					found = true
					break
				}

				output = append(output, top)
			}

			if !found {
				return nil, fmt.Errorf("mismatched parentheses in search expression")
			}
		}

		// Adiciona AND implícito entre termos consecutivos
		if i < len(tokens)-1 {
			next := tokens[i+1]
			if (token.Type == int(SearchTokenTerm) || token.Type == int(SearchTokenPhrase) || token.Type == int(SearchTokenCloseParen)) &&
				(next.Type == int(SearchTokenTerm) || next.Type == int(SearchTokenPhrase) || next.Type == int(SearchTokenOpenParen) || next.Type == int(SearchTokenNOT)) {
				// Adiciona AND implícito
				implicitAnd := &Token{
					Type:  int(SearchTokenAND),
					Value: "AND",
				}

				// Aplica a mesma lógica de precedência
				for len(operators) > 0 {
					top := operators[len(operators)-1]
					if top.Type == int(SearchTokenOpenParen) {
						break
					}

					if top.Type == int(SearchTokenNOT) {
						output = append(output, top)
						operators = operators[:len(operators)-1]
					} else {
						break
					}
				}
				operators = append(operators, implicitAnd)
			}
		}
	}

	// Pop operadores restantes
	for len(operators) > 0 {
		top := operators[len(operators)-1]
		operators = operators[:len(operators)-1]

		if top.Type == int(SearchTokenOpenParen) || top.Type == int(SearchTokenCloseParen) {
			return nil, fmt.Errorf("mismatched parentheses in search expression")
		}

		output = append(output, top)
	}

	return output, nil
}

// buildSearchTree constrói árvore de expressão a partir da notação postfix
func (p *SearchParser) buildSearchTree(ctx context.Context, postfix []*Token) (*SearchExpression, error) {
	var stack []*SearchExpression

	for _, token := range postfix {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		switch token.Type {
		case int(SearchTokenTerm):
			expr := &SearchExpression{
				Type:  SearchExpressionTerm,
				Value: token.Value,
			}
			stack = append(stack, expr)

		case int(SearchTokenPhrase):
			// Remove aspas da frase
			phrase := token.Value
			if len(phrase) >= 2 && phrase[0] == '"' && phrase[len(phrase)-1] == '"' {
				phrase = phrase[1 : len(phrase)-1]
			}

			expr := &SearchExpression{
				Type:  SearchExpressionPhrase,
				Value: phrase,
			}
			stack = append(stack, expr)

		case int(SearchTokenNOT):
			if len(stack) < 1 {
				return nil, fmt.Errorf("invalid search expression: NOT requires one operand")
			}

			operand := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			expr := &SearchExpression{
				Type:     SearchExpressionNOT,
				Value:    "NOT",
				Children: []*SearchExpression{operand},
			}
			stack = append(stack, expr)

		case int(SearchTokenAND):
			if len(stack) < 2 {
				return nil, fmt.Errorf("invalid search expression: AND requires two operands")
			}

			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			expr := &SearchExpression{
				Type:     SearchExpressionAND,
				Value:    "AND",
				Children: []*SearchExpression{left, right},
			}
			stack = append(stack, expr)

		case int(SearchTokenOR):
			if len(stack) < 2 {
				return nil, fmt.Errorf("invalid search expression: OR requires two operands")
			}

			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			expr := &SearchExpression{
				Type:     SearchExpressionOR,
				Value:    "OR",
				Children: []*SearchExpression{left, right},
			}
			stack = append(stack, expr)
		}
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("invalid search expression: expected single result, got %d", len(stack))
	}

	return stack[0], nil
}

// ValidateSearchExpression valida uma expressão de busca
func (p *SearchParser) ValidateSearchExpression(expr *SearchExpression) error {
	if expr == nil {
		return nil
	}

	switch expr.Type {
	case SearchExpressionTerm:
		if expr.Value == "" {
			return fmt.Errorf("search term cannot be empty")
		}

		// Valida wildcards
		if strings.Contains(expr.Value, "*") {
			if !p.isValidWildcard(expr.Value) {
				return fmt.Errorf("invalid wildcard pattern: %s", expr.Value)
			}
		}

	case SearchExpressionPhrase:
		if expr.Value == "" {
			return fmt.Errorf("search phrase cannot be empty")
		}

	case SearchExpressionNOT:
		if len(expr.Children) != 1 {
			return fmt.Errorf("NOT expression must have exactly one child")
		}

		return p.ValidateSearchExpression(expr.Children[0])

	case SearchExpressionAND, SearchExpressionOR:
		if len(expr.Children) != 2 {
			return fmt.Errorf("%s expression must have exactly two children", expr.Value)
		}

		for _, child := range expr.Children {
			if err := p.ValidateSearchExpression(child); err != nil {
				return err
			}
		}
	}

	return nil
}

// isValidWildcard verifica se um padrão wildcard é válido
func (p *SearchParser) isValidWildcard(pattern string) bool {
	// Wildcards simples são permitidos
	// Padrões como "test*", "*test", "te*st" são válidos
	// Padrões como "**" ou apenas "*" podem ser inválidos dependendo da implementação

	if pattern == "*" {
		return false // Wildcard sozinho não é permitido
	}

	// Verifica se não há wildcards consecutivos
	if strings.Contains(pattern, "**") {
		return false
	}

	return true
}

// GetSearchableProperties retorna propriedades que podem ser pesquisadas
func (p *SearchParser) GetSearchableProperties(metadata EntityMetadata) []PropertyMetadata {
	var searchableProps []PropertyMetadata

	for _, prop := range metadata.Properties {
		// Apenas propriedades de texto são pesquisáveis por padrão
		if p.isSearchableType(prop.Type) && !prop.IsNavigation {
			searchableProps = append(searchableProps, prop)
		}
	}

	return searchableProps
}

// isSearchableType verifica se um tipo é pesquisável
func (p *SearchParser) isSearchableType(propType string) bool {
	searchableTypes := []string{
		"string", "text", "varchar", "nvarchar", "char", "nchar",
		"clob", "nclob", "longtext", "mediumtext", "tinytext",
	}

	propTypeLower := strings.ToLower(propType)

	for _, searchableType := range searchableTypes {
		if strings.Contains(propTypeLower, searchableType) {
			return true
		}
	}

	return false
}

// ExtractSearchTerms extrai todos os termos de busca de uma expressão
func (p *SearchParser) ExtractSearchTerms(expr *SearchExpression) []string {
	if expr == nil {
		return []string{}
	}

	var terms []string

	switch expr.Type {
	case SearchExpressionTerm:
		terms = append(terms, expr.Value)

	case SearchExpressionPhrase:
		// Para frases, podemos extrair palavras individuais ou manter como frase
		terms = append(terms, expr.Value)

	case SearchExpressionNOT, SearchExpressionAND, SearchExpressionOR:
		for _, child := range expr.Children {
			childTerms := p.ExtractSearchTerms(child)
			terms = append(terms, childTerms...)
		}
	}

	return terms
}

// GetSearchComplexity calcula a complexidade de uma expressão de busca
func (p *SearchParser) GetSearchComplexity(expr *SearchExpression) int {
	if expr == nil {
		return 0
	}

	switch expr.Type {
	case SearchExpressionTerm, SearchExpressionPhrase:
		return 1

	case SearchExpressionNOT:
		return 1 + p.GetSearchComplexity(expr.Children[0])

	case SearchExpressionAND, SearchExpressionOR:
		complexity := 1
		for _, child := range expr.Children {
			complexity += p.GetSearchComplexity(child)
		}
		return complexity

	default:
		return 0
	}
}

// OptimizeSearchExpression otimiza uma expressão de busca
func (p *SearchParser) OptimizeSearchExpression(expr *SearchExpression) *SearchExpression {
	if expr == nil {
		return nil
	}

	switch expr.Type {
	case SearchExpressionTerm, SearchExpressionPhrase:
		return expr

	case SearchExpressionNOT:
		optimizedChild := p.OptimizeSearchExpression(expr.Children[0])
		return &SearchExpression{
			Type:     SearchExpressionNOT,
			Value:    "NOT",
			Children: []*SearchExpression{optimizedChild},
		}

	case SearchExpressionAND, SearchExpressionOR:
		optimizedChildren := make([]*SearchExpression, len(expr.Children))
		for i, child := range expr.Children {
			optimizedChildren[i] = p.OptimizeSearchExpression(child)
		}

		// Otimização: flatten operadores do mesmo tipo
		// (A AND B) AND C -> A AND B AND C
		if len(optimizedChildren) == 2 {
			left, right := optimizedChildren[0], optimizedChildren[1]

			// Se ambos os filhos são do mesmo tipo, podemos achatar
			if left.Type == expr.Type && right.Type == expr.Type {
				var flattenedChildren []*SearchExpression
				flattenedChildren = append(flattenedChildren, left.Children...)
				flattenedChildren = append(flattenedChildren, right.Children...)

				return &SearchExpression{
					Type:     expr.Type,
					Value:    expr.Value,
					Children: flattenedChildren,
				}
			}
		}

		return &SearchExpression{
			Type:     expr.Type,
			Value:    expr.Value,
			Children: optimizedChildren,
		}

	default:
		return expr
	}
}
