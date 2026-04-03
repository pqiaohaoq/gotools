package dynamicticker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDynamicTicker_NormalTick(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(180 * time.Millisecond)

	assert.True(t, atomic.LoadInt32(&count) >= 2, "expected at least 2 ticks, got %d", atomic.LoadInt32(&count))
}

func TestDynamicTicker_PauseWithZero(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(120 * time.Millisecond)

	beforePause := atomic.LoadInt32(&count)
	assert.True(t, beforePause >= 1, "should have ticked at least once before pause")

	dt.SetPeriod(0)
	time.Sleep(200 * time.Millisecond)

	afterPause := atomic.LoadInt32(&count)
	assert.Equal(t, beforePause, afterPause, "count should not increase after pause (period=0)")
}

func TestDynamicTicker_PauseWithNegative(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(120 * time.Millisecond)

	beforePause := atomic.LoadInt32(&count)
	assert.True(t, beforePause >= 1)

	dt.SetPeriod(-1 * time.Second)
	time.Sleep(200 * time.Millisecond)

	afterPause := atomic.LoadInt32(&count)
	assert.Equal(t, beforePause, afterPause, "count should not increase after pause (negative period)")
}

func TestDynamicTicker_ResumeAfterPause(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(120 * time.Millisecond)

	dt.SetPeriod(0)
	time.Sleep(150 * time.Millisecond)

	pausedCount := atomic.LoadInt32(&count)

	dt.SetPeriod(50 * time.Millisecond)
	time.Sleep(170 * time.Millisecond)

	resumedCount := atomic.LoadInt32(&count)
	assert.True(t, resumedCount > pausedCount, "should tick again after resume, got before=%d after=%d", pausedCount, resumedCount)
}

func TestDynamicTicker_StartWithZeroPeriod(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 0, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(0), atomic.LoadInt32(&count), "should not tick when started with period=0")

	// Resume
	dt.SetPeriod(50 * time.Millisecond)
	time.Sleep(170 * time.Millisecond)

	assert.True(t, atomic.LoadInt32(&count) >= 2, "should tick after setting a positive period")
}

func TestDynamicTicker_Stop(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(120 * time.Millisecond)

	dt.Stop()
	time.Sleep(100 * time.Millisecond)

	afterStop := atomic.LoadInt32(&count)
	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, afterStop, atomic.LoadInt32(&count), "count should not increase after Stop()")
}

func TestDynamicTicker_UpdatePeriod(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dt := NewDynamicTicker(ctx, 30*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(100 * time.Millisecond)

	fastCount := atomic.LoadInt32(&count)
	assert.True(t, fastCount >= 2, "expected at least 2 fast ticks, got %d", fastCount)

	dt.SetPeriod(200 * time.Millisecond)
	time.Sleep(250 * time.Millisecond)

	slowCount := atomic.LoadInt32(&count)
	assert.True(t, slowCount-fastCount <= 2, "slow period should produce fewer ticks")
}

func TestDynamicTicker_ContextCancel(t *testing.T) {
	var count int32
	ctx, cancel := context.WithCancel(context.Background())

	dt := NewDynamicTicker(ctx, 50*time.Millisecond, func() {
		atomic.AddInt32(&count, 1)
	})

	go dt.Run()
	time.Sleep(120 * time.Millisecond)

	cancel()
	time.Sleep(100 * time.Millisecond)

	afterCancel := atomic.LoadInt32(&count)
	time.Sleep(150 * time.Millisecond)

	assert.Equal(t, afterCancel, atomic.LoadInt32(&count), "count should not increase after context cancel")
}
