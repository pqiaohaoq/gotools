package cmap

import (
	"hash/fnv"
	"sync"
)

const DefaultShardNums = 128

type ConcurrentMap struct {
	shards   []*ConcurrentMapShard
	shardNum int
}

type ConcurrentMapShard struct {
	items map[string]interface{}
	sync.RWMutex
}

func New(shardNums int) *ConcurrentMap {
	if shardNums <= 0 {
		shardNums = DefaultShardNums
	}

	cm := &ConcurrentMap{
		shards:   make([]*ConcurrentMapShard, shardNums),
		shardNum: shardNums,
	}

	for i := 0; i < cm.shardNum; i++ {
		cm.shards[i] = &ConcurrentMapShard{items: make(map[string]interface{})}
	}

	return cm
}

func (cm *ConcurrentMap) GetShard(key string) *ConcurrentMapShard {
	hash := fnv.New32()
	_, _ = hash.Write([]byte(key))

	return cm.shards[hash.Sum32()%uint32(cm.shardNum)]
}

func (cm *ConcurrentMap) Get(key string) (interface{}, bool) {
	shard := cm.GetShard(key)

	shard.RLock()
	value, ok := shard.items[key]
	shard.RUnlock()

	return value, ok
}

func (cm *ConcurrentMap) Set(key string, value interface{}) {
	shard := cm.GetShard(key)

	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

func (cm *ConcurrentMap) Remove(key string) {
	shard := cm.GetShard(key)

	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

func (cm *ConcurrentMap) Count() int {
	count := 0

	for i := 0; i < cm.shardNum; i++ {
		shard := cm.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}

	return count
}

type Tuple struct {
	Key   string
	Value interface{}
}

func (cm *ConcurrentMap) IterBuffered() <-chan Tuple {
	chans := snapshot(cm)

	total := 0
	for _, c := range chans {
		total += cap(c)
	}

	ch := make(chan Tuple, total)
	go fanIn(chans, ch)

	return ch
}

func (cm *ConcurrentMap) Items() map[string]interface{} {
	temp := make(map[string]interface{})

	for item := range cm.IterBuffered() {
		temp[item.Key] = item.Value
	}

	return temp
}

// revive:disable:datarace
func snapshot(cm *ConcurrentMap) (chans []chan Tuple) {
	chans = make([]chan Tuple, cm.shardNum)

	wg := sync.WaitGroup{}
	wg.Add(cm.shardNum)
	for index, shard := range cm.shards {
		go func(index int, shard *ConcurrentMapShard) {
			shard.RLock()
			chans[index] = make(chan Tuple, len(shard.items))
			wg.Done()
			for key, value := range shard.items {
				chans[index] <- Tuple{Key: key, Value: value}
			}
			shard.RUnlock()

			close(chans[index])
		}(index, shard)
	}

	wg.Wait()
	return chans
}

func fanIn(chans []chan Tuple, out chan Tuple) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan Tuple) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}
