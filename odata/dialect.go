package odata

import (
	"strings"
	"time"
)

// SQLDialect define a interface para dialetos SQL específicos de cada provider
type SQLDialect interface {
	// GetName retorna o nome do dialeto (mysql, postgresql, oracle)
	GetName() string

	// SetupNodeMap configura o mapa de operadores OData para SQL
	SetupNodeMap() NodeMap

	// SetupPrepareMap configura o mapa de preparação de valores
	SetupPrepareMap() PrepareMap

	// BuildLimitClause constrói cláusula LIMIT/OFFSET (ou equivalente)
	BuildLimitClause(top, skip int) string

	// QuoteIdentifier adiciona quotes apropriados para identificadores
	QuoteIdentifier(identifier string) string

	// FormatDateTime formata um time.Time para o formato do banco
	FormatDateTime(t time.Time) string

	// BuildCeilingFunction constrói função CEILING/CEIL
	BuildCeilingFunction(arg string) string

	// BuildConcatFunction constrói função de concatenação
	BuildConcatFunction(args []string) string

	// BuildSubstringFunction constrói função SUBSTRING/SUBSTR
	BuildSubstringFunction(str, start, length string) string

	// BuildSubstringFromFunction constrói SUBSTRING sem length
	BuildSubstringFromFunction(str, start string) string

	// BuildDateExtractFunction constrói função de extração de data (YEAR, MONTH, etc)
	BuildDateExtractFunction(functionName, arg string) string

	// BuildNowFunction constrói função NOW/SYSDATE
	BuildNowFunction() string

	// SupportsFullTextSearch indica se o banco suporta full-text search
	SupportsFullTextSearch() bool

	// BuildFullTextSearchCondition constrói condição de full-text search
	BuildFullTextSearchCondition(column, term string) (string, interface{})

	// BuildFullTextPhraseCondition constrói condição de full-text phrase search
	BuildFullTextPhraseCondition(column, phrase string) (string, interface{})
}

// GetDialect retorna a implementação de dialect apropriada
func GetDialect(name string) SQLDialect {
	name = strings.ToLower(name)
	switch name {
	case "mysql":
		return &MySQLDialect{}
	case "postgresql", "postgres", "pgx":
		return &PostgreSQLDialect{}
	case "oracle":
		return &OracleDialect{}
	default:
		return &DefaultDialect{}
	}
}
