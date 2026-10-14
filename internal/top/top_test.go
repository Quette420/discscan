package top

import (
	"math/rand"
	"slices"
	"sync"
	"testing"
)

func TestTopKeepsLargest(t *testing.T) {
	tp := New(5, func(a, b int) bool { return a < b })

	values := rand.Perm(1000)

	var wg sync.WaitGroup
	for _, chunk := range [][]int{values[:500], values[500:]} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, v := range chunk {
				tp.Add(v)
			}
		}()
	}
	wg.Wait()

	got := tp.Sorted()
	want := []int{999, 998, 997, 996, 995}

	if !slices.Equal(got, want) {
		t.Fatalf("Sorted() = %v, want %v", got, want)
	}
}

func TestTopFewerThanLimit(t *testing.T) {
	tp := New(10, func(a, b int) bool { return a < b })
	tp.Add(3)
	tp.Add(1)
	tp.Add(2)

	if got := tp.Sorted(); !slices.Equal(got, []int{3, 2, 1}) {
		t.Fatalf("Sorted() = %v", got)
	}
}

func TestTopZeroLimit(t *testing.T) {
	tp := New(0, func(a, b int) bool { return a < b })
	tp.Add(1)

	if got := tp.Sorted(); len(got) != 0 {
		t.Fatalf("Sorted() = %v, want empty", got)
	}
}
