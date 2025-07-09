package odata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// NodeMap mapeia operadores OData para SQL
type NodeMap map[string]string

// PrepareMap mapeia funções para preparação de valores
type PrepareMap map[string]string

// NamedArgs gerencia argumentos nomeados para queries SQL usando sql.Named
type NamedArgs struct {
	args    []interface{}
	counter int
	dialect string
}

// NewNamedArgs cria uma nova instância de NamedArgs
func NewNamedArgs(dialect string) *NamedArgs {
	return &NamedArgs{
		args:    make([]interface{}, 0),
		counter: 0,
		dialect: strings.ToLower(dialect),
	}
}

// AddArg adiciona um argumento usando sql.Named e retorna o placeholder apropriado
func (na *NamedArgs) AddArg(value interface{}) string {
	na.counter++

	paramName := fmt.Sprintf("param%d", na.counter)

	namedArg := sql.Named(paramName, value)
	na.args = append(na.args, namedArg)

	return ":" + paramName
}

// GetArgs retorna os argumentos como slice de interface{}
func (na *NamedArgs) GetArgs() []interface{} {
	return na.args
}

// GetNamedArgs retorna os argumentos como slice para compatibilidade
func (na *NamedArgs) GetNamedArgs() []interface{} {
	return na.args
}

// QueryBuilder constrói queries SQL a partir de árvores de parse OData
type QueryBuilder struct {
	dialect    string
	nodeMap    NodeMap
	prepareMap PrepareMap
}

// NewQueryBuilder cria um novo QueryBuilder para o dialeto especificado
func NewQueryBuilder(dialect string) *QueryBuilder {
	qb := &QueryBuilder{
		dialect:    strings.ToLower(dialect),
		nodeMap:    make(NodeMap),
		prepareMap: make(PrepareMap),
	}

	// Configura os mapas baseado no dialeto
	switch qb.dialect {
	case "mysql":
		qb.setupMySQLMaps()
	case "postgresql":
		qb.setupPostgreSQLMaps()
	case "oracle":
		qb.setupOracleMaps()
	default:
		qb.setupDefaultMaps()
	}

	// Verifica se o nodeMap foi inicializado corretamente
	if qb.nodeMap == nil {
		panic("NodeMap is nil after setup")
	}

	if len(qb.nodeMap) == 0 {
		log.Printf("❌ QueryBuilder - WARNING: nodeMap is empty after setup")
	}

	return qb
}

// setupDefaultMaps configura mapas padrão
func (qb *QueryBuilder) setupDefaultMaps() {
	// Operadores de comparação
	qb.nodeMap["eq"] = "(%s = %s)"
	qb.nodeMap["ne"] = "(%s != %s)"
	qb.nodeMap["gt"] = "(%s > %s)"
	qb.nodeMap["ge"] = "(%s >= %s)"
	qb.nodeMap["lt"] = "(%s < %s)"
	qb.nodeMap["le"] = "(%s <= %s)"

	// Operadores lógicos
	qb.nodeMap["and"] = "(%s AND %s)"
	qb.nodeMap["or"] = "(%s OR %s)"
	qb.nodeMap["not"] = "(NOT %s)"

	// Operadores aritméticos
	qb.nodeMap["add"] = "(%s + %s)"
	qb.nodeMap["sub"] = "(%s - %s)"
	qb.nodeMap["mul"] = "(%s * %s)"
	qb.nodeMap["div"] = "(%s / %s)"
	qb.nodeMap["mod"] = "(%s %% %s)"

	// Funções de string
	qb.nodeMap["contains"] = "(%s LIKE %s)"
	qb.nodeMap["startswith"] = "(%s LIKE %s)"
	qb.nodeMap["endswith"] = "(%s LIKE %s)"
	qb.nodeMap["length"] = "LENGTH(%s)"
	qb.nodeMap["indexof"] = "LOCATE(%s, %s)"
	qb.nodeMap["substring"] = "SUBSTRING(%s, %s, %s)"
	qb.nodeMap["tolower"] = "LOWER(%s)"
	qb.nodeMap["toupper"] = "UPPER(%s)"
	qb.nodeMap["trim"] = "TRIM(%s)"
	qb.nodeMap["concat"] = "CONCAT(%s, %s)"

	// Funções de data/hora
	qb.nodeMap["year"] = "YEAR(%s)"
	qb.nodeMap["month"] = "MONTH(%s)"
	qb.nodeMap["day"] = "DAY(%s)"
	qb.nodeMap["hour"] = "HOUR(%s)"
	qb.nodeMap["minute"] = "MINUTE(%s)"
	qb.nodeMap["second"] = "SECOND(%s)"
	qb.nodeMap["now"] = "NOW()"
	qb.nodeMap["date"] = "DATE(%s)"
	qb.nodeMap["time"] = "TIME(%s)"

	// Funções matemáticas
	qb.nodeMap["round"] = "ROUND(%s)"
	qb.nodeMap["floor"] = "FLOOR(%s)"
	qb.nodeMap["ceiling"] = "CEIL(%s)"

	// Valores especiais
	qb.nodeMap["null"] = "NULL"

	// Prepare maps para LIKE
	qb.prepareMap["contains"] = "%%%s%%"
	qb.prepareMap["startswith"] = "%s%%"
	qb.prepareMap["endswith"] = "%%%s"
}

// setupMySQLMaps configura mapas para MySQL
func (qb *QueryBuilder) setupMySQLMaps() {
	qb.setupDefaultMaps() // Usa Default como padrão
}

