package cmap

import (
	"strconv"
	"sync"
	"testing"
)

type mapLock struct {
	data map[string]interface{}
	sync.RWMutex
}

func Benchmark_Strconv(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strconv.Itoa(i)
	}
}

func Benchmark_SingleSetCMap(b *testing.B) {
	m := New(128)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(strconv.Itoa(i), "value")
	}
}

func Benchmark_SingleSetMapLock(b *testing.B) {
	m := mapLock{data: make(map[string]interface{})}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Lock()
		m.data[strconv.Itoa(i)] = "value"
		m.Unlock()
	}
}

func Benchmark_SingleSetSyncMap(b *testing.B) {
	var m sync.Map

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(strconv.Itoa(i), "value")
	}
}

func Benchmark_SingleGetCMap(b *testing.B) {
	m := New(128)
	m.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get("key")
	}
}

func Benchmark_SingleGetMapLock(b *testing.B) {
	m := &mapLock{data: make(map[string]interface{})}
	m.data["key"] = "value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RLock()
		_ = m.data["key"]
		m.RUnlock()
	}
}

func Benchmark_SingleGetSyncMap(b *testing.B) {
	var m sync.Map
	m.Store("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Load("key")
	}
}

func Benchmark_MultiGetSameCMap(b *testing.B) {
	var wg sync.WaitGroup
	m := New(128)

	wg.Add(b.N)
	get, _ := GetSetCMap(m, &wg)
	m.Set("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go get("key", "value")
	}

	wg.Wait()
}

func Benchmark_MultiGetSameMapLock(b *testing.B) {
	var wg sync.WaitGroup
	m := &mapLock{data: make(map[string]interface{})}

	wg.Add(b.N)
	get, _ := GetSetMapLock(m, &wg)
	m.data["key"] = "value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go get("key", "value")
	}

	wg.Wait()
}

func Benchmark_MultiGetSameSycnMap(b *testing.B) {
	var (
		m  sync.Map
		wg sync.WaitGroup
	)

	wg.Add(b.N)
	get, _ := GetSetSyncMap(&m, &wg)
	m.Store("key", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go get("key", "value")
	}

	wg.Wait()
}

func Benchmark_MultiSetDifferentCMap(b *testing.B) {
	var wg sync.WaitGroup
	m := New(128)

	wg.Add(b.N)
	_, set := GetSetCMap(m, &wg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i), "value")
	}

	wg.Wait()
}

func Benchmark_MultiSetDifferentMapLock(b *testing.B) {
	var wg sync.WaitGroup
	m := &mapLock{data: make(map[string]interface{})}

	wg.Add(b.N)
	_, set := GetSetMapLock(m, &wg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i), "value")
	}

	wg.Wait()
}

func Benchmark_MultiSetDifferentSyncMap(b *testing.B) {
	var (
		m  sync.Map
		wg sync.WaitGroup
	)

	wg.Add(b.N)
	_, set := GetSetSyncMap(&m, &wg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i), "value")
	}

	wg.Wait()
}

func Benchmark_MultiGetSetDifferentCMap(b *testing.B) {
	var wg sync.WaitGroup
	m := New(128)

	wg.Add(2 * b.N)
	get, set := GetSetCMap(m, &wg)
	m.Set("-1", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i-1), "value")
		go get(strconv.Itoa(i), "value")
	}

	wg.Wait()
}

func Benchmark_MultiGetSetDifferentMapLock(b *testing.B) {
	var wg sync.WaitGroup
	m := &mapLock{data: make(map[string]interface{})}

	wg.Add(2 * b.N)
	get, set := GetSetMapLock(m, &wg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i-1), "value")
		go get(strconv.Itoa(i), "value")
	}

	wg.Wait()
}

func Benchmark_MultiGetSetDifferentSyncMap(b *testing.B) {
	var (
		m  sync.Map
		wg sync.WaitGroup
	)

	wg.Add(2 * b.N)
	get, set := GetSetSyncMap(&m, &wg)
	m.Store("-1", "value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i-1), "valuel")
		go get(strconv.Itoa(i), "value")
	}

	wg.Wait()
}

func GetSetCMap(m *ConcurrentMap, wg *sync.WaitGroup) (get func(key string, value interface{}), set func(key string, value interface{})) {
	get = func(key string, value interface{}) {
		for i := 0; i < 10; i++ {
			m.Get(key)
		}

		wg.Done()
	}

	set = func(key string, value interface{}) {
		for i := 0; i < 10; i++ {
			m.Set(key, value)
		}

		wg.Done()
	}

	return
}

func GetSetSyncMap(m *sync.Map, wg *sync.WaitGroup) (get func(key string, value interface{}), set func(key string, value interface{})) {
	get = func(key string, value interface{}) {
		for i := 0; i < 10; i++ {
			m.Load(key)
		}

		wg.Done()
	}

	set = func(key string, value interface{}) {
		for i := 0; i < 10; i++ {
			m.Store(key, value)
		}

		wg.Done()
	}

	return
}

func GetSetMapLock(m *mapLock, wg *sync.WaitGroup) (get func(key string, value interface{}), set func(key string, value interface{})) {
	get = func(key string, value interface{}) {
		for i := 0; i < 10; i++ {
			m.RLock()
			_ = m.data[key]
			m.RUnlock()
		}

		wg.Done()
	}

	set = func(key string, value interface{}) {
		for i := 0; i < 10; i++ {
			m.Lock()
			m.data[key] = value
			m.Unlock()
		}

		wg.Done()
	}

	return
}
