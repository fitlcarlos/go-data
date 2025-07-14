package odata

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"unicode"
)

// URLParser é um parser de URL otimizado que combina performance e robustez
type URLParser struct {
	// Cache para melhor performance
	normalizeCache sync.Map
	validateCache  sync.Map
	simpleCache    sync.Map

	// Configurações de compliance
	strictMode bool
}

// Mapa de palavras-chave OData suportadas (otimização por lookup O(1))
var supportedODataKeywords = map[string]bool{
	"$filter":      true,
	"$apply":       true,
	"$expand":      true,
	"$select":      true,
	"$orderby":     true,
	"$top":         true,
	"$skip":        true,
	"$count":       true,
	"$inlinecount": true,
	"$search":      true,
	"$compute":     true,
	"$format":      true,
}

// Configuração de compliance OData otimizada
type OptimizedComplianceConfig int

const (
	OptimizedComplianceStrict                  OptimizedComplianceConfig = 0
	OptimizedComplianceIgnoreDuplicateKeywords OptimizedComplianceConfig = 1 << iota
	OptimizedComplianceIgnoreUnknownKeywords
	OptimizedComplianceIgnoreInvalidComma
	OptimizedComplianceIgnoreAll OptimizedComplianceConfig = OptimizedComplianceIgnoreDuplicateKeywords |
		OptimizedComplianceIgnoreUnknownKeywords |
		OptimizedComplianceIgnoreInvalidComma
)

// NewURLParser cria um novo parser de URL
func NewURLParser() *URLParser {
	return &URLParser{
		strictMode: false, // modo flexível por padrão
	}
}

// NewOptimizedURLParser cria um novo parser otimizado (mantido para compatibilidade)
func NewOptimizedURLParser(strictMode bool) *URLParser {
	return &URLParser{
		strictMode: strictMode,
	}
}

// ParseQueryFast faz parsing rápido usando url.Values padrão quando possível
func (up *URLParser) ParseQueryFast(rawQuery string) (url.Values, error) {
	// Para queries simples, usa o parser padrão (mais rápido)
	if up.isSimpleQuery(rawQuery) {
		values, err := url.ParseQuery(rawQuery)
		if err != nil {
			return nil, err
		}

		// Valida rapidamente usando o mapa
		if err := up.fastValidate(values); err != nil {
			return nil, err
		}

		return values, nil
	}

	// Para queries complexas, usa nosso parser customizado
	return up.parseComplexQuery(rawQuery)
}

// isSimpleQuery verifica se a query é simples o suficiente para usar parser padrão
func (up *URLParser) isSimpleQuery(query string) bool {
	if query == "" {
		return true
	}

	// Cache check
	if cached, ok := up.simpleCache.Load(query); ok {
		return cached.(bool)
	}

	// Heurísticas para detectar queries simples (sem parênteses aninhados, aspas complexas, semicolons)
	simple := true
	parenthesesCount := 0
	inQuotes := false
	var quoteChar rune

Loop:
	for _, char := range query {
		switch char {
		case '\'', '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
			}
		case '(':
			if !inQuotes {
				parenthesesCount++
				if parenthesesCount > 2 { // Tolerância para cases simples
					simple = false
					break Loop
				}
			}
		case ')':
			if !inQuotes {
				parenthesesCount--
			}
		case ';':
			// Semicolons dentro de valores OData tornam a query complexa
			// O parser padrão do Go não suporta semicolons
			simple = false
			break Loop
		}
	}

	// Se há parênteses desbalanceados ou muitos níveis, é complexa
	if parenthesesCount != 0 || inQuotes {
		simple = false
	}

	// Cache result
	up.simpleCache.Store(query, simple)
	return simple
}

// fastValidate faz validação rápida usando mapa de keywords
func (up *URLParser) fastValidate(values url.Values) error {
	if !up.strictMode {
		return nil
	}

	for key, vals := range values {
		// Verifica se é keyword suportada (O(1) lookup)
		if _, ok := supportedODataKeywords[key]; !ok {
			return &QueryParseError{
				Message: fmt.Sprintf("Query parameter '%s' is not supported", key),
				Query:   key,
			}
		}

		// Verifica duplicatas
		if len(vals) > 1 {
			return &QueryParseError{
				Message: fmt.Sprintf("Query parameter '%s' cannot be specified more than once", key),
				Query:   key,
			}
		}
	}

	return nil
}