// setupPostgreSQLMaps configura mapas para PostgreSQL
func (qb *QueryBuilder) setupPostgreSQLMaps() {
	// Herda configuração Default e sobrescreve diferenças
	qb.setupDefaultMaps()

	// Diferenças específicas do PostgreSQL
	qb.nodeMap["mod"] = "(%s %% %s)"
	qb.nodeMap["indexof"] = "POSITION(%s IN %s)"
	qb.nodeMap["ceiling"] = "CEILING(%s)"
	qb.nodeMap["contains"] = "(%s ILIKE %s)" // Case insensitive
	qb.nodeMap["startswith"] = "(%s ILIKE %s)"
	qb.nodeMap["endswith"] = "(%s ILIKE %s)"
}

// setupOracleMaps configura mapas para Oracle
func (qb *QueryBuilder) setupOracleMaps() {
	// Herda configuração Default e sobrescreve diferenças
	qb.setupDefaultMaps()

	// Diferenças específicas do Oracle
	qb.nodeMap["mod"] = "MOD(%s, %s)"
	qb.nodeMap["indexof"] = "INSTR(%s, %s)"
	qb.nodeMap["substring"] = "SUBSTR(%s, %s, %s)"
	qb.nodeMap["now"] = "SYSDATE"
	qb.nodeMap["ceiling"] = "CEIL(%s)"
	qb.nodeMap["length"] = "LENGTH(%s)"
}

// BuildWhereClause constrói cláusula WHERE a partir de árvore de parse
func (qb *QueryBuilder) BuildWhereClause(ctx context.Context, tree *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	if tree == nil {
		return "", []interface{}{}, nil
	}

	namedArgs := NewNamedArgs(qb.dialect)
	sql, err := qb.buildNodeExpressionNamed(ctx, tree, metadata, namedArgs)
	if err != nil {
		return "", nil, err
	}

	return sql, namedArgs.GetArgs(), nil
}

// BuildWhereClauseNamed constrói cláusula WHERE usando argumentos nomeados
func (qb *QueryBuilder) BuildWhereClauseNamed(ctx context.Context, tree *ParseNode, metadata EntityMetadata, namedArgs *NamedArgs) (string, error) {
	if tree == nil {
		return "", nil
	}

	return qb.buildNodeExpressionNamed(ctx, tree, metadata, namedArgs)
}

// buildNodeExpression constrói expressão SQL para um nó
func (qb *QueryBuilder) buildNodeExpression(ctx context.Context, node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	if node == nil {
		return "", []interface{}{}, nil
	}

	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	switch node.Token.Type {
	case int(FilterTokenProperty):
		// Propriedade - mapear para nome da coluna
		return qb.buildPropertyExpression(node, metadata)

	case int(FilterTokenString):
		// String literal
		value := strings.Trim(node.Token.Value, "'")
		return "?", []interface{}{value}, nil

	case int(FilterTokenNumber):
		// Número literal - usa SemanticReference se disponível (valor tipado original)
		if node.Token.SemanticReference != nil {
			return "?", []interface{}{node.Token.SemanticReference}, nil
		}
		return "?", []interface{}{node.Token.Value}, nil

	case int(FilterTokenBoolean):
		// Boolean literal
		value := node.Token.Value == "true"
		return "?", []interface{}{value}, nil

	case int(FilterTokenNull):
		// Null literal
		return "NULL", []interface{}{}, nil

	case int(FilterTokenLogical), int(FilterTokenComparison), int(FilterTokenArithmetic):
		// Operadores binários
		return qb.buildBinaryOperatorExpression(ctx, node, metadata)

	case int(FilterTokenFunction):
		// Funções
		return qb.buildFunctionExpression(ctx, node, metadata)

	default:
		return "", nil, fmt.Errorf("unsupported token type: %v", node.Token.Type)
	}
}

