package odata

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

// BatchRequest representa uma requisição batch OData
type BatchRequest struct {
	Parts []*BatchPart
}

// BatchPart representa uma parte individual do batch (request ou changeset)
type BatchPart struct {
	IsChangeset bool
	Request     *BatchHTTPOperation
	Changeset   []*BatchHTTPOperation
}

// BatchHTTPOperation representa uma operação HTTP individual no batch
type BatchHTTPOperation struct {
	Method    string
	URL       string
	Headers   map[string]string
	Body      []byte
	ContentID string // Para referências dentro do batch
}

// BatchResponse representa a resposta de um batch
type BatchResponse struct {
	Parts []*BatchResponsePart
}

// BatchResponsePart representa uma parte da resposta
type BatchResponsePart struct {
	IsChangeset bool
	Response    *BatchOperationResponse
	Changeset   []*BatchOperationResponse
}

// BatchOperationResponse representa a resposta de uma operação
type BatchOperationResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	ContentID  string
}

// BatchProcessor processa requisições batch
type BatchProcessor struct {
	server *Server
}

// NewBatchProcessor cria um novo processador de batch
func NewBatchProcessor(server *Server) *BatchProcessor {
	return &BatchProcessor{
		server: server,
	}
}

// ParseBatchRequest faz o parsing de uma requisição batch multipart/mixed
func (bp *BatchProcessor) ParseBatchRequest(c fiber.Ctx) (*BatchRequest, error) {
	contentType := c.Get("Content-Type")
	if contentType == "" {
		return nil, fmt.Errorf("Content-Type header is required for batch requests")
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Type: %w", err)
	}

	if mediaType != "multipart/mixed" {
		return nil, fmt.Errorf("Content-Type must be multipart/mixed, got: %s", mediaType)
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("boundary parameter is required")
	}

	body := c.Body()
	reader := multipart.NewReader(bytes.NewReader(body), boundary)

	batchReq := &BatchRequest{
		Parts: make([]*BatchPart, 0),
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading multipart: %w", err)
		}

		partContentType := part.Header.Get("Content-Type")
		partBody, err := io.ReadAll(part)
		if err != nil {
			return nil, fmt.Errorf("error reading part body: %w", err)
		}

		// Verificar se é um changeset (multipart/mixed) ou uma requisição simples
		if strings.HasPrefix(partContentType, "multipart/mixed") {
			// É um changeset
			changesetPart, err := bp.parseChangeset(partContentType, partBody)
			if err != nil {
				return nil, fmt.Errorf("error parsing changeset: %w", err)
			}
			batchReq.Parts = append(batchReq.Parts, changesetPart)
		} else {
			// É uma requisição simples (GET)
			operation, err := bp.parseOperation(partBody)
			if err != nil {
				return nil, fmt.Errorf("error parsing operation: %w", err)
			}
			batchReq.Parts = append(batchReq.Parts, &BatchPart{
				IsChangeset: false,
				Request:     operation,
			})
		}
	}

	return batchReq, nil
}

// parseChangeset faz o parsing de um changeset (transacional)
func (bp *BatchProcessor) parseChangeset(contentType string, body []byte) (*BatchPart, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("invalid changeset Content-Type: %w", err)
	}

	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("changeset boundary parameter is required")
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)

	changeset := &BatchPart{
		IsChangeset: true,
		Changeset:   make([]*BatchHTTPOperation, 0),
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading changeset part: %w", err)
		}

		partBody, err := io.ReadAll(part)
		if err != nil {
			return nil, fmt.Errorf("error reading changeset part body: %w", err)
		}

		operation, err := bp.parseOperation(partBody)
		if err != nil {
			return nil, fmt.Errorf("error parsing changeset operation: %w", err)
		}

		changeset.Changeset = append(changeset.Changeset, operation)
	}

	return changeset, nil
}

