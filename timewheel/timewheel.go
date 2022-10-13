package timewheel

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"

	"github.com/qiaohao9/gotools/cmap"
)

var (
	_globalMu   sync.RWMutex
	_globalT, _ = New(100*time.Millisecond, 300)
)

func T() *TimeWheel {
	_globalMu.RLock()
	t := _globalT
	_globalMu.RUnlock()
	return t
}

func ReplaceGlobal(tt *TimeWheel) func() {
	_globalMu.Lock()
	prev := _globalT
	_globalT = tt
	_globalMu.RUnlock()

	return func() { ReplaceGlobal(prev) }
}

const (
	defaultMapShard       = 128
	defaultTaskConcurrent = 1000
)

type TimeoutCallbackFn func(Task)

type Task struct {
	Data            interface{}
	TimeoutCallback TimeoutCallbackFn

	key    string
	delay  time.Duration
	circle int

	elasped time.Duration
	now     time.Time
	end     time.Time
}

func (t *Task) Elasped() time.Duration {
	return t.elasped
}

type TimeWheel struct {
	slotNum         int
	currentPosition int

	timer *cmap.ConcurrentMap
	slots []*cmap.ConcurrentMap

	interval time.Duration
	ticker   *time.Ticker

	stopChannel chan bool

	taskGoroutine  *ants.Pool
	taskConcurrent int
}

type Option func(options *Options)

type Options struct {
	MapShard       int
	TaskConcurrent int
}

var defaultOptions = &Options{
	MapShard:       defaultMapShard,
	TaskConcurrent: defaultTaskConcurrent,
}

func WithMapShard(shard int) Option {
	return func(options *Options) {
		options.MapShard = shard
	}
}

func WithTaskConcurrent(concurrent int) Option {
	return func(options *Options) {
		options.TaskConcurrent = concurrent
	}
}

func evaluate(opts []Option) Options {
	o := *defaultOptions

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

func New(interval time.Duration, slotNum int, options ...Option) (*TimeWheel, error) {
	if interval <= 0 || slotNum <= 0 {
		return nil, errors.New("invalid parameter 'interval' or 'slotNum' must be large than zero")
	}

	o := evaluate(options)

	tw := &TimeWheel{
		slotNum:         slotNum,
		currentPosition: 0,

		timer: cmap.New(o.MapShard),
		slots: make([]*cmap.ConcurrentMap, slotNum),

		interval: interval,

		stopChannel:    make(chan bool),
		taskConcurrent: o.TaskConcurrent,
	}

	for i := 0; i < slotNum; i++ {
		tw.slots[i] = cmap.New(o.MapShard)
	}

	return tw, nil
}

func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.interval)
	tw.taskGoroutine, _ = ants.NewPool(tw.taskConcurrent)

	go tw.start()
}

func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandler()
		case <-tw.stopChannel:
			tw.ticker.Stop()
			tw.taskGoroutine.Release()

			return
		}
	}
}

func (tw *TimeWheel) Stop() {
	tw.stopChannel <- true
}

func (tw *TimeWheel) AddTask(delay time.Duration, key string, task *Task) error {
	if key == "" {
		return errors.New("parameter 'key' should not be empty")
	}

	if delay <= 0 {
		return errors.New("parameter 'delay' must be larger than zero")
	}

	if delay < tw.interval {
		return fmt.Errorf("parameter 'delay'=%d should not less than interval=%d", delay, tw.interval)
	}

	task.key = key
	task.delay = delay
	task.now = time.Now()

	tw.addTask(task)

	return nil
}

func (tw *TimeWheel) addTask(task *Task) {
	position, circle := tw.getPositionAndCircle(task.delay)
	task.circle = circle

	tw.slots[position].Set(task.key, task)
	tw.timer.Set(task.key, position)
}

func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (position int, circle int) {
	delaySeconds := d.Milliseconds()
	intervalSeconds := tw.interval.Milliseconds()

	circle = int((delaySeconds-intervalSeconds)/intervalSeconds) / tw.slotNum
	position = (tw.currentPosition - 1 + int(delaySeconds/intervalSeconds)) % tw.slotNum

	return
}

func (tw *TimeWheel) RemoveTask(key string) {
	if key == "" {
		return
	}

	tw.removeTask(key)
}

func (tw *TimeWheel) removeTask(key string) {
	position, ok := tw.timer.Get(key)
	if !ok {
		return
	}

	tw.timer.Remove(key)
	tw.slots[position.(int)].Remove(key)
}

func (tw *TimeWheel) HasTask(key string) bool {
	_, ok := tw.timer.Get(key)

	return ok
}

func (tw *TimeWheel) tickHandler() {
	tasks := tw.slots[tw.currentPosition]
	tw.scanAndRunTask(tasks)

	if tw.currentPosition == tw.slotNum-1 {
		tw.currentPosition = 0
	} else {
		tw.currentPosition++
	}
}

func (tw *TimeWheel) scanAndRunTask(tasks *cmap.ConcurrentMap) {
	for k, v := range tasks.Items() {
		key := k
		task := v.(*Task)

		if task.circle > 0 {
			task.circle--

			continue
		}

		if time.Since(task.now) < task.delay {
			continue
		}

		task.end = time.Now()
		task.elasped = task.end.Sub(task.now)

		_ = tw.taskGoroutine.Submit(func() {
			tw.timer.Remove(key)
			tasks.Remove(key)

			task.TimeoutCallback(*task)
		})

	}
}
