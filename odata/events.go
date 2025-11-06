package odata

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// EventType representa os tipos de eventos disponíveis
type EventType string

const (
	// Eventos de recuperação de dados
	EventEntityGet  EventType = "EntityGet"
	EventEntityList EventType = "EntityList"

	// Eventos de inserção (antes e depois)
	EventEntityInserting EventType = "EntityInserting"
	EventEntityInserted  EventType = "EntityInserted"

	// Eventos de atualização (antes e depois)
	EventEntityModifying EventType = "EntityModifying"
	EventEntityModified  EventType = "EntityModified"

	// Eventos de exclusão (antes e depois)
	EventEntityDeleting EventType = "EntityDeleting"
	EventEntityDeleted  EventType = "EntityDeleted"

	// Eventos de validação
	EventEntityValidating EventType = "EntityValidating"
	EventEntityValidated  EventType = "EntityValidated"

	// Eventos de exceção
	EventEntityError EventType = "EntityError"
)

// EventContext contém informações contextuais sobre o evento
type EventContext struct {
	Context          context.Context
	FiberContext     fiber.Ctx
	EntityName       string
	EntityType       string
	UserID           string
	UserRoles        []string
	UserScopes       []string
	RequestID        string
	Timestamp        int64
	Extra            map[string]interface{}
	DatabaseProvider DatabaseProvider // Para acesso direto ao provider
}

// EventArgs é a interface base para todos os argumentos de evento
type EventArgs interface {
	GetContext() *EventContext
	GetEventType() EventType
	GetEntityName() string
	GetEntity() interface{}
	SetEntity(interface{})
	CanCancel() bool
	Cancel(reason string)
	IsCanceled() bool
	GetCancelReason() string
	Manager() *ObjectManager
	GetManager() *ObjectManager
}

// BaseEventArgs implementa a funcionalidade comum para todos os eventos
type BaseEventArgs struct {
	Context      *EventContext
	EventType    EventType
	EntityName   string
	Entity       interface{}
	canCancel    bool
	canceled     bool
	cancelReason string
}

func (e *BaseEventArgs) GetContext() *EventContext    { return e.Context }
func (e *BaseEventArgs) GetEventType() EventType      { return e.EventType }
func (e *BaseEventArgs) GetEntityName() string        { return e.EntityName }
func (e *BaseEventArgs) GetEntity() interface{}       { return e.Entity }
func (e *BaseEventArgs) SetEntity(entity interface{}) { e.Entity = entity }
func (e *BaseEventArgs) CanCancel() bool              { return e.canCancel }
func (e *BaseEventArgs) IsCanceled() bool             { return e.canceled }
func (e *BaseEventArgs) GetCancelReason() string      { return e.cancelReason }

func (e *BaseEventArgs) Cancel(reason string) {
	if e.canCancel {
		e.canceled = true
		e.cancelReason = reason
	}
}

// Manager retorna ObjectManager para uso nos eventos
func (e *BaseEventArgs) Manager() *ObjectManager {
	return e.GetManager()
}

// GetManager retorna ObjectManager para uso nos eventos
func (e *BaseEventArgs) GetManager() *ObjectManager {
	if e.Context != nil && e.Context.DatabaseProvider != nil {
		return NewObjectManager(e.Context.DatabaseProvider, e.Context.Context)
	}
	return nil
}

// EntityGetArgs argumentos para evento OnEntityGet
type EntityGetArgs struct {
	*BaseEventArgs
	Keys        map[string]interface{}
	QueryParams map[string]interface{}
}

// EntityListArgs argumentos para evento OnEntityList
type EntityListArgs struct {
	*BaseEventArgs
	QueryOptions  QueryOptions
	Results       []interface{}
	TotalCount    int64
	FilterApplied bool
	CustomFilters map[string]interface{}
}

// EntityInsertingArgs argumentos para evento OnEntityInserting
type EntityInsertingArgs struct {
	*BaseEventArgs
	Data             map[string]interface{}
	ValidationErrors []string
}

// EntityInsertedArgs argumentos para evento OnEntityInserted
type EntityInsertedArgs struct {
	*BaseEventArgs
	CreatedEntity interface{}
	NewID         interface{}
}