// parseOperation faz o parsing de uma operação individual HTTP
func (bp *BatchProcessor) parseOperation(body []byte) (*BatchHTTPOperation, error) {
	lines := strings.Split(string(body), "\r\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid operation format")
	}

	// Primeira linha: METHOD URL HTTP/1.1
	requestLine := strings.Fields(lines[0])
	if len(requestLine) < 2 {
		return nil, fmt.Errorf("invalid request line: %s", lines[0])
	}

	method := requestLine[0]
	urlPath := requestLine[1]

	operation := &BatchHTTPOperation{
		Method:  method,
		URL:     urlPath,
		Headers: make(map[string]string),
	}

	// Parse headers
	i := 1
	for ; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			// Linha vazia marca o fim dos headers
			i++
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headerName := strings.TrimSpace(parts[0])
			headerValue := strings.TrimSpace(parts[1])
			operation.Headers[headerName] = headerValue

			// Extrair Content-ID se presente
			if strings.EqualFold(headerName, "Content-ID") {
				operation.ContentID = headerValue
			}
		}
	}

	// O resto é o body (se houver)
	if i < len(lines) {
		bodyLines := lines[i:]
		operation.Body = []byte(strings.Join(bodyLines, "\r\n"))
	}

	return operation, nil
}

// ExecuteBatch executa um batch request
func (bp *BatchProcessor) ExecuteBatch(ctx context.Context, batchReq *BatchRequest) (*BatchResponse, error) {
	batchResp := &BatchResponse{
		Parts: make([]*BatchResponsePart, 0),
	}

	// Mapa para armazenar referências de Content-ID
	contentIDMap := make(map[string]interface{})

	for _, part := range batchReq.Parts {
		if part.IsChangeset {
			// Executar changeset (transacional)
			changesetResp, err := bp.executeChangeset(ctx, part.Changeset, contentIDMap)
			if err != nil {
				// Se changeset falhar, retornar erro para todas as operações
				failedResp := make([]*BatchOperationResponse, len(part.Changeset))
				for i := range failedResp {
					failedResp[i] = &BatchOperationResponse{
						StatusCode: http.StatusInternalServerError,
						Headers:    map[string]string{"Content-Type": "application/json"},
						Body:       []byte(fmt.Sprintf(`{"error": {"message": "Changeset failed: %s"}}`, err.Error())),
					}
				}
				batchResp.Parts = append(batchResp.Parts, &BatchResponsePart{
					IsChangeset: true,
					Changeset:   failedResp,
				})
			} else {
				batchResp.Parts = append(batchResp.Parts, &BatchResponsePart{
					IsChangeset: true,
					Changeset:   changesetResp,
				})
			}
		} else {
			// Executar requisição simples
			resp, err := bp.executeOperation(ctx, part.Request, contentIDMap)
			if err != nil {
				resp = &BatchOperationResponse{
					StatusCode: http.StatusInternalServerError,
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       []byte(fmt.Sprintf(`{"error": {"message": "%s"}}`, err.Error())),
				}
			}
			batchResp.Parts = append(batchResp.Parts, &BatchResponsePart{
				IsChangeset: false,
				Response:    resp,
			})
		}
	}

	return batchResp, nil
}

// executeChangeset executa um changeset (transacional)
func (bp *BatchProcessor) executeChangeset(ctx context.Context, operations []*BatchHTTPOperation, contentIDMap map[string]interface{}) ([]*BatchOperationResponse, error) {
	responses := make([]*BatchOperationResponse, len(operations))

	// Obter database provider padrão
	provider := bp.server.provider
	if provider == nil {
		return nil, fmt.Errorf("no database provider configured")
	}

	// Iniciar transação
	tx, err := provider.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		// Rollback automático se não houver commit explícito
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Executar operações dentro da transação
	for i, op := range operations {
		resp, err := bp.executeOperationInTx(ctx, tx, op, contentIDMap)
		if err != nil {
			// Se uma operação falha, rollback automático via defer
			return nil, fmt.Errorf("operation %d failed (rolled back): %w", i, err)
		}

		// Se status code indica erro, rollback
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("operation %d returned error status %d (rolled back)", i, resp.StatusCode)
		}

		responses[i] = resp

		// Armazenar Content-ID para referências futuras
		if op.ContentID != "" {
			contentIDMap[op.ContentID] = resp
		}
	}

	// Se chegou aqui, todas as operações tiveram sucesso - commit
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Marcar tx como nil para evitar rollback no defer
	tx = nil

	return responses, nil
}

// executeOperation executa uma operação individual
func (bp *BatchProcessor) executeOperation(ctx context.Context, op *BatchHTTPOperation, contentIDMap map[string]interface{}) (*BatchOperationResponse, error) {
	// Resolver referências de Content-ID no URL
	_ = bp.resolveContentID(op.URL, contentIDMap)

	// TODO: Implementar execução real da operação
	// Por enquanto, retornar sucesso
	resp := &BatchOperationResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:      []byte(`{"@odata.context": "$metadata", "value": "Success"}`),
		ContentID: op.ContentID,
	}

	return resp, nil
}

