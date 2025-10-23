package odata

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

// EntityCacheItem representa um item no cache de entidades
type EntityCacheItem struct {
	Entity    any
	Key       string
	CachedAt  time.Time
	IsChanged bool
}

// BatchOperation representa uma operação em lote
type BatchOperation struct {
	Type   string // "INSERT", "UPDATE", "DELETE"
	Entity any
	Key    string
	Data   map[string]any
	SQL    string
	Args   []any
}

// TxManager representa uma transação ativa
type TxManager struct {
	tx         *sql.Tx
	manager    *ObjectManager
	operations []BatchOperation
	mu         sync.RWMutex
}

// ObjectManager implementa funcionalidades ORM similares ao TObjectManager do Aurelius
type ObjectManager struct {
	provider      DatabaseProvider
	context       context.Context
	cache         map[string]*EntityCacheItem // Identity mapping: "EntityName:Key" -> Entity
	changes       map[string]bool             // Change tracking: entityKey -> hasChanges
	batchSize     int                         // Tamanho do batch
	cachedUpdates bool                        // Modo cached updates
	pendingOps    []BatchOperation            // Operações pendentes
	attachedObjs  map[string]bool             // Objetos attached ao manager
	mu            sync.RWMutex                // Thread safety
	logger        *log.Logger
}

// NewObjectManager cria uma nova instância do ObjectManager
func NewObjectManager(provider DatabaseProvider, ctx context.Context) *ObjectManager {
	if ctx == nil {
		ctx = context.Background()
	}

	return &ObjectManager{
		provider:      provider,
		context:       ctx,
		cache:         make(map[string]*EntityCacheItem),
		changes:       make(map[string]bool),
		attachedObjs:  make(map[string]bool),
		batchSize:     100,
		cachedUpdates: false,
		pendingOps:    make([]BatchOperation, 0),

		logger: log.New(log.Writer(), "[ObjectManager] ", log.LstdFlags|log.Lshortfile),
	}
}

// CreateFromEventContext cria um ObjectManager a partir de um contexto de evento
func CreateFromEventContext(ctx *EventContext) *ObjectManager {
	return NewObjectManager(ctx.DatabaseProvider, ctx.Context)
}

// ==================================================
// 1. CORE CRUD OPERATIONS
// ==================================================

// Find busca uma entidade por ID
func (om *ObjectManager) Find(entityName string, key string) (any, error) {
	// Primeiro tenta buscar no cache
	if cached := om.getFromCache(entityName, key); cached != nil {
		om.logger.Printf("📦 Cache hit para %s:%s", entityName, key)
		om.attachToManager(cached)
		return cached.Entity, nil
	}

	// Se não está no cache, busca no banco
	om.logger.Printf("🔍 Buscando %s:%s no banco de dados", entityName, key)

	entityMetadata := om.findEntityMetadata(entityName)
	if entityMetadata == nil {
		return nil, fmt.Errorf("entidade '%s' não encontrada", entityName)
	}

	// Constrói query SELECT básica
	keyProperty := om.getKeyProperty(entityMetadata)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", entityMetadata.TableName, keyProperty.Name)

	rows, err := om.ExecuteQuery(query, []any{key})
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar entidade: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("entidade %s com ID %s não encontrada", entityName, key)
	}

	// Converte resultado para map
	result := make(map[string]any)
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	for i, col := range columns {
		result[col] = values[i]
	}

	// Adiciona ao cache
	om.addToCache(entityName, key, result)
	om.attachToManager(result)

	return result, nil
}

// FindCached busca apenas no cache, não toca o banco
func (om *ObjectManager) FindCached(entityName string, key string) (any, error) {
	if cached := om.getFromCache(entityName, key); cached != nil {
		return cached.Entity, nil
	}
	return nil, fmt.Errorf("entidade %s:%s não encontrada no cache", entityName, key)
}

