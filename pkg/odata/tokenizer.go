package odata

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// GlobalFilterTokenizer é o tokenizer singleton para filtros
var GlobalFilterTokenizer *Tokenizer

// Token representa um token no parsing
type Token struct {
	Type              int
	Value             string
	SemanticType      SemanticType
	SemanticReference interface{}
}

// SemanticType representa o tipo semântico de um token
type SemanticType int

const (
	SemanticTypeUnknown SemanticType = iota
	SemanticTypeProperty
	SemanticTypeFunction
	SemanticTypeOperator
	SemanticTypeValue
	SemanticTypeKeyword
)

// FilterTokenType representa os tipos de tokens para filtros
type FilterTokenType int

const (
	FilterTokenProperty FilterTokenType = iota + 1
	FilterTokenFunction
	FilterTokenArithmetic
	FilterTokenString
	FilterTokenNumber
	FilterTokenOpenParen
	FilterTokenCloseParen
	FilterTokenComma
	FilterTokenLogical
	FilterTokenComparison
	FilterTokenBoolean
	FilterTokenNull
	FilterTokenDateTime
	FilterTokenDate
	FilterTokenTime
	FilterTokenGuid
	FilterTokenDuration
	FilterTokenGeographyPoint
	FilterTokenGeometryPoint
)

// GetGlobalFilterTokenizer retorna o tokenizer global para filtros
func GetGlobalFilterTokenizer() *Tokenizer {
	if GlobalFilterTokenizer == nil {
		GlobalFilterTokenizer = createFilterTokenizer()
	}
	return GlobalFilterTokenizer
}

// createFilterTokenizer cria um tokenizer otimizado para filtros OData
func createFilterTokenizer() *Tokenizer {
	t := &Tokenizer{}

	// Operadores lógicos (precedência importante)
	t.Add(`^(?i)\b(and|or|not)\b`, int(FilterTokenLogical))

	// Operadores de comparação
	t.Add(`^(?i)\b(eq|ne|gt|ge|lt|le|has|in)\b`, int(FilterTokenComparison))

	// Operadores aritméticos
	t.Add(`^(?i)\b(add|sub|mul|div|divby|mod)\b`, int(FilterTokenArithmetic))

	// Funções (lista completa de funções OData)
	t.Add(`^(?i)\b(contains|startswith|endswith|length|indexof|substring|tolower|toupper|trim|concat|year|month|day|hour|minute|second|now|date|time|round|floor|ceiling|cast|isof)\b`, int(FilterTokenFunction))

	// Parênteses
	t.Add(`^\(`, int(FilterTokenOpenParen))
	t.Add(`^\)`, int(FilterTokenCloseParen))

	// Vírgulas
	t.Add(`^,`, int(FilterTokenComma))

	// Tipos de dados especiais

	// GUID: 12345678-1234-5678-9012-123456789012
	t.Add(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, int(FilterTokenGuid))

	// Geography Point: geography'POINT(-122.3 47.6)'
	t.Add(`^geography'[^']*'`, int(FilterTokenGeographyPoint))

	// Geometry Point: geometry'POINT(-122.3 47.6)'
	t.Add(`^geometry'[^']*'`, int(FilterTokenGeometryPoint))

	// DateTime: 2023-12-25T10:30:00Z ou 2023-12-25T10:30:00.000Z
	t.Add(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{3})?Z?`, int(FilterTokenDateTime))

	// Date: 2023-12-25
	t.Add(`^\d{4}-\d{2}-\d{2}`, int(FilterTokenDate))

	// Time: 14:30:00
	t.Add(`^\d{2}:\d{2}:\d{2}`, int(FilterTokenTime))

	// Duration: P1DT12H30M5S
	t.Add(`^P(\d+D)?(T(\d+H)?(\d+M)?(\d+S)?)?`, int(FilterTokenDuration))

	// Boolean
	t.Add(`^(?i)\b(true|false)\b`, int(FilterTokenBoolean))

	// Null
	t.Add(`^(?i)\bnull\b`, int(FilterTokenNull))

	// Strings (single quotes)
	t.Add(`^'([^'\\]|\\.)*'`, int(FilterTokenString))

	// Números (int, float, decimal)
	t.Add(`^-?\d+(\.\d+)?([eE][+-]?\d+)?[dDfFmM]?`, int(FilterTokenNumber))

	// Propriedades/Identificadores (deve vir por último)
	t.Add(`^[a-zA-Z_][a-zA-Z0-9_]*`, int(FilterTokenProperty))

	// Whitespace (skip)
	t.Add(`^\s+`, -1) // -1 indica que deve ser ignorado

	return t
}

