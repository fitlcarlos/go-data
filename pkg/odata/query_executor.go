package odata

import (
	"context"
	"database/sql"
	"fmt"
)

// =======================================================================================
// QUERY EXECUTION
// =======================================================================================

// executeQuery executa uma query SQL com contexto e retorna as rows
func (s *BaseEntityService) executeQuery(ctx context.Context, query string, args []any) (*sql.Rows, error) {
	// Verifica se a conexão está disponível
	conn := s.provider.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("database connection is nil - make sure the provider is properly connected")
	}

	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
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

	return conn.ExecContext(ctx, query, args...)
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
	row := s.provider.GetConnection().QueryRowContext(ctx, query, args...)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return count, nil
}

// parseFilterWithTimeout analisa uma string de filtro OData com timeout
func (s *BaseEntityService) parseFilterWithTimeout(ctx context.Context, filter string) (*GoDataFilterQuery, error) {
	return ParseFilterString(ctx, filter)
}
