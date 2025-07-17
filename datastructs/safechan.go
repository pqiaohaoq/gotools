package datastructs

import "sync"

type SafeChannel[T any] struct {
	ch     chan T
	closed bool
	mu     sync.Mutex
}

func NewSafeChannel[T any](size int) *SafeChannel[T] {
	return &SafeChannel[T]{
		ch: make(chan T, size),
	}
}

func (sc *SafeChannel[T]) Send(value T) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return false
	}

	sc.ch <- value

	return true
}

func (sc *SafeChannel[T]) Close() {
	sc.mu.Lock()
	sc.closed = true
	close(sc.ch)
	sc.mu.Unlock()
}

func (sc *SafeChannel[T]) RecvChan() <-chan T {
	return sc.ch
}
