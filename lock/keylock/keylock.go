package keylock

import (
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

var defaultConfig = &options{
	MaxPollInternval:    16 * time.Millisecond,
	InitialPollInterval: 1 * time.Millisecond,
}

type options struct {
	MaxPollInternval    time.Duration
	InitialPollInterval time.Duration
}

type Option func(*options)

func WithMaxPollInterval(d time.Duration) Option {
	return func(c *options) { c.MaxPollInternval = d }
}

func WithInitialPollInterval(d time.Duration) Option {
	return func(c *options) { c.InitialPollInterval = d }
}

func applyOpts(opts []Option) options {
	configCpy := *defaultConfig

	for _, opt := range opts {
		opt(&configCpy)
	}

	return configCpy
}

type lockEntry struct {
	holders int
	waiters int

	mu sync.RWMutex
}

type KeyLock struct {
	shards cmap.ConcurrentMap[string, *lockEntry]
	opts   options
}

func (kl *KeyLock) prepareLock(key string) *lockEntry {
	return kl.shards.Upsert(key, &lockEntry{}, func(exist bool, valueInMap, newValue *lockEntry) *lockEntry {
		le := newValue
		if exist {
			le = valueInMap
		}

		le.waiters++

		return le
	})
}

func (kl *KeyLock) commitLock(key string, le *lockEntry) {
	kl.shards.Upsert(key, le, func(exist bool, valueInMap, newValue *lockEntry) *lockEntry {
		lock := newValue
		if exist {
			lock = valueInMap
		}

		lock.waiters--
		lock.holders++

		return lock
	})
}

func (kl *KeyLock) releaseLock(key string, unlockFunc func(*sync.RWMutex)) {
	var le *lockEntry = nil

	kl.shards.RemoveCb(key, func(key string, v *lockEntry, exists bool) bool {
		if !exists {
			return false
		}

		le = v
		le.holders--

		return le.holders == 0 && le.waiters == 0
	})

	if le == nil {
		panic("keylock: unlock of unlocked key")
	}

	unlockFunc(&le.mu)
}

func (kl *KeyLock) cancelLock(key string) {
	kl.shards.RemoveCb(key, func(key string, v *lockEntry, exists bool) bool {
		if !exists {
			return false
		}

		v.waiters--

		return v.waiters == 0 && v.holders == 0
	})
}

func (kl *KeyLock) Lock(key string) {
	le := kl.prepareLock(key)
	le.mu.Lock()
	kl.commitLock(key, le)
}

func (kl *KeyLock) RLock(key string) {
	le := kl.prepareLock(key)
	le.mu.RLock()
	kl.commitLock(key, le)
}

func (kl *KeyLock) RUnlock(key string) {
	kl.releaseLock(key, (*sync.RWMutex).RUnlock)
}

func (kl *KeyLock) tryXLock(key string, tryXLockFunc func(*sync.RWMutex) bool) bool {
	le := kl.prepareLock(key)
	if tryXLockFunc(&le.mu) {
		kl.commitLock(key, le)
		return true
	}

	kl.cancelLock(key)

	return false
}

func (kl *KeyLock) TryLock(key string) bool {
	return kl.tryXLock(key, (*sync.RWMutex).TryLock)
}

func (kl *KeyLock) TryRLock(key string) bool {
	return kl.tryXLock(key, (*sync.RWMutex).TryRLock)
}

func (kl *KeyLock) TryLockWithTimeout(key string, timeout time.Duration) bool {
	return kl.TryXLockWithTimeout(key, timeout, (*sync.RWMutex).TryLock)
}

func (kl *KeyLock) TryRlockWithTimeout(key string, timeout time.Duration) bool {
	return kl.TryXLockWithTimeout(key, timeout, (*sync.RWMutex).TryRLock)
}

func (kl *KeyLock) TryXLockWithTimeout(key string, timeout time.Duration, tryXLockFunc func(*sync.RWMutex) bool) bool {
	le := kl.prepareLock(key)
	if tryXLockFunc(&le.mu) {
		kl.commitLock(key, le)
		return true
	}

	acquired := false
	defer func() {
		if !acquired {
			kl.cancelLock(key)
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	pollInterval := kl.opts.InitialPollInterval
	maxPollInterval := kl.opts.MaxPollInternval

	for {
		select {
		case <-timer.C:
		default:
			time.Sleep(pollInterval)
			if tryXLockFunc(&le.mu) {
				kl.commitLock(key, le)
				acquired = true

				return true
			}

			pollInterval *= 2
			if pollInterval >= maxPollInterval {
				pollInterval = maxPollInterval
			}
		}
	}
}

func (kl *KeyLock) Unlock(key string) {
	kl.releaseLock(key, (*sync.RWMutex).Unlock)
}

func (kl *KeyLock) Status(key string) (holders int, waiters int) {
	if le, ok := kl.shards.Get(key); ok {
		return le.holders, le.waiters
	}

	return 0, 0
}

func New(opts ...Option) *KeyLock {
	o := applyOpts(opts)

	return &KeyLock{
		shards: cmap.New[*lockEntry](),
		opts: options{
			MaxPollInternval:    o.MaxPollInternval,
			InitialPollInterval: o.InitialPollInterval,
		},
	}
}

var (
	_globalKeyLock   = New()
	_globalKeyLockMu sync.RWMutex
)

func ReplaceGlobalKeyLock(kl *KeyLock) {
	_globalKeyLockMu.Lock()
	_globalKeyLock = kl
	_globalKeyLockMu.Unlock()
}

func L() *KeyLock {
	_globalKeyLockMu.RLock()
	kl := _globalKeyLock
	_globalKeyLockMu.RUnlock()

	return kl
}
