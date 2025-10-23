package odata

import (
	"fmt"
	"strings"
	"time"
)

// OracleDialect implementa SQLDialect para Oracle
type OracleDialect struct{}

// GetName retorna o nome do dialeto
func (d *OracleDialect) GetName() string {
	return "oracle"
}

// SetupNodeMap configura o mapa de operadores OData para SQL
func (d *OracleDialect) SetupNodeMap() NodeMap {
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
	nodeMap["mod"] = "MOD(%s, %s)" // Oracle usa MOD como função

	// Funções de string
	nodeMap["contains"] = "(%s LIKE %s)"
	nodeMap["startswith"] = "(%s LIKE %s)"
	nodeMap["endswith"] = "(%s LIKE %s)"
	nodeMap["length"] = "LENGTH(%s)"
	nodeMap["indexof"] = "INSTR(%s, %s)"        // Oracle usa INSTR
	nodeMap["substring"] = "SUBSTR(%s, %s, %s)" // Oracle usa SUBSTR
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
	nodeMap["now"] = "SYSDATE" // Oracle usa SYSDATE
	nodeMap["date"] = "DATE(%s)"
	nodeMap["time"] = "TIME(%s)"

	// Funções matemáticas
	nodeMap["round"] = "ROUND(%s)"
	nodeMap["floor"] = "FLOOR(%s)"
	nodeMap["ceiling"] = "CEIL(%s)" // Oracle usa CEIL

	// Valores especiais
	nodeMap["null"] = "NULL"

	return nodeMap
}

// SetupPrepareMap configura o mapa de preparação de valores
func (d *OracleDialect) SetupPrepareMap() PrepareMap {
	prepareMap := make(PrepareMap)
	prepareMap["contains"] = "%%%s%%"
	prepareMap["startswith"] = "%s%%"
	prepareMap["endswith"] = "%%%s"
	return prepareMap
}

// BuildLimitClause constrói cláusula OFFSET/FETCH para Oracle
func (d *OracleDialect) BuildLimitClause(top, skip int) string {
	if top > 0 && skip > 0 {
		return fmt.Sprintf("OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", skip, top)
	} else if top > 0 {
		return fmt.Sprintf("FETCH NEXT %d ROWS ONLY", top)
	} else if skip > 0 {
		return fmt.Sprintf("OFFSET %d ROWS", skip)
	}
	return ""
}

// QuoteIdentifier adiciona aspas duplas para identificadores Oracle
func (d *OracleDialect) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}

// FormatDateTime formata um time.Time para Oracle
func (d *OracleDialect) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// BuildCeilingFunction constrói função CEIL para Oracle
func (d *OracleDialect) BuildCeilingFunction(arg string) string {
	return fmt.Sprintf("CEIL(%s)", arg)
}

// BuildConcatFunction constrói concatenação para Oracle
func (d *OracleDialect) BuildConcatFunction(args []string) string {
	// Oracle usa || para concatenação
	return strings.Join(args, " || ")
}

// BuildSubstringFunction constrói função SUBSTR para Oracle
func (d *OracleDialect) BuildSubstringFunction(str, start, length string) string {
	return fmt.Sprintf("SUBSTR(%s, %s, %s)", str, start, length)
}

// BuildSubstringFromFunction constrói SUBSTR sem length para Oracle
func (d *OracleDialect) BuildSubstringFromFunction(str, start string) string {
	return fmt.Sprintf("SUBSTR(%s, %s)", str, start)
}

// BuildDateExtractFunction constrói função de extração de data para Oracle
func (d *OracleDialect) BuildDateExtractFunction(functionName, arg string) string {
	return fmt.Sprintf("EXTRACT(%s FROM %s)", strings.ToUpper(functionName), arg)
}

// BuildNowFunction constrói função SYSDATE para Oracle
func (d *OracleDialect) BuildNowFunction() string {
	return "SYSDATE"
}

// SupportsFullTextSearch indica que Oracle suporta full-text search
func (d *OracleDialect) SupportsFullTextSearch() bool {
	return true
}

// BuildFullTextSearchCondition constrói condição de full-text search para Oracle
func (d *OracleDialect) BuildFullTextSearchCondition(column, term string) (string, interface{}) {
	// Oracle Text search
	return fmt.Sprintf("CONTAINS(%s, ?) > 0", column), term
}

// BuildFullTextPhraseCondition constrói condição de full-text phrase search para Oracle
func (d *OracleDialect) BuildFullTextPhraseCondition(column, phrase string) (string, interface{}) {
	// Oracle phrase search
	return fmt.Sprintf("CONTAINS(%s, ?) > 0", column), fmt.Sprintf(`"%s"`, phrase)
}
