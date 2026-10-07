// Package doublepriorityqueue provides a concurrency-safe double-ended
// priority queue. Each element is indexed simultaneously by a min-heap and
// a max-heap, so PopMin, PopMax, Update and Remove are all O(log n).
package doublepriorityqueue

import (
	"container/heap"
	"errors"
	"sort"
	"sync"
)

var (
	// ErrEmptyID is returned by Add when id is empty.
	ErrEmptyID = errors.New("doublepriorityqueue: id must be non-empty")
	// ErrDuplicateID is returned by Add when id already exists.
	ErrDuplicateID = errors.New("doublepriorityqueue: id already exists")
	// ErrNotFound is returned by Update and Remove for unknown ids.
	ErrNotFound = errors.New("doublepriorityqueue: id not found")
	// ErrClosed is returned by Add, Update and Remove after Close.
	ErrClosed = errors.New("doublepriorityqueue: queue is closed for mutations")
)

// Item is an immutable snapshot view of a single element.
type Item struct {
	ID       string
	Priority int
	Value    any
	Seq      uint64
}

// element is the internal node shared by both heaps and the id index.
type element struct {
	id       string
	priority int
	value    any
	seq      uint64
	minIndex int // position inside minHeap, -1 when absent
	maxIndex int // position inside maxHeap, -1 when absent
}

// minHeap pops the lowest priority first; ties break by smallest seq (FIFO).
type minHeap []*element

func (h minHeap) Len() int { return len(h) }

func (h minHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority < h[j].priority
	}
	return h[i].seq < h[j].seq
}

func (h minHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].minIndex = i
	h[j].minIndex = j
}

func (h *minHeap) Push(x any) {
	e := x.(*element)
	e.minIndex = len(*h)
	*h = append(*h, e)
}

func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	e.minIndex = -1
	*h = old[:n-1]
	return e
}

// maxHeap pops the highest priority first; ties break by smallest seq (FIFO).
type maxHeap []*element

func (h maxHeap) Len() int { return len(h) }

func (h maxHeap) Less(i, j int) bool {
	if h[i].priority != h[j].priority {
		return h[i].priority > h[j].priority
	}
	return h[i].seq < h[j].seq
}

func (h maxHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].maxIndex = i
	h[j].maxIndex = j
}

func (h *maxHeap) Push(x any) {
	e := x.(*element)
	e.maxIndex = len(*h)
	*h = append(*h, e)
}

func (h *maxHeap) Pop() any {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	e.maxIndex = -1
	*h = old[:n-1]
	return e
}

// Queue is a concurrency-safe double-ended priority queue.
// The zero value is not ready for use; create one with New.
type Queue struct {
	mu      sync.Mutex
	items   map[string]*element
	minH    minHeap
	maxH    maxHeap
	nextSeq uint64
	closed  bool
}

// New returns an empty, open Queue.
func New() *Queue {
	return &Queue{items: make(map[string]*element)}
}

// Add inserts a new element. id must be non-empty and unique.
// Each successful Add assigns the next internal Seq.
func (q *Queue) Add(id string, priority int, value any) error {
	if id == "" {
		return ErrEmptyID
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	if _, ok := q.items[id]; ok {
		return ErrDuplicateID
	}
	e := &element{id: id, priority: priority, value: value, seq: q.nextSeq}
	q.nextSeq++
	q.items[id] = e
	heap.Push(&q.minH, e)
	heap.Push(&q.maxH, e)
	return nil
}

// Update changes only the priority of an existing element; its Seq is kept.
func (q *Queue) Update(id string, priority int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	e, ok := q.items[id]
	if !ok {
		return ErrNotFound
	}
	e.priority = priority
	heap.Fix(&q.minH, e.minIndex)
	heap.Fix(&q.maxH, e.maxIndex)
	return nil
}

// Remove deletes an element from both heaps and the id index.
func (q *Queue) Remove(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	e, ok := q.items[id]
	if !ok {
		return ErrNotFound
	}
	q.removeLocked(e)
	return nil
}

// removeLocked deletes e from both heaps and the index. Caller holds q.mu.
func (q *Queue) removeLocked(e *element) {
	heap.Remove(&q.minH, e.minIndex)
	heap.Remove(&q.maxH, e.maxIndex)
	delete(q.items, e.id)
}

// PopMin removes and returns the element with the lowest priority
// (earliest Seq on ties). The second result is false when the queue is empty.
func (q *Queue) PopMin() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.minH) == 0 {
		return Item{}, false
	}
	e := heap.Pop(&q.minH).(*element)
	heap.Remove(&q.maxH, e.maxIndex)
	delete(q.items, e.id)
	return Item{ID: e.id, Priority: e.priority, Value: e.value, Seq: e.seq}, true
}

// PopMax removes and returns the element with the highest priority
// (earliest Seq on ties). The second result is false when the queue is empty.
func (q *Queue) PopMax() (Item, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.maxH) == 0 {
		return Item{}, false
	}
	e := heap.Pop(&q.maxH).(*element)
	heap.Remove(&q.minH, e.minIndex)
	delete(q.items, e.id)
	return Item{ID: e.id, Priority: e.priority, Value: e.value, Seq: e.seq}, true
}

// Snapshot returns an independent copy of all elements, ordered exactly as
// repeated PopMin calls would return them (priority asc, then Seq asc).
func (q *Queue) Snapshot() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Item, 0, len(q.items))
	for _, e := range q.items {
		out = append(out, Item{ID: e.id, Priority: e.priority, Value: e.value, Seq: e.seq})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].Seq < out[j].Seq
	})
	return out
}

// Len reports the number of elements currently in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// Close rejects further Add, Update and Remove calls. Existing elements can
// still be popped and read. Close is idempotent.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
}
