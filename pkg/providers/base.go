package providers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fitlcarlos/godata/pkg/odata"
)

// BaseProvider implementa funcionalidades comuns a todos os providers
type BaseProvider struct {
	db            *sql.DB
	driverName    string
	queryBuilder  *odata.QueryBuilder
	computeParser *odata.ComputeParser
	searchParser  *odata.SearchParser
}

// NewBaseProvider cria um novo BaseProvider
func NewBaseProvider(connection *sql.DB, driverName string) *BaseProvider {
	return &BaseProvider{
		db:         connection,
		driverName: driverName,
	}
}

// InitQueryBuilder inicializa o query builder
func (p *BaseProvider) InitQueryBuilder() {
	if p.queryBuilder == nil {
		p.queryBuilder = odata.NewQueryBuilder(p.driverName)
		if p.queryBuilder == nil {
			panic("Failed to initialize QueryBuilder")
		}
	}
}

// GetQueryBuilder retorna o query builder, inicializando se necessário
func (p *BaseProvider) GetQueryBuilder() *odata.QueryBuilder {
	if p.queryBuilder == nil {
		p.InitQueryBuilder()
	}
	return p.queryBuilder
}

// InitParsers inicializa os parsers de compute e search
func (p *BaseProvider) InitParsers() {
	if p.computeParser == nil {
		p.computeParser = odata.NewComputeParser()
	}
	if p.searchParser == nil {
		p.searchParser = odata.NewSearchParser()
	}
}

// GetConnection retorna a conexão com o banco
func (p *BaseProvider) GetConnection() *sql.DB {
	return p.db
}

// GetDriverName retorna o nome do driver
func (p *BaseProvider) GetDriverName() string {
	return p.driverName
}

// Close fecha a conexão com o banco
func (p *BaseProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// BuildSelectQueryOptimized constrói query SELECT otimizada usando o novo query builder
func (p *BaseProvider) BuildSelectQueryOptimized(ctx context.Context, metadata odata.EntityMetadata, options odata.QueryOptions) (string, []interface{}, error) {
	p.InitQueryBuilder()
	p.InitParsers()

	// Constrói a query base
	var query strings.Builder
	var args []interface{}

	// SELECT clause - inclui campos computados se houver
	selectClause, computeArgs, err := p.buildSelectWithCompute(ctx, metadata, options)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build select clause: %w", err)
	}
	query.WriteString("SELECT ")
	query.WriteString(selectClause)
	args = append(args, computeArgs...)

	// FROM clause
	tableName := metadata.TableName
	if tableName == "" {
		tableName = metadata.Name
	}
	query.WriteString(" FROM ")
	query.WriteString(tableName)

	// WHERE clause - combina filtro e busca
	whereClause, whereArgs, err := p.buildWhereWithSearch(ctx, metadata, options)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build where clause: %w", err)
	}
	if whereClause != "" {
		query.WriteString(" WHERE ")
		query.WriteString(whereClause)
		args = append(args, whereArgs...)
	}

	// ORDER BY clause
	if options.OrderBy != "" {
		orderByClause, err := p.BuildOrderByClause(options.OrderBy, metadata)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build order by clause: %w", err)
		}
		if orderByClause != "" {
			query.WriteString(" ORDER BY ")
			query.WriteString(orderByClause)
		}
	}

	// LIMIT/OFFSET clause
	topValue := odata.GetTopValue(options.Top)
	skipValue := odata.GetSkipValue(options.Skip)
	qb := p.GetQueryBuilder()
	limitClause := qb.BuildLimitClause(topValue, skipValue)
	if limitClause != "" {
		query.WriteString(" ")
		query.WriteString(limitClause)
	}

	return query.String(), args, nil
}