// resolveContentID resolve referências de Content-ID em URLs (ex: $1, $2)
func (bp *BatchProcessor) resolveContentID(url string, contentIDMap map[string]interface{}) string {
	// Procurar por referências $<id> no URL
	// Padrões suportados: $1, $2, ${1}, ${2}
	// Por exemplo: /Products($1)/Items ou /Orders(${1})

	// Se não há $ no URL, retornar original
	if !strings.Contains(url, "$") {
		return url
	}

	// Substituir todas as referências de Content-ID
	resolved := url
	for contentID, response := range contentIDMap {
		if batchResp, ok := response.(*BatchOperationResponse); ok {
			// Extrair ID da resposta JSON
			id := extractIDFromResponse(batchResp.Body)
			if id != "" {
				// Substituir $<contentID> e ${<contentID>}
				resolved = strings.ReplaceAll(resolved, fmt.Sprintf("$%s", contentID), id)
				resolved = strings.ReplaceAll(resolved, fmt.Sprintf("${%s}", contentID), id)
			}
		}
	}

	return resolved
}

// extractIDFromResponse extrai o ID de uma resposta JSON
func extractIDFromResponse(body []byte) string {
	// Parse JSON response
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	// Tentar encontrar ID em várias formas
	// Ordem de preferência: id, ID, Id, @odata.id
	if id, ok := data["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	if id, ok := data["ID"]; ok {
		return fmt.Sprintf("%v", id)
	}
	if id, ok := data["Id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	if odataID, ok := data["@odata.id"]; ok {
		// @odata.id pode ser uma URL completa como "/Products(123)"
		// Extrair apenas o ID
		idStr := fmt.Sprintf("%v", odataID)
		if strings.Contains(idStr, "(") && strings.Contains(idStr, ")") {
			start := strings.Index(idStr, "(")
			end := strings.Index(idStr, ")")
			if start < end {
				return strings.Trim(idStr[start+1:end], "'\"")
			}
		}
		return idStr
	}

	return ""
}

// executeOperationInTx executa uma operação dentro de uma transação
func (bp *BatchProcessor) executeOperationInTx(ctx context.Context, tx *sql.Tx, op *BatchHTTPOperation, contentIDMap map[string]interface{}) (*BatchOperationResponse, error) {
	// Resolver referências de Content-ID
	url := bp.resolveContentID(op.URL, contentIDMap)

	// Parse URL e extrair entidade/ID
	entityName, entityID, err := bp.parseOperationURL(url)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"BadRequest","message":"Invalid URL: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Obter entity service
	service := bp.server.GetEntityService(entityName)
	if service == nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusNotFound,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"NotFound","message":"Entity not found: %s"}}`, entityName)),
			ContentID:  op.ContentID,
		}, nil
	}

	// Executar operação baseado no método HTTP
	switch op.Method {
	case "POST":
		return bp.executeCreate(ctx, tx, service, op)
	case "PUT", "PATCH":
		return bp.executeUpdate(ctx, tx, service, entityID, op)
	case "DELETE":
		return bp.executeDelete(ctx, tx, service, entityID, op)
	default:
		return &BatchOperationResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"MethodNotAllowed","message":"Unsupported method in changeset: %s"}}`, op.Method)),
			ContentID:  op.ContentID,
		}, nil
	}
}

// parseOperationURL extrai entity name e ID de uma URL
func (bp *BatchProcessor) parseOperationURL(url string) (string, string, error) {
	// Remove prefixo /odata/ ou /api/v1/ se presente
	url = strings.TrimPrefix(url, "/odata/")
	url = strings.TrimPrefix(url, "/api/v1/")
	url = strings.TrimPrefix(url, "/")

	// Parse: /EntityName ou /EntityName(id) ou /EntityName('id')
	parts := strings.Split(url, "(")
	entityName := parts[0]

	var entityID string
	if len(parts) > 1 {
		// Extrair ID: (123) ou ('abc')
		idPart := strings.TrimSuffix(parts[1], ")")
		entityID = strings.Trim(idPart, "'\"")
	}

	return entityName, entityID, nil
}

