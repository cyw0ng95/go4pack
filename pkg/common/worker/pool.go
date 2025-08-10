package worker

import (
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

type Job func()

var (
	pool     *ants.Pool
	initOnce sync.Once
	mu       sync.RWMutex
	stats    = struct {
		Submitted uint64
		Completed uint64
		LastErr   string
		LastDur   time.Duration
		LastAt    time.Time
	}{}
)

// Init initializes the global worker pool with the given size. Safe to call multiple times.
func Init(size int) error {
	var err error
	initOnce.Do(func() {
		pool, err = ants.NewPool(size, ants.WithNonblocking(true))
	})
	return err
}

// Submit enqueues a job for asynchronous execution.
func Submit(j Job) error {
	if pool == nil {
		if err := Init(4); err != nil { // default size
			return err
		}
	}
	mu.Lock()
	stats.Submitted++
	mu.Unlock()
	return pool.Submit(func() {
		start := time.Now()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("worker panic recovered")
				mu.Lock()
				stats.LastErr = "panic"
				stats.Completed++
				stats.LastDur = time.Since(start)
				stats.LastAt = time.Now()
				mu.Unlock()
				return
			}
			mu.Lock()
			stats.Completed++
			stats.LastDur = time.Since(start)
			stats.LastAt = time.Now()
			mu.Unlock()
		}()
		j()
	})
}

// Cap returns pool capacity.
func Cap() int {
	if pool == nil {
		return 0
	}
	return pool.Cap()
}

// Running returns currently running goroutines.
func Running() int {
	if pool == nil {
		return 0
	}
	return pool.Running()
}

// Free returns free worker number.
func Free() int {
	if pool == nil {
		return 0
	}
	return pool.Free()
}

// StatsSnapshot returns a copy of current pool statistics.
func StatsSnapshot() map[string]any {
	mu.RLock()
	defer mu.RUnlock()
	return map[string]any{
		"capacity":         Cap(),
		"running":          Running(),
		"free":             Free(),
		"submitted":        stats.Submitted,
		"completed":        stats.Completed,
		"queued_est":       int(stats.Submitted - stats.Completed - uint64(Running())),
		"last_error":       stats.LastErr,
		"last_duration_ms": stats.LastDur.Milliseconds(),
		"last_finished_at": stats.LastAt,
	}
}
