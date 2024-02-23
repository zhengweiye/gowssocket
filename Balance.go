package gowssocket

import (
	"sync"
	"sync/atomic"
)

type Balance interface {
	Get(num uint32) uint32
}

type RoundRobinBalance struct {
	index uint32
	lock  sync.RWMutex
}

func newBalance() Balance {
	return &RoundRobinBalance{}
}

func (r *RoundRobinBalance) Get(num uint32) uint32 {
	newIndex := atomic.AddUint32(&r.index, 1)
	if newIndex > num {
		r.lock.Lock()
		defer r.lock.Unlock()
		if newIndex > num {
			newIndex = 1
			r.index = 1
		} else {
			newIndex = atomic.AddUint32(&r.index, 1)
		}
	}
	return newIndex % num
}
