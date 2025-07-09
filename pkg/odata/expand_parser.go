package odata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ExpandTokenType representa os tipos de tokens para parsing do $expand
type ExpandTokenType int

func (e ExpandTokenType) Value() int {
	return int(e)
}

const (
	ExpandTokenOpenParen ExpandTokenType = iota
	ExpandTokenCloseParen
	ExpandTokenNav
	ExpandTokenComma
	ExpandTokenSemicolon
	ExpandTokenEquals
	ExpandTokenLiteral
)

// GlobalExpandTokenizer é o tokenizer global singleton para $expand
var GlobalExpandTokenizer = NewExpandTokenizer()

// GoDataExpandQuery representa uma query de expansão OData
type GoDataExpandQuery struct {
	ExpandItems []*ExpandItem
	RawValue    string
}

// ExpandItem representa um item individual de expansão
type ExpandItem struct {
	Path    []*Token
	Filter  *GoDataFilterQuery
	At      *GoDataFilterQuery
	Search  *GoDataSearchQuery
	OrderBy *GoDataOrderByQuery
	Skip    *GoDataSkipQuery
	Top     *GoDataTopQuery
	Select  *GoDataSelectQuery
	Compute *GoDataComputeQuery
	Expand  *GoDataExpandQuery
	Levels  int
}

// NewExpandTokenizer cria um novo tokenizer para $expand
func NewExpandTokenizer() *Tokenizer {
	t := &Tokenizer{}
	t.Add("^\\(", int(ExpandTokenOpenParen))
	t.Add("^\\)", int(ExpandTokenCloseParen))
	t.Add("^/", int(ExpandTokenNav))
	t.Add("^,", int(ExpandTokenComma))
	t.Add("^;", int(ExpandTokenSemicolon))
	t.Add("^=", int(ExpandTokenEquals))
	t.Add("^[a-zA-Z0-9_\\.:\\$ \\*]+", int(ExpandTokenLiteral))
	return t
}

// ParseExpandString faz o parsing de uma string de $expand
func ParseExpandString(ctx context.Context, expand string) (*GoDataExpandQuery, error) {
	if expand == "" {
		return &GoDataExpandQuery{ExpandItems: []*ExpandItem{}, RawValue: expand}, nil
	}

	tokens, err := GlobalExpandTokenizer.Tokenize(ctx, expand)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize expand string: %w", err)
	}

	stack := NewTokenStack()
	queue := NewTokenQueue()
	items := make([]*ExpandItem, 0)

	for len(tokens) > 0 {
		token := tokens[0]
		tokens = tokens[1:]

		switch token.Value {
		case "(":
			queue.Enqueue(token)
			stack.Push(token)
		case ")":
			queue.Enqueue(token)
			stack.Pop()
		case ",":
			if stack.Empty() {
				// Vírgula no nível superior - parse este item e inicia nova queue
				item, err := ParseExpandItem(ctx, queue)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
				queue = NewTokenQueue()
			} else {
				// Vírgula dentro de expressão aninhada - mantém na queue
				queue.Enqueue(token)
			}
		default:
			queue.Enqueue(token)
		}
	}

	if !stack.Empty() {
		return nil, fmt.Errorf("mismatched parentheses in expand clause")
	}

	// Parse o último item
	if !queue.Empty() {
		item, err := ParseExpandItem(ctx, queue)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return &GoDataExpandQuery{ExpandItems: items, RawValue: expand}, nil
}

