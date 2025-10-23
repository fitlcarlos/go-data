package odata

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// AuditOperation representa o tipo de operação auditada
type AuditOperation string

const (
	AuditOpCreate       AuditOperation = "CREATE"
	AuditOpUpdate       AuditOperation = "UPDATE"
	AuditOpDelete       AuditOperation = "DELETE"
	AuditOpRead         AuditOperation = "READ"
	AuditOpAuthSuccess  AuditOperation = "AUTH_SUCCESS"
	AuditOpAuthFailure  AuditOperation = "AUTH_FAILURE"
	AuditOpAuthLogout   AuditOperation = "AUTH_LOGOUT"
	AuditOpUnauthorized AuditOperation = "UNAUTHORIZED"
)

// AuditLogConfig configurações de audit logging
type AuditLogConfig struct {
	// Habilita/desabilita audit logging
	Enabled bool

	// Tipo de logger: "file", "stdout", "stderr", "none"
	LogType string

	// Caminho do arquivo de log (quando LogType = "file")
	FilePath string

	// Formato do log: "json" ou "text"
	Format string

	// Buffer size para escrita assíncrona
	BufferSize int

	// Operações a serem logadas (se vazio, loga todas)
	LoggedOperations []AuditOperation

	// Incluir dados sensíveis no log (não recomendado em produção)
	IncludeSensitiveData bool
}

// DefaultAuditLogConfig retorna configuração padrão de audit logging
func DefaultAuditLogConfig() *AuditLogConfig {
	return &AuditLogConfig{
		Enabled:              false, // Desabilitado por padrão, deve ser explicitamente habilitado
		LogType:              "stdout",
		FilePath:             "audit.log",
		Format:               "json",
		BufferSize:           100,
		LoggedOperations:     []AuditOperation{}, // vazio = loga todas
		IncludeSensitiveData: false,
	}
}