// EntityModifyingArgs argumentos para evento OnEntityModifying
type EntityModifyingArgs struct {
	*BaseEventArgs
	Keys             map[string]interface{}
	Data             map[string]interface{}
	OriginalEntity   interface{}
	ValidationErrors []string
}

// EntityModifiedArgs argumentos para evento OnEntityModified
type EntityModifiedArgs struct {
	*BaseEventArgs
	Keys           map[string]interface{}
	UpdatedEntity  interface{}
	OriginalEntity interface{}
	ModifiedFields []string
}

// EntityDeletingArgs argumentos para evento OnEntityDeleting
type EntityDeletingArgs struct {
	*BaseEventArgs
	Keys           map[string]interface{}
	EntityToDelete interface{}
	CascadeDelete  bool
}

// EntityDeletedArgs argumentos para evento OnEntityDeleted
type EntityDeletedArgs struct {
	*BaseEventArgs
	Keys          map[string]interface{}
	DeletedEntity interface{}
}

// EntityValidatingArgs argumentos para evento OnEntityValidating
type EntityValidatingArgs struct {
	*BaseEventArgs
	Data             map[string]interface{}
	ValidationErrors []string
}

// EntityValidatedArgs argumentos para evento OnEntityValidated
type EntityValidatedArgs struct {
	*BaseEventArgs
	Data             map[string]interface{}
	ValidationErrors []string
	IsValid          bool
}

// EntityErrorArgs argumentos para evento OnEntityError
type EntityErrorArgs struct {
	*BaseEventArgs
	Error      error
	Operation  string
	StatusCode int
	CanRecover bool
}

// EventHandler representa um handler de evento
type EventHandler interface {
	Handle(args EventArgs) error
}

// EventHandlerFunc é uma função que pode ser usada como handler
type EventHandlerFunc func(args EventArgs) error

// Implementa a interface EventHandler
func (f EventHandlerFunc) Handle(args EventArgs) error {
	return f(args)
}

// EntityEventManager gerencia todos os eventos de entidade
type EntityEventManager struct {
	mu       sync.RWMutex
	handlers map[EventType]map[string][]EventHandler // EventType -> EntityName -> []Handler
	global   map[EventType][]EventHandler            // Handlers globais por tipo
	logger   *log.Logger
}

// NewEntityEventManager cria um novo gerenciador de eventos
func NewEntityEventManager(logger *log.Logger) *EntityEventManager {
	if logger == nil {
		logger = log.New(log.Writer(), "[EntityEvents] ", log.LstdFlags|log.Lshortfile)
	}

	return &EntityEventManager{
		handlers: make(map[EventType]map[string][]EventHandler),
		global:   make(map[EventType][]EventHandler),
		logger:   logger,
	}
}

// Subscribe registra um handler para um evento específico de uma entidade
func (em *EntityEventManager) Subscribe(eventType EventType, entityName string, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.handlers[eventType] == nil {
		em.handlers[eventType] = make(map[string][]EventHandler)
	}

	em.handlers[eventType][entityName] = append(em.handlers[eventType][entityName], handler)
	em.logger.Printf("Handler registrado para evento %s da entidade %s", eventType, entityName)
}

// SubscribeGlobal registra um handler global para um tipo de evento
func (em *EntityEventManager) SubscribeGlobal(eventType EventType, handler EventHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.global[eventType] = append(em.global[eventType], handler)
	em.logger.Printf("Handler global registrado para evento %s", eventType)
}

// SubscribeFunc registra uma função como handler
func (em *EntityEventManager) SubscribeFunc(eventType EventType, entityName string, handler func(args EventArgs) error) {
	em.Subscribe(eventType, entityName, EventHandlerFunc(handler))
}

// SubscribeGlobalFunc registra uma função como handler global
func (em *EntityEventManager) SubscribeGlobalFunc(eventType EventType, handler func(args EventArgs) error) {
	em.SubscribeGlobal(eventType, EventHandlerFunc(handler))
}

