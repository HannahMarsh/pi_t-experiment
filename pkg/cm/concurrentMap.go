package cm

import "sync"

// ConcurrentMap wraps around sync.Map
type ConcurrentMap[K comparable, V any] struct {
	m sync.Map
}

// Set adds or updates a value in the map for a given key.
func (cm *ConcurrentMap[K, V]) Set(key K, value V) {
	cm.m.Store(key, value)
}

// Get retrieves a value from the map for a given key.
func (cm *ConcurrentMap[K, V]) Get(key K) V {
	var zeroValue V
	value, ok := cm.m.LoadOrStore(key, zeroValue)
	if ok {
		return value.(V)
	}
	return zeroValue
}

func (cm *ConcurrentMap[K, V]) GetAndSet(key K, updateFunc func(V) V) {
	go func(key K, updateFunc func(V) V, cm *ConcurrentMap[K, V]) {
		for {
			currentValue := cm.Get(key)
			newValue := updateFunc(currentValue)
			if cm.m.CompareAndSwap(key, currentValue, newValue) {
				return
			}
		}
	}(key, updateFunc, cm)
}