// AuditLogEntry representa uma entrada de audit log
type AuditLogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	UserID       string                 `json:"user_id,omitempty"`
	Username     string                 `json:"username,omitempty"`
	IP           string                 `json:"ip"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	EntityName   string                 `json:"entity_name,omitempty"`
	Operation    AuditOperation         `json:"operation"`
	EntityID     string                 `json:"entity_id,omitempty"`
	Success      bool                   `json:"success"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Duration     int64                  `json:"duration_ms,omitempty"` // duração em milissegundos
	UserAgent    string                 `json:"user_agent,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// AuditLogger interface para audit logging
type AuditLogger interface {
	Log(entry AuditLogEntry) error
	Close() error
}

// FileAuditLogger implementa AuditLogger escrevendo em arquivo
type FileAuditLogger struct {
	config *AuditLogConfig
	file   *os.File
	mu     sync.Mutex
	buffer chan AuditLogEntry
	done   chan bool
	wg     sync.WaitGroup
}

// StdoutAuditLogger implementa AuditLogger escrevendo em stdout
type StdoutAuditLogger struct {
	config *AuditLogConfig
	mu     sync.Mutex
}

// StderrAuditLogger implementa AuditLogger escrevendo em stderr
type StderrAuditLogger struct {
	config *AuditLogConfig
	mu     sync.Mutex
}

// NoOpAuditLogger implementa AuditLogger mas não faz nada (quando audit logging está desabilitado)
type NoOpAuditLogger struct{}

// NewAuditLogger cria uma nova instância de AuditLogger baseado na configuração
func NewAuditLogger(config *AuditLogConfig) (AuditLogger, error) {
	if config == nil {
		config = DefaultAuditLogConfig()
	}

	if !config.Enabled {
		return &NoOpAuditLogger{}, nil
	}

	switch config.LogType {
	case "file":
		return NewFileAuditLogger(config)
	case "stdout":
		return &StdoutAuditLogger{config: config}, nil
	case "stderr":
		return &StderrAuditLogger{config: config}, nil
	case "none":
		return &NoOpAuditLogger{}, nil
	default:
		return nil, fmt.Errorf("unknown audit log type: %s", config.LogType)
	}
}

// NewFileAuditLogger cria um novo FileAuditLogger
func NewFileAuditLogger(config *AuditLogConfig) (*FileAuditLogger, error) {
	file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	logger := &FileAuditLogger{
		config: config,
		file:   file,
		buffer: make(chan AuditLogEntry, config.BufferSize),
		done:   make(chan bool),
	}

	// Inicia goroutine para escrita assíncrona
	logger.wg.Add(1)
	go logger.writeLoop()

	return logger, nil
}

// Log adiciona uma entrada ao log
func (l *FileAuditLogger) Log(entry AuditLogEntry) error {
	// Verifica se deve logar esta operação
	if !l.shouldLog(entry.Operation) {
		return nil
	}

	// Tenta adicionar ao buffer (non-blocking)
	select {
	case l.buffer <- entry:
		return nil
	default:
		// Buffer cheio, escreve diretamente (blocking)
		return l.writeEntry(entry)
	}
}

// writeLoop processa entradas do buffer
func (l *FileAuditLogger) writeLoop() {
	defer l.wg.Done()

	for {
		select {
		case entry := <-l.buffer:
			if err := l.writeEntry(entry); err != nil {
				log.Printf("Error writing audit log: %v", err)
			}
		case <-l.done:
			// Drena buffer antes de sair
			for len(l.buffer) > 0 {
				entry := <-l.buffer
				if err := l.writeEntry(entry); err != nil {
					log.Printf("Error writing audit log: %v", err)
				}
			}
			return
		}
	}
}

// writeEntry escreve uma entrada no arquivo
func (l *FileAuditLogger) writeEntry(entry AuditLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var line string
	var err error

	if l.config.Format == "json" {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal audit log entry: %w", err)
		}
		line = string(data) + "\n"
	} else {
		// Formato text
		line = l.formatTextEntry(entry) + "\n"
	}

	_, err = l.file.WriteString(line)
	if err != nil {
		return fmt.Errorf("failed to write audit log entry: %w", err)
	}

	// Flush para garantir escrita
	return l.file.Sync()
}

// formatTextEntry formata entrada como texto legível
func (l *FileAuditLogger) formatTextEntry(entry AuditLogEntry) string {
	status := "SUCCESS"
	if !entry.Success {
		status = "FAILURE"
	}

	return fmt.Sprintf("[%s] %s %s | User: %s (%s) | IP: %s | Entity: %s | ID: %s | Duration: %dms | Error: %s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Operation,
		status,
		entry.Username,
		entry.UserID,
		entry.IP,
		entry.EntityName,
		entry.EntityID,
		entry.Duration,
		entry.ErrorMessage,
	)
}

// shouldLog verifica se deve logar esta operação
func (l *FileAuditLogger) shouldLog(op AuditOperation) bool {
	if len(l.config.LoggedOperations) == 0 {
		return true // Loga todas se lista estiver vazia
	}

	for _, logged := range l.config.LoggedOperations {
		if logged == op {
			return true
		}
	}

	return false
}

// Close fecha o logger e aguarda escrita de buffer
func (l *FileAuditLogger) Close() error {
	close(l.done)
	l.wg.Wait()

	l.mu.Lock()
	defer l.mu.Unlock()

	return l.file.Close()
}

// Log implementa AuditLogger para StdoutAuditLogger
func (l *StdoutAuditLogger) Log(entry AuditLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.config.Format == "json" {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(l.formatTextEntry(entry))
	}

	return nil
}

// formatTextEntry formata entrada como texto
func (l *StdoutAuditLogger) formatTextEntry(entry AuditLogEntry) string {
	statusIcon := "✅"
	if !entry.Success {
		statusIcon = "❌"
	}

	return fmt.Sprintf("%s [AUDIT] %s %s | User: %s | IP: %s | Entity: %s(%s)",
		statusIcon,
		entry.Timestamp.Format("15:04:05"),
		entry.Operation,
		entry.Username,
		entry.IP,
		entry.EntityName,
		entry.EntityID,
	)
}

// Close implementa AuditLogger para StdoutAuditLogger
func (l *StdoutAuditLogger) Close() error {
	return nil
}

// Log implementa AuditLogger para StderrAuditLogger
func (l *StderrAuditLogger) Log(entry AuditLogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.config.Format == "json" {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintln(os.Stderr, l.formatTextEntry(entry))
	}

	return nil
}

// formatTextEntry formata entrada como texto
func (l *StderrAuditLogger) formatTextEntry(entry AuditLogEntry) string {
	status := "SUCCESS"
	if !entry.Success {
		status = "FAILURE"
	}

	return fmt.Sprintf("[AUDIT] %s %s %s | User: %s | IP: %s | Entity: %s(%s) | Error: %s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Operation,
		status,
		entry.Username,
		entry.IP,
		entry.EntityName,
		entry.EntityID,
		entry.ErrorMessage,
	)
}

// Close implementa AuditLogger para StderrAuditLogger
func (l *StderrAuditLogger) Close() error {
	return nil
}

// Log implementa AuditLogger para NoOpAuditLogger (não faz nada)
func (l *NoOpAuditLogger) Log(entry AuditLogEntry) error {
	return nil
}

// Close implementa AuditLogger para NoOpAuditLogger
func (l *NoOpAuditLogger) Close() error {
	return nil
}