// executeCreate executa um CREATE dentro da transação
func (bp *BatchProcessor) executeCreate(ctx context.Context, tx *sql.Tx, service EntityService, op *BatchHTTPOperation) (*BatchOperationResponse, error) {
	// Parse JSON body
	var entity map[string]interface{}
	if err := json.Unmarshal(op.Body, &entity); err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"BadRequest","message":"Invalid JSON: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Obter metadata
	metadata := service.GetMetadata()

	// Build INSERT query
	query, args, err := bp.server.provider.BuildInsertQuery(metadata, entity)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"InternalError","message":"Failed to build query: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Execute INSERT dentro da transação
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"InternalError","message":"Failed to insert: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Obter ID gerado
	lastID, err := result.LastInsertId()
	if err != nil {
		// Alguns bancos não suportam LastInsertId, usar ID do entity se disponível
		if id, ok := entity["id"]; ok {
			entity["ID"] = id
		} else if id, ok := entity["ID"]; ok {
			entity["ID"] = id
		}
	} else {
		entity["ID"] = lastID
	}

	// Serializar resposta
	respBody, err := json.Marshal(entity)
	if err != nil {
		respBody = []byte(fmt.Sprintf(`{"ID":%d}`, lastID))
	}

	return &BatchOperationResponse{
		StatusCode: http.StatusCreated,
		Headers: map[string]string{
			"Content-Type": "application/json",
			"Location":     fmt.Sprintf("/%s(%v)", metadata.Name, entity["ID"]),
		},
		Body:      respBody,
		ContentID: op.ContentID,
	}, nil
}

