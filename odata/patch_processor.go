package odata

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// PatchOperation representa uma operação a ser executada em um PATCH
type PatchOperation struct {
	Type          string                 // INSERT, UPDATE, DELETE
	Entity        map[string]interface{} // Dados da entidade
	Keys          map[string]interface{} // Chaves identificadoras
	NavigationPath string                // Caminho hierárquico (ex: "Itens", "Itens.Produtos")
	EntityName    string                 // Nome da entidade para lookup de serviço
}

// hasHierarchicalStructure verifica se o JSON tem estrutura hierárquica que requer processamento avançado
// Retorna true se:
// - Tem propriedades com @odata.removed
// - Tem propriedades de navegação com objetos/arrays aninhados
// - Tem @odata.id em objetos aninhados
func hasHierarchicalStructure(data map[string]interface{}, metadata EntityMetadata) bool {
	// Verifica se tem @odata.removed
	for key := range data {
		if key == "@odata.removed" || strings.HasSuffix(key, "@odata.removed") {
			return true
		}
	}

	// Verifica propriedades de navegação com objetos/arrays aninhados
	for _, prop := range metadata.Properties {
		if prop.IsNavigation {
			if value, exists := data[prop.Name]; exists {
				// Verifica se é array ou objeto
				if isNestedStructure(value) {
					return true
				}
			}
		}
	}

	// Verifica se tem @odata.id em objetos aninhados
	return hasNestedODataID(data)
}

// isNestedStructure verifica se o valor é uma estrutura aninhada (array ou objeto)
func isNestedStructure(value interface{}) bool {
	if value == nil {
		return false
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		// Verifica se é array de objetos
		if val.Len() > 0 {
			firstElem := val.Index(0).Interface()
			if isMapOrStruct(firstElem) {
				return true
			}
		}
		return false
	case reflect.Map:
		return true
	default:
		return false
	}
}

// isMapOrStruct verifica se o valor é um map ou struct
func isMapOrStruct(value interface{}) bool {
	if value == nil {
		return false
	}
	val := reflect.ValueOf(value)
	return val.Kind() == reflect.Map || val.Kind() == reflect.Struct
}

// hasNestedODataID verifica se há @odata.id em objetos aninhados
func hasNestedODataID(data map[string]interface{}) bool {
	for key, value := range data {
		// Verifica se a própria chave é @odata.id
		if key == "@odata.id" {
			return true
		}

		// Verifica recursivamente em objetos aninhados
		if valueMap, ok := value.(map[string]interface{}); ok {
			if hasNestedODataID(valueMap) {
				return true
			}
		}

		// Verifica em arrays
		if valueSlice, ok := value.([]interface{}); ok {
			for _, item := range valueSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if hasNestedODataID(itemMap) {
						return true
					}
				}
			}
		}
	}
	return false
}

// identifyOperation identifica o tipo de operação (INSERT/UPDATE/DELETE) baseado no objeto e metadados
func identifyOperation(entity map[string]interface{}, metadata EntityMetadata, removedFormat string) (string, error) {
	// Verifica se tem @odata.removed
	if hasRemovedAnnotation(entity, removedFormat) {
		return "DELETE", nil
	}

	// Verifica se tem todas as chaves (simples ou compostas)
	keys := extractKeysFromEntity(entity, metadata)
	if hasAllKeys(keys, metadata) {
		return "UPDATE", nil
	}

	// Caso contrário, é INSERT
	return "INSERT", nil
}

// hasRemovedAnnotation verifica se o objeto tem @odata.removed
func hasRemovedAnnotation(entity map[string]interface{}, removedFormat string) bool {
	// Verifica @odata.removed direto
	if removed, exists := entity["@odata.removed"]; exists {
		return isValidRemovedFormat(removed, removedFormat)
	}

	// Verifica propriedades com sufixo @odata.removed
	for key := range entity {
		if strings.HasSuffix(key, "@odata.removed") {
			return true
		}
	}

	return false
}

// isValidRemovedFormat valida o formato de @odata.removed conforme configuração
func isValidRemovedFormat(removed interface{}, format string) bool {
	if removed == nil {
		return false
	}

	switch format {
	case "empty":
		// Apenas objeto vazio {}
		if removedMap, ok := removed.(map[string]interface{}); ok {
			return len(removedMap) == 0
		}
		return false
	case "with_reason":
		// Apenas com propriedades como {"reason": "deleted"}
		if removedMap, ok := removed.(map[string]interface{}); ok {
			return len(removedMap) > 0
		}
		return false
	case "both", "":
		// Aceita ambos os formatos
		_, ok := removed.(map[string]interface{})
		return ok
	default:
		return false
	}
}

