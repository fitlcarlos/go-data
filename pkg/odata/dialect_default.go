package odata

import (
	"fmt"
	"strings"
	"time"
)

// DefaultDialect implementa SQLDialect para bancos genéricos
type DefaultDialect struct{}

// GetName retorna o nome do dialeto
func (d *DefaultDialect) GetName() string {
	return "default"
}

// SetupNodeMap configura o mapa de operadores OData para SQL
func (d *DefaultDialect) SetupNodeMap() NodeMap {
	nodeMap := make(NodeMap)

	// Operadores de comparação
	nodeMap["eq"] = "(%s = %s)"
	nodeMap["ne"] = "(%s != %s)"
	nodeMap["gt"] = "(%s > %s)"
	nodeMap["ge"] = "(%s >= %s)"
	nodeMap["lt"] = "(%s < %s)"
	nodeMap["le"] = "(%s <= %s)"

	// Operadores lógicos
	nodeMap["and"] = "(%s AND %s)"
	nodeMap["or"] = "(%s OR %s)"
	nodeMap["not"] = "(NOT %s)"

	// Operadores aritméticos
	nodeMap["add"] = "(%s + %s)"
	nodeMap["sub"] = "(%s - %s)"
	nodeMap["mul"] = "(%s * %s)"
	nodeMap["div"] = "(%s / %s)"
	nodeMap["mod"] = "(%s %% %s)"

	// Funções de string
	nodeMap["contains"] = "(%s LIKE %s)"
	nodeMap["startswith"] = "(%s LIKE %s)"
	nodeMap["endswith"] = "(%s LIKE %s)"
	nodeMap["length"] = "LENGTH(%s)"
	nodeMap["indexof"] = "LOCATE(%s, %s)"
	nodeMap["substring"] = "SUBSTRING(%s, %s, %s)"
	nodeMap["tolower"] = "LOWER(%s)"
	nodeMap["toupper"] = "UPPER(%s)"
	nodeMap["trim"] = "TRIM(%s)"
	nodeMap["concat"] = "CONCAT(%s, %s)"

	// Funções de data/hora
	nodeMap["year"] = "YEAR(%s)"
	nodeMap["month"] = "MONTH(%s)"
	nodeMap["day"] = "DAY(%s)"
	nodeMap["hour"] = "HOUR(%s)"
	nodeMap["minute"] = "MINUTE(%s)"
	nodeMap["second"] = "SECOND(%s)"
	nodeMap["now"] = "NOW()"
	nodeMap["date"] = "DATE(%s)"
	nodeMap["time"] = "TIME(%s)"

	// Funções matemáticas
	nodeMap["round"] = "ROUND(%s)"
	nodeMap["floor"] = "FLOOR(%s)"
	nodeMap["ceiling"] = "CEIL(%s)"

	// Valores especiais
	nodeMap["null"] = "NULL"

	return nodeMap
}

// SetupPrepareMap configura o mapa de preparação de valores
func (d *DefaultDialect) SetupPrepareMap() PrepareMap {
	prepareMap := make(PrepareMap)
	prepareMap["contains"] = "%%%s%%"
	prepareMap["startswith"] = "%s%%"
	prepareMap["endswith"] = "%%%s"
	return prepareMap
}

// BuildLimitClause constrói cláusula LIMIT/OFFSET genérica
func (d *DefaultDialect) BuildLimitClause(top, skip int) string {
	if top > 0 && skip > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", top, skip)
	} else if top > 0 {
		return fmt.Sprintf("LIMIT %d", top)
	} else if skip > 0 {
		return fmt.Sprintf("OFFSET %d", skip)
	}
	return ""
}

// QuoteIdentifier retorna identificador sem quotes (comportamento padrão)
func (d *DefaultDialect) QuoteIdentifier(identifier string) string {
	return identifier
}

// FormatDateTime formata um time.Time de forma genérica
func (d *DefaultDialect) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// BuildCeilingFunction constrói função CEILING
func (d *DefaultDialect) BuildCeilingFunction(arg string) string {
	return fmt.Sprintf("CEILING(%s)", arg)
}

// BuildConcatFunction constrói função CONCAT
func (d *DefaultDialect) BuildConcatFunction(args []string) string {
	return fmt.Sprintf("CONCAT(%s)", strings.Join(args, ", "))
}

// BuildSubstringFunction constrói função SUBSTRING
func (d *DefaultDialect) BuildSubstringFunction(str, start, length string) string {
	return fmt.Sprintf("SUBSTRING(%s, %s, %s)", str, start, length)
}

// BuildSubstringFromFunction constrói SUBSTRING sem length
func (d *DefaultDialect) BuildSubstringFromFunction(str, start string) string {
	return fmt.Sprintf("SUBSTRING(%s, %s)", str, start)
}

// BuildDateExtractFunction constrói função de extração de data
func (d *DefaultDialect) BuildDateExtractFunction(functionName, arg string) string {
	return fmt.Sprintf("%s(%s)", strings.ToUpper(functionName), arg)
}

// BuildNowFunction constrói função NOW
func (d *DefaultDialect) BuildNowFunction() string {
	return "NOW()"
}

// SupportsFullTextSearch indica que o banco genérico não suporta full-text search
func (d *DefaultDialect) SupportsFullTextSearch() bool {
	return false
}

// BuildFullTextSearchCondition constrói condição genérica de full-text search
func (d *DefaultDialect) BuildFullTextSearchCondition(column, term string) (string, interface{}) {
	// Fallback para LIKE genérico
	return fmt.Sprintf("%s LIKE ?", column), fmt.Sprintf("%%%s%%", term)
}

// BuildFullTextPhraseCondition constrói condição genérica de full-text phrase search
func (d *DefaultDialect) BuildFullTextPhraseCondition(column, phrase string) (string, interface{}) {
	// Fallback para LIKE genérico
	return fmt.Sprintf("%s LIKE ?", column), fmt.Sprintf("%%%s%%", phrase)
}
