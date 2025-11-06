package odata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

// OracleProvider implementa o DatabaseProvider para Oracle
type OracleProvider struct {
	*BaseProvider
}

// NewOracleProvider cria um novo OracleProvider
func NewOracleProvider(connection ...*sql.DB) *OracleProvider {
	var db *sql.DB

	// Se não recebeu conexão, tenta carregar do .env
	if len(connection) == 0 || connection[0] == nil {
		log.Printf("🔍 [PROVIDER] Nenhuma conexão passada, carregando do .env...")
		config, err := LoadEnvOrDefault()
		if err != nil {
			log.Printf("Aviso: Não foi possível carregar configurações do .env: %v", err)
			return &OracleProvider{
				BaseProvider: NewBaseProvider(nil, "oracle"),
			}
		}

		// Cria conexão com base no .env
		connectionString := config.BuildConnectionString()

		// Tenta conectar se há configurações suficientes
		if config.DBUser != "" && config.DBPassword != "" {
			var err error
			db, err = sql.Open("oracle", connectionString)
			if err != nil {
				log.Printf("Erro ao conectar ao Oracle usando .env: %v", err)
				return &OracleProvider{
					BaseProvider: NewBaseProvider(nil, "oracle"),
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
				log.Printf("❌ [BANCO DE DADOS] Falha ao conectar ao Oracle: %v", err)
				db.Close()
				return &OracleProvider{
					BaseProvider: NewBaseProvider(nil, "oracle"),
				}
			}
		}
	} else {
		db = connection[0]
	}

	provider := &OracleProvider{
		BaseProvider: NewBaseProvider(db, "oracle"),
	}

	// Inicializa query builder e parsers se há conexão
	if db != nil {
		provider.InitQueryBuilder()
		provider.InitParsers()
	} else {
		log.Printf("⚠️  [BANCO DE DADOS] Oracle Provider criado SEM conexão válida - Configure DB_USER e DB_PASSWORD no .env")
	}

	return provider
}

// GetDriverName retorna o nome do driver
func (p *OracleProvider) GetDriverName() string {
	return "oracle"
}

// Connect conecta ao banco Oracle com configurações otimizadas
func (p *OracleProvider) Connect(connectionString string) error {
	db, err := sql.Open("oracle", connectionString)
	if err != nil {
		return fmt.Errorf("failed to open Oracle connection: %w", err)
	}

	// Configurações otimizadas para Oracle com timeouts mais longos
	//db.SetMaxOpenConns(25)                  // Máximo de conexões abertas
	//db.SetMaxIdleConns(5)                   // Máximo de conexões inativas
	//db.SetConnMaxLifetime(10 * time.Minute) // Aumentado para 10 minutos
	//db.SetConnMaxIdleTime(5 * time.Minute)  // Aumentado para 5 minutos

	// Testa a conexão com timeout estendido
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping Oracle: %w", err)
	}

	p.db = db

	// Inicializa o query builder e parsers para Oracle
	p.InitQueryBuilder()
	p.InitParsers()

	log.Printf("✅ Oracle connection established with extended timeout settings")

	return nil
}

// BuildSelectQueryOptimized constrói query SELECT otimizada para Oracle com proteção contra timeouts
func (p *OracleProvider) BuildSelectQueryOptimized(ctx context.Context, entity EntityMetadata, options QueryOptions) (string, []interface{}, error) {
	// Verifica se o contexto já foi cancelado antes de começar
	select {
	case <-ctx.Done():
		return "", nil, fmt.Errorf("context cancelled before building query: %w", ctx.Err())
	default:
	}

	// Constrói a query usando o método otimizado do BaseProvider
	return p.BaseProvider.BuildSelectQueryOptimized(ctx, entity, options)
}