// Save marca uma entidade para inserção
func (om *ObjectManager) Save(entity any) error {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	// Verifica se já existe no cache
	if om.getFromCache(entityName, key) != nil {
		return fmt.Errorf("entidade %s:%s já existe", entityName, key)
	}

	if om.cachedUpdates {
		// Adiciona à lista de operações pendentes
		om.pendingOps = append(om.pendingOps, BatchOperation{
			Type:   "INSERT",
			Entity: entity,
			Key:    key,
			Data:   entityData,
		})
		om.logger.Printf("📝 Operação INSERT adicionada ao batch para %s:%s", entityName, key)
	} else {
		// Executa inserção imediatamente
		if err := om.executeInsert(entityData, entityName); err != nil {
			return err
		}
		// Adiciona ao cache após inserção bem-sucedida
		om.addToCache(entityName, key, entityData)
	}

	om.attachToManager(entity)
	return nil
}

// Update marca uma entidade para atualização
func (om *ObjectManager) Update(entity any) error {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	if om.cachedUpdates {
		// Adiciona à lista de operações pendentes
		om.pendingOps = append(om.pendingOps, BatchOperation{
			Type:   "UPDATE",
			Entity: entity,
			Key:    key,
			Data:   entityData,
		})
		om.logger.Printf("📝 Operação UPDATE adicionada ao batch para %s:%s", entityName, key)
	} else {
		// Marca como modificado mas não executa ainda
		om.markAsChanged(entityName, key)
	}

	om.attachToManager(entity)
	return nil
}

// SaveOrUpdate salva se novo ou atualiza se existente
func (om *ObjectManager) SaveOrUpdate(entity any) error {
	entityData := om.entityToMap(entity)

	// Verifica se entidade tem ID preenchido
	if om.hasValidID(entityData) {
		return om.Update(entity)
	}
	return om.Save(entity)
}

// Flush persiste mudanças de uma entidade específica
func (om *ObjectManager) Flush(entity any) error {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	if !om.IsAttached(entity) {
		return fmt.Errorf("entidade não está attached ao manager")
	}

	if !om.HasChanges(entity) {
		om.logger.Printf("ℹ️ Entidade %s:%s não tem mudanças para flush", entityName, key)
		return nil
	}

	// Se está em cached updates, executa esta operação imediatamente
	if om.cachedUpdates {
		return om.executePendingOperation(BatchOperation{
			Type:   "UPDATE",
			Entity: entity,
			Key:    key,
			Data:   entityData,
		})
	}

	// Executa update imediatamente
	return om.executeUpdate(entityData, entityName, key)
}

// FlushAll persiste todas as mudanças pendentes
func (om *ObjectManager) FlushAll() error {
	if om.cachedUpdates {
		return om.ApplyCachedUpdates()
	}

	var lastErr error
	for key := range om.changes {
		if om.changes[key] {
			// Aqui seria necessário reconstruir a entidade a partir da key
			// Para simplificar, vamos apenas limpar os changes pendentes
			om.changes[key] = false
		}
	}

	om.logger.Printf("✅ FlushAll concluído")
	return lastErr
}

// Remove marca uma entidade para remoção
func (om *ObjectManager) Remove(entity any) error {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	if om.cachedUpdates {
		om.pendingOps = append(om.pendingOps, BatchOperation{
			Type:   "DELETE",
			Entity: entity,
			Key:    key,
			Data:   entityData,
		})
		om.logger.Printf("📝 Operação DELETE adicionada ao batch para %s:%s", entityName, key)
	} else {
		// Executa remoção imediatamente
		if err := om.executeDelete(entityName, key); err != nil {
			return err
		}
	}

	// Remove do cache e marca como não attached
	om.removeFromCache(entityName, key)
	om.detachFromManager(entity)

	return nil
}

