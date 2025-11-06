package odata

import (
	"context"
	"fmt"
	"strings"
)

// Singleton global para filter parser
var GlobalFilterParser *ExpressionParser

// init inicializa os parsers globais para filtros
func init() {
	GlobalFilterTokenizer = GetGlobalFilterTokenizer()
	GlobalFilterParser = GetGlobalExpressionParser()
}

// GoDataFilterQuery representa uma query de filtro processada
type GoDataFilterQuery struct {
	Tree     *ParseNode
	RawValue string
}

// ParseFilterString converte uma string do parâmetro $filter da URL em uma árvore de parse
// que pode ser usada por providers para criar uma resposta.
func ParseFilterString(ctx context.Context, filter string) (*GoDataFilterQuery, error) {
	if filter == "" {
		return nil, nil
	}

	// Tokenize usando o tokenizer global
	tokens, err := GlobalFilterTokenizer.Tokenize(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize filter: %w", err)
	}

	if len(tokens) == 0 {
		return nil, nil
	}

	// Convert to postfix usando o parser global
	postfix, err := GlobalFilterParser.InfixToPostfix(ctx, tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to postfix: %w", err)
	}

	// Build tree
	tree, err := GlobalFilterParser.PostfixToTree(ctx, postfix)
	if err != nil {
		return nil, fmt.Errorf("failed to build parse tree: %w", err)
	}

	// Valida se é uma expressão booleana
	if tree == nil || tree.Token == nil || !GlobalFilterParser.isBooleanExpression(tree.Token) {
		return nil, fmt.Errorf("filter expression must be a boolean expression")
	}

	return &GoDataFilterQuery{
		Tree:     tree,
		RawValue: filter,
	}, nil
}

