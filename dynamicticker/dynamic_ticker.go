package dynamicticker

import (
	"context"
	"sync"
	"time"
)

type DynamicTicker struct {
	ctx    context.Context
	cancel context.CancelFunc
	period chan time.Duration
	task   func()
	mu     sync.Mutex
	cur    time.Duration
}

func NewDynamicTicker(ctx context.Context, d time.Duration, task func()) *DynamicTicker {
	childCtx, cancel := context.WithCancel(ctx)

	return &DynamicTicker{
		ctx:    childCtx,
		cancel: cancel,
		period: make(chan time.Duration, 1),
		cur:    d,
		task:   task,
	}
}

func (dt *DynamicTicker) Run() {
	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	if dt.cur > 0 {
		ticker = time.NewTicker(dt.cur)
	}

	for {
		select {
		case <-dt.ctx.Done():
			return

		case d := <-dt.period:
			if d <= 0 {
				if ticker != nil {
					ticker.Stop()
					ticker = nil
				}

				dt.mu.Lock()
				dt.cur = d
				dt.mu.Unlock()

				continue
			}

			dt.mu.Lock()
			dt.cur = d
			dt.mu.Unlock()

			if ticker == nil {
				ticker = time.NewTicker(d)
			} else {
				ticker.Reset(d)
			}

		case <-func() <-chan time.Time {
			if ticker == nil {
				return nil
			}
			return ticker.C
		}():
			dt.task()
		}
	}
}

func (dt *DynamicTicker) SetPeriod(d time.Duration) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if d == dt.cur {
		return
	}

	dt.period <- d
}

func (dt *DynamicTicker) Stop() {
	dt.cancel()
}