// parseComplexQuery usa nosso parser customizado para queries complexas
func (up *URLParser) parseComplexQuery(rawQuery string) (url.Values, error) {
	// Pre-processa a query (otimizado)
	processedQuery := up.fastPreprocess(rawQuery)

	// Parsing customizado (otimizado)
	return up.parseODataQueryOptimized(processedQuery)
}

// fastPreprocess faz pré-processamento otimizado
func (up *URLParser) fastPreprocess(query string) string {
	if query == "" {
		return ""
	}

	// Cache check
	if cached, ok := up.normalizeCache.Load(query); ok {
		return cached.(string)
	}

	// StringBuilder para melhor performance
	var result strings.Builder
	result.Grow(len(query)) // Pré-aloca espaço

	// Decodificação direta de caracteres mais comuns
	replacements := map[string]string{
		"%24": "$", "%28": "(", "%29": ")", "%2C": ",", "%3B": ";",
		"%3D": "=", "%26": "&", "%27": "'", "%22": "\"", "%20": " ", "+": " ",
	}

	processedQuery := query
	for encoded, decoded := range replacements {
		if strings.Contains(processedQuery, encoded) {
			processedQuery = strings.ReplaceAll(processedQuery, encoded, decoded)
		}
	}

	// Cache result
	up.normalizeCache.Store(query, processedQuery)
	return processedQuery
}

// parseODataQueryOptimized versão otimizada do parsing customizado
func (up *URLParser) parseODataQueryOptimized(query string) (url.Values, error) {
	values := make(url.Values)

	if query == "" {
		return values, nil
	}

	// Split otimizado com pool de strings
	params := up.splitQueryParamsOptimized(query)

	for _, param := range params {
		key, value := up.parseParamOptimized(param)
		if key != "" {
			// Decodifica apenas se necessário
			decodedValue := value
			if strings.Contains(value, "%") {
				if decoded, err := url.QueryUnescape(value); err == nil {
					decodedValue = decoded
				}
			}

			values.Add(key, decodedValue)
		}
	}

	// Validação rápida
	if err := up.fastValidate(values); err != nil {
		return nil, err
	}

	return values, nil
}

// splitQueryParamsOptimized versão otimizada do split de parâmetros
func (up *URLParser) splitQueryParamsOptimized(query string) []string {
	if query == "" {
		return nil
	}

	// Pré-aloca slice com capacidade estimada
	params := make([]string, 0, strings.Count(query, "&")+1)

	var start int
	var inQuotes bool
	var quoteChar rune
	var parenthesesLevel int

	for i, char := range query {
		switch char {
		case '\'', '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
		case '(':
			if !inQuotes {
				parenthesesLevel++
			}
		case ')':
			if !inQuotes {
				parenthesesLevel--
			}
		case '&':
			if !inQuotes && parenthesesLevel == 0 {
				// Fim do parâmetro atual
				if i > start {
					params = append(params, query[start:i])
				}
				start = i + 1
			}
			// Semicolons são válidos dentro de valores de parâmetros OData (como $expand)
			// Eles não são separadores no nível superior
		}
	}

	// Adiciona o último parâmetro
	if start < len(query) {
		params = append(params, query[start:])
	}

	return params
}

// parseParamOptimized versão otimizada do parsing de parâmetro
func (up *URLParser) parseParamOptimized(param string) (string, string) {
	if param == "" {
		return "", ""
	}

	// Busca rápida por '=' sem verificações complexas para cases simples
	if equalPos := strings.IndexByte(param, '='); equalPos != -1 {
		// Verificação simples se não há aspas/parênteses antes do =
		if equalPos < 50 && !strings.ContainsAny(param[:equalPos], "'\"()") {
			return param[:equalPos], param[equalPos+1:]
		}
	}

	// Fallback para parsing complexo
	return up.parseParamComplex(param)
}

