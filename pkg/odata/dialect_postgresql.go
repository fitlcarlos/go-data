package odata

import (
	"fmt"
	"strings"
	"time"
)

// PostgreSQLDialect implementa SQLDialect para PostgreSQL
type PostgreSQLDialect struct{}

// GetName retorna o nome do dialeto
func (d *PostgreSQLDialect) GetName() string {
	return "postgresql"
}

// SetupNodeMap configura o mapa de operadores OData para SQL
func (d *PostgreSQLDialect) SetupNodeMap() NodeMap {
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
	nodeMap["mod"] = "(%s %% %s)" // PostgreSQL usa % para módulo

	// Funções de string (case insensitive com ILIKE)
	nodeMap["contains"] = "(%s ILIKE %s)"
	nodeMap["startswith"] = "(%s ILIKE %s)"
	nodeMap["endswith"] = "(%s ILIKE %s)"
	nodeMap["length"] = "LENGTH(%s)"
	nodeMap["indexof"] = "POSITION(%s IN %s)" // PostgreSQL usa POSITION
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
	nodeMap["ceiling"] = "CEILING(%s)" // PostgreSQL usa CEILING

	// Valores especiais
	nodeMap["null"] = "NULL"

	return nodeMap
}

// SetupPrepareMap configura o mapa de preparação de valores
func (d *PostgreSQLDialect) SetupPrepareMap() PrepareMap {
	prepareMap := make(PrepareMap)
	prepareMap["contains"] = "%%%s%%"
	prepareMap["startswith"] = "%s%%"
	prepareMap["endswith"] = "%%%s"
	return prepareMap
}

// BuildLimitClause constrói cláusula LIMIT/OFFSET para PostgreSQL
func (d *PostgreSQLDialect) BuildLimitClause(top, skip int) string {
	if top > 0 && skip > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", top, skip)
	} else if top > 0 {
		return fmt.Sprintf("LIMIT %d", top)
	} else if skip > 0 {
		return fmt.Sprintf("OFFSET %d", skip)
	}
	return ""
}

// QuoteIdentifier adiciona aspas duplas para identificadores PostgreSQL
func (d *PostgreSQLDialect) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}

// FormatDateTime formata um time.Time para PostgreSQL
func (d *PostgreSQLDialect) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// BuildCeilingFunction constrói função CEILING para PostgreSQL
func (d *PostgreSQLDialect) BuildCeilingFunction(arg string) string {
	return fmt.Sprintf("CEIL(%s)", arg)
}

// BuildConcatFunction constrói função CONCAT para PostgreSQL
func (d *PostgreSQLDialect) BuildConcatFunction(args []string) string {
	return fmt.Sprintf("CONCAT(%s)", strings.Join(args, ", "))
}

// BuildSubstringFunction constrói função SUBSTRING para PostgreSQL
func (d *PostgreSQLDialect) BuildSubstringFunction(str, start, length string) string {
	return fmt.Sprintf("SUBSTRING(%s FROM %s FOR %s)", str, start, length)
}

// BuildSubstringFromFunction constrói SUBSTRING sem length para PostgreSQL
func (d *PostgreSQLDialect) BuildSubstringFromFunction(str, start string) string {
	return fmt.Sprintf("SUBSTRING(%s FROM %s)", str, start)
}

// BuildDateExtractFunction constrói função de extração de data para PostgreSQL
func (d *PostgreSQLDialect) BuildDateExtractFunction(functionName, arg string) string {
	return fmt.Sprintf("EXTRACT(%s FROM %s)", strings.ToUpper(functionName), arg)
}

// BuildNowFunction constrói função NOW para PostgreSQL
func (d *PostgreSQLDialect) BuildNowFunction() string {
	return "NOW()"
}

// SupportsFullTextSearch indica que PostgreSQL suporta full-text search
func (d *PostgreSQLDialect) SupportsFullTextSearch() bool {
	return true
}

// BuildFullTextSearchCondition constrói condição de full-text search para PostgreSQL
func (d *PostgreSQLDialect) BuildFullTextSearchCondition(column, term string) (string, interface{}) {
	// PostgreSQL full-text search
	return fmt.Sprintf("to_tsvector('english', %s) @@ plainto_tsquery('english', ?)", column), term
}

// BuildFullTextPhraseCondition constrói condição de full-text phrase search para PostgreSQL
func (d *PostgreSQLDialect) BuildFullTextPhraseCondition(column, phrase string) (string, interface{}) {
	// PostgreSQL phrase search
	return fmt.Sprintf("to_tsvector('english', %s) @@ phraseto_tsquery('english', ?)", column), phrase
}
