// Package top keeps the N best items out of an arbitrarily long stream.
package top

import (
	"container/heap"
	"sort"
	"sync"
)

// Top is a thread-safe bounded collection that retains the `limit` highest
// ranked items. Memory use is O(limit) regardless of how many items are added.
type Top[T any] struct {
	mu    sync.Mutex
	h     minHeap[T]
	limit int
}

// New creates a collector. less(a, b) reports whether a ranks lower than b.
func New[T any](limit int, less func(a, b T) bool) *Top[T] {
	return &Top[T]{
		h:     minHeap[T]{less: less},
		limit: limit,
	}
}

// Add offers an item; it is kept only if it ranks among the top `limit`.
func (t *Top[T]) Add(v T) {
	if t.limit <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.h.items) < t.limit {
		heap.Push(&t.h, v)
		return
	}

	// items[0] is the lowest ranked item currently kept.
	if t.h.less(t.h.items[0], v) {
		t.h.items[0] = v
		heap.Fix(&t.h, 0)
	}
}

// Sorted returns the kept items, highest ranked first.
func (t *Top[T]) Sorted() []T {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]T, len(t.h.items))
	copy(out, t.h.items)

	sort.SliceStable(out, func(i, j int) bool {
		return t.h.less(out[j], out[i])
	})

	return out
}

type minHeap[T any] struct {
	items []T
	less  func(a, b T) bool
}

func (h minHeap[T]) Len() int           { return len(h.items) }
func (h minHeap[T]) Less(i, j int) bool { return h.less(h.items[i], h.items[j]) }
func (h minHeap[T]) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *minHeap[T]) Push(x any) { h.items = append(h.items, x.(T)) }

func (h *minHeap[T]) Pop() any {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}
