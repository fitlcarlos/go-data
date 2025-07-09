package odata

import (
	"fmt"
	"strconv"
	"strings"
)

// GoDataCountQuery representa uma query de count processada
// Seguindo o padrão do goDataCisco, count é um boolean simples
type GoDataCountQuery bool

// ParseCountString converte uma string do parâmetro $count da URL em um boolean
// que indica se o count deve ser incluído na resposta.
func ParseCountString(count string) (*GoDataCountQuery, error) {
	if count == "" {
		return nil, nil
	}

	// Normaliza a string para lowercase
	countLower := strings.ToLower(strings.TrimSpace(count))

	// Parse usando strconv.ParseBool que aceita: 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False
	countBool, err := strconv.ParseBool(countLower)
	if err != nil {
		return nil, fmt.Errorf("invalid $count value '%s': must be 'true' or 'false'", count)
	}

	result := GoDataCountQuery(countBool)
	return &result, nil
}

// IsCountRequested verifica se o count foi solicitado
func IsCountRequested(count *GoDataCountQuery) bool {
	return count != nil && bool(*count)
}

// ValidateCountValue valida se o valor do count é válido
func ValidateCountValue(count string) error {
	if count == "" {
		return nil
	}

	_, err := ParseCountString(count)
	return err
}

// GetCountValue retorna o valor booleano do count
func GetCountValue(count *GoDataCountQuery) bool {
	if count == nil {
		return false
	}
	return bool(*count)
}

// SetCountValue define o valor do count
func SetCountValue(value bool) *GoDataCountQuery {
	result := GoDataCountQuery(value)
	return &result
}

// String retorna a representação string do count
func (c *GoDataCountQuery) String() string {
	if c == nil {
		return "false"
	}
	return strconv.FormatBool(bool(*c))
}

// MarshalJSON implementa json.Marshaler
func (c *GoDataCountQuery) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("false"), nil
	}
	return []byte(strconv.FormatBool(bool(*c))), nil
}

// UnmarshalJSON implementa json.Unmarshaler
func (c *GoDataCountQuery) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	value, err := strconv.ParseBool(str)
	if err != nil {
		return fmt.Errorf("invalid count value: %v", err)
	}
	*c = GoDataCountQuery(value)
	return nil
}

// ParseCountParameter é uma função auxiliar para parsing de parâmetros de URL
func ParseCountParameter(params map[string]string) (*GoDataCountQuery, error) {
	countStr, exists := params["$count"]
	if !exists {
		return nil, nil
	}

	return ParseCountString(countStr)
}

// ValidateCountParameter valida o parâmetro $count em uma URL
func ValidateCountParameter(params map[string]string) error {
	countStr, exists := params["$count"]
	if !exists {
		return nil
	}

	return ValidateCountValue(countStr)
}

// ApplyCountToQuery aplica a configuração de count a uma query
func ApplyCountToQuery(query interface{}, count *GoDataCountQuery) interface{} {
	// Esta função pode ser estendida para aplicar configurações específicas
	// baseadas no tipo de query recebido
	return query
}

// GetCountSQLFragment retorna o fragmento SQL para count se necessário
func GetCountSQLFragment(count *GoDataCountQuery, tableName string) string {
	if !IsCountRequested(count) {
		return ""
	}

	// Retorna um fragmento SQL básico para count
	return fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tableName)
}

// OptimizeCountQuery otimiza uma query de count
func OptimizeCountQuery(count *GoDataCountQuery) *GoDataCountQuery {
	// Por enquanto, apenas retorna o valor original
	// Futuras otimizações podem incluir cache de count, etc.
	return count
}

// IsCountEnabled verifica se o count está habilitado globalmente
func IsCountEnabled() bool {
	// Esta função pode ser usada para verificar configurações globais
	// Por enquanto, sempre retorna true
	return true
}

// GetCountLimit retorna o limite máximo para operações de count
func GetCountLimit() int {
	// Retorna um limite padrão para operações de count
	// Pode ser configurável via environment variables
	return 1000000 // 1 milhão como limite padrão
}

// ValidateCountLimit valida se o count está dentro dos limites permitidos
func ValidateCountLimit(estimatedCount int) error {
	limit := GetCountLimit()
	if estimatedCount > limit {
		return fmt.Errorf("count operation would exceed limit of %d records", limit)
	}
	return nil
}
