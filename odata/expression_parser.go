package odata

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Singleton global para expression parser otimizado
var (
	globalExpressionParser *ExpressionParser
	expressionParserOnce   sync.Once
)

// GetGlobalExpressionParser retorna instância singleton do expression parser
func GetGlobalExpressionParser() *ExpressionParser {
	expressionParserOnce.Do(func() {
		globalExpressionParser = &ExpressionParser{
			tokenizer: GlobalFilterTokenizer,
			operators: map[string]OperatorInfo{
				// Operadores lógicos (precedência mais baixa)
				"or":  {Precedence: 1, Associativity: AssocLeft},
				"and": {Precedence: 2, Associativity: AssocLeft},
				"not": {Precedence: 3, Associativity: AssocRight},

				// Operadores de comparação
				"eq":  {Precedence: 4, Associativity: AssocLeft},
				"ne":  {Precedence: 4, Associativity: AssocLeft},
				"gt":  {Precedence: 4, Associativity: AssocLeft},
				"ge":  {Precedence: 4, Associativity: AssocLeft},
				"lt":  {Precedence: 4, Associativity: AssocLeft},
				"le":  {Precedence: 4, Associativity: AssocLeft},
				"has": {Precedence: 4, Associativity: AssocLeft},
				"in":  {Precedence: 4, Associativity: AssocLeft},

				// Operadores aritméticos
				"add":   {Precedence: 5, Associativity: AssocLeft},
				"sub":   {Precedence: 5, Associativity: AssocLeft},
				"mul":   {Precedence: 6, Associativity: AssocLeft},
				"div":   {Precedence: 6, Associativity: AssocLeft},
				"divby": {Precedence: 6, Associativity: AssocLeft},
				"mod":   {Precedence: 6, Associativity: AssocLeft},
			},
		}
	})
	return globalExpressionParser
}

// ExpressionParser parser para expressões OData
type ExpressionParser struct {
	tokenizer *Tokenizer
	operators map[string]OperatorInfo
}

// NewExpressionParser cria um novo parser de expressões
func NewExpressionParser() *ExpressionParser {
	return GetGlobalExpressionParser()
}

// OperatorInfo informações sobre operadores
type OperatorInfo struct {
	Precedence    int
	Associativity Associativity
}

// Associativity associatividade dos operadores
type Associativity int

const (
	AssocLeft Associativity = iota
	AssocRight
)

// ParseNode representa um nó na árvore de parse
type ParseNode struct {
	Token    *Token
	Children []*ParseNode
	Parent   *ParseNode
}

// ParseFilterExpression analisa uma expressão de filtro
func (p *ExpressionParser) ParseFilterExpression(ctx context.Context, filter string) (*ParseNode, error) {
	if filter == "" {
		return nil, nil
	}

	// Tokenize
	tokens, err := p.tokenizer.Tokenize(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("tokenization error: %w", err)
	}

	if len(tokens) == 0 {
		return nil, nil
	}

	// Convert to postfix using optimized algorithm
	postfix, err := p.InfixToPostfix(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("infix to postfix conversion error: %w", err)
	}

	// Build tree from postfix
	tree, err := p.PostfixToTree(ctx, postfix)
	if err != nil {
		return nil, fmt.Errorf("tree building error: %w", err)
	}

	return tree, nil
}

