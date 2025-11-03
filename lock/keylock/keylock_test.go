package keylock

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ConcurrentLockUnlock(t *testing.T) {
	kl := New()

	numGoroutines := 10000
	lockKey := "test_key"
	counter := 0

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kl.Lock(lockKey)
			counter++
			holders, _ := kl.Status(lockKey)
			assert.Equal(t, 1, holders)
			kl.Unlock(lockKey)
		}()
	}

	wg.Wait()

	holders, waiters := kl.Status(lockKey)
	assert.Equal(t, numGoroutines, counter)
	assert.Equal(t, 0, holders)
	assert.Equal(t, 0, waiters)
}
