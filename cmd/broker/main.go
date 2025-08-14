package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go4pack/pkg/process"
	"go4pack/pkg/rpc"
)

// Broker uses the process model to manage a child go4pack API process
// and exposes simple RPC-style handlers (future transport TBD).
func main() {
	proc := process.New()

	// Prepare log file in same directory as executable
	exe, _ := os.Executable()
	dir := filepathDir(exe)
	logPath := dir + "/broker.log"
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// fallback to stdout
		lf = os.Stdout
	}
	logger := log.New(lf, "broker ", log.LstdFlags|log.Lmicroseconds)
	logger.Printf("starting broker (log=%s)", logPath)

	var mu sync.Mutex
	var child *exec.Cmd
	var childExit *rpc.RPCError
	var starting bool

	startChild := func(args []string) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		if child != nil && child.Process != nil && child.ProcessState == nil {
			return map[string]any{"status": "already_running", "pid": child.Process.Pid}, nil
		}
		if starting {
			return nil, nil
		}
		starting = true
		defer func() { starting = false }()
		selfDir, err := os.Executable()
		if err != nil {
			return nil, err
		}
		dir := filepathDir(selfDir)
		target := dir + "/go4pack"
		if _, err := os.Stat(target); err != nil {
			return nil, err
		}
		cmd := exec.Command(target, args...)
		cmd.Stdout = lf
		cmd.Stderr = lf
		cmd.Stdin = os.Stdin
		logger.Printf("starting child: %s pid=? args=%v", target, args)
		if err := cmd.Start(); err != nil {
			logger.Printf("child start error: %v", err)
			return nil, err
		}
		logger.Printf("child started pid=%d", cmd.Process.Pid)
		child = cmd
		go func() {
			err := cmd.Wait()
			mu.Lock()
			if err != nil {
				childExit = &rpc.RPCError{Code: "EXIT_ERROR", Message: err.Error()}
				logger.Printf("child exited (error) pid=%d err=%v", cmd.Process.Pid, err)
			} else if cmd.ProcessState != nil {
				childExit = &rpc.RPCError{Code: "EXITED", Message: cmd.ProcessState.String()}
				logger.Printf("child exited pid=%d state=%s", cmd.Process.Pid, cmd.ProcessState.String())
			}
			mu.Unlock()
		}()
		return map[string]any{"status": "started", "pid": cmd.Process.Pid}, nil
	}

	stopChild := func() (any, error) {
		mu.Lock()
		defer mu.Unlock()
		if child == nil || child.Process == nil || child.ProcessState != nil {
			return map[string]any{"status": "not_running"}, nil
		}
		logger.Printf("sending SIGTERM to child pid=%d", child.Process.Pid)
		if err := child.Process.Signal(syscall.SIGTERM); err != nil {
			logger.Printf("signal error: %v", err)
			return nil, err
		}
		return map[string]any{"status": "stopping"}, nil
	}

	status := func() any {
		mu.Lock()
		defer mu.Unlock()
		st := map[string]any{"running": false}
		if child != nil {
			if child.Process != nil && child.ProcessState == nil {
				st["running"] = true
				st["pid"] = child.Process.Pid
			}
			if childExit != nil {
				st["exit"] = childExit
			}
		}
		return st
	}

	proc.Start()
	_ = proc.Register("broker.start", func(ctx context.Context, m rpc.Message) (any, error) {
		args := []string{}
		if mp, ok := m.Payload.(map[string]any); ok {
			if raw, ok2 := mp["args"]; ok2 {
				if arr, ok3 := raw.([]any); ok3 {
					for _, v := range arr {
						if s, ok4 := v.(string); ok4 {
							args = append(args, s)
						}
					}
				}
			}
		}
		return startChild(args)
	})
	_ = proc.Register("broker.stop", func(ctx context.Context, m rpc.Message) (any, error) { return stopChild() })
	_ = proc.Register("broker.status", func(ctx context.Context, m rpc.Message) (any, error) { return status(), nil })
	_ = proc.Register("broker.ping", func(ctx context.Context, m rpc.Message) (any, error) {
		return map[string]any{"pong": true, "ts": time.Now().UnixMilli()}, nil
	})

	// Immediately start child with original CLI args (excluding program name)
	_, _ = startChild(os.Args[1:])

	// Signal forwarding
	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		sig := <-sigCh
		logger.Printf("received signal: %s", sig.String())
		if sig == syscall.SIGHUP {
			// Future: reload logic
			continue
		}
		mu.Lock()
		c := child
		mu.Unlock()
		if c != nil && c.Process != nil && c.ProcessState == nil {
			_ = c.Process.Signal(sig)
		}
		if sig == syscall.SIGINT || sig == syscall.SIGTERM {
			break
		}
	}
	logger.Printf("broker shutting down")
	// Allow child to exit gracefully
	time.Sleep(200 * time.Millisecond)
	proc.Stop()

	// Return child's exit code if available
	mu.Lock()
	if child != nil && child.ProcessState != nil {
		if ws, ok := child.ProcessState.Sys().(syscall.WaitStatus); ok {
			logger.Printf("exiting with child status=%d", ws.ExitStatus())
			os.Exit(ws.ExitStatus())
		}
	}
	mu.Unlock()
}

// small helper avoiding extra import just for filepath
func filepathDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}
