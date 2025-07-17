package safe

import (
	"sync"
)

type Map[K comparable, V any] struct {
	data map[K]V
	sync.RWMutex
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{data: make(map[K]V)}
}

func (sm *Map[K, V]) Get(key K) (V, bool) {
	sm.RLock()
	v, ok := sm.data[key]
	sm.RUnlock()

	return v, ok
}

func (sm *Map[K, V]) GetOrSet(key K, emptyFunc func() V) V {
	sm.Lock()
	v, ok := sm.data[key]
	if !ok {
		v = emptyFunc()
		sm.data[key] = v
	}
	sm.Unlock()

	return v
}

func (sm *Map[K, V]) Set(key K, value V) {
	sm.Lock()
	sm.data[key] = value
	sm.Unlock()
}

func (sm *Map[K, V]) Remove(key K) {
	sm.Lock()
	delete(sm.data, key)
	sm.Unlock()
}

type Tuple[K comparable, V any] struct {
	Key K
	Val V
}

func (sm *Map[K, V]) IterBuffered() <-chan Tuple[K, V] {
	buffered := snapshot(sm)

	return buffered
}

func snapshot[K comparable, V any](sm *Map[K, V]) chan Tuple[K, V] {
	sm.RLock()

	buffered := make(chan Tuple[K, V], len(sm.data))
	for key, val := range sm.data {
		buffered <- Tuple[K, V]{Key: key, Val: val}
	}

	sm.RUnlock()

	close(buffered)

	return buffered
}
