package utils

import (
	pq "github.com/emirpasic/gods/queues/priorityqueue"
	"sync"
)

type SafeHeap[T any] struct {
	p    *pq.Queue
	less func(a, b T) bool
	mu   sync.RWMutex
}

func NewSafeHeap[T any](less func(a, b T) bool) *SafeHeap[T] {
	return &SafeHeap[T]{
		p:    pq.NewWith(Comparator(less)),
		less: less,
	}
}

func (sh *SafeHeap[T]) Push(value T) {
	sh.mu.Lock()
	sh.p.Enqueue(value)
	sh.mu.Unlock()
}

func (sh *SafeHeap[T]) Pop() (*T, bool) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if sh.p.Empty() {
		return nil, false
	}
	if v, b := sh.p.Dequeue(); b || v == nil {
		return nil, false
	} else {
		return v.(*T), true
	}
}

func (sh *SafeHeap[T]) Size() int {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.p.Size()
}

func (sh *SafeHeap[T]) Drain() []T {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	values := make([]T, sh.p.Size())
	for i := 0; i < len(values); i++ {
		if v, b := sh.Pop(); b {
			values[i] = *v
		} else {
			return values[:i]
		}
	}
	return values
}

func (sh *SafeHeap[T]) Clear() {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.p.Clear()
}

func (sh *SafeHeap[T]) Values() []T {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	values := sh.p.Values()
	ret := make([]T, len(values))
	for i, v := range values {
		ret[i] = v.(T)
	}
	return ret
}

func (sh *SafeHeap[T]) Empty() bool {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.p.Empty()
}

func (sh *SafeHeap[T]) String() string {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.p.String()
}

func (sh *SafeHeap[T]) Peek() (*T, bool) {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	if v, b := sh.p.Peek(); b {
		return v.(*T), true
	} else {
		return nil, false
	}
}

func (sh *SafeHeap[T]) MapToInt(f func(T) int) []int {
	values := sh.Values()
	ret := make([]int, len(values))
	for i, v := range values {
		ret[i] = f(v)
	}
	return ret
}

func Comparator[T any](less func(T, T) bool) func(interface{}, interface{}) int {
	return func(a, b interface{}) int {
		if less(a.(T), b.(T)) {
			return -1
		} else if less(b.(T), a.(T)) {
			return 1
		}
		return 0
	}
}
