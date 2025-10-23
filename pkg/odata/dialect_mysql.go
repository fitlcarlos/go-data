package odata

import (
	"fmt"
	"strings"
	"time"
)

// MySQLDialect implementa SQLDialect para MySQL
type MySQLDialect struct{}

// GetName retorna o nome do dialeto
func (d *MySQLDialect) GetName() string {
	return "mysql"
}

// SetupNodeMap configura o mapa de operadores OData para SQL
func (d *MySQLDialect) SetupNodeMap() NodeMap {
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
func (d *MySQLDialect) SetupPrepareMap() PrepareMap {
	prepareMap := make(PrepareMap)
	prepareMap["contains"] = "%%%s%%"
	prepareMap["startswith"] = "%s%%"
	prepareMap["endswith"] = "%%%s"
	return prepareMap
}

// BuildLimitClause constrói cláusula LIMIT/OFFSET para MySQL
func (d *MySQLDialect) BuildLimitClause(top, skip int) string {
	if top > 0 && skip > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", top, skip)
	} else if top > 0 {
		return fmt.Sprintf("LIMIT %d", top)
	} else if skip > 0 {
		return fmt.Sprintf("OFFSET %d", skip)
	}
	return ""
}

// QuoteIdentifier adiciona backticks para identificadores MySQL
func (d *MySQLDialect) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf("`%s`", identifier)
}

// FormatDateTime formata um time.Time para MySQL
func (d *MySQLDialect) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// BuildCeilingFunction constrói função CEILING para MySQL
func (d *MySQLDialect) BuildCeilingFunction(arg string) string {
	return fmt.Sprintf("CEILING(%s)", arg)
}

// BuildConcatFunction constrói função CONCAT para MySQL
func (d *MySQLDialect) BuildConcatFunction(args []string) string {
	return fmt.Sprintf("CONCAT(%s)", strings.Join(args, ", "))
}

// BuildSubstringFunction constrói função SUBSTRING para MySQL
func (d *MySQLDialect) BuildSubstringFunction(str, start, length string) string {
	return fmt.Sprintf("SUBSTRING(%s, %s, %s)", str, start, length)
}

// BuildSubstringFromFunction constrói SUBSTRING sem length para MySQL
func (d *MySQLDialect) BuildSubstringFromFunction(str, start string) string {
	return fmt.Sprintf("SUBSTRING(%s, %s)", str, start)
}

// BuildDateExtractFunction constrói função de extração de data para MySQL
func (d *MySQLDialect) BuildDateExtractFunction(functionName, arg string) string {
	return fmt.Sprintf("%s(%s)", strings.ToUpper(functionName), arg)
}

// BuildNowFunction constrói função NOW para MySQL
func (d *MySQLDialect) BuildNowFunction() string {
	return "NOW()"
}

// SupportsFullTextSearch indica que MySQL suporta full-text search
func (d *MySQLDialect) SupportsFullTextSearch() bool {
	return true
}

// BuildFullTextSearchCondition constrói condição de full-text search para MySQL
func (d *MySQLDialect) BuildFullTextSearchCondition(column, term string) (string, interface{}) {
	// MySQL FULLTEXT search
	return fmt.Sprintf("MATCH(%s) AGAINST(? IN BOOLEAN MODE)", column), term
}

// BuildFullTextPhraseCondition constrói condição de full-text phrase search para MySQL
func (d *MySQLDialect) BuildFullTextPhraseCondition(column, phrase string) (string, interface{}) {
	// MySQL phrase search
	return fmt.Sprintf("MATCH(%s) AGAINST(? IN BOOLEAN MODE)", column), fmt.Sprintf(`"%s"`, phrase)
}