// buildSelectWithCompute constrói cláusula SELECT incluindo campos computados
func (p *BaseProvider) buildSelectWithCompute(ctx context.Context, metadata odata.EntityMetadata, options odata.QueryOptions) (string, []interface{}, error) {
	// Campos regulares
	selectFields := odata.GetSelectedProperties(options.Select)
	qb := p.GetQueryBuilder()
	baseSelect := qb.BuildSelectClause(metadata, selectFields)

	// Campos computados
	if options.Compute != nil {
		computeSQL, computeArgs, err := qb.BuildComputeSQL(ctx, options.Compute, metadata)
		if err != nil {
			return "", nil, err
		}

		if computeSQL != "" {
			if baseSelect != "" {
				baseSelect += ", " + computeSQL
			} else {
				baseSelect = computeSQL
			}
			return baseSelect, computeArgs, nil
		}
	}

	return baseSelect, nil, nil
}

// buildWhereWithSearch constrói cláusula WHERE combinando filtro e busca
func (p *BaseProvider) buildWhereWithSearch(ctx context.Context, metadata odata.EntityMetadata, options odata.QueryOptions) (string, []interface{}, error) {
	var filterSQL string
	var filterArgs []interface{}
	var searchSQL string
	var searchArgs []interface{}
	var err error

	qb := p.GetQueryBuilder()

	// Processa filtro se presente
	if options.Filter != nil && options.Filter.Tree != nil {
		tree := options.Filter.Tree

		filterSQL, filterArgs, err = qb.BuildWhereClause(ctx, tree, metadata)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build filter clause: %w", err)
		}
	}

	// Processa busca se presente
	if options.Search != nil {
		searchSQL, searchArgs, err = qb.BuildSearchSQL(ctx, options.Search, metadata)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build search clause: %w", err)
		}
	}

	// Combina filtro e busca
	return qb.CombineSearchWithFilter(ctx, searchSQL, filterSQL, searchArgs, filterArgs)
}

