package utils

import (
	"sort"
	"sync"
)

type MultiLock struct {
	types map[string]*lockType
}

type lockType struct {
	name            string
	mu              sync.RWMutex
	condition       sync.WaitGroup
	numThreads      int
	exclusiveGroups []exclusiveGroup
	dependencies    map[string]*lockType
	sorted          []string
}

type exclusiveGroup map[string]int

func newLockType(name string) *lockType {
	lt := &lockType{
		name:            name,
		numThreads:      0,
		exclusiveGroups: make([]exclusiveGroup, 0),
		dependencies:    make(map[string]*lockType),
	}
	return lt
}

func (l *MultiLock) numThreads(name string) int {
	l.types[name].mu.RLock()
	defer l.types[name].mu.RUnlock()
	return l.types[name].numThreads
}

func (l *MultiLock) lock(name string) func() {
	lt := l.types[name]
	lt.mu.Lock()
	lt.numThreads++
	allGood := true
	for _, group := range lt.exclusiveGroups {
		for t, maxThreads := range group {
			if l.numThreads(t) > maxThreads {
				allGood = false
				break
			}
		}
		if !allGood {

		}
	}

	return func() {
		lt.mu.Unlock()
	}
}

func NewMultiLock(exclusiveGroups ...[]string) *MultiLock {
	l := new(MultiLock)
	for _, group := range exclusiveGroups {
		var counts exclusiveGroup = make(map[string]int)
		for _, t := range group {
			if _, ok := counts[t]; !ok {
				counts[t] = 1
			} else {
				counts[t] = counts[t] + 1
			}
			if lType, ok := l.types[t]; !ok || lType == nil {
				l.types[t] = newLockType(t)
			}
		}
		for t, _ := range counts {
			l.types[t].exclusiveGroups = append(l.types[t].exclusiveGroups, counts)
			for t2, _ := range counts {
				l.types[t].dependencies[t2] = l.types[t2]
			}
		}
	}

	for _, lt := range l.types {
		var keys []string
		for k, _ := range lt.dependencies {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		lt.sorted = keys
	}
	return l
}

func (l *MultiLock) Lock(lockType string) func() {

	for _, t := range l.types[lockType].sorted {
		if t != lockType {
			l.types[t].mu.RLock()
		} else {
			l.types[t].mu.Lock()
		}
	}

	for _, group := range l.types[lockType].exclusiveGroups {
		lt := l.ifAllLockTypesAboveMaxThenGetATypeToWaitOn(group)
		for lt != nil {
			for _, t := range l.types[lockType].sorted {
				if t != lockType {
					l.types[t].mu.RUnlock()
				} else {
					l.types[t].mu.Unlock()
				}
			}
			lt.condition.Wait()
			for _, t := range l.types[lockType].sorted {
				if t != lockType {
					l.types[t].mu.RLock()
				} else {
					l.types[t].mu.Lock()
				}
			}
			lt = l.ifAllLockTypesAboveMaxThenGetATypeToWaitOn(group)
		}
	}

	l.types[lockType].numThreads++

	for _, t := range l.types[lockType].sorted {
		if t != lockType {
			l.types[t].mu.RUnlock()
		}
	}

	return func() {
		l.types[lockType].mu.Unlock()
	}
}

func (l *MultiLock) unlockAllExcept(lockType string, lt *lockType) {
	for _, t := range l.types[lockType].sorted {
		if t != lt.name {
			l.types[t].mu.Unlock()
		}
	}
}

func (l *MultiLock) lockAllExcept(lockType string, lt *lockType) {
	for _, t := range l.types[lockType].sorted {
		if t != lt.name {
			l.types[t].mu.Lock()
		}
	}
}

func (l *MultiLock) ifAllLockTypesAboveMaxThenGetATypeToWaitOn(group exclusiveGroup) *lockType {
	var lt *lockType = nil
	for t, maxThreads := range group {
		if l.numThreads(t) >= maxThreads {
			lt = l.types[t]
		} else {
			return nil
		}
	}
	return lt
}
