package dpq_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"doublepriorityqueue/dpq"
)

func mustAdd(t *testing.T, q *dpq.Queue, id string, priority int) {
	t.Helper()
	if err := q.Add(id, priority, nil); err != nil {
		t.Fatalf("Add(%q): %v", id, err)
	}
}

func popMinID(t *testing.T, q *dpq.Queue) string {
	t.Helper()
	item, ok := q.PopMin()
	if !ok {
		t.Fatal("PopMin on empty queue")
	}
	return item.ID
}

func popMaxID(t *testing.T, q *dpq.Queue) string {
	t.Helper()
	item, ok := q.PopMax()
	if !ok {
		t.Fatal("PopMax on empty queue")
	}
	return item.ID
}

func TestPopMinMaxOrder(t *testing.T) {
	q := dpq.New()
	for _, p := range []int{5, 1, 9, 3, 7} {
		mustAdd(t, q, fmt.Sprintf("p%d", p), p)
	}
	if got := popMinID(t, q); got != "p1" {
		t.Fatalf("PopMin = %s, want p1", got)
	}
	if got := popMaxID(t, q); got != "p9" {
		t.Fatalf("PopMax = %s, want p9", got)
	}
	for _, want := range []string{"p3", "p5", "p7"} {
		if got := popMinID(t, q); got != want {
			t.Fatalf("PopMin = %s, want %s", got, want)
		}
	}
	if _, ok := q.PopMin(); ok {
		t.Fatal("PopMin on drained queue should report ok=false")
	}
	if _, ok := q.PopMax(); ok {
		t.Fatal("PopMax on drained queue should report ok=false")
	}
}

func TestSamePriorityFIFO(t *testing.T) {
	q := dpq.New()
	for _, id := range []string{"a", "b", "c", "d"} {
		mustAdd(t, q, id, 4)
	}
	if got := popMinID(t, q); got != "a" {
		t.Fatalf("PopMin = %s, want a", got)
	}
	if got := popMaxID(t, q); got != "b" {
		t.Fatalf("PopMax = %s, want b (FIFO on ties)", got)
	}
	if got := popMinID(t, q); got != "c" {
		t.Fatalf("PopMin = %s, want c", got)
	}
	if got := popMaxID(t, q); got != "d" {
		t.Fatalf("PopMax = %s, want d", got)
	}
}

func TestUpdateAcrossPriorities(t *testing.T) {
	q := dpq.New()
	mustAdd(t, q, "x", 5)
	mustAdd(t, q, "y", 5)
	mustAdd(t, q, "z", 1)

	if err := q.Update("z", 5); err != nil {
		t.Fatalf("Update: %v", err)
	}
	// All at priority 5 now; FIFO follows original Seq: x, y, z.
	for _, want := range []string{"x", "y", "z"} {
		if got := popMinID(t, q); got != want {
			t.Fatalf("PopMin = %s, want %s", got, want)
		}
	}

	mustAdd(t, q, "lo", 1)
	mustAdd(t, q, "hi", 10)
	if err := q.Update("hi", -3); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got := popMinID(t, q); got != "hi" {
		t.Fatalf("PopMin = %s, want hi after Update to -3", got)
	}
	if got := popMaxID(t, q); got != "lo" {
		t.Fatalf("PopMax = %s, want lo", got)
	}
}

func TestRemoveConsistency(t *testing.T) {
	q := dpq.New()
	ids := []string{"a", "b", "c", "d", "e", "f"}
	for i, id := range ids {
		mustAdd(t, q, id, i+1) // a=1 .. f=6
	}
	for _, id := range []string{"b", "e"} {
		if err := q.Remove(id); err != nil {
			t.Fatalf("Remove(%q): %v", id, err)
		}
	}
	if err := q.Remove("b"); !errors.Is(err, dpq.ErrUnknownID) {
		t.Fatalf("re-Remove = %v, want ErrUnknownID", err)
	}
	if got := q.Len(); got != 4 {
		t.Fatalf("Len = %d, want 4", got)
	}
	// Alternating pops from both ends must return each survivor exactly
	// once, proving the two heaps and the id index stayed in sync.
	want := []string{"a", "f", "c", "d"}
	for i, w := range want {
		var got string
		if i%2 == 0 {
			got = popMinID(t, q)
		} else {
			got = popMaxID(t, q)
		}
		if got != w {
			t.Fatalf("pop %d = %s, want %s", i, got, w)
		}
	}
	if got := q.Len(); got != 0 {
		t.Fatalf("Len = %d, want 0", got)
	}
}

