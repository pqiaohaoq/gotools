package broadcast

import (
	"errors"
	"sync"

	"github.com/pqiaohaoq/gotools/datastructs"
	"github.com/pqiaohaoq/gotools/log"
	"go.uber.org/zap"
)

var (
	defaultOptions = &Options{
		logger: zap.NewNop().Sugar(),
	}
)

var (
	ErrTopicKeyIsEmpty    = errors.New("topic key is empty")
	ErrChannelNameIsEmpty = errors.New("channel name is empty")
	ErrChanIsNil          = errors.New("channel is nil")
)

type Broadcast struct {
	topics *datastructs.SafeMap[string, *datastructs.SafeMap[string, *datastructs.SafeChannel[any]]]
	logger log.Logger
}

type Option func(*Options)

type Options struct {
	logger log.Logger
}

func WithLogger(l log.Logger) Option { return func(o *Options) { o.logger = l } }

func applyOpts(opts []Option) Options {
	o := *defaultOptions

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

var (
	_globalBC = NewBroadcast()
	_globalMu sync.RWMutex
)

func ReplaceGlobals(bc *Broadcast) {
	_globalMu.Lock()
	_globalBC = bc
	_globalMu.Unlock()
}

func L() *Broadcast {
	_globalMu.RLock()
	bc := _globalBC
	_globalMu.RUnlock()

	return bc
}

func NewBroadcast(opts ...Option) *Broadcast {
	o := applyOpts(opts)

	return &Broadcast{
		topics: datastructs.NewSafeMap[string, *datastructs.SafeMap[string, *datastructs.SafeChannel[any]]](),
		logger: o.logger,
	}
}

func (bc *Broadcast) Subscribe(topicKey, chanName string, ch *datastructs.SafeChannel[any]) error {
	if topicKey == "" {
		return ErrTopicKeyIsEmpty
	}
	if chanName == "" {
		return ErrChannelNameIsEmpty
	}
	if ch == nil {
		return ErrChanIsNil
	}

	topic := bc.topics.GetOrSet(topicKey, func() *datastructs.SafeMap[string, *datastructs.SafeChannel[any]] {
		return datastructs.NewSafeMap[string, *datastructs.SafeChannel[any]]()
	})

	// TODO if duplicate topic and chan, will replace old
	topic.Set(chanName, ch)

	bc.logger.Infof("[broadcast] subscribe the topic %s with channel [name: %s pointer: %p]", topicKey, chanName, ch)

	return nil
}

func (bc *Broadcast) Unsubscribe(topicKey, chanName string) {
	if topicKey == "" || chanName == "" {
		return
	}

	topic, ok := bc.topics.Get(topicKey)
	if !ok {
		bc.logger.Infof("[broadcast] topic %s is not existed", topicKey)
		return
	}

	topic.Remove(chanName)

	bc.logger.Infof("[broadcast] chanName %s unsubscribe the topic %s", chanName, topicKey)
}

func (bc *Broadcast) PushMessage(topicKey string, message any) {
	bc.logger.Infof("[broadcast] topicKey %s, message: %+v", topicKey, message)

	topic, ok := bc.topics.Get(topicKey)
	if !ok {
		return
	}

	for tuple := range topic.IterBuffered() {
		chanName := tuple.Key
		ch := tuple.Val

		go func() {
			if !ch.Send(message) {
				bc.logger.Infof("[broadcast] channel [%s %p] is closed, purge it", chanName, ch)

				topic.Remove(chanName)
			}
		}()
	}
}