// BuildSelectQuery constrói uma query SELECT específica para Oracle
func (p *OracleProvider) BuildSelectQuery(entity EntityMetadata, options QueryOptions) (string, []interface{}, error) {
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

	// Constrói query básica sem paginação complexa
	query := fmt.Sprintf("SELECT %s FROM %s", selectClause, tableName)
	var args []interface{}

	// WHERE clause - usa argumentos nomeados para Oracle
	if options.Filter != nil && options.Filter.Tree != nil {
		// Sanitiza a query antes de construir WHERE para evitar caracteres inválidos
		whereClause, namedArgs, err := p.buildSanitizedWhereClauseNamed(context.Background(), options.Filter.Tree, entity)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build where clause: %w", err)
		}
		if whereClause != "" {
			query += " WHERE " + whereClause
			args = namedArgs
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

	// Aplicar TOP/SKIP usando ROWNUM (Oracle 11g compatible)
	topValue := GetTopValue(options.Top)
	skipValue := GetSkipValue(options.Skip)

	if topValue > 0 || skipValue > 0 {
		if skipValue > 0 {
			// Com SKIP: precisa de subquery com ROWNUM
			outerQuery := fmt.Sprintf("SELECT * FROM (SELECT ROWNUM rn, t.* FROM (%s) t) WHERE rn > %d",
				query, skipValue)
			query = outerQuery

			if topValue > 0 {
				query += fmt.Sprintf(" AND rn <= %d", skipValue+topValue)
			}
		} else {
			// Apenas TOP: mais simples
			query = fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", query, topValue)
		}
	}

	// Sanitiza a query final para Oracle
	query = p.sanitizeOracleQuery(query)

	log.Printf("🔍 DEBUG Oracle Query: %s", query)
	log.Printf("🔍 DEBUG Oracle Args: %v", args)

	return query, args, nil
}

// buildSanitizedWhereClause constrói cláusula WHERE com sanitização para Oracle
func (p *OracleProvider) buildSanitizedWhereClause(ctx context.Context, tree *ParseNode, entity EntityMetadata) (string, []interface{}, error) {
	// Verifica timeout antes de processar
	select {
	case <-ctx.Done():
		return "", nil, fmt.Errorf("context cancelled while building where clause: %w", ctx.Err())
	default:
	}

	qb := p.GetQueryBuilder()
	whereClause, whereArgs, err := qb.BuildWhereClause(ctx, tree, entity)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build where clause: %w", err)
	}

	// Sanitiza a cláusula WHERE para Oracle
	whereClause = p.sanitizeOracleQuery(whereClause)

	return whereClause, whereArgs, nil
}

// buildSanitizedWhereClauseNamed constrói cláusula WHERE com sanitização para Oracle usando argumentos nomeados
func (p *OracleProvider) buildSanitizedWhereClauseNamed(ctx context.Context, tree *ParseNode, entity EntityMetadata) (string, []interface{}, error) {
	// Verifica timeout antes de processar
	select {
	case <-ctx.Done():
		return "", nil, fmt.Errorf("context cancelled while building where clause: %w", ctx.Err())
	default:
	}

	namedArgs := NewNamedArgs("oracle")
	qb := p.GetQueryBuilder()
	whereClause, err := qb.BuildWhereClauseNamed(ctx, tree, entity, namedArgs)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build where clause: %w", err)
	}

	// Sanitiza a cláusula WHERE para Oracle
	whereClause = p.sanitizeOracleQuery(whereClause)

	return whereClause, namedArgs.GetArgs(), nil
}

// sanitizeOracleQuery remove caracteres inválidos e normaliza a query para Oracle
func (p *OracleProvider) sanitizeOracleQuery(query string) string {
	// Remove caracteres de controle que podem causar ORA-00911
	query = strings.ReplaceAll(query, "\x00", "") // Remove NULL bytes
	query = strings.ReplaceAll(query, "\r", " ")  // Remove carriage returns
	query = strings.ReplaceAll(query, "\n", " ")  // Remove newlines
	query = strings.ReplaceAll(query, "\t", " ")  // Remove tabs

	// Remove outros caracteres de controle problemáticos
	var sanitized strings.Builder
	for _, r := range query {
		// Mantém apenas caracteres ASCII válidos e espaços
		if (r >= 32 && r <= 126) || r == ' ' {
			sanitized.WriteRune(r)
		}
	}

	query = sanitized.String()

	// Remove múltiplos espaços consecutivos
	for strings.Contains(query, "  ") {
		query = strings.ReplaceAll(query, "  ", " ")
	}

	// Remove espaços no início e fim
	query = strings.TrimSpace(query)

	// Verifica se a query não termina com ponto e vírgula (Oracle não permite em alguns contextos)
	query = strings.TrimSuffix(query, ";")

	return query
}

