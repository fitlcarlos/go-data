package providers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/fitlcarlos/godata/pkg/odata"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLProvider implementa o provider para PostgreSQL
type PostgreSQLProvider struct {
	BaseProvider
}

// NewPostgreSQLProvider cria uma nova instância do provider PostgreSQL
func NewPostgreSQLProvider() *PostgreSQLProvider {
	return &PostgreSQLProvider{
		BaseProvider: BaseProvider{
			driverName: "pgx",
		},
	}
}

// Connect conecta ao banco PostgreSQL
func (p *PostgreSQLProvider) Connect(connectionString string) error {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Testa a conexão
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	p.db = db
	return nil
}

// BuildSelectQuery constrói uma query SELECT específica para PostgreSQL
func (p *PostgreSQLProvider) BuildSelectQuery(entity odata.EntityMetadata, options odata.QueryOptions) (string, []interface{}, error) {
	// SELECT clause
	selectFields := odata.GetSelectedProperties(options.Select)
	selectClause, err := p.BuildSelectClause(selectFields, entity)
	if err != nil {
		return "", nil, err
	}

	// FROM clause
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	query := fmt.Sprintf("SELECT %s FROM %s", selectClause, tableName)

	// WHERE clause
	var args []interface{}
	if options.Filter != nil && options.Filter.RawValue != "" {
		whereClause, whereArgs, err := p.BuildWhereClause(options.Filter.RawValue, entity)
		if err != nil {
			return "", nil, err
		}
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY clause
	if options.OrderBy != "" {
		orderByClause, err := p.BuildOrderByClause(options.OrderBy, entity)
		if err != nil {
			return "", nil, err
		}
		if orderByClause != "" {
			query += " ORDER BY " + orderByClause
		}
	}

	// LIMIT e OFFSET (PostgreSQL style)
	topValue := odata.GetTopValue(options.Top)
	skipValue := odata.GetSkipValue(options.Skip)
	if topValue > 0 {
		query += fmt.Sprintf(" LIMIT %d", topValue)
	}
	if skipValue > 0 {
		query += fmt.Sprintf(" OFFSET %d", skipValue)
	}

	return query, args, nil
}

// BuildInsertQuery constrói uma query INSERT específica para PostgreSQL
func (p *PostgreSQLProvider) BuildInsertQuery(entity odata.EntityMetadata, data map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var columns []string
	var placeholders []string
	var args []interface{}
	argIndex := 1

	for key, value := range data {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range entity.Properties {
			if p.Name == key {
				prop = &p
				break
			}
		}

		if prop == nil {
			continue // Ignora propriedades não encontradas
		}

		if prop.IsNavigation {
			continue // Ignora propriedades de navegação
		}

		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}

		columns = append(columns, columnName)
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
		argIndex++
	}

	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no valid columns found for insert")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return query, args, nil
}

// BuildUpdateQuery constrói uma query UPDATE específica para PostgreSQL
func (p *PostgreSQLProvider) BuildUpdateQuery(entity odata.EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var setClauses []string
	var args []interface{}
	argIndex := 1

	// Constrói as cláusulas SET
	for key, value := range data {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range entity.Properties {
			if p.Name == key {
				prop = &p
				break
			}
		}

		if prop == nil {
			continue // Ignora propriedades não encontradas
		}

		if prop.IsNavigation || prop.IsKey {
			continue // Ignora propriedades de navegação e chaves
		}

		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}

		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", columnName, argIndex))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
		argIndex++
	}

	if len(setClauses) == 0 {
		return "", nil, fmt.Errorf("no valid columns found for update")
	}

	// Constrói a cláusula WHERE baseada nas chaves
	var whereClauses []string
	for key, value := range keyValues {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range entity.Properties {
			if p.Name == key {
				prop = &p
				break
			}
		}

		if prop == nil {
			continue
		}

		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}

		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", columnName, argIndex))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
		argIndex++
	}

	if len(whereClauses) == 0 {
		return "", nil, fmt.Errorf("no valid keys found for update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s RETURNING *",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	return query, args, nil
}

// BuildDeleteQuery constrói uma query DELETE específica para PostgreSQL
func (p *PostgreSQLProvider) BuildDeleteQuery(entity odata.EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var whereClauses []string
	var args []interface{}
	argIndex := 1

	// Constrói a cláusula WHERE baseada nas chaves
	for key, value := range keyValues {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range entity.Properties {
			if p.Name == key {
				prop = &p
				break
			}
		}

		if prop == nil {
			continue
		}

		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}

		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", columnName, argIndex))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
		argIndex++
	}

	if len(whereClauses) == 0 {
		return "", nil, fmt.Errorf("no valid keys found for delete")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "))

	return query, args, nil
}

// MapGoTypeToSQL mapeia tipos Go para tipos PostgreSQL específicos
func (p *PostgreSQLProvider) MapGoTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "TEXT"
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "float32":
		return "REAL"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP"
	case "[]byte":
		return "BYTEA"
	default:
		return "TEXT"
	}
}

// FormatDateTime formata uma data/hora para PostgreSQL
func (p *PostgreSQLProvider) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// buildCondition sobrescreve o método base para usar placeholders do PostgreSQL
func (p *PostgreSQLProvider) buildCondition(expr odata.FilterExpression, prop odata.PropertyMetadata) (string, interface{}, error) {
	columnName := prop.ColumnName
	if columnName == "" {
		columnName = prop.Name
	}

	switch expr.Operator {
	case odata.FilterEq:
		return fmt.Sprintf("%s = $1", columnName), expr.Value, nil
	case odata.FilterNe:
		return fmt.Sprintf("%s != $1", columnName), expr.Value, nil
	case odata.FilterGt:
		return fmt.Sprintf("%s > $1", columnName), expr.Value, nil
	case odata.FilterGe:
		return fmt.Sprintf("%s >= $1", columnName), expr.Value, nil
	case odata.FilterLt:
		return fmt.Sprintf("%s < $1", columnName), expr.Value, nil
	case odata.FilterLe:
		return fmt.Sprintf("%s <= $1", columnName), expr.Value, nil
	case odata.FilterContains:
		return fmt.Sprintf("%s ILIKE $1", columnName), fmt.Sprintf("%%%s%%", expr.Value), nil
	case odata.FilterStartsWith:
		return fmt.Sprintf("%s ILIKE $1", columnName), fmt.Sprintf("%s%%", expr.Value), nil
	case odata.FilterEndsWith:
		return fmt.Sprintf("%s ILIKE $1", columnName), fmt.Sprintf("%%%s", expr.Value), nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
	}
}