// buildNodeExpressionNamed constrói expressão SQL para um nó usando argumentos nomeados
func (qb *QueryBuilder) buildNodeExpressionNamed(ctx context.Context, node *ParseNode, metadata EntityMetadata, namedArgs *NamedArgs) (string, error) {
	if node == nil {
		return "", nil
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	switch node.Token.Type {
	case int(FilterTokenProperty):
		// Propriedade - mapear para nome da coluna
		sql, _, err := qb.buildPropertyExpression(node, metadata)
		return sql, err

	case int(FilterTokenString):
		// String literal
		value := strings.Trim(node.Token.Value, "'")
		placeholder := namedArgs.AddArg(value)
		return placeholder, nil

	case int(FilterTokenNumber):
		// Número literal - usa SemanticReference se disponível (valor tipado original)
		if node.Token.SemanticReference != nil {
			placeholder := namedArgs.AddArg(node.Token.SemanticReference)
			return placeholder, nil
		}

		// Converte o valor string para o tipo numérico apropriado
		typedValue, err := qb.parseNumericValue(node.Token.Value)
		if err != nil {
			return "", fmt.Errorf("failed to parse numeric value '%s': %w", node.Token.Value, err)
		}
		placeholder := namedArgs.AddArg(typedValue)
		return placeholder, nil

	case int(FilterTokenBoolean):
		// Boolean literal
		value := node.Token.Value == "true"
		placeholder := namedArgs.AddArg(value)
		return placeholder, nil

	case int(FilterTokenNull):
		// Null literal
		return "NULL", nil

	case int(FilterTokenLogical), int(FilterTokenComparison), int(FilterTokenArithmetic):
		// Operadores binários
		return qb.buildBinaryOperatorExpressionNamed(ctx, node, metadata, namedArgs)

	case int(FilterTokenFunction):
		// Funções
		return qb.buildFunctionExpressionNamed(ctx, node, metadata, namedArgs)

	default:
		return "", fmt.Errorf("unsupported token type: %v", node.Token.Type)
	}
}

// buildPropertyExpression constrói expressão para propriedade
func (qb *QueryBuilder) buildPropertyExpression(node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	propertyName := node.Token.Value

	// Encontra a propriedade nos metadados (comparação case-insensitive)
	for _, prop := range metadata.Properties {
		if strings.EqualFold(prop.Name, propertyName) {
			if prop.ColumnName != "" {
				return prop.ColumnName, []interface{}{}, nil
			}
			return propertyName, []interface{}{}, nil
		}
	}

	return "", nil, fmt.Errorf("property %s not found in entity %s", propertyName, metadata.Name)
}

// buildBinaryOperatorExpression constrói expressão para operador binário
func (qb *QueryBuilder) buildBinaryOperatorExpression(ctx context.Context, node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	if len(node.Children) != 2 {
		return "", nil, fmt.Errorf("binary operator %s expects 2 children, got %d", node.Token.Value, len(node.Children))
	}

	operator := node.Token.Value
	template, exists := qb.nodeMap[operator]
	if !exists {
		return "", nil, fmt.Errorf("unsupported operator: %s", operator)
	}

	// Constrói expressões para os filhos
	leftExpr, leftArgs, err := qb.buildNodeExpression(ctx, node.Children[0], metadata)
	if err != nil {
		return "", nil, err
	}

	rightExpr, rightArgs, err := qb.buildNodeExpression(ctx, node.Children[1], metadata)
	if err != nil {
		return "", nil, err
	}

	// Aplica preparação especial para funções como LIKE
	if prepareTemplate, exists := qb.prepareMap[operator]; exists {
		// Aplica preparação se o argumento direito é um valor literal
		if len(rightArgs) == 1 {
			if strValue, ok := rightArgs[0].(string); ok {
				rightArgs[0] = fmt.Sprintf(prepareTemplate, strValue)
			}
		}
	}

	// Conversão de tipos baseada no contexto da propriedade
	if len(leftArgs) == 0 && len(rightArgs) == 1 {
		// Lado esquerdo é propriedade, lado direito é valor
		propertyName := node.Children[0].Token.Value
		if node.Children[0].Token.Type == int(FilterTokenProperty) {
			convertedValue, err := qb.convertValueToPropertyType(rightArgs[0], propertyName, metadata)
			if err != nil {
				return "", nil, fmt.Errorf("failed to convert value for property %s: %w", propertyName, err)
			}
			rightArgs[0] = convertedValue
		}
	} else if len(leftArgs) == 1 && len(rightArgs) == 0 {
		// Lado esquerdo é valor, lado direito é propriedade
		propertyName := node.Children[1].Token.Value
		if node.Children[1].Token.Type == int(FilterTokenProperty) {
			convertedValue, err := qb.convertValueToPropertyType(leftArgs[0], propertyName, metadata)
			if err != nil {
				return "", nil, fmt.Errorf("failed to convert value for property %s: %w", propertyName, err)
			}
			leftArgs[0] = convertedValue
		}
	}

	// Combina argumentos
	args := append(leftArgs, rightArgs...)

	// Aplica template
	expression := fmt.Sprintf(template, leftExpr, rightExpr)

	return expression, args, nil
}

// buildBinaryOperatorExpressionNamed constrói expressão para operador binário usando argumentos nomeados
func (qb *QueryBuilder) buildBinaryOperatorExpressionNamed(ctx context.Context, node *ParseNode, metadata EntityMetadata, namedArgs *NamedArgs) (string, error) {

	if len(node.Children) != 2 {
		return "", fmt.Errorf("binary operator %s expects 2 children, got %d", node.Token.Value, len(node.Children))
	}

	operator := node.Token.Value

	// Verifica se o nodeMap está inicializado
	if qb.nodeMap == nil {
		return "", fmt.Errorf("nodeMap is nil - QueryBuilder not properly initialized")
	}

	// Lista as chaves disponíveis no nodeMap para debug
	var availableKeys []string
	for k := range qb.nodeMap {
		availableKeys = append(availableKeys, k)
	}

	template, exists := qb.nodeMap[operator]
	if !exists {
		return "", fmt.Errorf("unsupported operator: %s (available: %v)", operator, availableKeys)
	}

	// Constrói expressões para os filhos
	leftExpr, err := qb.buildNodeExpressionNamed(ctx, node.Children[0], metadata, namedArgs)
	if err != nil {
		return "", err
	}

	rightExpr, err := qb.buildNodeExpressionNamed(ctx, node.Children[1], metadata, namedArgs)
	if err != nil {
		return "", err
	}

	// Aplica template
	expression := fmt.Sprintf(template, leftExpr, rightExpr)

	return expression, nil
}

// buildFunctionExpressionNamed constrói expressão para função usando argumentos nomeados
func (qb *QueryBuilder) buildFunctionExpressionNamed(ctx context.Context, node *ParseNode, metadata EntityMetadata, namedArgs *NamedArgs) (string, error) {
	functionName := node.Token.Value
	template, exists := qb.nodeMap[functionName]
	if !exists {
		return "", fmt.Errorf("unsupported function: %s", functionName)
	}

	// Constrói expressões para argumentos
	argExpressions := make([]string, len(node.Children))

	for i, child := range node.Children {
		expr, err := qb.buildNodeExpressionNamed(ctx, child, metadata, namedArgs)
		if err != nil {
			return "", err
		}
		argExpressions[i] = expr
	}

	// Aplica template baseado no número de argumentos
	var expression string
	switch len(argExpressions) {
	case 0:
		expression = template
	case 1:
		expression = fmt.Sprintf(template, argExpressions[0])
	case 2:
		expression = fmt.Sprintf(template, argExpressions[0], argExpressions[1])
	case 3:
		expression = fmt.Sprintf(template, argExpressions[0], argExpressions[1], argExpressions[2])
	default:
		return "", fmt.Errorf("function %s with %d arguments not supported", functionName, len(argExpressions))
	}

	return expression, nil
}

// buildFunctionExpression constrói expressão para função
func (qb *QueryBuilder) buildFunctionExpression(ctx context.Context, node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	functionName := node.Token.Value
	template, exists := qb.nodeMap[functionName]
	if !exists {
		return "", nil, fmt.Errorf("unsupported function: %s", functionName)
	}

	// Constrói expressões para argumentos
	argExpressions := make([]string, len(node.Children))
	allArgs := make([]interface{}, 0)

	for i, child := range node.Children {
		expr, args, err := qb.buildNodeExpression(ctx, child, metadata)
		if err != nil {
			return "", nil, err
		}
		argExpressions[i] = expr
		allArgs = append(allArgs, args...)
	}

	// Aplica template baseado no número de argumentos
	var expression string
	switch len(argExpressions) {
	case 0:
		expression = template
	case 1:
		expression = fmt.Sprintf(template, argExpressions[0])
	case 2:
		expression = fmt.Sprintf(template, argExpressions[0], argExpressions[1])
	case 3:
		expression = fmt.Sprintf(template, argExpressions[0], argExpressions[1], argExpressions[2])
	default:
		return "", nil, fmt.Errorf("function %s with %d arguments not supported", functionName, len(argExpressions))
	}

	return expression, allArgs, nil
}

// BuildSelectClause constrói cláusula SELECT
func (qb *QueryBuilder) BuildSelectClause(metadata EntityMetadata, selectOptions []string) string {
	if len(selectOptions) == 0 {
		// Seleciona todas as colunas não-navegacionais
		columns := make([]string, 0)
		for _, prop := range metadata.Properties {
			if !prop.IsNavigation {
				columnName := prop.ColumnName
				if columnName == "" {
					columnName = prop.Name
				}
				columns = append(columns, columnName)
			}
		}
		return strings.Join(columns, ", ")
	}

	// Seleciona apenas as colunas especificadas
	columns := make([]string, 0)
	for _, propName := range selectOptions {
		for _, prop := range metadata.Properties {
			if strings.EqualFold(prop.Name, propName) && !prop.IsNavigation {
				columnName := prop.ColumnName
				if columnName == "" {
					columnName = prop.Name
				}
				columns = append(columns, columnName)
				break
			}
		}
	}

	return strings.Join(columns, ", ")
}

// BuildOrderByClause constrói cláusula ORDER BY
func (qb *QueryBuilder) BuildOrderByClause(metadata EntityMetadata, orderByOptions []OrderByExpression) string {
	if len(orderByOptions) == 0 {
		return ""
	}

	clauses := make([]string, 0)
	for _, option := range orderByOptions {
		// Encontra a propriedade nos metadados
		for _, prop := range metadata.Properties {
			if strings.EqualFold(prop.Name, option.Property) {
				columnName := prop.ColumnName
				if columnName == "" {
					columnName = prop.Name
				}

				direction := "ASC"
				if option.Direction == OrderDesc {
					direction = "DESC"
				}

				clauses = append(clauses, fmt.Sprintf("%s %s", columnName, direction))
				break
			}
		}
	}

	return strings.Join(clauses, ", ")
}

// convertValueToPropertyType converte o valor para o tipo correto baseado nos metadados da propriedade
func (qb *QueryBuilder) convertValueToPropertyType(value interface{}, propertyName string, metadata EntityMetadata) (interface{}, error) {
	// Encontra a propriedade nos metadados (comparação case-insensitive)
	for _, prop := range metadata.Properties {
		if strings.EqualFold(prop.Name, propertyName) {
			// Converte o valor para o tipo correto
			switch prop.Type {
			case "int64":
				switch v := value.(type) {
				case int64:
					return v, nil
				case int:
					return int64(v), nil
				case int32:
					return int64(v), nil
				case string:
					parsed, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as int64: %w", v, err)
					}
					return parsed, nil
				case float64:
					return int64(v), nil
				default:
					return nil, fmt.Errorf("cannot convert %T to int64", value)
				}
			case "int32":
				switch v := value.(type) {
				case int32:
					return v, nil
				case int:
					return int32(v), nil
				case int64:
					return int32(v), nil
				case string:
					parsed, err := strconv.ParseInt(v, 10, 32)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as int32: %w", v, err)
					}
					return int32(parsed), nil
				case float64:
					return int32(v), nil
				default:
					return nil, fmt.Errorf("cannot convert %T to int32", value)
				}
			case "string":
				return fmt.Sprintf("%v", value), nil
			case "bool":
				switch v := value.(type) {
				case bool:
					return v, nil
				case string:
					parsed, err := strconv.ParseBool(v)
					if err != nil {
						return nil, fmt.Errorf("failed to parse string %s as bool: %w", v, err)
					}
					return parsed, nil
				default:
					return nil, fmt.Errorf("cannot convert %T to bool", value)
				}
			default:
				// Para tipos não mapeados, retorna o valor original
				return value, nil
			}
		}
	}

	// Se não encontrar a propriedade, retorna o valor original
	return value, nil
}