// Emit dispara um evento
func (em *EntityEventManager) Emit(args EventArgs) error {
	em.mu.RLock()
	defer em.mu.RUnlock()

	eventType := args.GetEventType()
	entityName := args.GetEntityName()

	// Executa handlers globais primeiro
	if globalHandlers, exists := em.global[eventType]; exists {
		for _, handler := range globalHandlers {
			if err := em.executeHandler(handler, args, "global"); err != nil {
				return err
			}

			// Verifica se o evento foi cancelado
			if args.IsCanceled() {
				return fmt.Errorf("evento cancelado: %s", args.GetCancelReason())
			}
		}
	}

	// Executa handlers específicos da entidade
	if entityHandlers, exists := em.handlers[eventType]; exists {
		if handlers, exists := entityHandlers[entityName]; exists {
			for _, handler := range handlers {
				if err := em.executeHandler(handler, args, entityName); err != nil {
					return err
				}

				// Verifica se o evento foi cancelado
				if args.IsCanceled() {
					return fmt.Errorf("evento cancelado: %s", args.GetCancelReason())
				}
			}
		}
	}

	return nil
}

// executeHandler executa um handler com tratamento de erro
func (em *EntityEventManager) executeHandler(handler EventHandler, args EventArgs, scope string) error {
	defer func() {
		if r := recover(); r != nil {
			em.logger.Printf("PANIC no handler %s do evento %s: %v", scope, args.GetEventType(), r)
		}
	}()

	if err := handler.Handle(args); err != nil {
		em.logger.Printf("Erro no handler %s do evento %s: %v", scope, args.GetEventType(), err)
		return fmt.Errorf("erro no handler %s: %w", scope, err)
	}

	return nil
}

// GetHandlerCount retorna o número de handlers registrados
func (em *EntityEventManager) GetHandlerCount(eventType EventType, entityName string) int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	count := 0

	// Conta handlers globais
	if globalHandlers, exists := em.global[eventType]; exists {
		count += len(globalHandlers)
	}

	// Conta handlers específicos da entidade
	if entityHandlers, exists := em.handlers[eventType]; exists {
		if handlers, exists := entityHandlers[entityName]; exists {
			count += len(handlers)
		}
	}

	return count
}

// ListSubscriptions lista todas as assinaturas de eventos
func (em *EntityEventManager) ListSubscriptions() map[string]interface{} {
	em.mu.RLock()
	defer em.mu.RUnlock()

	subscriptions := make(map[string]interface{})

	// Adiciona handlers globais
	globalInfo := make(map[string]int)
	for eventType, handlers := range em.global {
		globalInfo[string(eventType)] = len(handlers)
	}
	subscriptions["global"] = globalInfo

	// Adiciona handlers específicos por entidade
	entityInfo := make(map[string]map[string]int)
	for eventType, entityHandlers := range em.handlers {
		for entityName, handlers := range entityHandlers {
			if entityInfo[entityName] == nil {
				entityInfo[entityName] = make(map[string]int)
			}
			entityInfo[entityName][string(eventType)] = len(handlers)
		}
	}
	subscriptions["entities"] = entityInfo

	return subscriptions
}

// Clear remove todos os handlers
func (em *EntityEventManager) Clear() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.handlers = make(map[EventType]map[string][]EventHandler)
	em.global = make(map[EventType][]EventHandler)
	em.logger.Printf("Todos os handlers foram removidos")
}

// ClearEntity remove todos os handlers de uma entidade específica
func (em *EntityEventManager) ClearEntity(entityName string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	for eventType := range em.handlers {
		delete(em.handlers[eventType], entityName)
	}
	em.logger.Printf("Handlers da entidade %s foram removidos", entityName)
}

// Funções auxiliares para criar argumentos de evento

// NewEntityGetArgs cria argumentos para evento EntityGet
func NewEntityGetArgs(ctx *EventContext, keys map[string]interface{}, entity interface{}) *EntityGetArgs {
	return &EntityGetArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityGet,
			EntityName: ctx.EntityName,
			Entity:     entity,
			canCancel:  false,
		},
		Keys:        keys,
		QueryParams: make(map[string]interface{}),
	}
}

