package packeter

import (
  "sync"
)

type collectors struct {
	Map map[uint32]*collector
	sync.RWMutex
}

func newcollectors() *collectors {
	return &collectors{
		Map: make(map[uint32]*collector),
	}
}

func (t *collectors) get(key uint32) (*collector, bool) {
	t.RLock()
	k, b := t.Map[key]
	t.RUnlock()
	return k, b
}

func (t *collectors) set(key uint32, val *collector) {
	t.Lock()
	t.Map[key] = val
	t.Unlock()
}

func (t *collectors) delete(keys ...uint32) {
	t.Lock()
	for _, key := range keys {
		delete(t.Map, key)
	}
	t.Unlock()
}


