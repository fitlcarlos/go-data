package odata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventTypes_Constants(t *testing.T) {
	t.Run("All event types are defined", func(t *testing.T) {
		eventTypes := []EventType{
			EventEntityGet,
			EventEntityList,
			EventEntityInserting,
			EventEntityInserted,
			EventEntityModifying,
			EventEntityModified,
			EventEntityDeleting,
			EventEntityDeleted,
			EventEntityValidating,
			EventEntityValidated,
			EventEntityError,
		}

		// Verify all are non-empty strings
		for _, et := range eventTypes {
			assert.NotEmpty(t, string(et))
		}
	})

	t.Run("Event types have unique values", func(t *testing.T) {
		seen := make(map[EventType]bool)
		eventTypes := []EventType{
			EventEntityGet,
			EventEntityList,
			EventEntityInserting,
			EventEntityInserted,
			EventEntityModifying,
			EventEntityModified,
			EventEntityDeleting,
			EventEntityDeleted,
			EventEntityValidating,
			EventEntityValidated,
			EventEntityError,
		}

		for _, et := range eventTypes {
			assert.False(t, seen[et], "Duplicate event type: %s", et)
			seen[et] = true
		}
	})
}

func TestEventContext_Creation(t *testing.T) {
	t.Run("Create basic event context", func(t *testing.T) {
		ctx := &EventContext{
			Context:    context.Background(),
			EntityName: "Users",
			EntityType: "User",
			UserID:     "user123",
			Timestamp:  123456789,
		}

		assert.NotNil(t, ctx)
		assert.Equal(t, "Users", ctx.EntityName)
		assert.Equal(t, "User", ctx.EntityType)
		assert.Equal(t, "user123", ctx.UserID)
		assert.Equal(t, int64(123456789), ctx.Timestamp)
	})

	t.Run("Event context with roles and scopes", func(t *testing.T) {
		ctx := &EventContext{
			Context:    context.Background(),
			EntityName: "Orders",
			UserID:     "user456",
			UserRoles:  []string{"admin", "user"},
			UserScopes: []string{"read", "write"},
		}

		assert.Len(t, ctx.UserRoles, 2)
		assert.Len(t, ctx.UserScopes, 2)
		assert.Contains(t, ctx.UserRoles, "admin")
		assert.Contains(t, ctx.UserScopes, "write")
	})

	t.Run("Event context with extra data", func(t *testing.T) {
		ctx := &EventContext{
			Context:    context.Background(),
			EntityName: "Products",
			Extra: map[string]interface{}{
				"source":   "api",
				"priority": "high",
			},
		}

		assert.NotNil(t, ctx.Extra)
		assert.Equal(t, "api", ctx.Extra["source"])
		assert.Equal(t, "high", ctx.Extra["priority"])
	})
}

func TestBaseEventArgs_Implementation(t *testing.T) {
	t.Run("Create base event args", func(t *testing.T) {
		ctx := &EventContext{
			Context:    context.Background(),
			EntityName: "Users",
		}

		args := &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityInserting,
			EntityName: "Users",
			Entity:     map[string]interface{}{"Name": "John"},
			canCancel:  true,
		}

		assert.Equal(t, ctx, args.GetContext())
		assert.Equal(t, EventEntityInserting, args.GetEventType())
		assert.Equal(t, "Users", args.GetEntityName())
		assert.NotNil(t, args.GetEntity())
		assert.True(t, args.CanCancel())
		assert.False(t, args.IsCanceled())
	})

	t.Run("Set entity", func(t *testing.T) {
		args := &BaseEventArgs{
			EventType:  EventEntityInserted,
			EntityName: "Products",
		}

		newEntity := map[string]interface{}{"ID": 1, "Name": "Product"}
		args.SetEntity(newEntity)

		assert.Equal(t, newEntity, args.GetEntity())
	})

	t.Run("Cancel event", func(t *testing.T) {
		args := &BaseEventArgs{
			EventType: EventEntityInserting,
			canCancel: true,
		}

		assert.False(t, args.IsCanceled())

		args.Cancel("Business rule violation")

		assert.True(t, args.IsCanceled())
		assert.Equal(t, "Business rule violation", args.GetCancelReason())
	})

	t.Run("Cannot cancel non-cancelable event", func(t *testing.T) {
		args := &BaseEventArgs{
			EventType: EventEntityInserted,
			canCancel: false,
		}

		args.Cancel("Try to cancel")

		assert.False(t, args.IsCanceled())
		assert.Empty(t, args.GetCancelReason())
	})
}

func TestEventHandler_Function(t *testing.T) {
	t.Run("EventHandlerFunc works as handler", func(t *testing.T) {
		called := false
		handler := EventHandlerFunc(func(args EventArgs) error {
			called = true
			return nil
		})

		ctx := &EventContext{
			Context:    context.Background(),
			EntityName: "Users",
		}

		args := &BaseEventArgs{
			Context:    ctx,
			EventType:  EventEntityInserting,
			EntityName: "Users",
		}

		err := handler.Handle(args)

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("Handler can access entity data", func(t *testing.T) {
		var receivedEntity interface{}
		handler := EventHandlerFunc(func(args EventArgs) error {
			receivedEntity = args.GetEntity()
			return nil
		})

		entity := map[string]interface{}{"ID": 1, "Name": "Test"}
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Test",
			},
			EventType: EventEntityInserting,
			Entity:    entity,
		}

		handler.Handle(args)

		assert.Equal(t, entity, receivedEntity)
	})

	t.Run("Handler can modify entity", func(t *testing.T) {
		handler := EventHandlerFunc(func(args EventArgs) error {
			entity := args.GetEntity().(map[string]interface{})
			entity["Modified"] = true
			args.SetEntity(entity)
			return nil
		})

		entity := map[string]interface{}{"ID": 1}
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Test",
			},
			EventType: EventEntityInserting,
			Entity:    entity,
		}

		handler.Handle(args)

		modifiedEntity := args.GetEntity().(map[string]interface{})
		assert.True(t, modifiedEntity["Modified"].(bool))
	})
}

