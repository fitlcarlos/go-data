package odata

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// =======================================================================================
// TYPE CONVERSION
// =======================================================================================

// convertValueToPropertyType converte o valor para o tipo correto baseado nos metadados da propriedade
func (s *BaseEntityService) convertValueToPropertyType(value any, propertyName string, metadata EntityMetadata) (any, error) {
	// Se o valor é nil, retorna nil
	if value == nil {
		return nil, nil
	}

	// Encontra a propriedade nos metadados
	for _, prop := range metadata.Properties {
		if strings.EqualFold(prop.Name, propertyName) {
			// Converte o valor para o tipo correto
			switch prop.Type {
			case "int64":
				return s.convertToInt64(value)
			case "int32", "int":
				return s.convertToInt32(value)
			case "float64", "double":
				return s.convertToFloat64(value)
			case "float32", "single":
				return s.convertToFloat32(value)
			case "string":
				return s.convertToString(value), nil
			case "bool", "boolean":
				return s.convertToBool(value)
			case "[]byte", "binary":
				return s.convertToBytes(value)
			default:
				// Para tipos não mapeados ou personalizados, aplica conversão básica
				switch v := value.(type) {
				case []byte:
					// Por padrão, converte []byte para string se não for tipo binário
					return string(v), nil
				default:
					return value, nil
				}
			}
		}
	}

	// Se não encontrou nos metadados, retorna como está
	return value, nil
}

// convertToInt64 converte valor para int64
func (s *BaseEntityService) convertToInt64(value any) (any, error) {
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
	case float32:
		return int64(v), nil
	case []byte:
		str := string(v)
		parsed, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse []byte %s as int64: %w", str, err)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to int64", value)
	}
}

// convertToInt32 converte valor para int32
func (s *BaseEntityService) convertToInt32(value any) (any, error) {
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
	case float32:
		return int32(v), nil
	case []byte:
		str := string(v)
		parsed, err := strconv.ParseInt(str, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse []byte %s as int32: %w", str, err)
		}
		return int32(parsed), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to int32", value)
	}
}

// convertToFloat64 converte valor para float64
func (s *BaseEntityService) convertToFloat64(value any) (any, error) {
	switch v := value.(type) {
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
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse string %s as float64: %w", v, err)
		}
		return parsed, nil
	case []byte:
		str := string(v)
		parsed, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse []byte %s as float64: %w", str, err)
		}
		return parsed, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// convertToFloat32 converte valor para float32
func (s *BaseEntityService) convertToFloat32(value any) (any, error) {
	switch v := value.(type) {
	case float32:
		return v, nil
	case float64:
		return float32(v), nil
	case int:
		return float32(v), nil
	case int32:
		return float32(v), nil
	case int64:
		return float32(v), nil
	case string:
		parsed, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse string %s as float32: %w", v, err)
		}
		return float32(parsed), nil
	case []byte:
		str := string(v)
		parsed, err := strconv.ParseFloat(str, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse []byte %s as float32: %w", str, err)
		}
		return float32(parsed), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to float32", value)
	}
}

// convertToString converte valor para string
func (s *BaseEntityService) convertToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// convertToBool converte valor para bool
func (s *BaseEntityService) convertToBool(value any) (any, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse string %s as bool: %w", v, err)
		}
		return parsed, nil
	case []byte:
		str := string(v)
		parsed, err := strconv.ParseBool(str)
		if err != nil {
			return nil, fmt.Errorf("failed to parse []byte %s as bool: %w", str, err)
		}
		return parsed, nil
	case int:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// convertToBytes converte valor para []byte
func (s *BaseEntityService) convertToBytes(value any) (any, error) {
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("cannot convert %T to []byte", value)
	}
}

// =======================================================================================
// FILTER EVALUATION
// =======================================================================================

// entityMatchesFilter verifica se uma entidade atende ao filtro especificado
func (s *BaseEntityService) entityMatchesFilter(entity any, filter *GoDataFilterQuery, metadata EntityMetadata) bool {
	if filter == nil || filter.Tree == nil {
		return true
	}

	// Converte a entidade para OrderedEntity se necessário
	var orderedEntity *OrderedEntity
	if oe, ok := entity.(*OrderedEntity); ok {
		orderedEntity = oe
	} else {
		// Se não é OrderedEntity, tenta converter
		return false
	}

	// Avalia o filtro recursivamente
	return s.evaluateFilterNode(orderedEntity, filter.Tree, metadata)
}