// BuildInsertQuery constrói uma query INSERT específica para Oracle
func (p *OracleProvider) BuildInsertQuery(entity EntityMetadata, data map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var columns []string
	var placeholders []string
	namedArgs := NewNamedArgs("oracle")

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

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		placeholder := namedArgs.AddArg(convertedValue)
		placeholders = append(placeholders, placeholder)
	}

	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no valid columns found for insert")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	// Sanitiza a query
	query = p.sanitizeOracleQuery(query)

	// Retorna o map como um slice para compatibilidade
	return query, []interface{}{namedArgs.GetArgs()}, nil
}

// BuildUpdateQuery constrói uma query UPDATE específica para Oracle
func (p *OracleProvider) BuildUpdateQuery(entity EntityMetadata, data map[string]interface{}, keyValues map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var setClauses []string
	namedArgs := NewNamedArgs("oracle")

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

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		placeholder := namedArgs.AddArg(convertedValue)
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", columnName, placeholder))
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

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		placeholder := namedArgs.AddArg(convertedValue)
		whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", columnName, placeholder))
	}

	if len(whereClauses) == 0 {
		return "", nil, fmt.Errorf("no valid keys found for update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	// Retorna o map como um slice para compatibilidade
	return query, []interface{}{namedArgs.GetArgs()}, nil
}

// BuildDeleteQuery constrói uma query DELETE específica para Oracle
func (p *OracleProvider) BuildDeleteQuery(entity EntityMetadata, keyValues map[string]interface{}) (string, []interface{}, error) {
	tableName := entity.TableName
	if tableName == "" {
		tableName = entity.Name
	}

	var whereClauses []string
	namedArgs := NewNamedArgs("oracle")

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

		// Converte o valor para o tipo apropriado
		convertedValue, err := p.ConvertValue(value, prop.Type)
		if err != nil {
			return "", nil, err
		}

		placeholder := namedArgs.AddArg(convertedValue)
		whereClauses = append(whereClauses, fmt.Sprintf("%s = %s", columnName, placeholder))
	}

	if len(whereClauses) == 0 {
		return "", nil, fmt.Errorf("no valid keys found for delete")
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "))

	// Retorna o map como um slice para compatibilidade
	return query, []interface{}{namedArgs.GetArgs()}, nil
}

// ConvertValue implementa conversão de valor para Oracle sem conversão manual
func (p *OracleProvider) ConvertValue(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch targetType {
	case "int", "int32", "int64":
		switch v := value.(type) {
		case string:
			return strconv.ParseInt(v, 10, 64)
		case int:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		default:
			return value, nil // Deixa o driver Oracle lidar com isso
		}
	case "float64", "double":
		switch v := value.(type) {
		case string:
			return strconv.ParseFloat(v, 64)
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int:
			return float64(v), nil
		case int32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return value, nil // Deixa o driver Oracle lidar com isso
		}
	case "string":
		switch v := value.(type) {
		case string:
			return v, nil
		case []byte:
			return string(v), nil
		default:
			return fmt.Sprintf("%v", value), nil
		}
	case "bool", "boolean":
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		case int:
			return v != 0, nil
		case int32:
			return v != 0, nil
		case int64:
			return v != 0, nil
		default:
			return value, nil
		}
	default:
		return value, nil
	}
}

// MapGoTypeToSQL mapeia tipos Go para tipos Oracle específicos
func (p *OracleProvider) MapGoTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR2"
	case "int", "int32":
		return "NUMBER(10)"
	case "int64":
		return "NUMBER(19)"
	case "float32":
		return "NUMBER(7,2)"
	case "float64":
		return "NUMBER(15,2)"
	case "bool":
		return "NUMBER(1)"
	case "time.Time":
		return "DATE"
	case "[]byte":
		return "BLOB"
	default:
		return "VARCHAR2"
	}
}