// ParseExpandItem faz o parsing de um item individual de expansão
func ParseExpandItem(ctx context.Context, input *TokenQueue) (*ExpandItem, error) {
	item := &ExpandItem{}
	item.Path = []*Token{}

	stack := NewTokenStack()
	queue := NewTokenQueue()

	for !input.Empty() {
		token := input.Dequeue()

		switch token.Value {
		case "(":
			if !stack.Empty() {
				// Parêntese aninhado - vai para a queue
				queue.Enqueue(token)
			} else {
				// Parêntese de nível superior - finaliza parsing do path
				if !queue.Empty() {
					item.Path = append(item.Path, queue.Dequeue())
				}
			}
			stack.Push(token)
		case ")":
			stack.Pop()
			if !stack.Empty() {
				// Parêntese aninhado - vai para a queue
				queue.Enqueue(token)
			} else {
				// Parêntese de nível superior - parse as opções
				err := ParseExpandOption(ctx, queue, item)
				if err != nil {
					return nil, err
				}
				queue = NewTokenQueue()
			}
		case "/":
			if stack.Empty() {
				if queue.Empty() {
					return nil, fmt.Errorf("empty path segment in expand clause")
				}
				if input.Empty() {
					return nil, fmt.Errorf("empty path segment in expand clause")
				}
				// Barra no nível raiz - separa segmentos do path
				item.Path = append(item.Path, queue.Dequeue())
			} else {
				queue.Enqueue(token)
			}
		case ";":
			if stack.Size() == 1 {
				// Ponto e vírgula separa opções de expand no primeiro nível
				err := ParseExpandOption(ctx, queue, item)
				if err != nil {
					return nil, err
				}
				queue = NewTokenQueue()
			} else {
				queue.Enqueue(token)
			}
		case ",":
			if stack.Size() == 1 {
				// Vírgula também separa opções de expand no primeiro nível
				err := ParseExpandOption(ctx, queue, item)
				if err != nil {
					return nil, err
				}
				queue = NewTokenQueue()
			} else {
				queue.Enqueue(token)
			}
		default:
			queue.Enqueue(token)
		}
	}

	if !stack.Empty() {
		return nil, fmt.Errorf("mismatched parentheses in expand clause")
	}

	if !queue.Empty() {
		item.Path = append(item.Path, queue.Dequeue())
	}

	if len(item.Path) == 0 {
		return nil, fmt.Errorf("empty expand item")
	}

	return item, nil
}

// ParseExpandOption faz o parsing de uma opção de expansão
func ParseExpandOption(ctx context.Context, queue *TokenQueue, item *ExpandItem) error {
	if queue.Empty() {
		return fmt.Errorf("empty expand option")
	}

	head := queue.Dequeue().Value
	if queue.Empty() {
		return fmt.Errorf("invalid expand clause format")
	}

	equals := queue.Dequeue()
	if equals.Value != "=" {
		return fmt.Errorf("expected '=' in expand option")
	}

	body := queue.GetValueUntilSeparator()
	if body == "" {
		return fmt.Errorf("empty expand option value")
	}

	// Remove $ prefix se existir
	head = strings.TrimPrefix(head, "$")

	switch strings.ToLower(head) {
	case "filter":
		filter, err := ParseFilterString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse filter in expand: %w", err)
		}
		item.Filter = filter
	case "at":
		at, err := ParseFilterString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse at in expand: %w", err)
		}
		item.At = at
	case "search":
		search, err := ParseSearchString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse search in expand: %w", err)
		}
		item.Search = search
	case "orderby":
		orderby, err := ParseOrderByString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse orderby in expand: %w", err)
		}
		item.OrderBy = orderby
	case "skip":
		skip, err := ParseSkipString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse skip in expand: %w", err)
		}
		item.Skip = skip
	case "top":
		top, err := ParseTopString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse top in expand: %w", err)
		}
		item.Top = top
	case "select":
		sel, err := ParseSelectString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse select in expand: %w", err)
		}
		item.Select = sel
	case "compute":
		comp, err := ParseComputeString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse compute in expand: %w", err)
		}
		item.Compute = comp
	case "expand":
		expand, err := ParseExpandString(ctx, body)
		if err != nil {
			return fmt.Errorf("failed to parse nested expand: %w", err)
		}
		item.Expand = expand
	case "levels":
		levels, err := strconv.Atoi(body)
		if err != nil {
			return fmt.Errorf("invalid levels value: %w", err)
		}
		if levels < 0 {
			return fmt.Errorf("levels must be non-negative")
		}
		item.Levels = levels
	default:
		return fmt.Errorf("unsupported expand option: %s", head)
	}

	return nil
}

