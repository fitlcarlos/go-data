package odata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GoDataTopQuery representa uma query de $top OData
type GoDataTopQuery int

// GoDataSkipQuery representa uma query de $skip OData
type GoDataSkipQuery int

// Constantes para limites padrão
const (
	DefaultTopLimit  = DefaultMaxTopValue  // Usa constante centralizada
	DefaultSkipLimit = DefaultMaxSkipValue // Usa constante centralizada
	MinTopValue      = 0
	MinSkipValue     = 0
)

// ParseTopString faz o parsing de uma string de $top
func ParseTopString(ctx context.Context, top string) (*GoDataTopQuery, error) {
	if top == "" {
		return nil, nil
	}

	value, err := strconv.Atoi(top)
	if err != nil {
		return nil, fmt.Errorf("invalid $top value '%s': must be a non-negative integer", top)
	}

	if value < MinTopValue {
		return nil, fmt.Errorf("$top value must be non-negative, got %d", value)
	}

	if value > DefaultTopLimit {
		return nil, fmt.Errorf("$top value cannot exceed %d, got %d", DefaultTopLimit, value)
	}

	result := GoDataTopQuery(value)
	return &result, nil
}

// ParseSkipString faz o parsing de uma string de $skip
func ParseSkipString(ctx context.Context, skip string) (*GoDataSkipQuery, error) {
	if skip == "" {
		return nil, nil
	}

	value, err := strconv.Atoi(skip)
	if err != nil {
		return nil, fmt.Errorf("invalid $skip value '%s': must be a non-negative integer", skip)
	}

	if value < MinSkipValue {
		return nil, fmt.Errorf("$skip value must be non-negative, got %d", value)
	}

	if value > DefaultSkipLimit {
		return nil, fmt.Errorf("$skip value cannot exceed %d, got %d", DefaultSkipLimit, value)
	}

	result := GoDataSkipQuery(value)
	return &result, nil
}

// ValidateTopQuery valida uma query de $top
func ValidateTopQuery(top *GoDataTopQuery) error {
	if top == nil {
		return nil
	}

	value := int(*top)
	if value < MinTopValue {
		return fmt.Errorf("$top value must be non-negative, got %d", value)
	}

	if value > DefaultTopLimit {
		return fmt.Errorf("$top value cannot exceed %d, got %d", DefaultTopLimit, value)
	}

	return nil
}

// ValidateSkipQuery valida uma query de $skip
func ValidateSkipQuery(skip *GoDataSkipQuery) error {
	if skip == nil {
		return nil
	}

	value := int(*skip)
	if value < MinSkipValue {
		return fmt.Errorf("$skip value must be non-negative, got %d", value)
	}

	if value > DefaultSkipLimit {
		return fmt.Errorf("$skip value cannot exceed %d, got %d", DefaultSkipLimit, value)
	}

	return nil
}

// GetTopValue retorna o valor de $top
func GetTopValue(top *GoDataTopQuery) int {
	if top == nil {
		return 0
	}
	return int(*top)
}

// GetSkipValue retorna o valor de $skip
func GetSkipValue(skip *GoDataSkipQuery) int {
	if skip == nil {
		return 0
	}
	return int(*skip)
}

// SetTopValue define o valor de $top
func SetTopValue(top *GoDataTopQuery, value int) error {
	if value < MinTopValue {
		return fmt.Errorf("$top value must be non-negative, got %d", value)
	}

	if value > DefaultTopLimit {
		return fmt.Errorf("$top value cannot exceed %d, got %d", DefaultTopLimit, value)
	}

	*top = GoDataTopQuery(value)
	return nil
}

// SetSkipValue define o valor de $skip
func SetSkipValue(skip *GoDataSkipQuery, value int) error {
	if value < MinSkipValue {
		return fmt.Errorf("$skip value must be non-negative, got %d", value)
	}

	if value > DefaultSkipLimit {
		return fmt.Errorf("$skip value cannot exceed %d, got %d", DefaultSkipLimit, value)
	}

	*skip = GoDataSkipQuery(value)
	return nil
}

// String retorna a representação em string da query de $top
func (t *GoDataTopQuery) String() string {
	if t == nil {
		return ""
	}
	return strconv.Itoa(int(*t))
}

// String retorna a representação em string da query de $skip
func (s *GoDataSkipQuery) String() string {
	if s == nil {
		return ""
	}
	return strconv.Itoa(int(*s))
}

// ConvertTopSkipToSQL converte $top e $skip para SQL LIMIT e OFFSET
func ConvertTopSkipToSQL(top *GoDataTopQuery, skip *GoDataSkipQuery) (string, []interface{}) {
	var sql string
	var args []interface{}

	skipValue := GetSkipValue(skip)
	topValue := GetTopValue(top)

	if skipValue > 0 && topValue > 0 {
		sql = "LIMIT ? OFFSET ?"
		args = []interface{}{topValue, skipValue}
	} else if topValue > 0 {
		sql = "LIMIT ?"
		args = []interface{}{topValue}
	} else if skipValue > 0 {
		sql = "OFFSET ?"
		args = []interface{}{skipValue}
	}

	return sql, args
}