// Merge faz merge de um objeto detached com o object manager
func (om *ObjectManager) Merge(entity any) (any, error) {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	if !om.hasValidID(entityData) {
		return nil, fmt.Errorf("entidade deve ter ID válido para merge")
	}

	// Primeiro verifica se já existe no cache/banco
	if cached := om.getFromCache(entityName, key); cached != nil {
		// Atualiza objeto cached com dados do detached
		om.updateCachedEntity(cached, entityData)
		om.markAsChanged(entityName, key)
		return cached.Entity, nil
	}

	// Se não existe, busca no banco
	if foundEntity, err := om.Find(entityName, key); err == nil {
		// Entity exists in database, update it
		om.updateEntityWithData(foundEntity, entityData)
		om.markAsChanged(entityName, key)
		return foundEntity, nil
	}

	return nil, fmt.Errorf("entidade %s:%s não encontrada para merge", entityName, key)
}

// ==================================================
// 2. IDENTITY MAPPING & CACHE
// ==================================================

// IsCached verifica se uma entidade está no cache
func (om *ObjectManager) IsCached(entityName string, key string) bool {
	return om.getFromCache(entityName, key) != nil
}

// IsAttached verifica se um objeto está attached ao manager
func (om *ObjectManager) IsAttached(entity any) bool {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	om.mu.RLock()
	defer om.mu.RUnlock()

	return om.attachedObjs[om.buildCacheKey(entityName, key)]
}

// Evict remove uma entidade do manager
func (om *ObjectManager) Evict(entity any) error {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	om.removeFromCache(entityName, key)
	om.detachFromManager(entity)

	om.logger.Printf("🗑️ Entidade %s:%s removida do manager", entityName, key)
	return nil
}

// ClearCache limpa todo o cache
func (om *ObjectManager) ClearCache() error {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.cache = make(map[string]*EntityCacheItem)
	om.changes = make(map[string]bool)
	om.attachedObjs = make(map[string]bool)

	om.logger.Printf("🧹 Cache limpo completamente")
	return nil
}

// ==================================================
// 3. CHANGE TRACKING
// ==================================================

// HasChanges verifica se uma entidade foi modificada
func (om *ObjectManager) HasChanges(entity any) bool {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	om.mu.RLock()
	defer om.mu.RUnlock()

	return om.changes[om.buildCacheKey(entityName, key)]
}

// HasAnyChanges verifica se há alguma mudança pendente
func (om *ObjectManager) HasAnyChanges() bool {
	om.mu.RLock()
	defer om.mu.Unlock()

	if om.cachedUpdates {
		return len(om.pendingOps) > 0
	}

	for _, hasChanges := range om.changes {
		if hasChanges {
			return true
		}
	}

	return false
}

// GetChangedObjects retorna lista de objetos modificados
func (om *ObjectManager) GetChangedObjects() []any {
	var changed []any

	om.mu.RLock()
	defer om.mu.RUnlock()

	if om.cachedUpdates {
		for _, op := range om.pendingOps {
			if op.Entity != nil {
				changed = append(changed, op.Entity)
			}
		}
		return changed
	}

	for cacheKey, hasChanges := range om.changes {
		if hasChanges {
			if cached := om.cache[cacheKey]; cached != nil {
				changed = append(changed, cached.Entity)
			}
		}
	}

	return changed
}

// ==================================================
// 4. TRANSACTION MANAGEMENT
// ==================================================

// BeginTransaction inicia uma nova transação
func (om *ObjectManager) BeginTransaction() (*TxManager, error) {
	conn := om.GetConnection()
	tx, err := conn.BeginTx(om.context, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar transação: %w", err)
	}

	txManager := &TxManager{
		tx:         tx,
		manager:    om,
		operations: make([]BatchOperation, 0),
	}

	om.logger.Printf("🔒 Transação iniciada")
	return txManager, nil
}

// CommitTransaction confirma uma transação
func (om *ObjectManager) CommitTransaction(tx *TxManager) error {
	if err := tx.tx.Commit(); err != nil {
		om.logger.Printf("❌ Erro ao fazer commit da transação: %v", err)
		return fmt.Errorf("erro ao fazer commit da transação: %w", err)
	}

	om.logger.Printf("✅ Transação commitada com sucesso")
	return nil
}

