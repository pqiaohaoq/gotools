package datastructs

import (
	"sync"
)

type SafeMap[T any] struct {
	data map[string]T
	sync.RWMutex
}

func NewSafeMap[T any]() *SafeMap[T] {
	return &SafeMap[T]{
		data: make(map[string]T),
	}
}

func (sm *SafeMap[T]) Get(key string) (T, bool) {
	sm.RLock()
	v, ok := sm.data[key]
	sm.RUnlock()

	return v, ok
}

func (sm *SafeMap[T]) Set(key string, value T) {
	sm.Lock()
	sm.data[key] = value
	sm.Unlock()
}

func (sm *SafeMap[T]) Remove(key string) {
	sm.Lock()
	delete(sm.data, key)
	sm.Unlock()
}

type Tuple[T any] struct {
	Key string
	Val T
}

func (sm *SafeMap[T]) IterBuffered() <-chan Tuple[T] {
	buffered := snapshot(sm)

	return buffered
}

func snapshot[T any](sm *SafeMap[T]) chan Tuple[T] {
	sm.RLock()

	buffered := make(chan Tuple[T], len(sm.data))
	for key, val := range sm.data {
		buffered <- Tuple[T]{Key: key, Val: val}
	}

	sm.RUnlock()

	close(buffered)

	return buffered
}
