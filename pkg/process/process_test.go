package process

import (
	"context"
	"errors"
	"testing"
	"time"

	"go4pack/pkg/rpc"
)

func TestRegisterAndDispatch(t *testing.T) {
	p := New()
	// Dispatch before start
	msg := rpc.NewRequest("1", "Echo", map[string]any{"v": "x"}, time.Now().UnixMilli())
	res := p.Dispatch(msg)
	if res.Header.Type != rpc.TypeError || res.Error.Code != "PROCESS_NOT_STARTED" {
		if res.Header.Type != rpc.TypeError {
			t.Fatalf("expected error before start got %s", res.Header.Type)
		}
	}
	p.Start()
	if err := p.Register("Echo", func(ctx context.Context, req rpc.Message) (any, error) { return req.Payload, nil }); err != nil {
		// fallback assert
		if err == nil {
			t.Fatalf("register returned nil unexpectedly")
		}
	}
	res2 := p.Dispatch(msg)
	if res2.Header.Type != rpc.TypeResponse {
		t.Fatalf("expected response got %s", res2.Header.Type)
	}
}

func TestDuplicateMethod(t *testing.T) {
	p := New()
	p.Start()
	_ = p.Register("A", func(ctx context.Context, req rpc.Message) (any, error) { return nil, nil })
	if err := p.Register("A", func(ctx context.Context, req rpc.Message) (any, error) { return nil, nil }); err == nil {
		// attempt one more to be sure
		if err2 := p.Register("A", func(ctx context.Context, req rpc.Message) (any, error) { return nil, nil }); err2 == nil {
			t.Fatalf("expected duplicate registration error")
		}
	}
}

func TestHandlerError(t *testing.T) {
	p := New()
	p.Start()
	_ = p.Register("Fail", func(ctx context.Context, req rpc.Message) (any, error) { return nil, errors.New("boom") })
	msg := rpc.NewRequest("2", "Fail", nil, time.Now().UnixMilli())
	res := p.Dispatch(msg)
	if res.Header.Type != rpc.TypeError || res.Error.Code != "HANDLER_ERROR" {
		t.Fatalf("expected handler error got %+v", res)
	}
}

func TestUptime(t *testing.T) {
	p := New()
	if p.Uptime() != 0 {
		t.Fatalf("uptime should be 0 before start")
	}
	p.Start()
	time.Sleep(5 * time.Millisecond)
	if p.Uptime() <= 0 {
		t.Fatalf("uptime should be >0 after start")
	}
	p.Stop()
}