// FormatDateTime formata uma data/hora para Oracle
func (p *OracleProvider) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// BuildWhereClause sobrescreve o método base para usar placeholders do Oracle
func (p *OracleProvider) BuildWhereClause(filter string, metadata EntityMetadata) (string, []interface{}, error) {
	log.Printf("🔍 DEBUG Oracle BuildWhereClause called with filter: %s", filter)

	if filter == "" {
		return "", nil, nil
	}

	parser := NewODataParser()
	expressions, err := parser.ParseFilter(filter)
	if err != nil {
		log.Printf("❌ DEBUG Filter parsing error: %v", err)
		return "", nil, err
	}

	log.Printf("📊 DEBUG Parsed %d expressions", len(expressions))

	var conditions []string
	var args []interface{}

	for i, expr := range expressions {
		log.Printf("📝 DEBUG Expression %d: Property=%s, Operator=%s, Value=%v", i, expr.Property, expr.Operator, expr.Value)

		// Encontra a propriedade nos metadados
		var prop *PropertyMetadata
		for _, p := range metadata.Properties {
			if p.Name == expr.Property {
				prop = &p
				break
			}
		}

		if prop == nil {
			log.Printf("❌ DEBUG Property %s not found in entity %s", expr.Property, metadata.Name)
			return "", nil, fmt.Errorf("property %s not found in entity %s", expr.Property, metadata.Name)
		}

		condition, arg, err := p.buildOracleCondition(expr, *prop, i+1)
		if err != nil {
			log.Printf("❌ DEBUG buildOracleCondition error: %v", err)
			return "", nil, err
		}

		log.Printf("✅ DEBUG Generated condition: %s with arg: %v", condition, arg)
		conditions = append(conditions, condition)
		if arg != nil {
			args = append(args, arg)
		}
	}

	whereClause := strings.Join(conditions, " AND ")
	log.Printf("🎯 DEBUG Final WHERE clause: %s", whereClause)
	log.Printf("🎯 DEBUG Final args: %v", args)

	return whereClause, args, nil
}

// buildOracleCondition constrói uma condição individual do WHERE usando placeholders Oracle numerados
func (p *OracleProvider) buildOracleCondition(expr FilterExpression, prop PropertyMetadata, paramIndex int) (string, interface{}, error) {
	columnName := prop.ColumnName
	if columnName == "" {
		columnName = prop.Name
	}

	// Usa placeholder padrão "?" em vez dos numerados Oracle
	placeholder := "?"

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
		return fmt.Sprintf("UPPER(%s) LIKE UPPER(%s)", columnName, placeholder), fmt.Sprintf("%%%s%%", expr.Value), nil
	case FilterStartsWith:
		return fmt.Sprintf("UPPER(%s) LIKE UPPER(%s)", columnName, placeholder), fmt.Sprintf("%s%%", expr.Value), nil
	case FilterEndsWith:
		return fmt.Sprintf("UPPER(%s) LIKE UPPER(%s)", columnName, placeholder), fmt.Sprintf("%%%s", expr.Value), nil
	// Funções de string - delegam para o BaseProvider
	case FilterLength, FilterToLower, FilterToUpper, FilterTrim,
		FilterIndexOf, FilterConcat, FilterSubstring:
		return p.buildOracleStringFunctionCondition(expr, columnName, paramIndex)
	// Operadores matemáticos - tratamento específico para Oracle
	case FilterAdd, FilterSub, FilterMul, FilterDiv, FilterMod:
		return p.buildOracleMathFunctionCondition(expr, columnName, paramIndex)
	// Funções de data/hora - tratamento específico para Oracle
	case FilterYear, FilterMonth, FilterDay, FilterHour, FilterMinute, FilterSecond, FilterNow:
		return p.buildOracleDateTimeFunctionCondition(expr, columnName, paramIndex)
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
	}
}

// buildOracleStringFunctionCondition constrói condições para funções de string no Oracle
func (p *OracleProvider) buildOracleStringFunctionCondition(expr FilterExpression, columnName string, paramIndex int) (string, interface{}, error) {
	placeholder := "?"

	switch expr.Operator {
	case FilterLength:
		return fmt.Sprintf("LENGTH(%s) = %s", columnName, placeholder), expr.Value, nil
	case FilterToLower:
		return fmt.Sprintf("LOWER(%s) = %s", columnName, placeholder), expr.Value, nil
	case FilterToUpper:
		return fmt.Sprintf("UPPER(%s) = %s", columnName, placeholder), expr.Value, nil
	case FilterTrim:
		return fmt.Sprintf("TRIM(%s) = %s", columnName, placeholder), expr.Value, nil
	case FilterIndexOf:
		// Oracle usa INSTR() que retorna posição base 1, subtraímos 1 para base 0
		return fmt.Sprintf("INSTR(%s, %s) - 1", columnName, placeholder), expr.Value, nil
	case FilterConcat:
		// Oracle usa || para concatenação
		if len(expr.Arguments) == 0 {
			return "", nil, fmt.Errorf("concat function requires arguments")
		}
		concatExpr := columnName
		for range expr.Arguments {
			concatExpr += " || " + placeholder
		}
		concatExpr += " = " + placeholder
		return concatExpr, expr.Arguments, nil
	case FilterSubstring:
		// Oracle usa SUBSTR()
		if len(expr.Arguments) == 1 {
			return fmt.Sprintf("SUBSTR(%s, %s) = %s", columnName, placeholder, placeholder), expr.Arguments[0], nil
		} else if len(expr.Arguments) == 2 {
			return fmt.Sprintf("SUBSTR(%s, %s, %s) = %s", columnName, placeholder, placeholder, placeholder), expr.Arguments, nil
		}
		return "", nil, fmt.Errorf("substring function requires 1 or 2 arguments")
	default:
		return "", nil, fmt.Errorf("unsupported string function: %s", expr.Operator)
	}
}