// extractKeysFromEntity extrai chaves primárias de um objeto JSON
// Suporta chaves simples e compostas
// Verifica também @odata.id se presente
func extractKeysFromEntity(entity map[string]interface{}, metadata EntityMetadata) map[string]interface{} {
	keys := make(map[string]interface{})

	// Primeiro, verifica se tem @odata.id
	if odataID, exists := entity["@odata.id"]; exists {
		// Extrai ID do @odata.id (formato: "/EntityName(id)" ou "/EntityName(key1=value1,key2=value2)")
		if idStr, ok := odataID.(string); ok {
			extractedKeys := parseODataID(idStr, metadata)
			if len(extractedKeys) > 0 {
				return extractedKeys
			}
		}
	}

	// Extrai chaves primárias dos metadados
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			if value, exists := entity[prop.Name]; exists {
				keys[prop.Name] = value
			}
		}
	}

	return keys
}

// parseODataID extrai chaves de um @odata.id
// Formato: "/EntityName(id)" ou "/EntityName(key1=value1,key2=value2)"
func parseODataID(odataID string, metadata EntityMetadata) map[string]interface{} {
	keys := make(map[string]interface{})

	// Remove barra inicial se presente
	odataID = strings.TrimPrefix(odataID, "/")

	// Encontra parênteses
	start := strings.Index(odataID, "(")
	end := strings.LastIndex(odataID, ")")
	if start == -1 || end == -1 || start >= end {
		return keys
	}

	keyString := odataID[start+1 : end]

	// Identifica as chaves primárias dos metadados
	var primaryKeys []PropertyMetadata
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			primaryKeys = append(primaryKeys, prop)
		}
	}

	if len(primaryKeys) == 0 {
		return keys
	}

	// Se há apenas uma chave primária, assume que o valor é para ela
	if len(primaryKeys) == 1 {
		key := primaryKeys[0]
		value := parseKeyValue(keyString, key.Type)
		if value != nil {
			keys[key.Name] = value
		}
		return keys
	}

	// Para chaves compostas, precisa analisar pares chave=valor
	pairs := strings.Split(keyString, ",")
	for _, pair := range pairs {
		kv := strings.Split(strings.TrimSpace(pair), "=")
		if len(kv) != 2 {
			continue
		}

		keyName := strings.TrimSpace(kv[0])
		keyValue := strings.TrimSpace(kv[1])

		// Encontra a propriedade correspondente
		for _, prop := range primaryKeys {
			if prop.Name == keyName {
				value := parseKeyValue(keyValue, prop.Type)
				if value != nil {
					keys[keyName] = value
				}
				break
			}
		}
	}

	return keys
}

// parseKeyValue converte uma string em valor do tipo apropriado
func parseKeyValue(value, dataType string) interface{} {
	// Remove aspas se presentes
	value = strings.Trim(value, `"'`)

	switch dataType {
	case "int", "int32", "int64":
		if intValue, err := parseInt(value); err == nil {
			return intValue
		}
	case "float32", "float64":
		if floatValue, err := parseFloat(value); err == nil {
			return floatValue
		}
	case "bool":
		if boolValue, err := parseBool(value); err == nil {
			return boolValue
		}
	default:
		// String ou outros tipos
		return value
	}

	return value
}

// parseInt tenta converter string para int
func parseInt(s string) (interface{}, error) {
	// Tenta int64 primeiro
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	// Tenta int32
	if i, err := strconv.ParseInt(s, 10, 32); err == nil {
		return int32(i), nil
	}
	// Tenta int
	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}
	return nil, fmt.Errorf("cannot parse int: %s", s)
}

// parseFloat tenta converter string para float
func parseFloat(s string) (interface{}, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("cannot parse float: %s", s)
}

// parseBool tenta converter string para bool
func parseBool(s string) (interface{}, error) {
	if b, err := strconv.ParseBool(s); err == nil {
		return b, nil
	}
	return nil, fmt.Errorf("cannot parse bool: %s", s)
}

// hasAllKeys verifica se todas as chaves primárias estão presentes
func hasAllKeys(keys map[string]interface{}, metadata EntityMetadata) bool {
	// Conta quantas chaves primárias existem
	keyCount := 0
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			keyCount++
			// Verifica se a chave está presente
			if _, exists := keys[prop.Name]; !exists {
				return false
			}
		}
	}

	// Se não há chaves primárias definidas, considera que não tem todas
	if keyCount == 0 {
		return false
	}

	return len(keys) == keyCount
}

