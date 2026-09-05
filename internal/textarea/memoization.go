// Package textarea - memoization helpers (inlined from charmbracelet/bubbles v1 textarea/memoization).
package textarea

import (
	"container/list"
	"sync"
)

// memoHasher is an interface that requires a Hash method.
type memoHasher interface {
	Hash() string
}

// memoEntry is a struct that holds a key-value pair.
type memoEntry[T any] struct {
	key   string
	value T
}

// memoCache is a simple LRU cache.
type memoCache[H memoHasher, T any] struct {
	capacity     int
	mutex        sync.Mutex
	cache        map[string]*list.Element
	evictionList *list.List
}

func newMemoCache[H memoHasher, T any](capacity int) *memoCache[H, T] {
	return &memoCache[H, T]{
		capacity:     capacity,
		cache:        make(map[string]*list.Element),
		evictionList: list.New(),
	}
}

func (m *memoCache[H, T]) Capacity() int {
	return m.capacity
}

func (m *memoCache[H, T]) Get(h H) (T, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hashedKey := h.Hash()
	if element, found := m.cache[hashedKey]; found {
		m.evictionList.MoveToFront(element)
		entry, ok := element.Value.(*memoEntry[T])
		if !ok {
			panic("memoCache: list element has unexpected type")
		}
		return entry.value, true
	}
	var result T
	return result, false
}

func (m *memoCache[H, T]) Set(h H, value T) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	hashedKey := h.Hash()
	if element, found := m.cache[hashedKey]; found {
		m.evictionList.MoveToFront(element)
		entry, ok := element.Value.(*memoEntry[T])
		if !ok {
			panic("memoCache: list element has unexpected type")
		}
		entry.value = value
		return
	}

	if m.evictionList.Len() >= m.capacity {
		toEvict := m.evictionList.Back()
		if toEvict != nil {
			removed := m.evictionList.Remove(toEvict)
			evictedEntry, ok := removed.(*memoEntry[T])
			if !ok {
				panic("memoCache: list element has unexpected type")
			}
			delete(m.cache, evictedEntry.key)
		}
	}

	newEntry := &memoEntry[T]{
		key:   hashedKey,
		value: value,
	}
	element := m.evictionList.PushFront(newEntry)
	m.cache[hashedKey] = element
}