// Tokenizer é responsável por tokenizar strings
type Tokenizer struct {
	patterns []tokenPattern
}

// tokenPattern representa um padrão de token
type tokenPattern struct {
	regex     *regexp.Regexp
	tokenType int
}

// Add adiciona um padrão de token ao tokenizer
func (t *Tokenizer) Add(pattern string, tokenType int) {
	regex := regexp.MustCompile(pattern)
	t.patterns = append(t.patterns, tokenPattern{
		regex:     regex,
		tokenType: tokenType,
	})
}

// Tokenize tokeniza uma string em tokens
func (t *Tokenizer) Tokenize(ctx context.Context, input string) ([]*Token, error) {
	var tokens []*Token
	remaining := strings.TrimSpace(input)

	for len(remaining) > 0 {
		matched := false

		for _, pattern := range t.patterns {
			if pattern.regex.MatchString(remaining) {
				match := pattern.regex.FindString(remaining)
				if match != "" {
					tokens = append(tokens, &Token{
						Type:  pattern.tokenType,
						Value: strings.TrimSpace(match),
					})
					remaining = strings.TrimSpace(remaining[len(match):])
					matched = true
					break
				}
			}
		}

		if !matched {
			return nil, fmt.Errorf("unable to tokenize: '%s'", remaining)
		}
	}

	return tokens, nil
}

// TokenStack representa uma pilha de tokens
type TokenStack struct {
	tokens []*Token
}

// NewTokenStack cria uma nova pilha de tokens
func NewTokenStack() *TokenStack {
	return &TokenStack{
		tokens: make([]*Token, 0),
	}
}

// Push adiciona um token ao topo da pilha
func (s *TokenStack) Push(token *Token) {
	s.tokens = append(s.tokens, token)
}

// Pop remove e retorna o token do topo da pilha
func (s *TokenStack) Pop() *Token {
	if len(s.tokens) == 0 {
		return nil
	}

	index := len(s.tokens) - 1
	token := s.tokens[index]
	s.tokens = s.tokens[:index]
	return token
}

// Peek retorna o token do topo da pilha sem removê-lo
func (s *TokenStack) Peek() *Token {
	if len(s.tokens) == 0 {
		return nil
	}
	return s.tokens[len(s.tokens)-1]
}

// Empty verifica se a pilha está vazia
func (s *TokenStack) Empty() bool {
	return len(s.tokens) == 0
}

// Size retorna o tamanho da pilha
func (s *TokenStack) Size() int {
	return len(s.tokens)
}

// TokenQueue representa uma fila de tokens
type TokenQueue struct {
	tokens []*Token
	head   int
}

// NewTokenQueue cria uma nova fila de tokens
func NewTokenQueue() *TokenQueue {
	return &TokenQueue{
		tokens: make([]*Token, 0),
		head:   0,
	}
}

// Enqueue adiciona um token ao final da fila
func (q *TokenQueue) Enqueue(token *Token) {
	q.tokens = append(q.tokens, token)
}

// Dequeue remove e retorna o token do início da fila
func (q *TokenQueue) Dequeue() *Token {
	if q.head >= len(q.tokens) {
		return nil
	}

	token := q.tokens[q.head]
	q.head++
	return token
}

// Peek retorna o token do início da fila sem removê-lo
func (q *TokenQueue) Peek() *Token {
	if q.head >= len(q.tokens) {
		return nil
	}
	return q.tokens[q.head]
}