// processNavigationProperty processa propriedades de navegação recursivamente
func processNavigationProperty(
	ctx context.Context,
	server *Server,
	entity map[string]interface{},
	prop PropertyMetadata,
	navigationPath string,
	removedFormat string,
	operations *[]PatchOperation,
) error {
	if !prop.IsNavigation {
		return nil
	}

	value, exists := entity[prop.Name]
	if !exists {
		return nil
	}

	newPath := prop.Name
	if navigationPath != "" {
		newPath = navigationPath + "." + prop.Name
	}

	// Processa coleção (array)
	if prop.IsCollection {
		if valueSlice, ok := value.([]interface{}); ok {
			for _, item := range valueSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					// Obtém metadados da entidade relacionada
					relatedMetadata, err := getRelatedEntityMetadata(server, prop.RelatedType)
					if err != nil {
						log.Printf("Warning: Failed to get metadata for %s: %v", prop.RelatedType, err)
						continue
					}

					// Identifica operação
					opType, err := identifyOperation(itemMap, relatedMetadata, removedFormat)
					if err != nil {
						log.Printf("Warning: Failed to identify operation: %v", err)
						continue
					}

					// Extrai chaves
					keys := extractKeysFromEntity(itemMap, relatedMetadata)

					// Remove @odata.removed do objeto antes de processar
					cleanEntity := cleanRemovedAnnotation(itemMap)

					*operations = append(*operations, PatchOperation{
						Type:          opType,
						Entity:        cleanEntity,
						Keys:          keys,
						NavigationPath: newPath,
						EntityName:    prop.RelatedType,
					})

					// Processa recursivamente propriedades de navegação aninhadas
					if err := processPatchRecursive(ctx, server, itemMap, relatedMetadata, newPath, removedFormat, operations); err != nil {
						log.Printf("Warning: Failed to process nested navigation: %v", err)
					}
				}
			}
		}
	} else {
		// Processa entidade única
		if valueMap, ok := value.(map[string]interface{}); ok {
			// Obtém metadados da entidade relacionada
			relatedMetadata, err := getRelatedEntityMetadata(server, prop.RelatedType)
			if err != nil {
				return fmt.Errorf("failed to get metadata for %s: %w", prop.RelatedType, err)
			}

			// Identifica operação
			opType, err := identifyOperation(valueMap, relatedMetadata, removedFormat)
			if err != nil {
				return fmt.Errorf("failed to identify operation: %w", err)
			}

			// Extrai chaves
			keys := extractKeysFromEntity(valueMap, relatedMetadata)

			// Remove @odata.removed do objeto antes de processar
			cleanEntity := cleanRemovedAnnotation(valueMap)

			*operations = append(*operations, PatchOperation{
				Type:          opType,
				Entity:        cleanEntity,
				Keys:          keys,
				NavigationPath: newPath,
				EntityName:    prop.RelatedType,
			})

			// Processa recursivamente propriedades de navegação aninhadas
			return processPatchRecursive(ctx, server, valueMap, relatedMetadata, newPath, removedFormat, operations)
		}
	}

	return nil
}

// cleanRemovedAnnotation remove @odata.removed do objeto
func cleanRemovedAnnotation(entity map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{})
	for key, value := range entity {
		if key != "@odata.removed" && !strings.HasSuffix(key, "@odata.removed") {
			clean[key] = value
		}
	}
	return clean
}

// getRelatedEntityMetadata obtém metadados de uma entidade relacionada
func getRelatedEntityMetadata(server *Server, entityName string) (EntityMetadata, error) {
	service := server.GetEntityService(entityName)
	if service == nil {
		return EntityMetadata{}, fmt.Errorf("entity service not found: %s", entityName)
	}
	return service.GetMetadata(), nil
}

// processPatchRecursive processa recursivamente o JSON hierárquico
func processPatchRecursive(
	ctx context.Context,
	server *Server,
	entity map[string]interface{},
	metadata EntityMetadata,
	navigationPath string,
	removedFormat string,
	operations *[]PatchOperation,
) error {
	// Processa propriedades de navegação
	for _, prop := range metadata.Properties {
		if prop.IsNavigation {
			if err := processNavigationProperty(ctx, server, entity, prop, navigationPath, removedFormat, operations); err != nil {
				log.Printf("Warning: Failed to process navigation property %s: %v", prop.Name, err)
			}
		}
	}

	return nil
}