// ValidateTopSkipCombination valida a combinação de $top e $skip
func ValidateTopSkipCombination(top *GoDataTopQuery, skip *GoDataSkipQuery) error {
	if err := ValidateTopQuery(top); err != nil {
		return err
	}

	if err := ValidateSkipQuery(skip); err != nil {
		return err
	}

	// Validações adicionais para a combinação
	skipValue := GetSkipValue(skip)
	topValue := GetTopValue(top)

	if skipValue > 0 && topValue == 0 {
		return fmt.Errorf("$skip requires $top to be specified")
	}

	// Verifica se a combinação pode resultar em muitos dados
	if skipValue > DefaultMaxSkipValue && topValue > DefaultMaxTopValue {
		return fmt.Errorf("combination of $skip (%d) and $top (%d) may result in excessive data", skipValue, topValue)
	}

	return nil
}

// GetPaginationInfo retorna informações de paginação
func GetPaginationInfo(top *GoDataTopQuery, skip *GoDataSkipQuery) PaginationInfo {
	return PaginationInfo{
		PageSize:   GetTopValue(top),
		Offset:     GetSkipValue(skip),
		HasTop:     top != nil,
		HasSkip:    skip != nil,
		PageNumber: calculatePageNumber(top, skip),
	}
}

// PaginationInfo contém informações de paginação
type PaginationInfo struct {
	PageSize   int
	Offset     int
	HasTop     bool
	HasSkip    bool
	PageNumber int
}

// calculatePageNumber calcula o número da página baseado em top e skip
func calculatePageNumber(top *GoDataTopQuery, skip *GoDataSkipQuery) int {
	if top == nil || skip == nil {
		return 1
	}

	topValue := GetTopValue(top)
	skipValue := GetSkipValue(skip)

	if topValue == 0 {
		return 1
	}

	return (skipValue / topValue) + 1
}

// IsFirstPage verifica se é a primeira página
func (p PaginationInfo) IsFirstPage() bool {
	return p.PageNumber <= 1
}

// HasPagination verifica se há paginação
func (p PaginationInfo) HasPagination() bool {
	return p.HasTop || p.HasSkip
}

// GetNextOffset calcula o offset para a próxima página
func (p PaginationInfo) GetNextOffset() int {
	if p.PageSize == 0 {
		return p.Offset
	}
	return p.Offset + p.PageSize
}

// GetPreviousOffset calcula o offset para a página anterior
func (p PaginationInfo) GetPreviousOffset() int {
	if p.PageSize == 0 || p.Offset < p.PageSize {
		return 0
	}
	return p.Offset - p.PageSize
}

// ParseTopSkipFromURL faz o parsing de $top e $skip de parâmetros de URL
func ParseTopSkipFromURL(topStr, skipStr string) (*GoDataTopQuery, *GoDataSkipQuery, error) {
	ctx := context.Background()

	top, err := ParseTopString(ctx, topStr)
	if err != nil {
		return nil, nil, err
	}

	skip, err := ParseSkipString(ctx, skipStr)
	if err != nil {
		return nil, nil, err
	}

	return top, skip, nil
}

// ApplyDefaultLimits aplica limites padrão se não especificados
func ApplyDefaultLimits(top *GoDataTopQuery, skip *GoDataSkipQuery, defaultTop int) (*GoDataTopQuery, *GoDataSkipQuery) {
	// Aplica top padrão se não especificado
	if top == nil && defaultTop > 0 {
		defaultTopQuery := GoDataTopQuery(defaultTop)
		top = &defaultTopQuery
	}

	// Skip permanece nil se não especificado
	return top, skip
}

// FormatTopSkipForURL formata $top e $skip para URL
func FormatTopSkipForURL(top *GoDataTopQuery, skip *GoDataSkipQuery) string {
	var parts []string

	if top != nil {
		parts = append(parts, fmt.Sprintf("$top=%d", int(*top)))
	}

	if skip != nil {
		parts = append(parts, fmt.Sprintf("$skip=%d", int(*skip)))
	}

	if len(parts) == 0 {
		return ""
	}

	return "&" + strings.Join(parts, "&")
}

// GetTopSkipComplexity calcula a complexidade de $top e $skip
func GetTopSkipComplexity(top *GoDataTopQuery, skip *GoDataSkipQuery) int {
	complexity := 0

	if top != nil {
		complexity += 1
		if int(*top) > 1000 {
			complexity += 1 // Valores grandes adicionam complexidade
		}
	}

	if skip != nil {
		complexity += 1
		if int(*skip) > DefaultMaxTopValue {
			complexity += 2 // Skip grande adiciona mais complexidade
		}
	}

	return complexity
}
