package timewheel

import (
	"errors"
	"sync"
	"time"

	"github.com/pqiaohaoq/gotools/datastructs"
	"github.com/pqiaohaoq/gotools/log"
	"go.uber.org/zap"
)

var (
	ErrTaskKeyIsEmpty                = errors.New("task key is empty")
	ErrTaskDelayLessThanTickInterval = errors.New("task delay duration is less than 10ms")
	ErrDelayLessThanTickInterval     = errors.New("task delay duration is less than tick interval")
	ErrTaskDuplicatedKey             = errors.New("duplicated task key")
)

var (
	defaultOptions = &Options{
		slotNum:        600, // 1min/circle
		tickerInterval: 100 * time.Millisecond,
		logger:         zap.NewNop().Sugar(),
	}
)

type TimeWheel struct {
	currentPosition int

	slotNum     int
	slots       []*datastructs.SafeMap[string, *Task]
	keyPosition *datastructs.SafeMap[string, int]

	tickInterval time.Duration
	ticker       *time.Ticker

	stopChannel chan struct{}

	logger log.Logger
}

type TaskFunc func()

type Task struct {
	delay   time.Duration
	circle  int
	addTime time.Time

	runFunc TaskFunc
}

type Option func(*Options)

type Options struct {
	slotNum        int
	tickerInterval time.Duration
	logger         log.Logger
}

func WithTickerInterval(d time.Duration) Option { return func(o *Options) { o.tickerInterval = d } }
func WithSlotNum(n int) Option                  { return func(o *Options) { o.slotNum = n } }
func WithLogger(l log.Logger) Option            { return func(o *Options) { o.logger = l } }

func applyOpts(opts []Option) Options {
	o := *defaultOptions

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

var (
	_globalTW = NewTimeWheel()
	_globalMu sync.RWMutex
)

func ReplaceGlobals(tw *TimeWheel) {
	_globalMu.Lock()
	_globalTW = tw
	_globalMu.Unlock()
}

func L() *TimeWheel {
	_globalMu.RLock()
	tw := _globalTW
	_globalMu.RUnlock()

	return tw
}

func NewTimeWheel(options ...Option) *TimeWheel {
	o := applyOpts(options)

	tw := &TimeWheel{
		slotNum: o.slotNum,

		keyPosition: datastructs.NewSafeMap[string, int](),

		tickInterval: o.tickerInterval,
		ticker:       time.NewTicker(o.tickerInterval),

		stopChannel: make(chan struct{}),

		logger: o.logger,
	}

	tw.slots = make([]*datastructs.SafeMap[string, *Task], tw.slotNum)
	for i := range tw.slotNum {
		tw.slots[i] = datastructs.NewSafeMap[string, *Task]()
	}

	return tw
}

func (tw *TimeWheel) Start() {
	go tw.start()

	tw.logger.Infof("start the timewheel with tick interval: %s", tw.tickInterval)
}

func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandler()
		case <-tw.stopChannel:
			tw.ticker.Stop()
			return
		}
	}
}

func (tw *TimeWheel) tickHandler() {
	tw.logger.Debugf("tick the slot position %d", tw.currentPosition)

	tw.scanAndRunTask(tw.currentPosition)

	tw.currentPosition++
	if tw.currentPosition > tw.slotNum-1 {
		tw.currentPosition = 0
	}
}

func (tw *TimeWheel) Stop() {
	tw.stopChannel <- struct{}{}

	tw.logger.Infof("stop the timewheel")
}

func (tw *TimeWheel) AddTask(delay time.Duration, key string, taskFunc TaskFunc) error {
	if delay < 10*time.Millisecond {
		return ErrTaskDelayLessThanTickInterval
	}
	if delay < tw.tickInterval {
		return ErrDelayLessThanTickInterval
	}
	if key == "" {
		return ErrTaskKeyIsEmpty
	}

	if _, ok := tw.keyPosition.Get(key); ok {
		return ErrTaskDuplicatedKey
	}

	position, circle := tw.getPositionAndCircle(delay)

	task := &Task{
		delay:   delay,
		circle:  circle,
		addTime: time.Now(),
		runFunc: taskFunc,
	}

	tw.keyPosition.Set(key, position)
	slot := tw.slots[position]
	slot.Set(key, task)

	tw.logger.Debugf("add the task %s with delay %s into the slots (position: %d, circle: %d)", key, delay, position, circle)

	return nil
}

func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (int, int) {
	delay := d.Milliseconds()
	interval := tw.tickInterval.Milliseconds()

	circle := int((delay+interval-1)/interval) / tw.slotNum
	position := (tw.currentPosition + int((delay+interval-1)/interval)) % tw.slotNum

	return position, circle
}

func (tw *TimeWheel) RemoveTask(key string) {
	position, ok := tw.keyPosition.Get(key)
	if !ok {
		return
	}

	tw.keyPosition.Remove(key)
	slot := tw.slots[position]
	slot.Remove(key)
}

func (tw *TimeWheel) scanAndRunTask(position int) {
	slot := tw.slots[position]

	for tuple := range slot.IterBuffered() {
		key := tuple.Key
		task := tuple.Val

		tw.logger.Debugf("scan the task %s, delay: %s, circle: %d, addTime: %s", key, task.delay, task.circle, task.addTime)

		if task.circle > 0 {
			task.circle--

			continue
		}

		go func() {
			slot.Remove(key)
			tw.keyPosition.Remove(key)

			tw.logger.Debugf("execute the task %s", key)
			task.runFunc()
		}()
	}
}