// BuildLimitClause constrói cláusula LIMIT/OFFSET
func (qb *QueryBuilder) BuildLimitClause(top, skip int) string {
	switch strings.ToLower(qb.dialect) {
	case "mysql", "postgresql":
		if top > 0 && skip > 0 {
			return fmt.Sprintf("LIMIT %d OFFSET %d", top, skip)
		} else if top > 0 {
			return fmt.Sprintf("LIMIT %d", top)
		} else if skip > 0 {
			return fmt.Sprintf("OFFSET %d", skip)
		}
	case "oracle":
		if top > 0 && skip > 0 {
			return fmt.Sprintf("OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", skip, top)
		} else if top > 0 {
			return fmt.Sprintf("FETCH FIRST %d ROWS ONLY", top)
		} else if skip > 0 {
			return fmt.Sprintf("OFFSET %d ROWS", skip)
		}
	}
	return ""
}

// BuildCompleteQuery constrói query SQL completa
func (qb *QueryBuilder) BuildCompleteQuery(ctx context.Context, metadata EntityMetadata, options QueryOptions) (string, []interface{}, error) {
	var query strings.Builder
	var args []interface{}

	// SELECT clause
	selectFields := GetSelectedProperties(options.Select)
	selectClause := qb.BuildSelectClause(metadata, selectFields)
	query.WriteString("SELECT ")
	query.WriteString(selectClause)

	// FROM clause
	tableName := metadata.TableName
	if tableName == "" {
		tableName = metadata.Name
	}
	query.WriteString(" FROM ")
	query.WriteString(tableName)

	// WHERE clause
	if options.Filter != nil && options.Filter.Tree != nil {
		// Usa a árvore já parseada do filter
		tree := options.Filter.Tree

		whereClause, whereArgs, err := qb.BuildWhereClause(ctx, tree, metadata)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build where clause: %w", err)
		}

		if whereClause != "" {
			query.WriteString(" WHERE ")
			query.WriteString(whereClause)
			args = append(args, whereArgs...)
		}
	}

	// ORDER BY clause
	if options.OrderBy != "" {
		// TODO: Parse orderBy string to OrderByExpression slice
		// Por enquanto, adiciona diretamente a string
		query.WriteString(" ORDER BY ")
		query.WriteString(options.OrderBy)
	}

	// LIMIT/OFFSET clause
	topValue := GetTopValue(options.Top)
	skipValue := GetSkipValue(options.Skip)
	limitClause := qb.BuildLimitClause(topValue, skipValue)
	if limitClause != "" {
		query.WriteString(" ")
		query.WriteString(limitClause)
	}

	return query.String(), args, nil
}

