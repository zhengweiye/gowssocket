package gowssocket

import "sync/atomic"

type Balance interface {
	Get(num uint32) uint32
}

type RoundRobinBalance struct {
	index uint32
}

func newBalance() Balance {
	return RoundRobinBalance{}
}

func (r RoundRobinBalance) Get(num uint32) uint32 {
	newIndex := atomic.AddUint32(&r.index, 1)
	if newIndex > num {
		newIndex = 0
		r.index = 0
	}
	return newIndex % num
}
