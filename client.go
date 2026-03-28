package esphome_apiclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/richard87/esphome-apiclient/pb"
	"github.com/richard87/esphome-apiclient/transport"
	"google.golang.org/protobuf/proto"
)

// Option is a configuration option for the Client.
type Option func(*Client)

// WithClientInfo sets the client_info sent in the HelloRequest.
func WithClientInfo(info string) Option {
	return func(c *Client) {
		c.clientInfo = info
	}
}

// WithEncryptionKey configures Noise protocol encryption using a base64-encoded
// 32-byte pre-shared key (the encryption.key from ESPHome config).
func WithEncryptionKey(key string) Option {
	return func(c *Client) {
		c.encryptionKey = key
	}
}

// WithExpectedName sets the expected device name validated during the Noise handshake.
// If empty, no name validation is performed.
func WithExpectedName(name string) Option {
	return func(c *Client) {
		c.expectedName = name
	}
}

// WithKeepalive configures the keepalive interval for PingRequests.
// Default is 20 seconds. Set to 0 to disable keepalive.
func WithKeepalive(interval time.Duration) Option {
	return func(c *Client) {
		c.keepaliveInterval = interval
	}
}

// WithKeepaliveTimeout configures the timeout for waiting for PingResponse.
// Default is 10 seconds.
func WithKeepaliveTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.keepaliveTimeout = timeout
	}
}

// WithReconnect enables automatic reconnection with the given base interval.
// Exponential backoff is applied (up to 5 minutes). Set to 0 to disable reconnect.
func WithReconnect(interval time.Duration) Option {
	return func(c *Client) {
		c.reconnectInterval = interval
	}
}

// WithOnConnect registers a callback that fires after (re)connection.
func WithOnConnect(fn func()) Option {
	return func(c *Client) {
		c.onConnect = fn
	}
}

// WithOnDisconnect registers a callback that fires on unexpected disconnection.
func WithOnDisconnect(fn func()) Option {
	return func(c *Client) {
		c.onDisconnect = fn
	}
}