// BuildComputeSQL constrói SQL para expressões $compute
func (qb *QueryBuilder) BuildComputeSQL(ctx context.Context, computeOption *ComputeOption, metadata EntityMetadata) (string, []interface{}, error) {
	if computeOption == nil || len(computeOption.Expressions) == 0 {
		return "", nil, nil
	}

	var computeFields []string
	var params []interface{}

	for _, expr := range computeOption.Expressions {
		sql, exprParams, err := qb.buildComputeExpression(ctx, expr, metadata)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build compute expression '%s': %w", expr.Expression, err)
		}

		computeFields = append(computeFields, fmt.Sprintf("(%s) AS %s", sql, qb.QuoteIdentifier(expr.Alias)))
		params = append(params, exprParams...)
	}

	return strings.Join(computeFields, ", "), params, nil
}

// buildComputeExpression constrói SQL para uma expressão de compute individual
func (qb *QueryBuilder) buildComputeExpression(ctx context.Context, expr ComputeExpression, metadata EntityMetadata) (string, []interface{}, error) {
	if expr.ParseTree == nil {
		return "", nil, fmt.Errorf("compute expression has no parse tree")
	}

	return qb.buildComputeNode(ctx, expr.ParseTree, metadata)
}

// buildComputeNode constrói SQL para um nó da árvore de compute
func (qb *QueryBuilder) buildComputeNode(ctx context.Context, node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	if node == nil {
		return "", nil, fmt.Errorf("compute node is nil")
	}

	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	switch node.Token.Type {
	case int(FilterTokenProperty):
		// Mapeia propriedade para coluna
		columnName, err := qb.getColumnName(node.Token.Value, metadata)
		if err != nil {
			return "", nil, err
		}
		return qb.QuoteIdentifier(columnName), nil, nil

	case int(FilterTokenString):
		// String literal
		value := node.Token.Value
		if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1] // Remove aspas
		}
		return "?", []interface{}{value}, nil

	case int(FilterTokenNumber):
		// Número literal
		return "?", []interface{}{node.Token.Value}, nil

	case int(FilterTokenFunction):
		// Função
		return qb.buildComputeFunction(ctx, node, metadata)

	case int(FilterTokenArithmetic):
		// Operador aritmético
		return qb.buildComputeArithmetic(ctx, node, metadata)

	default:
		return "", nil, fmt.Errorf("unsupported compute token type: %v", node.Token.Type)
	}
}