// Empty verifica se a fila está vazia
func (q *TokenQueue) Empty() bool {
	return q.head >= len(q.tokens)
}

// Size retorna o tamanho da fila
func (q *TokenQueue) Size() int {
	return len(q.tokens) - q.head
}

// GetValue retorna o valor concatenado de todos os tokens restantes na fila
func (q *TokenQueue) GetValue() string {
	if q.Empty() {
		return ""
	}

	var values []string
	for i := q.head; i < len(q.tokens); i++ {
		values = append(values, q.tokens[i].Value)
	}

	return strings.Join(values, " ")
}

// GetValueUntilSeparator retorna o valor concatenado dos tokens até encontrar um separador (; ou ,)
func (q *TokenQueue) GetValueUntilSeparator() string {
	if q.Empty() {
		return ""
	}

	var values []string
	for i := q.head; i < len(q.tokens); i++ {
		// Para no ponto e vírgula ou vírgula
		if q.tokens[i].Value == ";" || q.tokens[i].Value == "," {
			break
		}
		values = append(values, q.tokens[i].Value)
	}

	return strings.Join(values, " ")
}

// Reset reinicia a fila
func (q *TokenQueue) Reset() {
	q.tokens = make([]*Token, 0)
	q.head = 0
}

// ToSlice retorna todos os tokens restantes como slice
func (q *TokenQueue) ToSlice() []*Token {
	if q.Empty() {
		return []*Token{}
	}

	return q.tokens[q.head:]
}

// GoDataSearchQuery representa uma query de busca OData (placeholder)
type GoDataSearchQuery struct {
	Tree     *ParseNode
	RawValue string
}

// Query retorna a query de busca original
func (q *GoDataSearchQuery) Query() string {
	if q == nil {
		return ""
	}
	return q.RawValue
}

// String retorna a representação string da query
func (q *GoDataSearchQuery) String() string {
	if q == nil {
		return ""
	}
	return q.RawValue
}

// GetTerms extrai todos os termos de busca da query
func (q *GoDataSearchQuery) GetTerms() []string {
	if q == nil || q.RawValue == "" {
		return []string{}
	}

	// Divide por espaços e remove termos vazios
	terms := strings.Fields(q.RawValue)

	// Remove operadores lógicos e parênteses
	filtered := make([]string, 0, len(terms))
	for _, term := range terms {
		upper := strings.ToUpper(term)
		if upper != "AND" && upper != "OR" && upper != "NOT" &&
			term != "(" && term != ")" {
			// Remove aspas se houver
			term = strings.Trim(term, "\"'")
			if term != "" {
				filtered = append(filtered, term)
			}
		}
	}

	return filtered
}

// GoDataOrderByQuery representa uma query de ordenação OData (placeholder)
type GoDataOrderByQuery struct {
	OrderByItems []*OrderByItem
	RawValue     string
}

// OrderByItem representa um item de ordenação (placeholder)
type OrderByItem struct {
	Property  string
	Direction OrderByDirection
}

// GoDataComputeQuery representa uma query de compute OData (placeholder)
type GoDataComputeQuery struct {
	ComputeItems []*ComputeItem
	RawValue     string
}

// ComputeItem representa um item de compute (placeholder)
type ComputeItem struct {
	Expression string
	Alias      string
}

// Funções placeholder para parsers que ainda não foram implementados
func ParseSearchString(ctx context.Context, search string) (*GoDataSearchQuery, error) {
	// Retorna nil para string vazia (convenção Go)
	if search == "" {
		return nil, nil
	}

	// TODO: Implementar parser de search completo
	return &GoDataSearchQuery{RawValue: search}, nil
}

func ParseOrderByString(ctx context.Context, orderBy string) (*GoDataOrderByQuery, error) {
	// TODO: Implementar parser de orderby
	return &GoDataOrderByQuery{RawValue: orderBy}, nil
}

func ParseComputeString(ctx context.Context, compute string) (*GoDataComputeQuery, error) {
	// TODO: Implementar parser de compute
	return &GoDataComputeQuery{RawValue: compute}, nil
}