// evaluateFilterNode avalia um nó do filtro recursivamente
func (s *BaseEntityService) evaluateFilterNode(entity *OrderedEntity, node *ParseNode, metadata EntityMetadata) bool {
	if node == nil {
		return true
	}

	switch node.Token.Type {
	case int(FilterTokenLogical):
		// Operadores lógicos: and, or, not
		switch node.Token.Value {
		case "and":
			if len(node.Children) != 2 {
				return false
			}
			return s.evaluateFilterNode(entity, node.Children[0], metadata) && s.evaluateFilterNode(entity, node.Children[1], metadata)
		case "or":
			if len(node.Children) != 2 {
				return false
			}
			return s.evaluateFilterNode(entity, node.Children[0], metadata) || s.evaluateFilterNode(entity, node.Children[1], metadata)
		case "not":
			if len(node.Children) != 1 {
				return false
			}
			return !s.evaluateFilterNode(entity, node.Children[0], metadata)
		}
	case int(FilterTokenComparison):
		// Operadores de comparação: eq, ne, gt, lt, ge, le
		if len(node.Children) != 2 {
			return false
		}

		leftValue := s.evaluateFilterValue(entity, node.Children[0], metadata)
		rightValue := s.evaluateFilterValue(entity, node.Children[1], metadata)

		return s.compareValues(leftValue, rightValue, node.Token.Value)
	}

	return false
}

// evaluateFilterValue avalia um valor no filtro (propriedade ou literal)
func (s *BaseEntityService) evaluateFilterValue(entity *OrderedEntity, node *ParseNode, metadata EntityMetadata) any {
	if node == nil {
		return nil
	}

	switch node.Token.Type {
	case int(FilterTokenString), int(FilterTokenNumber), int(FilterTokenBoolean), int(FilterTokenNull):
		// Valor literal (string, número, booleano)
		return s.parseFilterLiteral(node.Token.Value)
	case int(FilterTokenProperty):
		// Nome de propriedade
		propertyName := node.Token.Value

		// Busca o valor na entidade
		if value, exists := entity.Get(propertyName); exists {
			return value
		}

		// Se não encontrou, tenta busca case-insensitive
		for _, prop := range entity.Properties {
			if strings.EqualFold(prop.Name, propertyName) {
				return prop.Value
			}
		}

		return nil
	}

	return nil
}

// parseFilterLiteral converte um literal string para o tipo apropriado
func (s *BaseEntityService) parseFilterLiteral(literal string) any {
	// Remove aspas se for string
	if len(literal) >= 2 && literal[0] == '\'' && literal[len(literal)-1] == '\'' {
		return literal[1 : len(literal)-1]
	}

	// Tenta converter para número
	if intVal, err := strconv.ParseInt(literal, 10, 64); err == nil {
		return intVal
	}

	// Tenta converter para float
	if floatVal, err := strconv.ParseFloat(literal, 64); err == nil {
		return floatVal
	}

	// Tenta converter para booleano
	if boolVal, err := strconv.ParseBool(literal); err == nil {
		return boolVal
	}

	// Retorna como string se não conseguir converter
	return literal
}

// compareValues compara dois valores usando o operador especificado
func (s *BaseEntityService) compareValues(left, right any, operator string) bool {
	// Converte para string para comparação
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch operator {
	case "eq":
		return leftStr == rightStr
	case "ne":
		return leftStr != rightStr
	case "gt":
		return leftStr > rightStr
	case "lt":
		return leftStr < rightStr
	case "ge":
		return leftStr >= rightStr
	case "le":
		return leftStr <= rightStr
	}

	return false
}

// =======================================================================================
// COMPUTE PROCESSING
// =======================================================================================

// applyComputeToResults aplica campos computados aos resultados seguindo a ordem OData v4
func (s *BaseEntityService) applyComputeToResults(ctx context.Context, results []any, computeOption *ComputeOption) ([]any, error) {
	if computeOption == nil || len(computeOption.Expressions) == 0 {
		return results, nil
	}

	// Para cada resultado, calcula os campos computados
	for i, result := range results {
		orderedEntity, ok := result.(*OrderedEntity)
		if !ok {
			continue
		}

		// Calcula cada expressão computada
		for _, expr := range computeOption.Expressions {
			computedValue, err := s.evaluateComputeExpression(ctx, expr, orderedEntity)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate compute expression '%s': %w", expr.Expression, err)
			}

			// Adiciona o campo computado ao resultado
			orderedEntity.Set(expr.Alias, computedValue)
		}

		results[i] = orderedEntity
	}

	return results, nil
}