// ValidateExpandQuery valida uma query de expansão contra metadados
func ValidateExpandQuery(expand *GoDataExpandQuery, service *EntityService, entity *EntityMetadata) error {
	if expand == nil {
		return nil
	}

	for _, item := range expand.ExpandItems {
		if err := ValidateExpandItem(item, service, entity); err != nil {
			return err
		}
	}

	return nil
}

// ValidateExpandItem valida um item de expansão
func ValidateExpandItem(item *ExpandItem, service *EntityService, entity *EntityMetadata) error {
	if len(item.Path) == 0 {
		return fmt.Errorf("empty expand path")
	}

	// Valida o primeiro segmento do path
	pathSegment := item.Path[0].Value
	var navigationProperty *PropertyMetadata

	for _, prop := range entity.Properties {
		if strings.EqualFold(prop.Name, pathSegment) {
			if !prop.IsNavigation {
				return fmt.Errorf("property '%s' is not a navigation property", pathSegment)
			}
			navigationProperty = &prop
			break
		}
	}

	if navigationProperty == nil {
		return fmt.Errorf("navigation property '%s' not found in entity '%s'", pathSegment, entity.Name)
	}

	// TODO: Validar segmentos de path adicionais para navegação aninhada
	// TODO: Validar opções de expand (filter, select, etc.) contra a entidade relacionada

	return nil
}

// ConvertExpandToSQL converte uma query de expansão para SQL
func ConvertExpandToSQL(expand *GoDataExpandQuery, entity *EntityMetadata) ([]string, error) {
	if expand == nil || len(expand.ExpandItems) == 0 {
		return []string{}, nil
	}

	var joins []string

	for _, item := range expand.ExpandItems {
		join, err := ConvertExpandItemToSQL(item, entity)
		if err != nil {
			return nil, err
		}
		if join != "" {
			joins = append(joins, join)
		}
	}

	return joins, nil
}

// ConvertExpandItemToSQL converte um item de expansão para SQL JOIN
func ConvertExpandItemToSQL(item *ExpandItem, entity *EntityMetadata) (string, error) {
	if len(item.Path) == 0 {
		return "", fmt.Errorf("empty expand path")
	}

	pathSegment := item.Path[0].Value

	// Encontra a propriedade de navegação
	var navigationProperty *PropertyMetadata
	for _, prop := range entity.Properties {
		if strings.EqualFold(prop.Name, pathSegment) && prop.IsNavigation {
			navigationProperty = &prop
			break
		}
	}

	if navigationProperty == nil {
		return "", fmt.Errorf("navigation property '%s' not found", pathSegment)
	}

	// Gera o JOIN baseado no tipo de relacionamento
	if navigationProperty.Association != nil {
		// Relacionamento simples (1:1 ou N:1)
		join := fmt.Sprintf("LEFT JOIN %s AS %s ON %s.%s = %s.%s",
			navigationProperty.RelatedType,
			pathSegment,
			entity.TableName,
			navigationProperty.Association.ForeignKey,
			pathSegment,
			navigationProperty.Association.References)
		return join, nil
	}

	if navigationProperty.ManyAssociation != nil {
		// Relacionamento múltiplo (1:N ou N:N)
		if navigationProperty.ManyAssociation.JoinTable != "" {
			// Relacionamento N:N com tabela de junção
			join := fmt.Sprintf("LEFT JOIN %s ON %s.%s = %s.%s LEFT JOIN %s AS %s ON %s.%s = %s.%s",
				navigationProperty.ManyAssociation.JoinTable,
				entity.TableName,
				navigationProperty.ManyAssociation.ForeignKey,
				navigationProperty.ManyAssociation.JoinTable,
				navigationProperty.ManyAssociation.JoinColumn,
				navigationProperty.RelatedType,
				pathSegment,
				navigationProperty.ManyAssociation.JoinTable,
				navigationProperty.ManyAssociation.InverseJoinColumn,
				pathSegment,
				navigationProperty.ManyAssociation.References)
			return join, nil
		} else {
			// Relacionamento 1:N simples
			join := fmt.Sprintf("LEFT JOIN %s AS %s ON %s.%s = %s.%s",
				navigationProperty.RelatedType,
				pathSegment,
				entity.TableName,
				navigationProperty.ManyAssociation.ForeignKey,
				pathSegment,
				navigationProperty.ManyAssociation.References)
			return join, nil
		}
	}

	return "", fmt.Errorf("invalid navigation property configuration for '%s'", pathSegment)
}

