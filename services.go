package esphome_apiclient

import (
	"fmt"
	"sync"

	"github.com/richard87/esphome-apiclient/pb"
)

// ServiceDefinition represents a discovered ESPHome service with its argument schema.
type ServiceDefinition struct {
	Name             string
	Key              uint32
	Args             []*pb.ListEntitiesServicesArgument
	SupportsResponse pb.SupportsResponseType
}

// ServiceRegistry caches discovered services from ListEntitiesServicesResponse messages.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[uint32]*ServiceDefinition
	byName   map[string]*ServiceDefinition
}

// NewServiceRegistry creates an empty service registry.
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[uint32]*ServiceDefinition),
		byName:   make(map[string]*ServiceDefinition),
	}
}

// HandleServiceDefinition processes a ListEntitiesServicesResponse message.
func (r *ServiceRegistry) HandleServiceDefinition(msg *pb.ListEntitiesServicesResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()
	svc := &ServiceDefinition{
		Name:             msg.Name,
		Key:              msg.Key,
		Args:             msg.Args,
		SupportsResponse: msg.SupportsResponse,
	}
	r.services[msg.Key] = svc
	r.byName[msg.Name] = svc
}

// ByKey returns a service definition by key, or nil if not found.
func (r *ServiceRegistry) ByKey(key uint32) *ServiceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[key]
}

// ByName returns a service definition by name, or nil if not found.
func (r *ServiceRegistry) ByName(name string) *ServiceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[name]
}

// All returns a copy of all service definitions.
func (r *ServiceRegistry) All() []*ServiceDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*ServiceDefinition, 0, len(r.services))
	for _, svc := range r.services {
		result = append(result, svc)
	}
	return result
}

// Clear removes all services.
func (r *ServiceRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services = make(map[uint32]*ServiceDefinition)
	r.byName = make(map[string]*ServiceDefinition)
}

// ExecuteService sends an ExecuteServiceRequest to the device.
// The args slice should contain one ExecuteServiceArgument per declared argument.
func (c *Client) ExecuteService(key uint32, args []*pb.ExecuteServiceArgument) error {
	req := &pb.ExecuteServiceRequest{
		Key:  key,
		Args: args,
	}
	// ExecuteServiceRequest has message type ID 42
	if err := c.SendMessage(req, 42); err != nil {
		return fmt.Errorf("ExecuteService: failed to send request: %w", err)
	}
	return nil
}

// ExecuteServiceByName looks up a service by name and executes it.
func (c *Client) ExecuteServiceByName(name string, args []*pb.ExecuteServiceArgument) error {
	svc := c.services.ByName(name)
	if svc == nil {
		return fmt.Errorf("ExecuteServiceByName: service %q not found", name)
	}
	return c.ExecuteService(svc.Key, args)
}

// Services returns the client's service registry.
func (c *Client) Services() *ServiceRegistry {
	return c.services
}