func TestDuplicateAndUnknownID(t *testing.T) {
	q := dpq.New()
	if err := q.Add("", 1, nil); !errors.Is(err, dpq.ErrEmptyID) {
		t.Fatalf("Add empty id = %v, want ErrEmptyID", err)
	}
	mustAdd(t, q, "a", 1)
	if err := q.Add("a", 2, nil); !errors.Is(err, dpq.ErrDuplicateID) {
		t.Fatalf("Add duplicate = %v, want ErrDuplicateID", err)
	}
	if err := q.Update("ghost", 1); !errors.Is(err, dpq.ErrUnknownID) {
		t.Fatalf("Update unknown = %v, want ErrUnknownID", err)
	}
	if err := q.Remove("ghost"); !errors.Is(err, dpq.ErrUnknownID) {
		t.Fatalf("Remove unknown = %v, want ErrUnknownID", err)
	}
	if got := q.Len(); got != 1 {
		t.Fatalf("Len = %d, want 1 (failed ops must not mutate)", got)
	}
}

func TestSnapshot(t *testing.T) {
	q := dpq.New()
	mustAdd(t, q, "b", 2)
	mustAdd(t, q, "a", 1)
	mustAdd(t, q, "c", 1)

	snap := q.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("Snapshot len = %d, want 3", len(snap))
	}
	// PopMin order: priority asc, Seq asc on ties.
	wantIDs := []string{"a", "c", "b"}
	for i, item := range snap {
		if item.ID != wantIDs[i] {
			t.Fatalf("snap[%d].ID = %s, want %s", i, item.ID, wantIDs[i])
		}
		if item.Seq == 0 {
			t.Fatalf("snap[%d].Seq must be assigned", i)
		}
	}
	// The copy is independent: mutating it must not affect the queue.
	snap[0].Priority = 999
	snap[1].ID = "corrupted"
	again := q.Snapshot()
	if again[0].ID != "a" || again[0].Priority != 1 {
		t.Fatalf("queue changed after mutating snapshot: %+v", again[0])
	}
	if got := q.Len(); got != 3 {
		t.Fatalf("Len = %d, want 3 (Snapshot must not drain)", got)
	}
}

func TestClose(t *testing.T) {
	q := dpq.New()
	mustAdd(t, q, "a", 1)
	mustAdd(t, q, "b", 2)

	q.Close()
	q.Close() // idempotent, must not panic

	if err := q.Add("c", 3, nil); !errors.Is(err, dpq.ErrClosed) {
		t.Fatalf("Add after Close = %v, want ErrClosed", err)
	}
	if err := q.Update("a", 9); !errors.Is(err, dpq.ErrClosed) {
		t.Fatalf("Update after Close = %v, want ErrClosed", err)
	}
	if err := q.Remove("a"); !errors.Is(err, dpq.ErrClosed) {
		t.Fatalf("Remove after Close = %v, want ErrClosed", err)
	}
	// Reads and pops still work on the remaining elements.
	if got := q.Len(); got != 2 {
		t.Fatalf("Len after Close = %d, want 2", got)
	}
	if got := len(q.Snapshot()); got != 2 {
		t.Fatalf("Snapshot after Close len = %d, want 2", got)
	}
	if got := popMinID(t, q); got != "a" {
		t.Fatalf("PopMin after Close = %s, want a", got)
	}
	if got := popMaxID(t, q); got != "b" {
		t.Fatalf("PopMax after Close = %s, want b", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	q := dpq.New()
	const workers = 4
	const perWorker = 25

	done := make(chan struct{})
	var readers sync.WaitGroup
	readers.Add(1)
	go func() { // concurrent read-only traffic
		defer readers.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = q.Len()
				_ = q.Snapshot()
			}
		}
	}()

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				id := fmt.Sprintf("w%d-%d", w, i)
				if err := q.Add(id, i%7, nil); err != nil {
					t.Errorf("Add(%q): %v", id, err)
				}
			}
			for i := 0; i < perWorker; i += 5 { // update own ids
				if err := q.Update(fmt.Sprintf("w%d-%d", w, i), 100+i); err != nil {
					t.Errorf("Update: %v", err)
				}
			}
			for i := 1; i < perWorker; i += 5 { // remove own ids
				if err := q.Remove(fmt.Sprintf("w%d-%d", w, i)); err != nil {
					t.Errorf("Remove: %v", err)
				}
			}
		}(w)
	}
	wg.Wait()
	close(done)
	readers.Wait()

	const wantRemaining = workers * (perWorker - 5) // 5 removed per worker
	if got := q.Len(); got != wantRemaining {
		t.Fatalf("Len = %d, want %d", got, wantRemaining)
	}

	var mu sync.Mutex
	popped := make(map[string]int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for {
				var item dpq.Item
				var ok bool
				if w%2 == 0 {
					item, ok = q.PopMin()
				} else {
					item, ok = q.PopMax()
				}
				if !ok {
					return
				}
				mu.Lock()
				popped[item.ID]++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	if len(popped) != wantRemaining {
		t.Fatalf("popped %d distinct ids, want %d", len(popped), wantRemaining)
	}
	for id, n := range popped {
		if n != 1 {
			t.Fatalf("id %s popped %d times (duplicate across heaps?)", id, n)
		}
	}
	if got := q.Len(); got != 0 {
		t.Fatalf("Len = %d after drain, want 0", got)
	}
}
