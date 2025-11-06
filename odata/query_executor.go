package odata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// =======================================================================================
// QUERY EXECUTION
// =======================================================================================

// executeQuery executa uma query SQL com contexto e retorna as rows
func (s *BaseEntityService) executeQuery(ctx context.Context, query string, args []any) (*sql.Rows, error) {
	// Verifica se a conexão está disponível (GetConnection já faz ping e valida)
	conn := s.provider.GetConnection()

	if conn == nil {
		return nil, fmt.Errorf("database connection is nil - make sure the provider is properly connected")
	}

	// Log da query SQL se DB_LOG_SQL estiver habilitado
	if s.shouldLogSQL() {
		log.Printf("🔍 [SQL] QUERY: %s", query)
		if len(args) > 0 {
			log.Printf("🔍 [SQL] ARGS: %v", args)
		}
	}

	// Executa a query
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		if s.shouldLogSQL() {
			log.Printf("❌ [SQL] ERRO: %v", err)
		}
		return nil, err
	}

	return rows, nil
}

// executeExec executa um comando SQL (INSERT, UPDATE, DELETE) com contexto
func (s *BaseEntityService) executeExec(ctx context.Context, query string, args []any) (sql.Result, error) {
	// Verifica se a conexão está disponível
	conn := s.provider.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("database connection is nil - make sure the provider is properly connected")
	}

	// Log da query SQL se DB_LOG_SQL estiver habilitado
	if s.shouldLogSQL() {
		log.Printf("🔍 [SQL] EXEC: %s", query)
		if len(args) > 0 {
			log.Printf("🔍 [SQL] ARGS: %v", args)
		}
	}

	result, err := conn.ExecContext(ctx, query, args...)
	if err != nil && s.shouldLogSQL() {
		log.Printf("❌ [SQL] ERRO: %v", err)
	}
	return result, err
}

// GetCount retorna a contagem de registros que atendem às opções de consulta
func (s *BaseEntityService) GetCount(ctx context.Context, options QueryOptions) (int64, error) {
	// Constrói a query de count usando o provider
	tableName := s.metadata.TableName
	if tableName == "" {
		tableName = s.metadata.Name
	}

	// Usa o provider para construir a cláusula WHERE corretamente
	var whereClause string
	var args []any
	var err error

	if options.Filter != nil && options.Filter.Tree != nil {
		whereClause, args, err = ConvertFilterToSQL(ctx, options.Filter, s.metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to build where clause for count: %w", err)
		}
	}

	// Constrói a query COUNT com o provider
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	// Executa a query
	conn := s.provider.GetConnection()
	if conn == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	// Log da query SQL se DB_LOG_SQL estiver habilitado
	if s.shouldLogSQL() {
		log.Printf("🔍 [SQL] COUNT: %s", query)
		if len(args) > 0 {
			log.Printf("🔍 [SQL] ARGS: %v", args)
		}
	}

	row := conn.QueryRowContext(ctx, query, args...)

	var count int64
	if err := row.Scan(&count); err != nil {
		if s.shouldLogSQL() {
			log.Printf("❌ [SQL] ERRO: %v", err)
		}
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return count, nil
}

// shouldLogSQL verifica se logs SQL devem ser habilitados
func (s *BaseEntityService) shouldLogSQL() bool {
	if s.server == nil {
		return false
	}
	
	// Pega o config do server
	config := s.server.GetConfig()
	if config == nil {
		return false
	}
	
	// Verifica se DB_LOG_SQL está habilitado
	return config.DBLogSQL
}

// parseFilterWithTimeout analisa uma string de filtro OData com timeout
func (s *BaseEntityService) parseFilterWithTimeout(ctx context.Context, filter string) (*GoDataFilterQuery, error) {
	return ParseFilterString(ctx, filter)
}