// executeUpdate executa um UPDATE dentro da transação
func (bp *BatchProcessor) executeUpdate(ctx context.Context, tx *sql.Tx, service EntityService, entityID string, op *BatchHTTPOperation) (*BatchOperationResponse, error) {
	// Parse JSON body
	var updates map[string]interface{}
	if err := json.Unmarshal(op.Body, &updates); err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"BadRequest","message":"Invalid JSON: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Obter metadata
	metadata := service.GetMetadata()

	// Determinar chave primária
	var keyColumn string
	var keyProperty string
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			keyColumn = prop.ColumnName
			if keyColumn == "" {
				keyColumn = prop.Name
			}
			keyProperty = prop.Name
			break
		}
	}

	if keyColumn == "" {
		// Fallback para "id" ou "ID"
		keyColumn = "id"
		keyProperty = "ID"
	}

	// Criar mapa de chaves
	keyValues := map[string]interface{}{
		keyProperty: entityID,
	}

	// Build UPDATE query
	query, args, err := bp.server.provider.BuildUpdateQuery(metadata, updates, keyValues)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"InternalError","message":"Failed to build query: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Execute UPDATE dentro da transação
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"InternalError","message":"Failed to update: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Verificar se alguma linha foi afetada
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return &BatchOperationResponse{
			StatusCode: http.StatusNotFound,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"error":{"code":"NotFound","message":"Entity not found"}}`),
			ContentID:  op.ContentID,
		}, nil
	}

	// Adicionar ID ao resultado
	updates[keyProperty] = entityID

	// Serializar resposta
	respBody, err := json.Marshal(updates)
	if err != nil {
		respBody = []byte(fmt.Sprintf(`{"%s":"%s"}`, keyProperty, entityID))
	}

	return &BatchOperationResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:      respBody,
		ContentID: op.ContentID,
	}, nil
}

// executeDelete executa um DELETE dentro da transação
func (bp *BatchProcessor) executeDelete(ctx context.Context, tx *sql.Tx, service EntityService, entityID string, op *BatchHTTPOperation) (*BatchOperationResponse, error) {
	// Obter metadata
	metadata := service.GetMetadata()

	// Determinar chave primária
	var keyColumn string
	var keyProperty string
	for _, prop := range metadata.Properties {
		if prop.IsKey {
			keyColumn = prop.ColumnName
			if keyColumn == "" {
				keyColumn = prop.Name
			}
			keyProperty = prop.Name
			break
		}
	}

	if keyColumn == "" {
		// Fallback para "id" ou "ID"
		keyColumn = "id"
		keyProperty = "ID"
	}

	// Criar mapa de chaves
	keyValues := map[string]interface{}{
		keyProperty: entityID,
	}

	// Build DELETE query
	query, args, err := bp.server.provider.BuildDeleteQuery(metadata, keyValues)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"InternalError","message":"Failed to build query: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Execute DELETE dentro da transação
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return &BatchOperationResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":{"code":"InternalError","message":"Failed to delete: %s"}}`, err.Error())),
			ContentID:  op.ContentID,
		}, nil
	}

	// Verificar se alguma linha foi afetada
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return &BatchOperationResponse{
			StatusCode: http.StatusNotFound,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"error":{"code":"NotFound","message":"Entity not found"}}`),
			ContentID:  op.ContentID,
		}, nil
	}

	// DELETE bem sucedido retorna 204 No Content
	return &BatchOperationResponse{
		StatusCode: http.StatusNoContent,
		Headers:    map[string]string{},
		Body:       []byte{},
		ContentID:  op.ContentID,
	}, nil
}

// WriteBatchResponse escreve a resposta batch no formato multipart/mixed
func (bp *BatchProcessor) WriteBatchResponse(c fiber.Ctx, batchResp *BatchResponse) error {
	boundary := fmt.Sprintf("batchresponse_%d", time.Now().UnixNano())

	c.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.SetBoundary(boundary)

	for partIndex, part := range batchResp.Parts {
		if part.IsChangeset {
			// Escrever changeset
			changesetBoundary := fmt.Sprintf("changeset_%d_%d", time.Now().UnixNano(), partIndex)

			partWriter, err := writer.CreatePart(map[string][]string{
				"Content-Type": {fmt.Sprintf("multipart/mixed; boundary=%s", changesetBoundary)},
			})
			if err != nil {
				return err
			}

			changesetBuf := &bytes.Buffer{}
			changesetWriter := multipart.NewWriter(changesetBuf)
			changesetWriter.SetBoundary(changesetBoundary)

			for _, opResp := range part.Changeset {
				if err := bp.writeOperationResponse(changesetWriter, opResp); err != nil {
					return err
				}
			}

			changesetWriter.Close()
			partWriter.Write(changesetBuf.Bytes())
		} else {
			// Escrever resposta simples
			if err := bp.writeOperationResponse(writer, part.Response); err != nil {
				return err
			}
		}
	}

	writer.Close()

	return c.Send(buf.Bytes())
}

// writeOperationResponse escreve uma resposta de operação no multipart writer
func (bp *BatchProcessor) writeOperationResponse(writer *multipart.Writer, resp *BatchOperationResponse) error {
	headers := map[string][]string{
		"Content-Type":              {"application/http"},
		"Content-Transfer-Encoding": {"binary"},
	}

	if resp.ContentID != "" {
		headers["Content-ID"] = []string{resp.ContentID}
	}

	partWriter, err := writer.CreatePart(headers)
	if err != nil {
		return err
	}

	// Escrever linha de status HTTP
	fmt.Fprintf(partWriter, "HTTP/1.1 %d %s\r\n", resp.StatusCode, http.StatusText(resp.StatusCode))

	// Escrever headers
	for key, value := range resp.Headers {
		fmt.Fprintf(partWriter, "%s: %s\r\n", key, value)
	}

	// Linha vazia entre headers e body
	fmt.Fprint(partWriter, "\r\n")

	// Escrever body
	partWriter.Write(resp.Body)

	return nil
}

// HandleBatch é o handler para requisições $batch
func (s *Server) HandleBatch(c fiber.Ctx) error {
	ctx := c.Context()

	processor := NewBatchProcessor(s)

	// Parse batch request
	batchReq, err := processor.ParseBatchRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "BadRequest",
				"message": fmt.Sprintf("Invalid batch request: %v", err),
			},
		})
	}

	// Execute batch
	batchResp, err := processor.ExecuteBatch(ctx, batchReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "InternalServerError",
				"message": fmt.Sprintf("Error executing batch: %v", err),
			},
		})
	}

	// Write batch response
	return processor.WriteBatchResponse(c, batchResp)
}

// BatchConfig configurações para batch requests
type BatchConfig struct {
	MaxOperations      int           // Máximo de operações por batch
	MaxChangesets      int           // Máximo de changesets por batch
	Timeout            time.Duration // Timeout para execução do batch
	EnableTransactions bool          // Habilitar transações para changesets
}

// DefaultBatchConfig retorna configuração padrão
func DefaultBatchConfig() *BatchConfig {
	return &BatchConfig{
		MaxOperations:      100,
		MaxChangesets:      10,
		Timeout:            30 * time.Second,
		EnableTransactions: true,
	}
}
