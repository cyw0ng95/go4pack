package worker

import (
	"sync"

	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
)

type Job func()

var (
	pool     *ants.Pool
	initOnce sync.Once
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
	return pool.Submit(func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("worker panic recovered")
			}
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