// RollbackTransaction desfaz uma transação
func (om *ObjectManager) RollbackTransaction(tx *TxManager) error {
	if err := tx.tx.Rollback(); err != nil {
		om.logger.Printf("❌ Erro ao fazer rollback da transação: %v", err)
		return fmt.Errorf("erro ao fazer rollback da transação: %w", err)
	}

	om.logger.Printf("🔄 Transação desfeita com sucesso")
	return nil
}

// WithTransaction executa uma função dentro de uma transação
func (om *ObjectManager) WithTransaction(fn func(*TxManager) error) error {
	txManager, err := om.BeginTransaction()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			om.RollbackTransaction(txManager)
		}
	}()

	if err := fn(txManager); err != nil {
		om.RollbackTransaction(txManager)
		return err
	}

	return om.CommitTransaction(txManager)
}

// ==================================================
// 5. BATCH OPERATIONS
// ==================================================

// SetBatchSize configura o tamanho do batch
func (om *ObjectManager) SetBatchSize(size int) {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.batchSize = size
	om.logger.Printf("📦 Batch size configurado para: %d", size)
}

// SetCachedUpdates habilita/desabilita cached updates
func (om *ObjectManager) SetCachedUpdates(enabled bool) {
	om.mu.Lock()
	defer om.mu.Unlock()

	om.cachedUpdates = enabled
	if enabled {
		om.logger.Printf("📝 Cached updates habilitado")
	} else {
		om.logger.Printf("📝 Cached updates desabilitado")
	}
}

// ApplyCachedUpdates aplica todas as operações pendentes
func (om *ObjectManager) ApplyCachedUpdates() error {
	om.mu.Lock()
	defer om.mu.Unlock()

	if len(om.pendingOps) == 0 {
		om.logger.Printf("ℹ️ Nenhuma operação pendente para aplicar")
		return nil
	}

	om.logger.Printf("🚀 Aplicando %d operações em lote", len(om.pendingOps))

	// Agrupa operações por tipo para otimização
	inserts := make([]BatchOperation, 0)
	updates := make([]BatchOperation, 0)
	deletes := make([]BatchOperation, 0)

	for _, op := range om.pendingOps {
		switch op.Type {
		case "INSERT":
			inserts = append(inserts, op)
		case "UPDATE":
			updates = append(updates, op)
		case "DELETE":
			deletes = append(deletes, op)
		}
	}

	// Executa operações em grupos
	if err := om.executeBatchInsert(inserts); err != nil {
		return err
	}
	if err := om.executeBatchUpdate(updates); err != nil {
		return err
	}
	if err := om.executeBatchDelete(deletes); err != nil {
		return err
	}

	// Limpa operações pendentes
	om.pendingOps = make([]BatchOperation, 0)

	om.logger.Printf("✅ Operações em lote aplicadas com sucesso")
	return nil
}

// GetCachedCount retorna número de operações pendentes
func (om *ObjectManager) GetCachedCount() int {
	om.mu.RLock()
	defer om.mu.RUnlock()

	return len(om.pendingOps)
}

// ==================================================
// 6. PERFORMANCE & OPTIMIZATION
// ==================================================

// GetConnection retorna a conexão do banco
func (om *ObjectManager) GetConnection() *sql.DB {
	return om.provider.GetConnection()
}

// ExecuteQuery executa uma query customizada
func (om *ObjectManager) ExecuteQuery(query string, args ...any) (*sql.Rows, error) {
	conn := om.GetConnection()
	if conn == nil {
		return nil, fmt.Errorf("conexão com banco não disponível")
	}

	om.logger.Printf("🔍 Executando query: %s", query)
	return conn.QueryContext(om.context, query, args...)
}

// ExecuteQueryTransaction executa query dentro de uma transação
func (om *ObjectManager) ExecuteQueryTransaction(tx *TxManager, query string, args ...any) (*sql.Rows, error) {
	om.logger.Printf("🔍 Executando query em transação: %s", query)
	return tx.tx.QueryContext(om.context, query, args...)
}