// evaluateComputeExpression avalia uma expressão computada
func (s *BaseEntityService) evaluateComputeExpression(ctx context.Context, expr ComputeExpression, entity *OrderedEntity) (any, error) {
	if expr.ParseTree == nil {
		return nil, fmt.Errorf("compute expression has no parse tree")
	}

	return s.evaluateComputeNode(ctx, expr.ParseTree, entity)
}

// evaluateComputeNode avalia um nó da árvore de compute
func (s *BaseEntityService) evaluateComputeNode(ctx context.Context, node *ParseNode, entity *OrderedEntity) (any, error) {
	if node == nil {
		return nil, fmt.Errorf("compute node is nil")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	switch node.Token.Type {
	case int(FilterTokenProperty):
		// Obtém valor da propriedade
		value, exists := entity.Get(node.Token.Value)
		if !exists {
			return nil, fmt.Errorf("property %s not found", node.Token.Value)
		}
		return value, nil

	case int(FilterTokenString):
		// String literal
		value := node.Token.Value
		if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
			value = value[1 : len(value)-1] // Remove aspas
		}
		return value, nil

	case int(FilterTokenNumber):
		// Número literal
		return node.Token.Value, nil

	case int(FilterTokenArithmetic):
		// Operador aritmético
		if len(node.Children) != 2 {
			return nil, fmt.Errorf("arithmetic operator requires 2 operands")
		}

		left, err := s.evaluateComputeNode(ctx, node.Children[0], entity)
		if err != nil {
			return nil, err
		}

		right, err := s.evaluateComputeNode(ctx, node.Children[1], entity)
		if err != nil {
			return nil, err
		}

		return s.evaluateArithmeticOperation(node.Token.Value, left, right)

	default:
		return nil, fmt.Errorf("unsupported compute token type: %v", node.Token.Type)
	}
}

// evaluateArithmeticOperation avalia operações aritméticas
func (s *BaseEntityService) evaluateArithmeticOperation(operator string, left, right any) (any, error) {
	// Converte para números
	leftNum, err := s.convertToNumber(left)
	if err != nil {
		return nil, fmt.Errorf("left operand is not a number: %w", err)
	}

	rightNum, err := s.convertToNumber(right)
	if err != nil {
		return nil, fmt.Errorf("right operand is not a number: %w", err)
	}

	switch operator {
	case "add":
		return leftNum + rightNum, nil
	case "sub":
		return leftNum - rightNum, nil
	case "mul":
		return leftNum * rightNum, nil
	case "div":
		if rightNum == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return leftNum / rightNum, nil
	default:
		return nil, fmt.Errorf("unsupported arithmetic operator: %s", operator)
	}
}

// convertToNumber converte um valor para número
func (s *BaseEntityService) convertToNumber(value any) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		if num, err := strconv.ParseFloat(v, 64); err == nil {
			return num, nil
		}
		return 0, fmt.Errorf("cannot convert string '%s' to number", v)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// =======================================================================================
// SELECT PROCESSING
// =======================================================================================

// applySelectToResults filtra apenas os campos selecionados nos resultados
func (s *BaseEntityService) applySelectToResults(results []any, selectQuery *GoDataSelectQuery) ([]any, error) {
	if selectQuery == nil {
		return results, nil
	}

	selectedFields := GetSelectedProperties(selectQuery)
	if len(selectedFields) == 0 {
		return results, nil
	}

	// Para cada resultado, filtra apenas os campos selecionados
	for i, result := range results {
		orderedEntity, ok := result.(*OrderedEntity)
		if !ok {
			continue
		}

		// Cria nova entidade com apenas os campos selecionados
		filteredEntity := NewOrderedEntity()
		for _, field := range selectedFields {
			if value, exists := orderedEntity.Get(field); exists {
				filteredEntity.Set(field, value)
			}
		}

		results[i] = filteredEntity
	}

	return results, nil
}