// NewEntityListArgs cria argumentos para evento EntityList
func NewEntityListArgs(ctx *EventContext, options QueryOptions, results []interface{}) *EntityListArgs {
	// Calcular TotalCount baseado nos resultados se não for fornecido
	totalCount := int64(0)
	if results != nil {
		totalCount = int64(len(results))
	}

	return &EntityListArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityList,
			EntityName: ctx.EntityName,
			Entity:     results,
			canCancel:  true,
		},
		QueryOptions:  options,
		Results:       results,
		TotalCount:    totalCount,
		FilterApplied: options.Filter != nil,
		CustomFilters: make(map[string]interface{}),
	}
}

// NewEntityInsertingArgs cria argumentos para evento EntityInserting
func NewEntityInsertingArgs(ctx *EventContext, data map[string]interface{}) *EntityInsertingArgs {
	return &EntityInsertingArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityInserting,
			EntityName: ctx.EntityName,
			Entity:     data,
			canCancel:  true,
		},
		Data:             data,
		ValidationErrors: make([]string, 0),
	}
}

// NewEntityInsertedArgs cria argumentos para evento EntityInserted
func NewEntityInsertedArgs(ctx *EventContext, entity interface{}) *EntityInsertedArgs {
	return &EntityInsertedArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityInserted,
			EntityName: ctx.EntityName,
			Entity:     entity,
			canCancel:  false,
		},
		CreatedEntity: entity,
	}
}

// NewEntityModifyingArgs cria argumentos para evento EntityModifying
func NewEntityModifyingArgs(ctx *EventContext, keys map[string]interface{}, data map[string]interface{}, original interface{}) *EntityModifyingArgs {
	return &EntityModifyingArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityModifying,
			EntityName: ctx.EntityName,
			Entity:     data,
			canCancel:  true,
		},
		Keys:             keys,
		Data:             data,
		OriginalEntity:   original,
		ValidationErrors: make([]string, 0),
	}
}

// NewEntityModifiedArgs cria argumentos para evento EntityModified
func NewEntityModifiedArgs(ctx *EventContext, keys map[string]interface{}, updated, original interface{}) *EntityModifiedArgs {
	return &EntityModifiedArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityModified,
			EntityName: ctx.EntityName,
			Entity:     updated,
			canCancel:  false,
		},
		Keys:           keys,
		UpdatedEntity:  updated,
		OriginalEntity: original,
		ModifiedFields: make([]string, 0),
	}
}

// NewEntityDeletingArgs cria argumentos para evento EntityDeleting
func NewEntityDeletingArgs(ctx *EventContext, keys map[string]interface{}, entity interface{}) *EntityDeletingArgs {
	return &EntityDeletingArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityDeleting,
			EntityName: ctx.EntityName,
			Entity:     entity,
			canCancel:  true,
		},
		Keys:           keys,
		EntityToDelete: entity,
	}
}

// NewEntityDeletedArgs cria argumentos para evento EntityDeleted
func NewEntityDeletedArgs(ctx *EventContext, keys map[string]interface{}, entity interface{}) *EntityDeletedArgs {
	return &EntityDeletedArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityDeleted,
			EntityName: ctx.EntityName,
			Entity:     entity,
			canCancel:  false,
		},
		Keys:          keys,
		DeletedEntity: entity,
	}
}

// NewEntityErrorArgs cria argumentos para evento EntityError
func NewEntityErrorArgs(ctx *EventContext, err error, operation string, statusCode int) *EntityErrorArgs {
	return &EntityErrorArgs{
		BaseEventArgs: &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityError,
			EntityName: ctx.EntityName,
			Entity:     nil,
			canCancel:  true,
		},
		Error:      err,
		Operation:  operation,
		StatusCode: statusCode,
		CanRecover: false,
	}
}

// createEventContext cria um contexto de evento a partir do contexto do Fiber
func createEventContext(c fiber.Ctx, entityName string) *EventContext {
	ctx := &EventContext{
		Context:      c.Context(),
		FiberContext: c,
		EntityName:   entityName,
		EntityType:   reflect.TypeOf(entityName).String(),
		RequestID:    c.Get("X-Request-ID", ""),
		Timestamp:    time.Now().Unix(),
		Extra:        make(map[string]interface{}),
	}

	// Extrai informações do usuário se disponível
	if user := GetCurrentUser(c); user != nil {
		ctx.UserID = user.Username
		ctx.UserRoles = user.Roles
		ctx.UserScopes = user.Scopes
	}

	return ctx
}