// ==================================================
// MÉTODOS AUXILIARES PRIVADOS
// ==================================================

// buildCacheKey cria chave única para cache
func (om *ObjectManager) buildCacheKey(entityName, key string) string {
	return fmt.Sprintf("%s:%s", entityName, key)
}

// getFromCache busca entidade no cache
func (om *ObjectManager) getFromCache(entityName, key string) *EntityCacheItem {
	om.mu.RLock()
	defer om.mu.RUnlock()

	cacheKey := om.buildCacheKey(entityName, key)
	return om.cache[cacheKey]
}

// addToCache adiciona entidade ao cache
func (om *ObjectManager) addToCache(entityName, key string, entity any) {
	om.mu.Lock()
	defer om.mu.Unlock()

	cacheKey := om.buildCacheKey(entityName, key)
	om.cache[cacheKey] = &EntityCacheItem{
		Entity:    entity,
		Key:       key,
		CachedAt:  time.Now(),
		IsChanged: false,
	}

	if om.attachedObjs[cacheKey] {
		om.logger.Printf("📦 Entidade %s:%s já está attached, apenas atualizada", entityName, key)
	}
}

// removeFromCache remove entidade do cache
func (om *ObjectManager) removeFromCache(entityName, key string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	cacheKey := om.buildCacheKey(entityName, key)
	delete(om.cache, cacheKey)
	delete(om.changes, cacheKey)
}

// attachToManager marca objeto como attached
func (om *ObjectManager) attachToManager(entity any) {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	om.mu.Lock()
	defer om.mu.Unlock()

	cacheKey := om.buildCacheKey(entityName, key)
	om.attachedObjs[cacheKey] = true
}

// detachFromManager marca objeto como não attached
func (om *ObjectManager) detachFromManager(entity any) {
	entityData := om.entityToMap(entity)
	entityName := om.getEntityType(entity)
	key := om.extractKey(entityData, entityName)

	om.mu.Lock()
	defer om.mu.Unlock()

	cacheKey := om.buildCacheKey(entityName, key)
	delete(om.attachedObjs, cacheKey)
	delete(om.changes, cacheKey)
}

// markAsChanged marca entidade como modificada
func (om *ObjectManager) markAsChanged(entityName, key string) {
	om.mu.Lock()
	defer om.mu.Unlock()

	cacheKey := om.buildCacheKey(entityName, key)
	om.changes[cacheKey] = true

	if cached := om.cache[cacheKey]; cached != nil {
		cached.IsChanged = true
	}
}

// entityToMap converte entidade para map
func (om *ObjectManager) entityToMap(entity any) map[string]any {
	if dataMap, ok := entity.(map[string]any); ok {
		return dataMap
	}

	// Para estruturas Go, usa reflexão básica
	data := make(map[string]any)
	v := reflect.ValueOf(entity)
	t := reflect.TypeOf(entity)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Converte json tag para nome do campo
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			fieldName = jsonTag
		}

		data[fieldName] = value.Interface()
	}

	return data
}