// BuildWhereClause constrói a cláusula WHERE baseada no filtro OData (método legado)
func (p *BaseProvider) BuildWhereClause(filter string, metadata odata.EntityMetadata) (string, []interface{}, error) {
	if filter == "" {
		return "", nil, nil
	}

	parser := odata.NewODataParser()
	expressions, err := parser.ParseFilter(filter)
	if err != nil {
		return "", nil, err
	}

	var conditions []string
	var args []interface{}

	for _, expr := range expressions {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range metadata.Properties {
			if p.Name == expr.Property {
				prop = &p
				break
			}
		}

		if prop == nil {
			return "", nil, fmt.Errorf("property %s not found in entity %s", expr.Property, metadata.Name)
		}

		condition, arg, err := p.buildCondition(expr, *prop)
		if err != nil {
			return "", nil, err
		}

		conditions = append(conditions, condition)
		if arg != nil {
			// Verifica se arg é um slice de interface{} (para múltiplos argumentos)
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

// buildCondition constrói uma condição individual do WHERE
func (p *BaseProvider) buildCondition(expr odata.FilterExpression, prop odata.PropertyMetadata) (string, interface{}, error) {
	columnName := prop.ColumnName
	if columnName == "" {
		columnName = prop.Name
	}

	switch expr.Operator {
	case odata.FilterEq:
		return fmt.Sprintf("%s = ?", columnName), expr.Value, nil
	case odata.FilterNe:
		return fmt.Sprintf("%s != ?", columnName), expr.Value, nil
	case odata.FilterGt:
		return fmt.Sprintf("%s > ?", columnName), expr.Value, nil
	case odata.FilterGe:
		return fmt.Sprintf("%s >= ?", columnName), expr.Value, nil
	case odata.FilterLt:
		return fmt.Sprintf("%s < ?", columnName), expr.Value, nil
	case odata.FilterLe:
		return fmt.Sprintf("%s <= ?", columnName), expr.Value, nil
	case odata.FilterContains:
		return fmt.Sprintf("%s LIKE ?", columnName), fmt.Sprintf("%%%s%%", expr.Value), nil
	case odata.FilterStartsWith:
		return fmt.Sprintf("%s LIKE ?", columnName), fmt.Sprintf("%s%%", expr.Value), nil
	case odata.FilterEndsWith:
		return fmt.Sprintf("%s LIKE ?", columnName), fmt.Sprintf("%%%s", expr.Value), nil
	// Funções de string - estas são tratadas de forma especial
	case odata.FilterLength, odata.FilterToLower, odata.FilterToUpper, odata.FilterTrim,
		odata.FilterIndexOf, odata.FilterConcat, odata.FilterSubstring:
		return p.buildStringFunctionCondition(expr, columnName)
	// Operadores matemáticos - tratados de forma especial
	case odata.FilterAdd, odata.FilterSub, odata.FilterMul, odata.FilterDiv, odata.FilterMod:
		return p.buildMathFunctionCondition(expr, columnName)
	// Funções de data/hora - tratadas de forma especial
	case odata.FilterYear, odata.FilterMonth, odata.FilterDay, odata.FilterHour, odata.FilterMinute, odata.FilterSecond, odata.FilterNow:
		return p.buildDateTimeFunctionCondition(expr, columnName)
	default:
		return "", nil, fmt.Errorf("unsupported operator: %s", expr.Operator)
	}
}

// BuildOrderByClause constrói a cláusula ORDER BY baseada no orderBy OData
func (p *BaseProvider) BuildOrderByClause(orderBy string, metadata odata.EntityMetadata) (string, error) {
	if orderBy == "" {
		return "", nil
	}

	parser := odata.NewODataParser()
	expressions, err := parser.ParseOrderBy(orderBy)
	if err != nil {
		return "", err
	}

	var orderClauses []string

	for _, expr := range expressions {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range metadata.Properties {
			if strings.EqualFold(p.Name, expr.Property) {
				prop = &p
				break
			}
		}

		if prop == nil {
			return "", fmt.Errorf("property %s not found in entity %s", expr.Property, metadata.Name)
		}

		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}

		direction := "ASC"
		if expr.Direction == odata.OrderDesc {
			direction = "DESC"
		}

		orderClauses = append(orderClauses, fmt.Sprintf("%s %s", columnName, direction))
	}

	return strings.Join(orderClauses, ", "), nil
}

// buildStringFunctionCondition constrói condições para funções de string
func (p *BaseProvider) buildStringFunctionCondition(expr odata.FilterExpression, columnName string) (string, interface{}, error) {
	switch expr.Operator {
	case odata.FilterLength:
		// length(Property) eq value
		if expr.Value == nil {
			return "", nil, fmt.Errorf("length function requires a value to compare")
		}
		return fmt.Sprintf("LENGTH(%s) = ?", columnName), expr.Value, nil

	case odata.FilterToLower:
		// tolower(Property) eq value
		if expr.Value == nil {
			return "", nil, fmt.Errorf("tolower function requires a value to compare")
		}
		return fmt.Sprintf("LOWER(%s) = ?", columnName), expr.Value, nil

	case odata.FilterToUpper:
		// toupper(Property) eq value
		if expr.Value == nil {
			return "", nil, fmt.Errorf("toupper function requires a value to compare")
		}
		return fmt.Sprintf("UPPER(%s) = ?", columnName), expr.Value, nil

	case odata.FilterTrim:
		// trim(Property) eq value
		if expr.Value == nil {
			return "", nil, fmt.Errorf("trim function requires a value to compare")
		}
		return fmt.Sprintf("TRIM(%s) = ?", columnName), expr.Value, nil

	case odata.FilterIndexOf:
		// indexof(Property, substring) eq position
		// expr.Value contém a substring a procurar
		// A comparação seria feita em uma expressão maior como: indexof(Name, 'test') eq 0
		return fmt.Sprintf("POSITION(? IN %s) - 1", columnName), expr.Value, nil

	case odata.FilterConcat:
		// concat(Property, arg1, arg2, ...) eq value
		if len(expr.Arguments) == 0 {
			return "", nil, fmt.Errorf("concat function requires arguments")
		}
		if expr.Value == nil {
			return "", nil, fmt.Errorf("concat function requires a value to compare")
		}

		// Constrói a expressão CONCAT
		concatExpr := fmt.Sprintf("CONCAT(%s", columnName)
		var args []interface{}
		for _, arg := range expr.Arguments {
			concatExpr += ", ?"
			args = append(args, arg)
		}
		concatExpr += ") = ?"
		args = append(args, expr.Value)
		return concatExpr, args, nil

	case odata.FilterSubstring:
		// substring(Property, start[, length]) eq value
		if len(expr.Arguments) == 0 {
			return "", nil, fmt.Errorf("substring function requires arguments")
		}
		if expr.Value == nil {
			return "", nil, fmt.Errorf("substring function requires a value to compare")
		}

		var args []interface{}
		var substringExpr string

		if len(expr.Arguments) == 1 {
			// substring(string, start) - do start até o final
			substringExpr = fmt.Sprintf("SUBSTRING(%s, ?) = ?", columnName)
			args = []interface{}{expr.Arguments[0], expr.Value}
		} else if len(expr.Arguments) == 2 {
			// substring(string, start, length)
			substringExpr = fmt.Sprintf("SUBSTRING(%s, ?, ?) = ?", columnName)
			args = []interface{}{expr.Arguments[0], expr.Arguments[1], expr.Value}
		} else {
			return "", nil, fmt.Errorf("substring function requires 1 or 2 arguments")
		}

		return substringExpr, args, nil

	default:
		return "", nil, fmt.Errorf("unsupported string function: %s", expr.Operator)
	}
}

// buildMathFunctionCondition constrói condições para operadores matemáticos
func (p *BaseProvider) buildMathFunctionCondition(expr odata.FilterExpression, columnName string) (string, interface{}, error) {
	if len(expr.Arguments) != 1 {
		return "", nil, fmt.Errorf("math function %s requires exactly 1 argument", expr.Operator)
	}

	if expr.Value == nil {
		return "", nil, fmt.Errorf("math function %s requires a value to compare", expr.Operator)
	}

	// Converte o argumento para o tipo apropriado
	var sqlOperator string
	switch expr.Operator {
	case odata.FilterAdd:
		sqlOperator = "+"
	case odata.FilterSub:
		sqlOperator = "-"
	case odata.FilterMul:
		sqlOperator = "*"
	case odata.FilterDiv:
		sqlOperator = "/"
	case odata.FilterMod:
		sqlOperator = "%"
	default:
		return "", nil, fmt.Errorf("unsupported math operator: %s", expr.Operator)
	}

	// Constrói a expressão SQL: (column + value) = comparisionValue
	mathExpression := fmt.Sprintf("(%s %s ?) = ?", columnName, sqlOperator)

	// Retorna slice com argumentos: [argument, comparisonValue]
	args := []interface{}{expr.Arguments[0], expr.Value}
	return mathExpression, args, nil
}

// buildDateTimeFunctionCondition constrói condições para funções de data/hora
func (p *BaseProvider) buildDateTimeFunctionCondition(expr odata.FilterExpression, columnName string) (string, interface{}, error) {
	if expr.Value == nil && expr.Operator != odata.FilterNow {
		return "", nil, fmt.Errorf("datetime function %s requires a value to compare", expr.Operator)
	}

	switch expr.Operator {
	case odata.FilterYear:
		// year(DateField) eq 2023
		return fmt.Sprintf("EXTRACT(YEAR FROM %s) = ?", columnName), expr.Value, nil

	case odata.FilterMonth:
		// month(DateField) eq 12
		return fmt.Sprintf("EXTRACT(MONTH FROM %s) = ?", columnName), expr.Value, nil

	case odata.FilterDay:
		// day(DateField) eq 25
		return fmt.Sprintf("EXTRACT(DAY FROM %s) = ?", columnName), expr.Value, nil

	case odata.FilterHour:
		// hour(DateTimeField) eq 14
		return fmt.Sprintf("EXTRACT(HOUR FROM %s) = ?", columnName), expr.Value, nil

	case odata.FilterMinute:
		// minute(DateTimeField) eq 30
		return fmt.Sprintf("EXTRACT(MINUTE FROM %s) = ?", columnName), expr.Value, nil

	case odata.FilterSecond:
		// second(DateTimeField) eq 45
		return fmt.Sprintf("EXTRACT(SECOND FROM %s) = ?", columnName), expr.Value, nil

	case odata.FilterNow:
		// now() eq '2023-12-25T10:30:00'
		return "CURRENT_TIMESTAMP = ?", expr.Value, nil

	default:
		return "", nil, fmt.Errorf("unsupported datetime function: %s", expr.Operator)
	}
}

// BuildSelectClause constrói a cláusula SELECT baseada no select OData
func (p *BaseProvider) BuildSelectClause(selectFields []string, metadata odata.EntityMetadata) (string, error) {
	if len(selectFields) == 0 {
		// Seleciona todos os campos não-navegação
		var columns []string
		for _, prop := range metadata.Properties {
			if !prop.IsNavigation {
				columnName := prop.ColumnName
				if columnName == "" {
					columnName = prop.Name
				}
				columns = append(columns, columnName)
			}
		}
		return strings.Join(columns, ", "), nil
	}

	var columns []string
	for _, field := range selectFields {
		// Encontra a propriedade nos metadados
		var prop *odata.PropertyMetadata
		for _, p := range metadata.Properties {
			if p.Name == field {
				prop = &p
				break
			}
		}

		if prop == nil {
			return "", fmt.Errorf("property %s not found in entity %s", field, metadata.Name)
		}

		if prop.IsNavigation {
			return "", fmt.Errorf("navigation property %s cannot be selected directly", field)
		}

		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}

		columns = append(columns, columnName)
	}

	return strings.Join(columns, ", "), nil
}

// BuildLimitClause constrói a cláusula LIMIT baseada no skip e top OData
func (p *BaseProvider) BuildLimitClause(skip, top int) string {
	if skip > 0 && top > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", top, skip)
	} else if top > 0 {
		return fmt.Sprintf("LIMIT %d", top)
	} else if skip > 0 {
		return fmt.Sprintf("OFFSET %d", skip)
	}
	return ""
}

