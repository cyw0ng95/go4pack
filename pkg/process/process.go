package process

import (
	"context"
	"errors"
	"sync"
	"time"

	"go4pack/pkg/rpc"
)

// Handler is a callback invoked for an RPC request message.
// It returns a response payload or an error which will be converted to rpc.RPCError.
type Handler func(ctx context.Context, req rpc.Message) (any, error)

// Process encapsulates application lifecycle and RPC callback registry.
type Process struct {
	mu        sync.RWMutex
	started   bool
	stopped   bool
	startTime time.Time
	methods   map[string]Handler
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new unstarted Process.
func New() *Process {
	return &Process{methods: make(map[string]Handler)}
}

// Register associates a handler with a method name. Returns error if method already exists or process stopped.
func (p *Process) Register(method string, h Handler) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return errors.New("process stopped")
	}
	if _, exists := p.methods[method]; exists {
		return errors.New("method already registered")
	}
	p.methods[method] = h
	return nil
}

// Start marks the process as started and prepares context.
func (p *Process) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return
	}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.startTime = time.Now()
	p.started = true
}

// Stop cancels context and marks process stopped.
func (p *Process) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	if p.cancel != nil {
		p.cancel()
	}
	p.stopped = true
}

// Dispatch processes an incoming RPC message and returns an outgoing message (response / error / ack).
func (p *Process) Dispatch(msg rpc.Message) rpc.Message {
	if msg.Header.Type != rpc.TypeRequest {
		// For now only handle requests; other types just echo ack
		return rpc.NewAck(msg.Header.ID, msg.Header.Seq, time.Now().UnixMilli(), true)
	}
	p.mu.RLock()
	h, ok := p.methods[msg.Header.Method]
	ctx := p.ctx
	started := p.started
	p.mu.RUnlock()
	if !started {
		return rpc.NewError(msg.Header.ID, "PROCESS_NOT_STARTED", "process not started", nil, time.Now().UnixMilli())
	}
	if !ok {
		return rpc.NewError(msg.Header.ID, "METHOD_NOT_FOUND", "unknown method", nil, time.Now().UnixMilli())
	}
	respPayload, err := h(ctx, msg)
	if err != nil {
		return rpc.NewError(msg.Header.ID, "HANDLER_ERROR", err.Error(), nil, time.Now().UnixMilli())
	}
	return rpc.NewResponse(msg.Header.ID, respPayload, time.Now().UnixMilli(), true)
}

// Uptime returns duration since start, zero if not started.
func (p *Process) Uptime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.started {
		return 0
	}
	return time.Since(p.startTime)
}