// WithLogger sets a custom logger for the client.
func WithLogger(l *log.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// Client handles communication with an ESPHome device.
type Client struct {
	framer   Framer
	router   *Router
	entities *EntityRegistry
	services *ServiceRegistry
	done     chan struct{}
	writeMu  sync.Mutex

	clientInfo      string
	encryptionKey   string
	expectedName    string
	apiVersionMajor uint32
	apiVersionMinor uint32
	serverInfo      string
	name            string

	// Keepalive & Reconnect
	keepaliveInterval time.Duration
	keepaliveTimeout  time.Duration
	reconnectInterval time.Duration
	onConnect         func()
	onDisconnect      func()
	logger            *log.Logger

	// Connection management
	address   string
	timeout   time.Duration
	opts      []Option
	ctx       context.Context
	cancel    context.CancelFunc
	connected atomic.Bool
	closeMu   sync.Mutex
	closed    bool

	// State subscription handler (re-registered on reconnect)
	stateHandler func(msg proto.Message)
	stateMu      sync.Mutex
}

// Connected returns true if the client currently has an active connection.
func (c *Client) Connected() bool {
	return c.connected.Load()
}

// Name returns the device name from the HelloResponse.
func (c *Client) Name() string {
	return c.name
}

// ServerInfo returns the server info from the HelloResponse.
func (c *Client) ServerInfo() string {
	return c.serverInfo
}

// APIVersion returns the negotiated API version.
func (c *Client) APIVersion() (major, minor uint32) {
	return c.apiVersionMajor, c.apiVersionMinor
}

// Dial connects to the specified address.
func Dial(address string, timeout time.Duration, opts ...Option) (*Client, error) {
	return DialWithContext(context.Background(), address, timeout, opts...)
}

// DialWithContext connects to the specified address with a parent context.
// When the context is cancelled, all goroutines (keepalive, reconnect, read loop) exit cleanly.
func DialWithContext(ctx context.Context, address string, timeout time.Duration, opts ...Option) (*Client, error) {
	childCtx, cancel := context.WithCancel(ctx)

	c := &Client{
		router:            NewRouter(),
		entities:          NewEntityRegistry(),
		services:          NewServiceRegistry(),
		done:              make(chan struct{}),
		clientInfo:        "esphome-apiclient-go",
		apiVersionMajor:   1,
		apiVersionMinor:   10,
		keepaliveInterval: 20 * time.Second,
		keepaliveTimeout:  10 * time.Second,
		address:           address,
		timeout:           timeout,
		opts:              opts,
		ctx:               childCtx,
		cancel:            cancel,
	}

	for _, opt := range opts {
		opt(c)
	}

	if err := c.connect(); err != nil {
		cancel()
		return nil, err
	}

	c.connected.Store(true)
	go c.readLoop()

	if c.keepaliveInterval > 0 {
		go c.keepaliveLoop()
	}

	if c.onConnect != nil {
		c.onConnect()
	}

	return c, nil
}

// connect establishes the TCP connection (plain or noise) and performs the handshake.
func (c *Client) connect() error {
	if c.encryptionKey != "" {
		psk, err := base64.StdEncoding.DecodeString(c.encryptionKey)
		if err != nil {
			return fmt.Errorf("invalid encryption key: %w", err)
		}
		nt, err := transport.DialNoise(c.address, c.timeout, psk, c.expectedName)
		if err != nil {
			return err
		}
		c.framer = &NoiseFramer{transport: nt}
	} else {
		trans, err := transport.Dial(c.address, c.timeout)
		if err != nil {
			return err
		}
		c.framer = NewPlainFramer(trans)
	}

	if err := c.handshake(); err != nil {
		c.framer.Close()
		return err
	}

	return nil
}

func (c *Client) handshake() error {
	req := &pb.HelloRequest{
		ClientInfo:      c.clientInfo,
		ApiVersionMajor: c.apiVersionMajor,
		ApiVersionMinor: c.apiVersionMinor,
	}

	// HelloRequest has id = 1
	if err := c.SendMessage(req, 1); err != nil {
		return fmt.Errorf("failed to send HelloRequest: %w", err)
	}

	msg, msgType, err := c.RecvMessage()
	if err != nil {
		return fmt.Errorf("failed to receive HelloResponse: %w", err)
	}

	if msgType != 2 {
		return fmt.Errorf("expected HelloResponse (msgType 2), got %d", msgType)
	}

	resp, ok := msg.(*pb.HelloResponse)
	if !ok {
		return fmt.Errorf("expected HelloResponse, got %T", msg)
	}

	if resp.ApiVersionMajor != c.apiVersionMajor {
		return fmt.Errorf("unsupported API version %d.%d (expected %d.x)", resp.ApiVersionMajor, resp.ApiVersionMinor, c.apiVersionMajor)
	}

	c.apiVersionMajor = resp.ApiVersionMajor
	c.apiVersionMinor = resp.ApiVersionMinor
	c.serverInfo = resp.ServerInfo
	c.name = resp.Name

	return nil
}

// Close closes the connection to the ESPHome device and stops all background goroutines.
func (c *Client) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	c.connected.Store(false)

	if c.cancel != nil {
		c.cancel()
	}
	if c.framer != nil {
		return c.framer.Close()
	}
	return nil
}

// Done returns a channel that is closed when the read loop exits.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// On registers a handler for a specific message type.
func (c *Client) On(msgType uint32, handler MessageHandler) func() {
	return c.router.On(msgType, handler)
}

// requestResponse sends a request message and waits for a single response of the
// given type. It returns the first matching message or an error on timeout / close.
func (c *Client) requestResponse(reqMsg proto.Message, reqType uint32, respType uint32, timeout time.Duration) (proto.Message, error) {
	ch := make(chan proto.Message, 1)

	remove := c.On(respType, func(msg proto.Message) {
		select {
		case ch <- msg:
		default:
		}
	})
	defer remove()

	if err := c.SendMessage(reqMsg, reqType); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for response type %d", respType)
	case <-c.Done():
		return nil, fmt.Errorf("client closed while waiting for response type %d", respType)
	}
}