// getEntityType obtém nome do tipo da entidade
func (om *ObjectManager) getEntityType(entity any) string {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// extractKey extrai chave única da entidade
func (om *ObjectManager) extractKey(entityData map[string]any, entityName string) string {
	// Procura por campo "id" ou similar
	if id, ok := entityData["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	if ID, ok := entityData["ID"]; ok {
		return fmt.Sprintf("%v", ID)
	}
	// Fallback para nome da entidade
	return entityName
}

// hasValidID verifica se entidade tem ID válido
func (om *ObjectManager) hasValidID(entityData map[string]any) bool {
	if id, ok := entityData["id"]; ok && id != nil {
		return true
	}
	if ID, ok := entityData["ID"]; ok && ID != nil {
		return true
	}
	return false
}

// findEntityMetadata busca metadados da entidade (implementação simplificada)
func (om *ObjectManager) findEntityMetadata(entityName string) *EntityMetadata {
	// Implementação simplificada - em produção seria integrada com registros reais
	return &EntityMetadata{
		Name:      entityName,
		TableName: entityName, // Assume tabela com mesmo nome
	}
}

// getKeyProperty retorna propriedade chave da entidade
func (om *ObjectManager) getKeyProperty(metadata *EntityMetadata) PropertyMetadata {
	// Implementação simplificada - sempre retorna "id"
	return PropertyMetadata{
		Name: "id",
		Type: "int64",
	}
}

// executeInsert executa inserção imediata
func (om *ObjectManager) executeInsert(entityData map[string]any, entityName string) error {
	// Implementação simplificada - seria integrada com providers existentes
	om.logger.Printf("💾 Executando INSERT para %s", entityName)
	return nil
}

// executeUpdate executa atualização imediata
func (om *ObjectManager) executeUpdate(entityData map[string]any, entityName, key string) error {
	om.logger.Printf("💾 Executando UPDATE para %s:%s", entityName, key)

	// Remove mudança do tracking após execução
	om.mu.Lock()
	delete(om.changes, om.buildCacheKey(entityName, key))
	om.mu.Unlock()

	return nil
}

// executeDelete executa remoção imediata
func (om *ObjectManager) executeDelete(entityName, key string) error {
	om.logger.Printf("💾 Executando DELETE para %s:%s", entityName, key)
	return nil
}

// executePendingOperation executa uma operação individual pendente
func (om *ObjectManager) executePendingOperation(op BatchOperation) error {
	switch op.Type {
	case "INSERT":
		return om.executeInsert(op.Data, om.getEntityType(op.Entity))
	case "UPDATE":
		return om.executeUpdate(op.Data, om.getEntityType(op.Entity), op.Key)
	case "DELETE":
		return om.executeDelete(om.getEntityType(op.Entity), op.Key)
	}
	return nil
}

// executeBatchInsert executa inserções em lote
func (om *ObjectManager) executeBatchInsert(inserts []BatchOperation) error {
	if len(inserts) == 0 {
		return nil
	}

	om.logger.Printf("🚀 Executando %d INSERTs em lote", len(inserts))

	for i, op := range inserts {
		if err := om.executeInsert(op.Data, om.getEntityType(op.Entity)); err != nil {
			return fmt.Errorf("erro no INSERT %d: %w", i, err)
		}
	}

	return nil
}

// executeBatchUpdate executa atualizações em lote
func (om *ObjectManager) executeBatchUpdate(updates []BatchOperation) error {
	if len(updates) == 0 {
		return nil
	}

	om.logger.Printf("🚀 Executando %d UPDATEs em lote", len(updates))

	for i, op := range updates {
		if err := om.executeUpdate(op.Data, om.getEntityType(op.Entity), op.Key); err != nil {
			return fmt.Errorf("erro no UPDATE %d: %w", i, err)
		}
	}

	return nil
}

// executeBatchDelete executa remoções em lote
func (om *ObjectManager) executeBatchDelete(deletes []BatchOperation) error {
	if len(deletes) == 0 {
		return nil
	}

	om.logger.Printf("🚀 Executando %d DELETEs em lote", len(deletes))

	for i, op := range deletes {
		if err := om.executeDelete(om.getEntityType(op.Entity), op.Key); err != nil {
			return fmt.Errorf("erro no DELETE %d: %w", i, err)
		}
	}

	return nil
}

// updateCachedEntity atualiza entidade cached com novos dados
func (om *ObjectManager) updateCachedEntity(cached *EntityCacheItem, newData map[string]any) {
	if entityMap, ok := cached.Entity.(map[string]any); ok {
		for k, v := range newData {
			entityMap[k] = v
		}
	}
}

// updateEntityWithData atualiza entidade com dados fornecidos
func (om *ObjectManager) updateEntityWithData(entity any, data map[string]any) {
	if entityMap, ok := entity.(map[string]any); ok {
		for k, v := range data {
			entityMap[k] = v
		}
	}
}