// buildOracleMathFunctionCondition constrói condições para operadores matemáticos no Oracle
func (p *OracleProvider) buildOracleMathFunctionCondition(expr FilterExpression, columnName string, paramIndex int) (string, interface{}, error) {
	if len(expr.Arguments) != 1 {
		return "", nil, fmt.Errorf("math function %s requires exactly 1 argument", expr.Operator)
	}

	if expr.Value == nil {
		return "", nil, fmt.Errorf("math function %s requires a value to compare", expr.Operator)
	}

	placeholder1 := "?"
	placeholder2 := "?"

	// Oracle tem sintaxe específica para alguns operadores
	var mathExpression string
	switch expr.Operator {
	case FilterAdd:
		mathExpression = fmt.Sprintf("(%s + %s) = %s", columnName, placeholder1, placeholder2)
	case FilterSub:
		mathExpression = fmt.Sprintf("(%s - %s) = %s", columnName, placeholder1, placeholder2)
	case FilterMul:
		mathExpression = fmt.Sprintf("(%s * %s) = %s", columnName, placeholder1, placeholder2)
	case FilterDiv:
		mathExpression = fmt.Sprintf("(%s / %s) = %s", columnName, placeholder1, placeholder2)
	case FilterMod:
		// Oracle usa a função MOD() em vez do operador %
		mathExpression = fmt.Sprintf("MOD(%s, %s) = %s", columnName, placeholder1, placeholder2)
	default:
		return "", nil, fmt.Errorf("unsupported math operator: %s", expr.Operator)
	}

	// Retorna a expressão com os argumentos
	args := []interface{}{expr.Arguments[0], expr.Value}
	return mathExpression, args, nil
}

// buildOracleDateTimeFunctionCondition constrói condições para funções de data/hora no Oracle
func (p *OracleProvider) buildOracleDateTimeFunctionCondition(expr FilterExpression, columnName string, paramIndex int) (string, interface{}, error) {
	if expr.Value == nil && expr.Operator != FilterNow {
		return "", nil, fmt.Errorf("datetime function %s requires a value to compare", expr.Operator)
	}

	placeholder := "?"

	switch expr.Operator {
	case FilterYear:
		// Oracle usa EXTRACT(YEAR FROM date_column)
		return fmt.Sprintf("EXTRACT(YEAR FROM %s) = %s", columnName, placeholder), expr.Value, nil

	case FilterMonth:
		// Oracle usa EXTRACT(MONTH FROM date_column)
		return fmt.Sprintf("EXTRACT(MONTH FROM %s) = %s", columnName, placeholder), expr.Value, nil

	case FilterDay:
		// Oracle usa EXTRACT(DAY FROM date_column)
		return fmt.Sprintf("EXTRACT(DAY FROM %s) = %s", columnName, placeholder), expr.Value, nil

	case FilterHour:
		// Oracle usa EXTRACT(HOUR FROM timestamp_column)
		return fmt.Sprintf("EXTRACT(HOUR FROM %s) = %s", columnName, placeholder), expr.Value, nil

	case FilterMinute:
		// Oracle usa EXTRACT(MINUTE FROM timestamp_column)
		return fmt.Sprintf("EXTRACT(MINUTE FROM %s) = %s", columnName, placeholder), expr.Value, nil

	case FilterSecond:
		// Oracle usa EXTRACT(SECOND FROM timestamp_column)
		return fmt.Sprintf("EXTRACT(SECOND FROM %s) = %s", columnName, placeholder), expr.Value, nil

	case FilterNow:
		// Oracle usa SYSDATE para data/hora atual
		return fmt.Sprintf("SYSDATE = %s", placeholder), expr.Value, nil

	default:
		return "", nil, fmt.Errorf("unsupported datetime function: %s", expr.Operator)
	}
}