// DeviceInfo fetches device information.
func (c *Client) DeviceInfo() (*pb.DeviceInfoResponse, error) {
	resp, err := c.requestResponse(&pb.DeviceInfoRequest{}, 9, 10, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("DeviceInfo: %w", err)
	}
	return resp.(*pb.DeviceInfoResponse), nil
}

// Entities returns the client's entity registry.
func (c *Client) Entities() *EntityRegistry {
	return c.entities
}

// ListEntities fetches all entities from the device.
// It sends a ListEntitiesRequest and collects all entity response messages until
// the device sends ListEntitiesDoneResponse. Entities are also registered in
// the client's EntityRegistry.
func (c *Client) ListEntities() ([]proto.Message, error) {
	return c.ListEntitiesWithTimeout(10 * time.Second)
}

// ListEntitiesWithTimeout is like ListEntities but with a configurable timeout.
func (c *Client) ListEntitiesWithTimeout(timeout time.Duration) ([]proto.Message, error) {
	var entities []proto.Message
	var mu sync.Mutex
	done := make(chan struct{})

	// Register handler for ListEntitiesDoneResponse
	removeDone := c.On(pb.ListEntitiesDoneResponseID, func(msg proto.Message) {
		close(done)
	})
	defer removeDone()

	// Register handlers for all entity response types
	var removes []func()
	for _, id := range pb.ListEntityResponseIDs {
		removes = append(removes, c.On(id, func(msg proto.Message) {
			mu.Lock()
			entities = append(entities, msg)
			mu.Unlock()
			// Populate the entity registry
			c.entities.HandleListEntityMessage(msg)
			// Also handle service definitions
			if svcMsg, ok := msg.(*pb.ListEntitiesServicesResponse); ok {
				c.services.HandleServiceDefinition(svcMsg)
			}
		}))
	}
	defer func() {
		for _, rem := range removes {
			rem()
		}
	}()

	if err := c.SendMessage(&pb.ListEntitiesRequest{}, 11); err != nil {
		return nil, fmt.Errorf("ListEntities: failed to send request: %w", err)
	}

	select {
	case <-done:
		mu.Lock()
		defer mu.Unlock()
		return entities, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("ListEntities: timeout waiting for ListEntitiesDoneResponse")
	case <-c.Done():
		return nil, fmt.Errorf("ListEntities: client closed")
	}
}

// SubscribeStates subscribes to state updates.
// The handler is called for every incoming state response message.
// State updates also automatically update the entity registry.
// Returns an unsubscribe function that removes all registered handlers.
// The handler is saved so it can be re-registered on reconnect.
func (c *Client) SubscribeStates(handler func(msg proto.Message)) (unsubscribe func(), err error) {
	// Save handler for reconnect
	c.stateMu.Lock()
	c.stateHandler = handler
	c.stateMu.Unlock()

	var removes []func()
	for _, id := range pb.StateResponseIDs {
		removes = append(removes, c.On(id, func(msg proto.Message) {
			// Update entity registry state cache
			c.entities.HandleStateMessage(msg)
			// Call user handler
			if handler != nil {
				handler(msg)
			}
		}))
	}

	unsubscribe = func() {
		c.stateMu.Lock()
		c.stateHandler = nil
		c.stateMu.Unlock()
		for _, rem := range removes {
			rem()
		}
	}

	if err := c.SendMessage(&pb.SubscribeStatesRequest{}, 20); err != nil {
		unsubscribe()
		return nil, fmt.Errorf("SubscribeStates: failed to send request: %w", err)
	}

	return unsubscribe, nil
}

// Ping sends a PingRequest and waits for a PingResponse.
func (c *Client) Ping() error {
	return c.PingWithTimeout(5 * time.Second)
}

// PingWithTimeout sends a PingRequest and waits for a PingResponse with a configurable timeout.
func (c *Client) PingWithTimeout(timeout time.Duration) error {
	_, err := c.requestResponse(&pb.PingRequest{}, 7, 8, timeout)
	if err != nil {
		return fmt.Errorf("Ping: %w", err)
	}
	return nil
}

