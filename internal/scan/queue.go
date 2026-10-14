package scan

import "sync"

// dirQueue is a LIFO work queue of directories shared by the workers.
//
// pending counts directories that are queued or currently being processed;
// when it drops to zero the walk is complete and the queue closes.
type dirQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	dirs    []string
	pending int
	closed  bool
}

func newDirQueue(root string) *dirQueue {
	q := &dirQueue{
		dirs:    []string{root},
		pending: 1,
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// pop blocks until a directory is available or the walk is finished.
func (q *dirQueue) pop() (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.dirs) == 0 && !q.closed {
		q.cond.Wait()
	}

	if q.closed {
		return "", false
	}

	dir := q.dirs[len(q.dirs)-1]
	q.dirs = q.dirs[:len(q.dirs)-1]

	return dir, true
}

func (q *dirQueue) add(dir string) {
	q.mu.Lock()
	q.pending++
	q.dirs = append(q.dirs, dir)
	q.cond.Signal()
	q.mu.Unlock()
}

// done marks one popped directory as fully processed.
func (q *dirQueue) done() {
	q.mu.Lock()

	q.pending--
	if q.pending == 0 {
		q.closed = true
		q.cond.Broadcast()
	}

	q.mu.Unlock()
}
