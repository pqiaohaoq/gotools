package safe

import "sync"

type Channel[T any] struct {
	C      chan T
	closed bool
	mu     sync.Mutex
}

func NewChannel[T any](size int) *Channel[T] {
	return &Channel[T]{
		C: make(chan T, size),
	}
}

func (sc *Channel[T]) Send(value T) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return false
	}

	sc.C <- value

	return true
}

func (sc *Channel[T]) Close() {
	sc.mu.Lock()
	sc.closed = true
	close(sc.C)
	sc.mu.Unlock()
}