// parseParamComplex versão complexa apenas quando necessário
func (up *URLParser) parseParamComplex(param string) (string, string) {
	var inQuotes bool
	var quoteChar rune
	var parenthesesLevel int

	for i, char := range param {
		switch char {
		case '\'', '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
		case '(':
			if !inQuotes {
				parenthesesLevel++
			}
		case ')':
			if !inQuotes {
				parenthesesLevel--
			}
		case '=':
			if !inQuotes && parenthesesLevel == 0 {
				return strings.TrimSpace(param[:i]), strings.TrimSpace(param[i+1:])
			}
		}
	}

	return strings.TrimSpace(param), ""
}

// ValidateODataQueryFast validação otimizada
func (up *URLParser) ValidateODataQueryFast(query string) error {
	if query == "" {
		return nil
	}

	// Cache check
	if cached, ok := up.validateCache.Load(query); ok {
		if err, isErr := cached.(error); isErr {
			return err
		}
		return nil
	}

	// Validação rápida apenas se strictMode
	if !up.strictMode {
		up.validateCache.Store(query, nil)
		return nil
	}

	// Verifica balanceamento básico (otimizado)
	if err := up.fastBalanceCheck(query); err != nil {
		up.validateCache.Store(query, err)
		return err
	}

	up.validateCache.Store(query, nil)
	return nil
}

// fastBalanceCheck verificação rápida de balanceamento
func (up *URLParser) fastBalanceCheck(query string) error {
	var parentheses, singleQuotes, doubleQuotes int

	for _, char := range query {
		switch char {
		case '(':
			parentheses++
		case ')':
			parentheses--
			if parentheses < 0 {
				return &QueryParseError{
					Message: "Unbalanced parentheses in query",
					Query:   query,
				}
			}
		case '\'':
			singleQuotes++
		case '"':
			doubleQuotes++
		}
	}

	if parentheses != 0 {
		return &QueryParseError{
			Message: "Unbalanced parentheses in query",
			Query:   query,
		}
	}

	if singleQuotes%2 != 0 || doubleQuotes%2 != 0 {
		return &QueryParseError{
			Message: "Unbalanced quotes in query",
			Query:   query,
		}
	}

	return nil
}

// NormalizeODataQueryFast normalização otimizada
func (up *URLParser) NormalizeODataQueryFast(query string) string {
	if query == "" {
		return ""
	}

	// Cache check
	if cached, ok := up.normalizeCache.Load(query); ok {
		return cached.(string)
	}

	// Normalização otimizada
	result := up.fastNormalize(query)
	up.normalizeCache.Store(query, result)
	return result
}

