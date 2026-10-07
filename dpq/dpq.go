// Package dpq implements a concurrent double-ended priority queue.
//
// Every element is indexed simultaneously by a min-heap and a max-heap,
// so PopMin and PopMax are both O(log n). Elements with equal priority
// are ordered FIFO by an internal monotonically increasing sequence
// number assigned on Add.
package dpq

import (
	"errors"
	"sort"
	"sync"
)

var (
	ErrEmptyID     = errors.New("dpq: id must be non-empty")
	ErrDuplicateID = errors.New("dpq: id already exists")
	ErrUnknownID   = errors.New("dpq: id not found")
	ErrClosed      = errors.New("dpq: queue is closed for mutation")
)

// Item is the public view of a queued element.
type Item struct {
	ID       string
	Priority int
	Value    any
	Seq      uint64
}

// element is the internal node shared by both heaps and the id index.
type element struct {
	item     Item
	minIndex int // position in the min-heap, -1 when not present
	maxIndex int // position in the max-heap, -1 when not present
}

// Queue is a goroutine-safe double-ended priority queue.
type Queue struct {
	mu     sync.RWMutex
	minH   heap
	maxH   heap
	byID   map[string]*element
	seq    uint64
	closed bool
}

// New returns an empty Queue.
func New() *Queue {
	q := &Queue{byID: make(map[string]*element)}
	q.minH = heap{
		less: func(a, b *element) bool {
			if a.item.Priority != b.item.Priority {
				return a.item.Priority < b.item.Priority
			}
			return a.item.Seq < b.item.Seq
		},
		setIndex: func(e *element, i int) { e.minIndex = i },
	}
	q.maxH = heap{
		less: func(a, b *element) bool {
			if a.item.Priority != b.item.Priority {
				return a.item.Priority > b.item.Priority
			}
			return a.item.Seq < b.item.Seq
		},
		setIndex: func(e *element, i int) { e.maxIndex = i },
	}
	return q
}

// Add inserts a new element. id must be non-empty and not already present.
// A new, ever-increasing Seq is assigned on every successful Add.
func (q *Queue) Add(id string, priority int, value any) error {
	if id == "" {
		return ErrEmptyID
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	if _, ok := q.byID[id]; ok {
		return ErrDuplicateID
	}
	q.seq++
	e := &element{item: Item{ID: id, Priority: priority, Value: value, Seq: q.seq}}
	q.minH.push(e)
	q.maxH.push(e)
	q.byID[id] = e
	return nil
}

// Update changes the priority of an existing element in O(log n).
// The element keeps its original Seq, so FIFO order among equal
// priorities still reflects the original Add order.
func (q *Queue) Update(id string, priority int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	e, ok := q.byID[id]
	if !ok {
		return ErrUnknownID
	}
	if e.item.Priority == priority {
		return nil
	}
	e.item.Priority = priority
	q.minH.fix(e.minIndex)
	q.maxH.fix(e.maxIndex)
	return nil
}

// Remove deletes an element by id in O(log n).
func (q *Queue) Remove(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	e, ok := q.byID[id]
	if !ok {
		return ErrUnknownID
	}
	q.removeBoth(e)
	return nil
}

// PopMin removes and returns the lowest-priority element (earliest Seq
// on ties). ok is false when the queue is empty.
func (q *Queue) PopMin() (item Item, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.minH.len() == 0 {
		return Item{}, false
	}
	e := q.minH.pop()
	q.maxH.removeAt(e.maxIndex)
	delete(q.byID, e.item.ID)
	return e.item, true
}

// PopMax removes and returns the highest-priority element (earliest Seq
// on ties). ok is false when the queue is empty.
func (q *Queue) PopMax() (item Item, ok bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.maxH.len() == 0 {
		return Item{}, false
	}
	e := q.maxH.pop()
	q.minH.removeAt(e.minIndex)
	delete(q.byID, e.item.ID)
	return e.item, true
}

// Snapshot returns an independent copy of all items in PopMin order.
func (q *Queue) Snapshot() []Item {
	q.mu.RLock()
	defer q.mu.RUnlock()
	items := make([]Item, len(q.minH.items))
	for i, e := range q.minH.items {
		items[i] = e.item
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].Seq < items[j].Seq
	})
	return items
}

// Len reports the number of elements currently in the queue.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.minH.len()
}

// Close marks the queue closed for mutation: Add, Update and Remove
// return ErrClosed afterwards. PopMin, PopMax, Snapshot and Len keep
// working on the remaining elements. Close is idempotent.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
}

// removeBoth deletes e from both heaps and the id index.
// Caller must hold q.mu.
func (q *Queue) removeBoth(e *element) {
	q.minH.removeAt(e.minIndex)
	q.maxH.removeAt(e.maxIndex)
	delete(q.byID, e.item.ID)
}