func TestEventLifecycle_Inserting(t *testing.T) {
	t.Run("Inserting event is cancelable", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityInserting,
			canCancel: true,
		}

		assert.True(t, args.CanCancel())
		assert.Equal(t, EventEntityInserting, args.GetEventType())
	})

	t.Run("Handler can cancel inserting", func(t *testing.T) {
		handler := EventHandlerFunc(func(args EventArgs) error {
			if args.CanCancel() {
				args.Cancel("Validation failed")
			}
			return nil
		})

		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityInserting,
			canCancel: true,
		}

		handler.Handle(args)

		assert.True(t, args.IsCanceled())
		assert.Equal(t, "Validation failed", args.GetCancelReason())
	})
}

func TestEventLifecycle_Inserted(t *testing.T) {
	t.Run("Inserted event is not cancelable", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityInserted,
			canCancel: false,
		}

		assert.False(t, args.CanCancel())
		assert.Equal(t, EventEntityInserted, args.GetEventType())
	})

	t.Run("Inserted event can access created entity", func(t *testing.T) {
		entity := map[string]interface{}{"ID": 1, "Name": "John"}
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityInserted,
			Entity:    entity,
		}

		assert.Equal(t, entity, args.GetEntity())
	})
}

func TestEventLifecycle_Modifying(t *testing.T) {
	t.Run("Modifying event is cancelable", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Products",
			},
			EventType: EventEntityModifying,
			canCancel: true,
		}

		assert.True(t, args.CanCancel())
	})

	t.Run("Handler can modify data before update", func(t *testing.T) {
		handler := EventHandlerFunc(func(args EventArgs) error {
			entity := args.GetEntity().(map[string]interface{})
			entity["UpdatedAt"] = "2025-10-18"
			args.SetEntity(entity)
			return nil
		})

		entity := map[string]interface{}{"ID": 1, "Name": "Product"}
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Products",
			},
			EventType: EventEntityModifying,
			Entity:    entity,
		}

		handler.Handle(args)

		modifiedEntity := args.GetEntity().(map[string]interface{})
		assert.Equal(t, "2025-10-18", modifiedEntity["UpdatedAt"])
	})
}

func TestEventLifecycle_Modified(t *testing.T) {
	t.Run("Modified event is not cancelable", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Products",
			},
			EventType: EventEntityModified,
			canCancel: false,
		}

		assert.False(t, args.CanCancel())
	})
}

func TestEventLifecycle_Deleting(t *testing.T) {
	t.Run("Deleting event is cancelable", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Orders",
			},
			EventType: EventEntityDeleting,
			canCancel: true,
		}

		assert.True(t, args.CanCancel())
	})

	t.Run("Handler can prevent deletion", func(t *testing.T) {
		handler := EventHandlerFunc(func(args EventArgs) error {
			entity := args.GetEntity().(map[string]interface{})
			if entity["Status"] == "active" {
				args.Cancel("Cannot delete active entities")
			}
			return nil
		})

		entity := map[string]interface{}{"ID": 1, "Status": "active"}
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Orders",
			},
			EventType: EventEntityDeleting,
			Entity:    entity,
			canCancel: true,
		}

		handler.Handle(args)

		assert.True(t, args.IsCanceled())
		assert.Contains(t, args.GetCancelReason(), "Cannot delete")
	})
}

func TestEventLifecycle_Deleted(t *testing.T) {
	t.Run("Deleted event is not cancelable", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Orders",
			},
			EventType: EventEntityDeleted,
			canCancel: false,
		}

		assert.False(t, args.CanCancel())
	})
}

func TestEventLifecycle_Validation(t *testing.T) {
	t.Run("Validating event exists", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityValidating,
		}

		assert.Equal(t, EventEntityValidating, args.GetEventType())
	})

	t.Run("Validated event exists", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityValidated,
		}

		assert.Equal(t, EventEntityValidated, args.GetEventType())
	})
}

func TestEventLifecycle_Error(t *testing.T) {
	t.Run("Error event can be created", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityError,
		}

		assert.Equal(t, EventEntityError, args.GetEventType())
	})
}

func TestEventLifecycle_GetAndList(t *testing.T) {
	t.Run("Get event", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityGet,
		}

		assert.Equal(t, EventEntityGet, args.GetEventType())
	})

	t.Run("List event", func(t *testing.T) {
		args := &BaseEventArgs{
			Context: &EventContext{
				Context:    context.Background(),
				EntityName: "Users",
			},
			EventType: EventEntityList,
		}

		assert.Equal(t, EventEntityList, args.GetEventType())
	})
}