// SemanticizeFilterQuery adiciona informações semânticas à query de filtro
// baseado no contexto do serviço e entidade
func SemanticizeFilterQuery(
	filter *GoDataFilterQuery,
	metadata EntityMetadata,
) error {
	if filter == nil || filter.Tree == nil {
		return nil
	}

	var semanticizeFilterNode func(node *ParseNode) error
	semanticizeFilterNode = func(node *ParseNode) error {
		if node == nil || node.Token == nil {
			return nil
		}

		// Se é uma propriedade, valida se existe na entidade (case-insensitive)
		if node.Token.Type == int(FilterTokenProperty) {
			propertyName := node.Token.Value
			found := false

			for _, prop := range metadata.Properties {
				// Comparação case-insensitive
				if strings.EqualFold(prop.Name, propertyName) {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("property '%s' not found in entity '%s'", propertyName, metadata.Name)
			}
		}

		// Processa filhos recursivamente
		for _, child := range node.Children {
			if err := semanticizeFilterNode(child); err != nil {
				return err
			}
		}

		return nil
	}

	return semanticizeFilterNode(filter.Tree)
}

// ValidateFilterExpression valida uma expressão de filtro
func ValidateFilterExpression(ctx context.Context, filter string, metadata EntityMetadata) error {
	if filter == "" {
		return nil
	}

	// Parse a expressão
	filterQuery, err := ParseFilterString(ctx, filter)
	if err != nil {
		return fmt.Errorf("invalid filter expression: %w", err)
	}

	// Valida semanticamente
	if err := SemanticizeFilterQuery(filterQuery, metadata); err != nil {
		return fmt.Errorf("semantic validation failed: %w", err)
	}

	return nil
}

// ConvertFilterToSQL converte uma GoDataFilterQuery para SQL
func ConvertFilterToSQL(ctx context.Context, filter *GoDataFilterQuery, metadata EntityMetadata) (string, []interface{}, error) {
	if filter == nil || filter.Tree == nil {
		return "", nil, nil
	}

	// Usa o query builder existente para converter
	qb := NewQueryBuilder("mysql") // Usa MySQL como padrão, pode ser parametrizado
	return qb.BuildWhereClause(ctx, filter.Tree, metadata)
}

// GetFilterProperties retorna todas as propriedades usadas no filtro
func GetFilterProperties(filter *GoDataFilterQuery) []string {
	if filter == nil || filter.Tree == nil {
		return []string{}
	}

	properties := make(map[string]bool)
	var collectProperties func(node *ParseNode)

	collectProperties = func(node *ParseNode) {
		if node == nil || node.Token == nil {
			return
		}

		if node.Token.Type == int(FilterTokenProperty) {
			properties[node.Token.Value] = true
		}

		for _, child := range node.Children {
			collectProperties(child)
		}
	}

	collectProperties(filter.Tree)

	result := make([]string, 0, len(properties))
	for prop := range properties {
		result = append(result, prop)
	}

	return result
}

// OptimizeFilterExpression otimiza uma expressão de filtro removendo redundâncias
func OptimizeFilterExpression(filter *GoDataFilterQuery) *GoDataFilterQuery {
	if filter == nil || filter.Tree == nil {
		return filter
	}

	// Por enquanto, retorna a expressão original
	// Futuras otimizações podem incluir:
	// - Remoção de expressões sempre verdadeiras/falsas
	// - Combinação de condições AND/OR
	// - Reordenação para melhor performance

	return filter
}

// IsSimpleFilter verifica se o filtro é uma expressão simples (propriedade operador valor)
func IsSimpleFilter(filter *GoDataFilterQuery) bool {
	if filter == nil || filter.Tree == nil {
		return false
	}

	// Verifica se é um operador de comparação com dois filhos
	if filter.Tree.Token.Type == int(FilterTokenComparison) && len(filter.Tree.Children) == 2 {
		left := filter.Tree.Children[0]
		right := filter.Tree.Children[1]

		// Lado esquerdo deve ser propriedade, lado direito deve ser valor
		return left.Token.Type == int(FilterTokenProperty) &&
			(right.Token.Type == int(FilterTokenString) ||
				right.Token.Type == int(FilterTokenNumber) ||
				right.Token.Type == int(FilterTokenBoolean) ||
				right.Token.Type == int(FilterTokenDateTime) ||
				right.Token.Type == int(FilterTokenDate) ||
				right.Token.Type == int(FilterTokenTime) ||
				right.Token.Type == int(FilterTokenGuid))
	}

	return false
}

// GetFilterComplexity retorna um score de complexidade do filtro
func GetFilterComplexity(filter *GoDataFilterQuery) int {
	if filter == nil || filter.Tree == nil {
		return 0
	}

	var calculateComplexity func(node *ParseNode) int
	calculateComplexity = func(node *ParseNode) int {
		if node == nil {
			return 0
		}

		complexity := 1 // Base complexity for each node

		// Adiciona complexidade baseada no tipo de token
		switch node.Token.Type {
		case int(FilterTokenFunction):
			complexity += 3 // Funções são mais complexas
		case int(FilterTokenLogical):
			if strings.ToLower(node.Token.Value) == "not" {
				complexity += 2 // NOT é mais complexo
			} else {
				complexity += 1 // AND/OR
			}
		case int(FilterTokenComparison), int(FilterTokenArithmetic):
			complexity += 1
		}

		// Soma complexidade dos filhos
		for _, child := range node.Children {
			complexity += calculateComplexity(child)
		}

		return complexity
	}

	return calculateComplexity(filter.Tree)
}

// FormatFilterExpression formata uma expressão de filtro para exibição
func FormatFilterExpression(filter *GoDataFilterQuery) string {
	if filter == nil {
		return ""
	}

	if filter.RawValue != "" {
		return filter.RawValue
	}

	if filter.Tree == nil {
		return ""
	}

	// Reconstrói a expressão a partir da árvore
	var formatNode func(node *ParseNode) string
	formatNode = func(node *ParseNode) string {
		if node == nil || node.Token == nil {
			return ""
		}

		switch node.Token.Type {
		case int(FilterTokenProperty), int(FilterTokenString), int(FilterTokenNumber),
			int(FilterTokenBoolean), int(FilterTokenDateTime), int(FilterTokenDate),
			int(FilterTokenTime), int(FilterTokenGuid):
			return node.Token.Value

		case int(FilterTokenLogical), int(FilterTokenComparison), int(FilterTokenArithmetic):
			if len(node.Children) == 2 {
				left := formatNode(node.Children[0])
				right := formatNode(node.Children[1])
				return fmt.Sprintf("(%s %s %s)", left, node.Token.Value, right)
			} else if len(node.Children) == 1 && strings.ToLower(node.Token.Value) == "not" {
				operand := formatNode(node.Children[0])
				return fmt.Sprintf("not (%s)", operand)
			}

		case int(FilterTokenFunction):
			if len(node.Children) > 0 {
				args := make([]string, len(node.Children))
				for i, child := range node.Children {
					args[i] = formatNode(child)
				}
				return fmt.Sprintf("%s(%s)", node.Token.Value, strings.Join(args, ", "))
			}
			return fmt.Sprintf("%s()", node.Token.Value)
		}

		return node.Token.Value
	}

	return formatNode(filter.Tree)
}