// GetExpandedProperties retorna as propriedades expandidas
func GetExpandedProperties(expand *GoDataExpandQuery) []string {
	if expand == nil {
		return []string{}
	}

	var properties []string
	for _, item := range expand.ExpandItems {
		if len(item.Path) > 0 {
			properties = append(properties, item.Path[0].Value)
		}
	}

	return properties
}

// GetExpandComplexity calcula a complexidade de uma query de expansão
func GetExpandComplexity(expand *GoDataExpandQuery) int {
	if expand == nil {
		return 0
	}

	complexity := 0
	for _, item := range expand.ExpandItems {
		complexity += calculateExpandItemComplexity(item)
	}

	return complexity
}

// calculateExpandItemComplexity calcula a complexidade de um item de expansão
func calculateExpandItemComplexity(item *ExpandItem) int {
	complexity := 1 // Base complexity for the expand item

	// Add complexity for nested options
	if item.Filter != nil {
		complexity += GetFilterComplexity(item.Filter)
	}
	if item.OrderBy != nil {
		complexity += 1
	}
	if item.Select != nil {
		complexity += 1
	}
	if item.Expand != nil {
		complexity += GetExpandComplexity(item.Expand) * 2 // Nested expands are more complex
	}
	if item.Levels > 1 {
		complexity += item.Levels * 2 // Multiple levels increase complexity
	}

	return complexity
}

// IsSimpleExpand verifica se é uma expansão simples (sem opções aninhadas)
func IsSimpleExpand(expand *GoDataExpandQuery) bool {
	if expand == nil {
		return true
	}

	for _, item := range expand.ExpandItems {
		if item.Filter != nil || item.OrderBy != nil || item.Select != nil ||
			item.Expand != nil || item.Skip != nil || item.Top != nil ||
			item.Compute != nil || item.Search != nil || item.Levels > 0 {
			return false
		}
	}

	return true
}

// FormatExpandExpression formata uma expressão de expansão para exibição
func FormatExpandExpression(expand *GoDataExpandQuery) string {
	if expand == nil {
		return ""
	}

	var parts []string
	for _, item := range expand.ExpandItems {
		parts = append(parts, formatExpandItem(item))
	}

	return strings.Join(parts, ", ")
}

// formatExpandItem formata um item de expansão
func formatExpandItem(item *ExpandItem) string {
	if len(item.Path) == 0 {
		return ""
	}

	path := item.Path[0].Value

	var options []string
	if item.Filter != nil {
		options = append(options, fmt.Sprintf("$filter=%s", item.Filter.RawValue))
	}
	if item.OrderBy != nil {
		options = append(options, fmt.Sprintf("$orderby=%s", item.OrderBy.RawValue))
	}
	if item.Select != nil {
		options = append(options, fmt.Sprintf("$select=%s", item.Select.RawValue))
	}
	if item.Skip != nil {
		options = append(options, fmt.Sprintf("$skip=%d", int(*item.Skip)))
	}
	if item.Top != nil {
		options = append(options, fmt.Sprintf("$top=%d", int(*item.Top)))
	}
	if item.Levels > 0 {
		options = append(options, fmt.Sprintf("$levels=%d", item.Levels))
	}
	if item.Expand != nil {
		options = append(options, fmt.Sprintf("$expand=%s", FormatExpandExpression(item.Expand)))
	}

	if len(options) > 0 {
		return fmt.Sprintf("%s(%s)", path, strings.Join(options, ";"))
	}

	return path
}

// String retorna a representação em string da query de expansão
func (e *GoDataExpandQuery) String() string {
	if e == nil {
		return ""
	}
	return e.RawValue
}