// InfixToPostfix converte expressão infix para postfix usando algoritmo Shunting Yard otimizado
func (p *ExpressionParser) InfixToPostfix(ctx context.Context, tokens []*Token) ([]*Token, error) {
	var output []*Token
	var operatorStack []*Token

	for _, token := range tokens {
		// Verifica cancelamento do contexto
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		switch token.Type {
		case int(FilterTokenProperty), int(FilterTokenString), int(FilterTokenNumber), int(FilterTokenBoolean), int(FilterTokenNull),
			int(FilterTokenDateTime), int(FilterTokenDate), int(FilterTokenTime), int(FilterTokenGuid), int(FilterTokenDuration),
			int(FilterTokenGeographyPoint), int(FilterTokenGeometryPoint):
			// Operandos vão direto para output
			output = append(output, token)

		case int(FilterTokenFunction):
			// Funções vão para stack
			operatorStack = append(operatorStack, token)

		case int(FilterTokenComma):
			// Vírgula: pop até encontrar parêntese aberto
			// EXCEÇÃO: não fazer pop do operador "in" - ele deve permanecer na stack
			for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1].Type != int(FilterTokenOpenParen) {
				top := operatorStack[len(operatorStack)-1]
				// Se o topo é "in", não fazer pop - apenas pular
				if top.Type == int(FilterTokenComparison) && strings.ToLower(top.Value) == "in" {
					break
				}
				output = append(output, top)
				operatorStack = operatorStack[:len(operatorStack)-1]
			}

		case int(FilterTokenOpenParen):
			// Parêntese aberto vai para stack
			operatorStack = append(operatorStack, token)

		case int(FilterTokenCloseParen):
			// Parêntese fechado: pop até encontrar parêntese aberto
			for len(operatorStack) > 0 && operatorStack[len(operatorStack)-1].Type != int(FilterTokenOpenParen) {
				output = append(output, operatorStack[len(operatorStack)-1])
				operatorStack = operatorStack[:len(operatorStack)-1]
			}

			if len(operatorStack) == 0 {
				return nil, fmt.Errorf("mismatched parentheses")
			}

			// Remove parêntese aberto
			operatorStack = operatorStack[:len(operatorStack)-1]

			// Se há função no topo, move para output
			if len(operatorStack) > 0 && operatorStack[len(operatorStack)-1].Type == int(FilterTokenFunction) {
				output = append(output, operatorStack[len(operatorStack)-1])
				operatorStack = operatorStack[:len(operatorStack)-1]
			}

		case int(FilterTokenLogical), int(FilterTokenComparison), int(FilterTokenArithmetic):
			// Operadores: aplica regras de precedência
			op1 := strings.ToLower(token.Value)
			op1Info, exists := p.operators[op1]
			if !exists {
				return nil, fmt.Errorf("unknown operator: %s", op1)
			}

			for len(operatorStack) > 0 {
				top := operatorStack[len(operatorStack)-1]
				if top.Type == int(FilterTokenOpenParen) {
					break
				}

				if top.Type == int(FilterTokenFunction) {
					output = append(output, top)
					operatorStack = operatorStack[:len(operatorStack)-1]
					continue
				}

				op2 := strings.ToLower(top.Value)
				op2Info, exists := p.operators[op2]
				if !exists {
					break
				}

				if (op1Info.Associativity == AssocLeft && op1Info.Precedence <= op2Info.Precedence) ||
					(op1Info.Associativity == AssocRight && op1Info.Precedence < op2Info.Precedence) {
					output = append(output, top)
					operatorStack = operatorStack[:len(operatorStack)-1]
				} else {
					break
				}
			}

			operatorStack = append(operatorStack, token)

		default:
			return nil, fmt.Errorf("unexpected token type: %v", token.Type)
		}
	}

	// Pop remaining operators
	for len(operatorStack) > 0 {
		top := operatorStack[len(operatorStack)-1]
		if top.Type == int(FilterTokenOpenParen) || top.Type == int(FilterTokenCloseParen) {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		output = append(output, top)
		operatorStack = operatorStack[:len(operatorStack)-1]
	}

	return output, nil
}

// PostfixToTree constrói árvore de parse a partir de expressão postfix
func (p *ExpressionParser) PostfixToTree(ctx context.Context, postfix []*Token) (*ParseNode, error) {
	if len(postfix) == 0 {
		return nil, nil
	}

	var stack []*ParseNode

	for _, token := range postfix {
		// Verifica cancelamento do contexto
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		node := &ParseNode{
			Token:    token,
			Children: make([]*ParseNode, 0),
		}

		switch token.Type {
		case int(FilterTokenProperty), int(FilterTokenString), int(FilterTokenNumber), int(FilterTokenBoolean), int(FilterTokenNull),
			int(FilterTokenDateTime), int(FilterTokenDate), int(FilterTokenTime), int(FilterTokenGuid), int(FilterTokenDuration),
			int(FilterTokenGeographyPoint), int(FilterTokenGeometryPoint):
			// Operandos: nós folha
			stack = append(stack, node)

		case int(FilterTokenFunction):
			// Funções: determina número de argumentos
			argCount := p.getFunctionArgCount(token.Value)
			if len(stack) < argCount {
				return nil, fmt.Errorf("insufficient arguments for function %s", token.Value)
			}

			// Pop argumentos
			for i := 0; i < argCount; i++ {
				child := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				child.Parent = node
				// Insere no início para manter ordem
				node.Children = append([]*ParseNode{child}, node.Children...)
			}

			stack = append(stack, node)

		case int(FilterTokenComparison):
			if strings.ToLower(token.Value) == "in" {
				// Operador IN: precisa de 1 propriedade + múltiplos valores
				// EX: id_user in (1, 2, 3, 4, 5) -> postfix: id_user 1 2 3 4 5 in
				// Stack esperado ao chegar no IN: [1, 2, 3, 4, 5, id_user]
				// Último item da pilha = propriedade
				// Os anteriores = valores em ordem
				if len(stack) < 2 {
					return nil, fmt.Errorf("insufficient operands for operator %s", token.Value)
				}

				// A propriedade está no índice 0, os valores nos índices 1 até n
				property := stack[0]
				values := stack[1:] // Todos os valores

				// Configurar parent
				property.Parent = node
				for i := range values {
					values[i].Parent = node
				}

				// Resultado: [property, val1, val2, val3, val4, val5]
				node.Children = append([]*ParseNode{property}, values...)

				// Limpar stack - remover todos os elementos que foram usados
				stack = []*ParseNode{}

				// Adiciona o nó construído de volta ao stack
				stack = append(stack, node)
			} else {
				// Outros operadores de comparação são binários normais
				if len(stack) < 2 {
					return nil, fmt.Errorf("insufficient operands for operator %s", token.Value)
				}

				// Pop dois operandos
				right := stack[len(stack)-1]
				left := stack[len(stack)-2]
				stack = stack[:len(stack)-2]

				// Configura filhos
				left.Parent = node
				right.Parent = node
				node.Children = []*ParseNode{left, right}

				stack = append(stack, node)
			}

		case int(FilterTokenLogical), int(FilterTokenArithmetic):
			// Operadores binários
			if len(stack) < 2 {
				return nil, fmt.Errorf("insufficient operands for operator %s", token.Value)
			}

			// Pop dois operandos
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// Configura filhos
			left.Parent = node
			right.Parent = node
			node.Children = []*ParseNode{left, right}

			stack = append(stack, node)

		default:
			return nil, fmt.Errorf("unexpected token in postfix: %v", token.Type)
		}
	}

	if len(stack) != 1 {
		return nil, fmt.Errorf("invalid expression: expected single result, got %d", len(stack))
	}

	return stack[0], nil
}