// MapGoTypeToSQL mapeia tipos Go para tipos SQL genéricos
func (p *BaseProvider) MapGoTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR"
	case "int", "int32", "int64":
		return "INTEGER"
	case "float32", "float64":
		return "DECIMAL"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		return "TIMESTAMP"
	case "[]byte":
		return "BLOB"
	default:
		return "VARCHAR"
	}
}

// FormatDateTime formata uma data/hora para SQL
func (p *BaseProvider) FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// ConvertValue converte um valor para o tipo apropriado
func (p *BaseProvider) ConvertValue(value interface{}, targetType string) (interface{}, error) {
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
		default:
			return nil, fmt.Errorf("cannot convert %T to int", value)
		}
	case "float32", "float64":
		switch v := value.(type) {
		case string:
			return strconv.ParseFloat(v, 64)
		case float32:
			return float64(v), nil
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case int64:
			return float64(v), nil
		default:
			return nil, fmt.Errorf("cannot convert %T to float", value)
		}
	case "bool":
		switch v := value.(type) {
		case string:
			return strconv.ParseBool(v)
		case bool:
			return v, nil
		case int:
			return v != 0, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to bool", value)
		}
	case "time.Time":
		switch v := value.(type) {
		case string:
			return time.Parse("2006-01-02T15:04:05", v)
		case time.Time:
			return v, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to time.Time", value)
		}
	case "string":
		return fmt.Sprintf("%v", value), nil
	default:
		return value, nil
	}
}
