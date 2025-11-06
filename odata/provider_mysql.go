package odata

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLProvider implementa o provider para MySQL
type MySQLProvider struct {
	BaseProvider
}

// NewMySQLProvider cria uma nova instância do provider MySQL
func NewMySQLProvider(connection ...*sql.DB) *MySQLProvider {
	var db *sql.DB

	// Se não recebeu conexão, tenta carregar do .env
	if len(connection) == 0 || connection[0] == nil {
		log.Printf("🔍 [PROVIDER] Nenhuma conexão passada, carregando do .env...")
		config, err := LoadEnvOrDefault()
		if err != nil {
			log.Printf("Aviso: Não foi possível carregar configurações do .env: %v", err)
			return &MySQLProvider{
				BaseProvider: BaseProvider{
					driverName: "mysql",
				},
			}
		}

		// Cria conexão com base no .env
		connectionString := config.BuildConnectionString()

		// Tenta conectar se há configurações suficientes
		if config.DBUser != "" && config.DBPassword != "" {
			var err error
			db, err = sql.Open("mysql", connectionString)
			if err != nil {
				log.Printf("Erro ao conectar ao MySQL usando .env: %v", err)
				return &MySQLProvider{
					BaseProvider: BaseProvider{
						driverName: "mysql",
					},
				}
			}

			// Configura pool de conexões
			db.SetMaxOpenConns(config.DBMaxOpenConns)
			db.SetMaxIdleConns(config.DBMaxIdleConns)

			// Se DBConnMaxLifetime for 0, usa 5 minutos como padrão para evitar conexões expiradas imediatamente
			lifetime := config.DBConnMaxLifetime
			if lifetime == 0 {
				lifetime = 5 * time.Minute
			}
			db.SetConnMaxLifetime(lifetime)

			// Se DBConnMaxIdleTime for 0, usa 5 minutos como padrão
			idleTime := config.DBConnMaxIdleTime
			if idleTime == 0 {
				idleTime = 5 * time.Minute
			}
			db.SetConnMaxIdleTime(idleTime)

			// Testa conexão (sem contexto com timeout para não cancelar a conexão)
			if err := db.Ping(); err != nil {
				log.Printf("❌ [BANCO DE DADOS] Falha ao conectar ao MySQL: %v", err)
				db.Close()
				return &MySQLProvider{
					BaseProvider: BaseProvider{
						driverName: "mysql",
					},
				}
			}
		}
	} else {
		db = connection[0]
	}

	provider := &MySQLProvider{
		BaseProvider: BaseProvider{
			driverName: "mysql",
			db:         db,
		},
	}

	// Inicializa query builder e parsers se há conexão
	if db != nil {
		provider.InitQueryBuilder()
		provider.InitParsers()
	} else {
		log.Printf("⚠️  [BANCO DE DADOS] MySQL Provider criado SEM conexão válida - Configure DB_USER e DB_PASSWORD no .env")
	}

	return provider
}

// Connect conecta ao banco MySQL
func (p *MySQLProvider) Connect(connectionString string) error {
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Testa a conexão
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL: %w", err)
	}

	p.db = db
	return nil
}

// BuildSelectQuery constrói uma query SELECT específica para MySQL
func (p *MySQLProvider) BuildSelectQuery(entity EntityMetadata, options QueryOptions) (string, []interface{}, error) {
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

	// LIMIT e OFFSET (MySQL style)
	topValue := GetTopValue(options.Top)
	skipValue := GetSkipValue(options.Skip)
	if topValue > 0 && skipValue > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", topValue, skipValue)
	} else if topValue > 0 {
		query += fmt.Sprintf(" LIMIT %d", topValue)
	} else if skipValue > 0 {
		query += fmt.Sprintf(" LIMIT 18446744073709551615 OFFSET %d", skipValue)
	}

	return query, args, nil
}

// BuildInsertQuery constrói uma query INSERT específica para MySQL
func (p *MySQLProvider) BuildInsertQuery(entity EntityMetadata, data map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var columns []string
	var placeholders []string
	var args []interface{}

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
		placeholders = append(placeholders, "?")

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
	}

	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no valid columns found for insert")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	return query, args, nil
}

// BuildUpdateQuery constrói uma query UPDATE específica para MySQL
func (p *MySQLProvider) BuildUpdateQuery(entity EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var setClauses []string
	var args []interface{}

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

		setClauses = append(setClauses, fmt.Sprintf("%s = ?", columnName))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
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

		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", columnName))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
	}

	if len(whereClauses) == 0 {
		return "", nil, fmt.Errorf("no valid keys found for update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	return query, args, nil
}

// BuildDeleteQuery constrói uma query DELETE específica para MySQL
func (p *MySQLProvider) BuildDeleteQuery(entity EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var whereClauses []string
	var args []interface{}

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

		whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", columnName))

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		args = append(args, convertedValue)
	}

	if len(whereClauses) == 0 {
		return "", nil, fmt.Errorf("no valid keys found for delete")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "))

	return query, args, nil
}

// MapGoTypeToSQL mapeia tipos Go para tipos MySQL específicos
func (p *MySQLProvider) MapGoTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR(255)"
	case "int", "int32":
		return "INT"
	case "int64":
		return "BIGINT"
	case "float32":
		return "FLOAT"
	case "float64":
		return "DOUBLE"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "DATETIME"
	case "[]byte":
		return "BLOB"
	default:
		return "VARCHAR(255)"
	}
}

// FormatDateTime formata uma data/hora para MySQL
func (p *MySQLProvider) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