// getFunctionArgCount retorna o número de argumentos esperados para uma função
func (p *ExpressionParser) getFunctionArgCount(funcName string) int {
	switch strings.ToLower(funcName) {
	case "contains", "startswith", "endswith", "indexof":
		return 2
	case "substring":
		return 3 // pode ser 2 ou 3, mas assumimos 3 por padrão
	case "concat":
		return 2 // pode ser variável, mas assumimos 2 por padrão
	case "length", "tolower", "toupper", "trim", "year", "month", "day", "hour", "minute", "second", "round", "floor", "ceiling":
		return 1
	case "now":
		return 0
	default:
		return 1 // padrão para funções desconhecidas
	}
}

// ConvertToFilterExpressions converte árvore de parse para FilterExpression (compatibilidade)
func (p *ExpressionParser) ConvertToFilterExpressions(node *ParseNode) ([]FilterExpression, error) {
	if node == nil {
		return []FilterExpression{}, nil
	}

	expressions := make([]FilterExpression, 0)

	// Percorre a árvore e converte para formato antigo
	err := p.traverseNode(node, &expressions)
	if err != nil {
		return nil, err
	}

	return expressions, nil
}

// traverseNode percorre recursivamente a árvore
func (p *ExpressionParser) traverseNode(node *ParseNode, expressions *[]FilterExpression) error {
	if node == nil {
		return nil
	}

	// Processa nó atual
	if node.Token.Type == int(FilterTokenLogical) || node.Token.Type == int(FilterTokenComparison) || node.Token.Type == int(FilterTokenArithmetic) {
		if len(node.Children) >= 2 {
			expr := FilterExpression{
				Operator: FilterOperator(node.Token.Value),
			}

			// Extrai propriedade do lado esquerdo
			if node.Children[0].Token.Type == int(FilterTokenProperty) {
				expr.Property = node.Children[0].Token.Value
			}

			// Extrai valor do lado direito
			if node.Children[1].Token.Type == int(FilterTokenString) {
				expr.Value = strings.Trim(node.Children[1].Token.Value, "'")
			} else {
				expr.Value = node.Children[1].Token.Value
			}

			*expressions = append(*expressions, expr)
		}
	}

	// Processa filhos
	for _, child := range node.Children {
		err := p.traverseNode(child, expressions)
		if err != nil {
			return err
		}
	}

	return nil
}

// isBooleanExpression verifica se um token representa uma expressão booleana
func (p *ExpressionParser) isBooleanExpression(token *Token) bool {
	if token == nil {
		return false
	}

	switch token.Type {
	case int(FilterTokenLogical), int(FilterTokenComparison):
		return true
	case int(FilterTokenBoolean):
		return true
	case int(FilterTokenFunction):
		// Algumas funções retornam boolean
		funcName := strings.ToLower(token.Value)
		return funcName == "contains" || funcName == "startswith" || funcName == "endswith"
	default:
		return false
	}
}
