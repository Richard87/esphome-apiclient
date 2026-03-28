package esphome_apiclient

import (
	"sync"

	"google.golang.org/protobuf/proto"
)

// MessageHandler is a callback for a specific protobuf message type.
type MessageHandler func(msg proto.Message)

// Router maintains a registry mapping message type IDs to handlers.
type Router struct {
	mu       sync.RWMutex
	handlers map[uint32]map[uint64]MessageHandler
	nextID   uint64
}

// NewRouter creates a new MessageRouter.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint32]map[uint64]MessageHandler),
	}
}

// On registers a handler for a specific message type.
// It returns a function that can be called to remove the handler.
func (r *Router) On(msgType uint32, handler MessageHandler) func() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.handlers[msgType] == nil {
		r.handlers[msgType] = make(map[uint64]MessageHandler)
	}

	r.nextID++
	id := r.nextID
	r.handlers[msgType][id] = handler

	return func() {
		r.Remove(msgType, id)
	}
}

// Remove unregisters a handler by ID.
func (r *Router) Remove(msgType uint32, id uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.handlers[msgType]; ok {
		delete(m, id)
	}
}

// Dispatch calls all registered handlers for the given message type.
func (r *Router) Dispatch(msgType uint32, msg proto.Message) {
	r.mu.RLock()
	handlersMap := r.handlers[msgType]

	if len(handlersMap) == 0 {
		r.mu.RUnlock()
		return
	}

	// Copy handlers to avoid holding the lock during dispatch
	handlers := make([]MessageHandler, 0, len(handlersMap))
	for _, h := range handlersMap {
		handlers = append(handlers, h)
	}
	r.mu.RUnlock()

	for _, h := range handlers {
		// Handlers should not block, but we execute them in the same goroutine for now.
		h(msg)
	}
}