// Disconnect sends a DisconnectRequest, waits for DisconnectResponse, then closes the transport.
// It also cancels the context, stopping keepalive and reconnect goroutines.
func (c *Client) Disconnect() error {
	// Disable reconnect during intentional disconnect
	c.reconnectInterval = 0

	defer c.Close()
	_, err := c.requestResponse(&pb.DisconnectRequest{}, 5, 6, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Disconnect: %w", err)
	}
	return nil
}

func (c *Client) readLoop() {
	defer close(c.done)
	for {
		msg, msgType, err := c.RecvMessage()
		if err != nil {
			c.connected.Store(false)
			// Check if context was cancelled (clean shutdown)
			if c.ctx != nil {
				select {
				case <-c.ctx.Done():
					return
				default:
				}
			}

			// Unexpected disconnect
			if c.onDisconnect != nil {
				c.onDisconnect()
			}

			if c.reconnectInterval > 0 {
				go c.reconnectLoop()
			}
			return
		}

		if msg != nil {
			c.router.Dispatch(msgType, msg)
		}
		// Unknown message types (msg == nil) are silently skipped
	}
}

// keepaliveLoop periodically sends PingRequest messages and monitors for responses.
func (c *Client) keepaliveLoop() {
	ticker := time.NewTicker(c.keepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
			if !c.connected.Load() {
				return
			}
			err := c.PingWithTimeout(c.keepaliveTimeout)
			if err != nil {
				if c.logger != nil {
					c.logger.Printf("keepalive ping failed: %v", err)
				}
				// Connection is dead — close and trigger reconnect via readLoop
				c.closeMu.Lock()
				if c.framer != nil {
					c.framer.Close()
				}
				c.closeMu.Unlock()
				return
			}
		}
	}
}

// reconnectLoop attempts to reconnect with exponential backoff.
func (c *Client) reconnectLoop() {
	backoff := c.reconnectInterval
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(backoff):
		}

		if c.logger != nil {
			c.logger.Printf("attempting reconnect to %s...", c.address)
		}

		// Reset state for reconnect
		c.done = make(chan struct{})
		c.entities.Clear()
		c.services.Clear()

		err := c.connect()
		if err != nil {
			if c.logger != nil {
				c.logger.Printf("reconnect failed: %v", err)
			}
			// Exponential backoff
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		c.connected.Store(true)
		c.closeMu.Lock()
		c.closed = false
		c.closeMu.Unlock()

		go c.readLoop()

		if c.keepaliveInterval > 0 {
			go c.keepaliveLoop()
		}

		// Re-discover entities and re-subscribe
		if _, err := c.ListEntities(); err != nil {
			if c.logger != nil {
				c.logger.Printf("reconnect: failed to list entities: %v", err)
			}
		}

		// Re-subscribe to states if a handler was registered
		c.stateMu.Lock()
		handler := c.stateHandler
		c.stateMu.Unlock()
		if handler != nil {
			if _, err := c.SubscribeStates(handler); err != nil {
				if c.logger != nil {
					c.logger.Printf("reconnect: failed to resubscribe states: %v", err)
				}
			}
		}

		if c.onConnect != nil {
			c.onConnect()
		}

		if c.logger != nil {
			c.logger.Printf("reconnected to %s successfully", c.address)
		}
		return
	}
}

// SendMessage encodes and sends a protobuf message over the framer.
// It is safe to call from multiple goroutines.
func (c *Client) SendMessage(msg proto.Message, msgType uint32) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.framer.WriteFrame(msgType, data)
}

// RecvMessage reads a frame from the framer and unmarshals it into a protobuf message.
// If the message type is unknown, it returns a nil message and the type ID, without an error.
func (c *Client) RecvMessage() (proto.Message, uint32, error) {
	msgType, data, err := c.framer.ReadFrame()
	if err != nil {
		return nil, 0, err
	}

	factory, ok := pb.MessageRegistry[msgType]
	if !ok {
		// Log + skip unknown types instead of failing
		return nil, msgType, nil
	}

	msg := factory()
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, msgType, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return msg, msgType, nil
}
