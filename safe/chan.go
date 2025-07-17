package safe

import "sync"

type Channel[T any] struct {
	ch     chan T
	closed bool
	mu     sync.Mutex
}

func NewChannel[T any](size int) *Channel[T] {
	return &Channel[T]{
		ch: make(chan T, size),
	}
}

func (sc *Channel[T]) Send(value T) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.closed {
		return false
	}

	sc.ch <- value

	return true
}

func (sc *Channel[T]) Close() {
	sc.mu.Lock()
	sc.closed = true
	close(sc.ch)
	sc.mu.Unlock()
}

func (sc *Channel[T]) RecvChan() <-chan T {
	return sc.ch
}