// fastNormalize normalização rápida
func (up *URLParser) fastNormalize(query string) string {
	var result strings.Builder
	result.Grow(len(query))

	var inQuotes bool
	var quoteChar rune
	var lastWasSpace bool

	for _, char := range query {
		switch char {
		case '\'', '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
			result.WriteRune(char)
			lastWasSpace = false
		case ' ', '\t', '\n', '\r':
			if inQuotes {
				result.WriteRune(char)
				lastWasSpace = false
			} else if !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
		default:
			if unicode.IsControl(char) && char != '\t' && char != '\n' {
				continue // Remove caracteres de controle
			}
			result.WriteRune(char)
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(result.String())
}

// ClearCache limpa o cache para liberar memória
func (up *URLParser) ClearCache() {
	up.normalizeCache = sync.Map{}
	up.validateCache = sync.Map{}
	up.simpleCache = sync.Map{}
}

// GetCacheStats retorna estatísticas do cache
func (up *URLParser) GetCacheStats() (normalizeEntries, validateEntries, simpleEntries int) {
	up.normalizeCache.Range(func(key, value interface{}) bool {
		normalizeEntries++
		return true
	})

	up.validateCache.Range(func(key, value interface{}) bool {
		validateEntries++
		return true
	})

	up.simpleCache.Range(func(key, value interface{}) bool {
		simpleEntries++
		return true
	})

	return normalizeEntries, validateEntries, simpleEntries
}

// QueryParseError representa um erro de parsing de query
type QueryParseError struct {
	Message string
	Query   string
}

func (e *QueryParseError) Error() string {
	return e.Message + ": " + e.Query
}

// Métodos de compatibilidade para manter a API original do URLParser

// ParseQuery é um alias para ParseQueryFast para compatibilidade
func (up *URLParser) ParseQuery(rawQuery string) (url.Values, error) {
	return up.ParseQueryFast(rawQuery)
}

// ValidateODataQuery é um alias para ValidateODataQueryFast para compatibilidade
func (up *URLParser) ValidateODataQuery(query string) error {
	return up.ValidateODataQueryFast(query)
}

// NormalizeODataQuery é um alias para NormalizeODataQueryFast para compatibilidade
func (up *URLParser) NormalizeODataQuery(query string) string {
	return up.NormalizeODataQueryFast(query)
}

// ExtractODataSystemParams extrai parâmetros do sistema OData ($filter, $orderby, etc.)
func (up *URLParser) ExtractODataSystemParams(values url.Values) map[string]string {
	systemParams := make(map[string]string)

	systemKeys := []string{
		"$filter", "$orderby", "$top", "$skip", "$count",
		"$select", "$expand", "$search", "$compute", "$apply",
	}

	for _, key := range systemKeys {
		// Busca case-insensitive
		for actualKey, vals := range values {
			if strings.EqualFold(actualKey, key) && len(vals) > 0 {
				systemParams[key] = vals[0]
				break
			}
		}
	}

	return systemParams
}

// CleanODataValue limpa um valor OData removendo caracteres desnecessários
func (up *URLParser) CleanODataValue(value string) string {
	if value == "" {
		return ""
	}

	// Remove espaços no início e fim
	value = strings.TrimSpace(value)

	// Remove caracteres de controle
	var result strings.Builder
	result.Grow(len(value))

	for _, char := range value {
		if !unicode.IsControl(char) || char == '\t' || char == '\n' {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// ParseExpandValue faz parsing específico de valores $expand
func (up *URLParser) ParseExpandValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	// Valida a estrutura do expand
	if err := up.ValidateODataQueryFast(value); err != nil {
		return "", err
	}

	// Normaliza o valor
	normalized := up.NormalizeODataQueryFast(value)

	// Limpa o valor
	cleaned := up.CleanODataValue(normalized)

	return cleaned, nil
}

// ParseFilterValue faz parsing específico de valores $filter
func (up *URLParser) ParseFilterValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	// Valida a estrutura do filter
	if err := up.ValidateODataQueryFast(value); err != nil {
		return "", err
	}

	// Normaliza o valor
	normalized := up.NormalizeODataQueryFast(value)

	// Limpa o valor
	cleaned := up.CleanODataValue(normalized)

	return cleaned, nil
}

// ParseODataURL faz o parsing completo de uma URL OData
func (up *URLParser) ParseODataURL(rawURL string) (*url.URL, url.Values, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, err
	}

	// Faz o parsing customizado da query
	values, err := up.ParseQueryFast(parsedURL.RawQuery)
	if err != nil {
		return nil, nil, err
	}

	return parsedURL, values, nil
}

// Métodos auxiliares para compatibilidade

// isBalanced verifica se os caracteres estão balanceados
func (up *URLParser) isBalanced(query string, open, close rune) bool {
	level := 0
	var inQuotes bool
	var quoteChar rune

	for _, char := range query {
		switch char {
		case '\'', '"':
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			}
		default:
			if !inQuotes {
				if char == open {
					level++
				} else if char == close {
					level--
					if level < 0 {
						return false
					}
				}
			}
		}
	}

	return level == 0
}

// isQuotesBalanced verifica se as aspas estão balanceadas
func (up *URLParser) isQuotesBalanced(query string) bool {
	var inSingleQuote, inDoubleQuote bool

	for _, char := range query {
		switch char {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		}
	}

	return !inSingleQuote && !inDoubleQuote
}

// Métodos adicionais para compatibilidade com testes

// splitQueryParams é um alias para splitQueryParamsOptimized para compatibilidade com testes
func (up *URLParser) splitQueryParams(query string) []string {
	return up.splitQueryParamsOptimized(query)
}

// parseParam é um alias para parseParamOptimized para compatibilidade com testes
func (up *URLParser) parseParam(param string) (string, string) {
	return up.parseParamOptimized(param)
}