// buildComputeFunction constrói SQL para funções em compute
func (qb *QueryBuilder) buildComputeFunction(ctx context.Context, node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	functionName := node.Token.Value
	var params []interface{}

	switch functionName {
	case "round":
		if len(node.Children) < 1 || len(node.Children) > 2 {
			return "", nil, fmt.Errorf("round function requires 1 or 2 arguments")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		if len(node.Children) == 2 {
			precisionSQL, precisionParams, err := qb.buildComputeNode(ctx, node.Children[1], metadata)
			if err != nil {
				return "", nil, err
			}
			params = append(params, precisionParams...)
			return fmt.Sprintf("ROUND(%s, %s)", argSQL, precisionSQL), params, nil
		}

		return fmt.Sprintf("ROUND(%s)", argSQL), params, nil

	case "floor":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("floor function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("FLOOR(%s)", argSQL), params, nil

	case "ceiling":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("ceiling function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		// Mapeia para função específica do banco
		switch qb.dialect {
		case "mysql":
			return fmt.Sprintf("CEILING(%s)", argSQL), params, nil
		case "postgresql":
			return fmt.Sprintf("CEIL(%s)", argSQL), params, nil
		case "oracle":
			return fmt.Sprintf("CEIL(%s)", argSQL), params, nil
		default:
			return fmt.Sprintf("CEILING(%s)", argSQL), params, nil
		}

	case "abs":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("abs function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("ABS(%s)", argSQL), params, nil

	case "sqrt":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("sqrt function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("SQRT(%s)", argSQL), params, nil

	case "length":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("length function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("LENGTH(%s)", argSQL), params, nil

	case "tolower":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("tolower function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("LOWER(%s)", argSQL), params, nil

	case "toupper":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("toupper function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("UPPER(%s)", argSQL), params, nil

	case "trim":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("trim function requires 1 argument")
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		return fmt.Sprintf("TRIM(%s)", argSQL), params, nil

	case "concat":
		if len(node.Children) < 2 {
			return "", nil, fmt.Errorf("concat function requires at least 2 arguments")
		}

		var argSQLs []string
		for _, child := range node.Children {
			argSQL, argParams, err := qb.buildComputeNode(ctx, child, metadata)
			if err != nil {
				return "", nil, err
			}
			argSQLs = append(argSQLs, argSQL)
			params = append(params, argParams...)
		}

		// Mapeia para função específica do banco
		switch qb.dialect {
		case "mysql":
			return fmt.Sprintf("CONCAT(%s)", strings.Join(argSQLs, ", ")), params, nil
		case "postgresql":
			return fmt.Sprintf("CONCAT(%s)", strings.Join(argSQLs, ", ")), params, nil
		case "oracle":
			return strings.Join(argSQLs, " || "), params, nil
		default:
			return fmt.Sprintf("CONCAT(%s)", strings.Join(argSQLs, ", ")), params, nil
		}

	case "substring":
		if len(node.Children) < 2 || len(node.Children) > 3 {
			return "", nil, fmt.Errorf("substring function requires 2 or 3 arguments")
		}

		strSQL, strParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, strParams...)

		startSQL, startParams, err := qb.buildComputeNode(ctx, node.Children[1], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, startParams...)

		if len(node.Children) == 3 {
			lengthSQL, lengthParams, err := qb.buildComputeNode(ctx, node.Children[2], metadata)
			if err != nil {
				return "", nil, err
			}
			params = append(params, lengthParams...)

			// Mapeia para função específica do banco
			switch qb.dialect {
			case "mysql":
				return fmt.Sprintf("SUBSTRING(%s, %s, %s)", strSQL, startSQL, lengthSQL), params, nil
			case "postgresql":
				return fmt.Sprintf("SUBSTRING(%s FROM %s FOR %s)", strSQL, startSQL, lengthSQL), params, nil
			case "oracle":
				return fmt.Sprintf("SUBSTR(%s, %s, %s)", strSQL, startSQL, lengthSQL), params, nil
			default:
				return fmt.Sprintf("SUBSTRING(%s, %s, %s)", strSQL, startSQL, lengthSQL), params, nil
			}
		} else {
			// Mapeia para função específica do banco
			switch qb.dialect {
			case "mysql":
				return fmt.Sprintf("SUBSTRING(%s, %s)", strSQL, startSQL), params, nil
			case "postgresql":
				return fmt.Sprintf("SUBSTRING(%s FROM %s)", strSQL, startSQL), params, nil
			case "oracle":
				return fmt.Sprintf("SUBSTR(%s, %s)", strSQL, startSQL), params, nil
			default:
				return fmt.Sprintf("SUBSTRING(%s, %s)", strSQL, startSQL), params, nil
			}
		}

	case "year", "month", "day", "hour", "minute", "second":
		if len(node.Children) != 1 {
			return "", nil, fmt.Errorf("%s function requires 1 argument", functionName)
		}

		argSQL, argParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
		if err != nil {
			return "", nil, err
		}
		params = append(params, argParams...)

		// Mapeia para função específica do banco
		switch qb.dialect {
		case "mysql":
			return fmt.Sprintf("%s(%s)", strings.ToUpper(functionName), argSQL), params, nil
		case "postgresql":
			return fmt.Sprintf("EXTRACT(%s FROM %s)", strings.ToUpper(functionName), argSQL), params, nil
		case "oracle":
			return fmt.Sprintf("EXTRACT(%s FROM %s)", strings.ToUpper(functionName), argSQL), params, nil
		default:
			return fmt.Sprintf("%s(%s)", strings.ToUpper(functionName), argSQL), params, nil
		}

	case "now":
		if len(node.Children) != 0 {
			return "", nil, fmt.Errorf("now function requires no arguments")
		}

		// Mapeia para função específica do banco
		switch qb.dialect {
		case "mysql":
			return "NOW()", nil, nil
		case "postgresql":
			return "NOW()", nil, nil
		case "oracle":
			return "SYSDATE", nil, nil
		default:
			return "NOW()", nil, nil
		}

	default:
		return "", nil, fmt.Errorf("unsupported compute function: %s", functionName)
	}
}

// buildComputeArithmetic constrói SQL para operadores aritméticos em compute
func (qb *QueryBuilder) buildComputeArithmetic(ctx context.Context, node *ParseNode, metadata EntityMetadata) (string, []interface{}, error) {
	if len(node.Children) != 2 {
		return "", nil, fmt.Errorf("arithmetic operator requires 2 operands")
	}

	leftSQL, leftParams, err := qb.buildComputeNode(ctx, node.Children[0], metadata)
	if err != nil {
		return "", nil, err
	}

	rightSQL, rightParams, err := qb.buildComputeNode(ctx, node.Children[1], metadata)
	if err != nil {
		return "", nil, err
	}

	var params []interface{}
	params = append(params, leftParams...)
	params = append(params, rightParams...)

	operator := node.Token.Value
	var sqlOperator string

	switch operator {
	case "add":
		sqlOperator = "+"
	case "sub":
		sqlOperator = "-"
	case "mul":
		sqlOperator = "*"
	case "div":
		sqlOperator = "/"
	case "mod":
		sqlOperator = "%"
	default:
		return "", nil, fmt.Errorf("unsupported arithmetic operator: %s", operator)
	}

	return fmt.Sprintf("(%s %s %s)", leftSQL, sqlOperator, rightSQL), params, nil
}

// getColumnName obtém o nome da coluna para uma propriedade
func (qb *QueryBuilder) getColumnName(propertyName string, metadata EntityMetadata) (string, error) {
	for _, prop := range metadata.Properties {
		if strings.EqualFold(prop.Name, propertyName) {
			if prop.ColumnName != "" {
				return prop.ColumnName, nil
			}
			return prop.Name, nil
		}
	}
	return "", fmt.Errorf("property '%s' not found", propertyName)
}

// QuoteIdentifier adiciona aspas aos identificadores quando necessário
func (qb *QueryBuilder) QuoteIdentifier(identifier string) string {
	switch qb.dialect {
	case "mysql":
		return fmt.Sprintf("`%s`", identifier)
	case "postgresql":
		return fmt.Sprintf(`"%s"`, identifier)
	case "oracle":
		return fmt.Sprintf(`"%s"`, identifier)
	default:
		return identifier
	}
}

// BuildSearchSQL constrói SQL para expressões $search
func (qb *QueryBuilder) BuildSearchSQL(ctx context.Context, searchOption *SearchOption, metadata EntityMetadata) (string, []interface{}, error) {
	if searchOption == nil || searchOption.Expression == nil {
		return "", nil, nil
	}

	// Obtém propriedades pesquisáveis
	searchableProps := qb.getSearchableProperties(metadata)
	if len(searchableProps) == 0 {
		return "", nil, fmt.Errorf("no searchable properties found in entity %s", metadata.Name)
	}

	// Constrói SQL para a expressão de busca
	return qb.buildSearchExpression(ctx, searchOption.Expression, searchableProps)
}

// buildSearchExpression constrói SQL para uma expressão de busca
func (qb *QueryBuilder) buildSearchExpression(ctx context.Context, expr *SearchExpression, searchableProps []PropertyMetadata) (string, []interface{}, error) {
	if expr == nil {
		return "", nil, nil
	}

	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	switch expr.Type {
	case SearchExpressionTerm:
		return qb.buildSearchTerm(ctx, expr.Value, searchableProps)

	case SearchExpressionPhrase:
		return qb.buildSearchPhrase(ctx, expr.Value, searchableProps)

	case SearchExpressionAND:
		return qb.buildSearchBinaryOperator(ctx, expr, "AND", searchableProps)

	case SearchExpressionOR:
		return qb.buildSearchBinaryOperator(ctx, expr, "OR", searchableProps)

	case SearchExpressionNOT:
		return qb.buildSearchUnaryOperator(ctx, expr, "NOT", searchableProps)

	default:
		return "", nil, fmt.Errorf("unsupported search expression type: %v", expr.Type)
	}
}

// buildSearchTerm constrói SQL para um termo de busca
func (qb *QueryBuilder) buildSearchTerm(ctx context.Context, term string, searchableProps []PropertyMetadata) (string, []interface{}, error) {
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	if term == "" {
		return "", nil, fmt.Errorf("empty search term")
	}

	var conditions []string
	var params []interface{}

	// Verifica se o termo tem wildcards
	hasWildcard := strings.Contains(term, "*")

	for _, prop := range searchableProps {
		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}
		quotedColumn := qb.QuoteIdentifier(columnName)

		if hasWildcard {
			// Converte wildcards OData para SQL
			sqlPattern := strings.ReplaceAll(term, "*", "%")

			// Usa full-text search se disponível, senão LIKE
			if qb.supportsFullTextSearch() {
				condition, param := qb.buildFullTextSearchCondition(quotedColumn, sqlPattern)
				conditions = append(conditions, condition)
				params = append(params, param)
			} else {
				conditions = append(conditions, fmt.Sprintf("%s LIKE ?", quotedColumn))
				params = append(params, sqlPattern)
			}
		} else {
			// Busca exata ou contém
			if qb.supportsFullTextSearch() {
				condition, param := qb.buildFullTextSearchCondition(quotedColumn, term)
				conditions = append(conditions, condition)
				params = append(params, param)
			} else {
				// Busca por substring
				conditions = append(conditions, fmt.Sprintf("%s LIKE ?", quotedColumn))
				params = append(params, "%"+term+"%")
			}
		}
	}

	if len(conditions) == 0 {
		return "", nil, fmt.Errorf("no search conditions generated")
	}

	// Combina condições com OR (o termo deve ser encontrado em qualquer propriedade)
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), params, nil
}

// buildSearchPhrase constrói SQL para uma frase de busca
func (qb *QueryBuilder) buildSearchPhrase(ctx context.Context, phrase string, searchableProps []PropertyMetadata) (string, []interface{}, error) {
	if phrase == "" {
		return "", nil, fmt.Errorf("empty search phrase")
	}

	var conditions []string
	var params []interface{}

	for _, prop := range searchableProps {
		columnName := prop.ColumnName
		if columnName == "" {
			columnName = prop.Name
		}
		quotedColumn := qb.QuoteIdentifier(columnName)

		// Para frases, usa busca exata
		if qb.supportsFullTextSearch() {
			condition, param := qb.buildFullTextPhraseCondition(quotedColumn, phrase)
			conditions = append(conditions, condition)
			params = append(params, param)
		} else {
			// Busca por substring exata
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", quotedColumn))
			params = append(params, "%"+phrase+"%")
		}
	}

	if len(conditions) == 0 {
		return "", nil, fmt.Errorf("no search conditions generated")
	}

	// Combina condições com OR
	return fmt.Sprintf("(%s)", strings.Join(conditions, " OR ")), params, nil
}

// buildSearchBinaryOperator constrói SQL para operadores binários (AND, OR)
func (qb *QueryBuilder) buildSearchBinaryOperator(ctx context.Context, expr *SearchExpression, operator string, searchableProps []PropertyMetadata) (string, []interface{}, error) {
	if len(expr.Children) != 2 {
		return "", nil, fmt.Errorf("%s operator requires exactly 2 operands", operator)
	}

	leftSQL, leftParams, err := qb.buildSearchExpression(ctx, expr.Children[0], searchableProps)
	if err != nil {
		return "", nil, err
	}

	rightSQL, rightParams, err := qb.buildSearchExpression(ctx, expr.Children[1], searchableProps)
	if err != nil {
		return "", nil, err
	}

	var params []interface{}
	params = append(params, leftParams...)
	params = append(params, rightParams...)

	return fmt.Sprintf("(%s %s %s)", leftSQL, operator, rightSQL), params, nil
}

// buildSearchUnaryOperator constrói SQL para operadores unários (NOT)
func (qb *QueryBuilder) buildSearchUnaryOperator(ctx context.Context, expr *SearchExpression, operator string, searchableProps []PropertyMetadata) (string, []interface{}, error) {
	if len(expr.Children) != 1 {
		return "", nil, fmt.Errorf("%s operator requires exactly 1 operand", operator)
	}

	operandSQL, operandParams, err := qb.buildSearchExpression(ctx, expr.Children[0], searchableProps)
	if err != nil {
		return "", nil, err
	}

	return fmt.Sprintf("%s (%s)", operator, operandSQL), operandParams, nil
}

// getSearchableProperties obtém propriedades pesquisáveis de um metadata
func (qb *QueryBuilder) getSearchableProperties(metadata EntityMetadata) []PropertyMetadata {
	var searchableProps []PropertyMetadata

	for _, prop := range metadata.Properties {
		if qb.isSearchableProperty(prop) {
			searchableProps = append(searchableProps, prop)
		}
	}

	return searchableProps
}

// isSearchableProperty verifica se uma propriedade é pesquisável
func (qb *QueryBuilder) isSearchableProperty(prop PropertyMetadata) bool {
	if prop.IsNavigation {
		return false
	}

	// Verifica tipos de dados pesquisáveis
	propType := strings.ToLower(prop.Type)
	searchableTypes := []string{
		"string", "text", "varchar", "nvarchar", "char", "nchar",
		"clob", "nclob", "longtext", "mediumtext", "tinytext",
	}

	for _, searchableType := range searchableTypes {
		if strings.Contains(propType, searchableType) {
			return true
		}
	}

	return false
}

// supportsFullTextSearch verifica se o banco suporta full-text search
func (qb *QueryBuilder) supportsFullTextSearch() bool {
	switch qb.dialect {
	case "mysql", "postgresql", "oracle":
		return true
	default:
		return false
	}
}

// buildFullTextSearchCondition constrói condição de full-text search
func (qb *QueryBuilder) buildFullTextSearchCondition(column, term string) (string, interface{}) {
	switch qb.dialect {
	case "mysql":
		// MySQL FULLTEXT search
		return fmt.Sprintf("MATCH(%s) AGAINST(? IN BOOLEAN MODE)", column), term

	case "postgresql":
		// PostgreSQL full-text search
		return fmt.Sprintf("to_tsvector('english', %s) @@ plainto_tsquery('english', ?)", column), term

	case "oracle":
		// Oracle Text search
		return fmt.Sprintf("CONTAINS(%s, ?) > 0", column), term

	default:
		// Fallback para LIKE
		return fmt.Sprintf("%s LIKE ?", column), "%" + term + "%"
	}
}

// buildFullTextPhraseCondition constrói condição de busca por frase
func (qb *QueryBuilder) buildFullTextPhraseCondition(column, phrase string) (string, interface{}) {
	switch qb.dialect {
	case "mysql":
		// MySQL phrase search
		return fmt.Sprintf("MATCH(%s) AGAINST(? IN BOOLEAN MODE)", column), fmt.Sprintf(`"%s"`, phrase)

	case "postgresql":
		// PostgreSQL phrase search
		return fmt.Sprintf("to_tsvector('english', %s) @@ phraseto_tsquery('english', ?)", column), phrase

	case "oracle":
		// Oracle phrase search
		return fmt.Sprintf("CONTAINS(%s, ?) > 0", column), fmt.Sprintf(`"%s"`, phrase)

	default:
		// Fallback para LIKE
		return fmt.Sprintf("%s LIKE ?", column), "%" + phrase + "%"
	}
}

// BuildSearchWhereClause constrói cláusula WHERE para busca
func (qb *QueryBuilder) BuildSearchWhereClause(ctx context.Context, searchOption *SearchOption, metadata EntityMetadata) (string, []interface{}, error) {
	if searchOption == nil || searchOption.Expression == nil {
		return "", nil, nil
	}

	return qb.BuildSearchSQL(ctx, searchOption, metadata)
}

// CombineSearchWithFilter combina busca com filtro existente
func (qb *QueryBuilder) CombineSearchWithFilter(ctx context.Context, searchSQL, filterSQL string, searchParams, filterParams []interface{}) (string, []interface{}, error) {
	if searchSQL == "" && filterSQL == "" {
		return "", nil, nil
	}

	if searchSQL == "" {
		return filterSQL, filterParams, nil
	}

	if filterSQL == "" {
		return searchSQL, searchParams, nil
	}

	// Combina com AND
	combinedSQL := fmt.Sprintf("(%s) AND (%s)", searchSQL, filterSQL)
	var combinedParams []interface{}
	combinedParams = append(combinedParams, searchParams...)
	combinedParams = append(combinedParams, filterParams...)

	return combinedSQL, combinedParams, nil
}

// parseNumericValue converte um valor string para o tipo numérico apropriado
func (qb *QueryBuilder) parseNumericValue(value string) (interface{}, error) {
	// Remove sufixos específicos de tipo se presentes
	cleanValue := value
	suffixes := []string{"d", "D", "f", "F", "m", "M"}
	for _, suffix := range suffixes {
		cleanValue = strings.TrimSuffix(cleanValue, suffix)
	}

	// Tenta converter para int64 primeiro (mais comum)
	if intVal, err := strconv.ParseInt(cleanValue, 10, 64); err == nil {
		return intVal, nil
	}

	// Tenta converter para float64 se não for inteiro
	if floatVal, err := strconv.ParseFloat(cleanValue, 64); err == nil {
		return floatVal, nil
	}

	return nil, fmt.Errorf("invalid numeric value: %s", value)
}
