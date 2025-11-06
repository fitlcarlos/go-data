package odata

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	maxPayloadSize = 10000 // Limite para truncamento de payloads muito grandes (10KB)
)

// PayloadLoggerMiddleware retorna um middleware que loga request e response payloads quando habilitado
func (s *Server) PayloadLoggerMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Verifica se o logging de payloads está habilitado
		config, err := LoadEnvOrDefault()
		if err != nil || !config.LogPayloads {
			// Se desabilitado ou erro ao carregar config, apenas continua sem logar
			return c.Next()
		}

		method := c.Method()
		path := c.Path()

		// Captura o request body
		requestBody := c.Body()
		requestContentType := c.Get("Content-Type")

		// Log do request
		if len(requestBody) > 0 {
			formattedRequest := formatPayload(requestBody, requestContentType)
			log.Printf("━━━━ [REQUEST] %s %s ━━━━\nContent-Type: %s\n%s",
				method, path, requestContentType, formattedRequest)
		} else {
			log.Printf("━━━━ [REQUEST] %s %s ━━━━\n(empty body)",
				method, path)
		}

		// Executa o próximo handler
		err = c.Next()

		// Captura a response após processamento
		// No Fiber v3, o body da response pode ser obtido após Next()
		responseContentType := c.Get("Content-Type")
		statusCode := c.Response().StatusCode()

		// Tenta obter o body da response
		// Nota: No Fiber, após c.Next() o body pode já ter sido enviado
		// Vamos tentar capturar do Response().Body()
		responseBody := c.Response().Body()

		if len(responseBody) > 0 {
			formattedResponse := formatPayload(responseBody, responseContentType)
			log.Printf("━━━━ [RESPONSE] %s %s - Status: %d ━━━━\nContent-Type: %s\n%s",
				method, path, statusCode, responseContentType, formattedResponse)
		} else {
			log.Printf("━━━━ [RESPONSE] %s %s - Status: %d ━━━━\n(empty body)",
				method, path, statusCode)
		}

		return err
	}
}

// formatPayload formata o payload baseado no Content-Type
func formatPayload(body []byte, contentType string) string {
	// Trunca se muito grande
	if len(body) > maxPayloadSize {
		truncated := body[:maxPayloadSize]
		return fmt.Sprintf("%s\n\n... [truncated, original size: %d bytes]", formatPayloadContent(truncated, contentType), len(body))
	}

	return formatPayloadContent(body, contentType)
}

// formatPayloadContent formata o conteúdo baseado no tipo
func formatPayloadContent(body []byte, contentType string) string {
	if len(body) == 0 {
		return "(empty)"
	}

	// Remove parâmetros do Content-Type (ex: "application/json; charset=utf-8")
	baseContentType := strings.Split(contentType, ";")[0]
	baseContentType = strings.TrimSpace(baseContentType)

	// Tenta formatar JSON
	if strings.Contains(baseContentType, "json") || strings.Contains(baseContentType, "application/json") {
		var jsonValue interface{}
		if err := json.Unmarshal(body, &jsonValue); err == nil {
			// Conseguiu fazer parse como JSON, formata com indentação
			prettyJSON, err := json.MarshalIndent(jsonValue, "", "  ")
			if err == nil {
				return string(prettyJSON)
			}
		}
		// Se não conseguiu parsear/formatar, retorna como string
		return string(body)
	}

	// Para outros tipos, tenta decodificar como string UTF-8
	// Se falhar, mostra como bytes hexadecimais ou string com caracteres especiais
	if utf8String := string(body); isPrintableUTF8(utf8String) {
		return utf8String
	}

	// Se não for texto legível, mostra como hex
	return fmt.Sprintf("[binary data, %d bytes]", len(body))
}

// isPrintableUTF8 verifica se a string contém apenas caracteres UTF-8 imprimíveis
func isPrintableUTF8(s string) bool {
	for _, r := range s {
		if r > 0x7E || (r < 0x20 && r != '\n' && r != '\r' && r != '\t') {
			return false
		}
	}
	return true
}
