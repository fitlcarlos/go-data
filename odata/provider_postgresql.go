package odata

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLProvider implementa o provider para PostgreSQL
type PostgreSQLProvider struct {
	BaseProvider
}

// NewPostgreSQLProvider cria uma nova instância do provider PostgreSQL
func NewPostgreSQLProvider(connection ...*sql.DB) *PostgreSQLProvider {
	var db *sql.DB

	// Se não recebeu conexão, tenta carregar do .env
	if len(connection) == 0 || connection[0] == nil {
		log.Printf("🔍 [PROVIDER] Nenhuma conexão passada, carregando do .env...")
		config, err := LoadEnvOrDefault()
		if err != nil {
			log.Printf("Aviso: Não foi possível carregar configurações do .env: %v", err)
			return &PostgreSQLProvider{
				BaseProvider: BaseProvider{
					driverName: "pgx",
				},
			}
		}

		// Cria conexão com base no .env
		connectionString := config.BuildConnectionString()

		// Tenta conectar se há configurações suficientes
		if config.DBUser != "" && config.DBPassword != "" {
			var err error
			db, err = sql.Open("pgx", connectionString)
			if err != nil {
				log.Printf("Erro ao conectar ao PostgreSQL usando .env: %v", err)
				return &PostgreSQLProvider{
					BaseProvider: BaseProvider{
						driverName: "pgx",
					},
				}
			}

			// Configura pool de conexões
			db.SetMaxOpenConns(config.DBMaxOpenConns)
			db.SetMaxIdleConns(config.DBMaxIdleConns)

			// Se DBConnMaxLifetime for 0, configura para 1 hora (padrão razoável)
			lifetime := config.DBConnMaxLifetime
			if lifetime == 0 {
				lifetime = time.Hour
			}
			db.SetConnMaxLifetime(lifetime)

			// Se DBConnMaxIdleTime for 0, configura para 10 minutos (padrão razoável)
			// IMPORTANTE: Quando MaxIdleTime é 0, o Go usa 90 segundos (padrão do SO)
			idleTime := config.DBConnMaxIdleTime
			if idleTime == 0 {
				idleTime = 10 * time.Minute
			}
			db.SetConnMaxIdleTime(idleTime)

			// Testa conexão
			if err := db.Ping(); err != nil {
				log.Printf("❌ [BANCO DE DADOS] Falha ao conectar ao PostgreSQL: %v", err)
				db.Close()
				return &PostgreSQLProvider{
					BaseProvider: BaseProvider{
						driverName: "pgx",
					},
				}
			}
		}
	} else {
		db = connection[0]
	}

	provider := &PostgreSQLProvider{
		BaseProvider: BaseProvider{
			driverName: "pgx",
			db:         db,
		},
	}

	// Inicializa query builder e parsers se há conexão
	if db != nil {
		provider.InitQueryBuilder()
		provider.InitParsers()
	} else {
		log.Printf("⚠️  [BANCO DE DADOS] PostgreSQL Provider criado SEM conexão válida - Configure DB_USER e DB_PASSWORD no .env")
	}

	return provider
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
func (p *PostgreSQLProvider) BuildSelectQuery(entity EntityMetadata, options QueryOptions) (string, []interface{}, error) {
	// SELECT clause
	selectFields := GetSelectedProperties(options.Select)
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
	topValue := GetTopValue(options.Top)
	skipValue := GetSkipValue(options.Skip)
	if topValue > 0 {
		query += fmt.Sprintf(" LIMIT %d", topValue)
	}
	if skipValue > 0 {
		query += fmt.Sprintf(" OFFSET %d", skipValue)
	}

	return query, args, nil
}

// BuildInsertQuery constrói uma query INSERT específica para PostgreSQL
func (p *PostgreSQLProvider) BuildInsertQuery(entity EntityMetadata, data map[string]interface{}) (string, []interface{}, error) {
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
		var prop *PropertyMetadata
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
func (p *PostgreSQLProvider) BuildUpdateQuery(entity EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error) {
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
		var prop *PropertyMetadata
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
		var prop *PropertyMetadata
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
func (p *PostgreSQLProvider) BuildDeleteQuery(entity EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error) {
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
		var prop *PropertyMetadata
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

// BuildWhereClause constrói a cláusula WHERE específica para PostgreSQL (usa $1, $2, etc.)
func (p *PostgreSQLProvider) BuildWhereClause(filter string, metadata EntityMetadata) (string, []interface{}, error) {
	if filter == "" {
		return "", nil, nil
	}

	parser := NewODataParser()
	expressions, err := parser.ParseFilter(filter)
	if err != nil {
		return "", nil, err
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	for _, expr := range expressions {
		// Encontra a propriedade nos metadados
		var prop *PropertyMetadata
		for _, pm := range metadata.Properties {
			if pm.Name == expr.Property {
				prop = &pm
				break
			}
		}

		if prop == nil {
			return "", nil, fmt.Errorf("property %s not found in entity %s", expr.Property, metadata.Name)
		}

		condition, arg, err := p.buildConditionWithIndex(expr, *prop, &argIndex)
		if err != nil {
			return "", nil, err
		}

		conditions = append(conditions, condition)
		if arg != nil {
			if argSlice, ok := arg.([]interface{}); ok {
				args = append(args, argSlice...)
			} else {
				args = append(args, arg)
			}
		}
	}

	whereClause := strings.Join(conditions, " AND ")
	return whereClause, args, nil
}

// buildConditionWithIndex constrói condição individual usando placeholders PostgreSQL ($1, $2...)
func (p *PostgreSQLProvider) buildConditionWithIndex(expr FilterExpression, prop PropertyMetadata, argIndex *int) (string, interface{}, error) {
	columnName := prop.ColumnName
	if columnName == "" {
		columnName = prop.Name
	}

	placeholder := fmt.Sprintf("$%d", *argIndex)
	*argIndex++

	switch expr.Operator {
	case FilterEq:
		return fmt.Sprintf("%s = %s", columnName, placeholder), expr.Value, nil
	case FilterNe:
		return fmt.Sprintf("%s != %s", columnName, placeholder), expr.Value, nil
	case FilterGt:
		return fmt.Sprintf("%s > %s", columnName, placeholder), expr.Value, nil
	case FilterGe:
		return fmt.Sprintf("%s >= %s", columnName, placeholder), expr.Value, nil
	case FilterLt:
		return fmt.Sprintf("%s < %s", columnName, placeholder), expr.Value, nil
	case FilterLe:
		return fmt.Sprintf("%s <= %s", columnName, placeholder), expr.Value, nil
	case FilterContains:
		return fmt.Sprintf("%s ILIKE %s", columnName, placeholder), fmt.Sprintf("%%%s%%", expr.Value), nil
	case FilterStartsWith:
		return fmt.Sprintf("%s ILIKE %s", columnName, placeholder), fmt.Sprintf("%s%%", expr.Value), nil
	case FilterEndsWith:
		return fmt.Sprintf("%s ILIKE %s", columnName, placeholder), fmt.Sprintf("%%%s", expr.Value), nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
	}
}

// buildCondition sobrescreve o método base para usar placeholders do PostgreSQL (mantido para compatibilidade)
func (p *PostgreSQLProvider) buildCondition(expr FilterExpression, prop PropertyMetadata) (string, interface{}, error) {
	columnName := prop.ColumnName
	if columnName == "" {
		columnName = prop.Name
	}

	switch expr.Operator {
	case FilterEq:
		return fmt.Sprintf("%s = $1", columnName), expr.Value, nil
	case FilterNe:
		return fmt.Sprintf("%s != $1", columnName), expr.Value, nil
	case FilterGt:
		return fmt.Sprintf("%s > $1", columnName), expr.Value, nil
	case FilterGe:
		return fmt.Sprintf("%s >= $1", columnName), expr.Value, nil
	case FilterLt:
		return fmt.Sprintf("%s < $1", columnName), expr.Value, nil
	case FilterLe:
		return fmt.Sprintf("%s <= $1", columnName), expr.Value, nil
	case FilterContains:
		return fmt.Sprintf("%s ILIKE $1", columnName), fmt.Sprintf("%%%s%%", expr.Value), nil
	case FilterStartsWith:
		return fmt.Sprintf("%s ILIKE $1", columnName), fmt.Sprintf("%s%%", expr.Value), nil
	case FilterEndsWith:
		return fmt.Sprintf("%s ILIKE $1", columnName), fmt.Sprintf("%%%s", expr.Value), nil
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
	}
}
