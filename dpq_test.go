package doublepriorityqueue

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// validateConsistency checks that both heaps and the id index agree with
// each other: same elements, correct stored indices, valid heap ordering.
func validateConsistency(t *testing.T, q *Queue) {
	t.Helper()
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) != len(q.minH) || len(q.items) != len(q.maxH) {
		t.Fatalf("size mismatch: index=%d minHeap=%d maxHeap=%d",
			len(q.items), len(q.minH), len(q.maxH))
	}
	seenMin := make(map[string]int)
	seenMax := make(map[string]int)
	for i, e := range q.minH {
		if e.minIndex != i {
			t.Fatalf("minHeap[%d] id=%q has minIndex=%d", i, e.id, e.minIndex)
		}
		seenMin[e.id]++
		if q.items[e.id] != e {
			t.Fatalf("minHeap element %q not the indexed one", e.id)
		}
		if i > 0 {
			p := (i - 1) / 2
			if q.minH.Less(i, p) {
				t.Fatalf("minHeap order violated: child %q before parent %q", e.id, q.minH[p].id)
			}
		}
	}
	for i, e := range q.maxH {
		if e.maxIndex != i {
			t.Fatalf("maxHeap[%d] id=%q has maxIndex=%d", i, e.id, e.maxIndex)
		}
		seenMax[e.id]++
		if q.items[e.id] != e {
			t.Fatalf("maxHeap element %q not the indexed one", e.id)
		}
		if i > 0 {
			p := (i - 1) / 2
			if q.maxH.Less(i, p) {
				t.Fatalf("maxHeap order violated: child %q before parent %q", e.id, q.maxH[p].id)
			}
		}
	}
	for id := range q.items {
		if seenMin[id] != 1 || seenMax[id] != 1 {
			t.Fatalf("id %q appears min=%d max=%d times, want exactly 1 each",
				id, seenMin[id], seenMax[id])
		}
	}
}

func mustAdd(t *testing.T, q *Queue, id string, priority int, value any) {
	t.Helper()
	if err := q.Add(id, priority, value); err != nil {
		t.Fatalf("Add(%q): %v", id, err)
	}
}

func TestPopMinMaxOrder(t *testing.T) {
	q := New()
	for i, p := range []int{5, 1, 9, 3, 7} {
		mustAdd(t, q, fmt.Sprintf("id-%d", p), p, i)
	}
	validateConsistency(t, q)

	it, ok := q.PopMin()
	if !ok || it.ID != "id-1" || it.Priority != 1 {
		t.Fatalf("PopMin = %+v, %v; want id-1", it, ok)
	}
	it, ok = q.PopMax()
	if !ok || it.ID != "id-9" || it.Priority != 9 {
		t.Fatalf("PopMax = %+v, %v; want id-9", it, ok)
	}
	validateConsistency(t, q)

	var got []int
	for q.Len() > 0 {
		it, _ := q.PopMin()
		got = append(got, it.Priority)
	}
	want := []int{3, 5, 7}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("drain order = %v, want %v", got, want)
	}
	if _, ok := q.PopMin(); ok {
		t.Fatal("PopMin on empty queue should report false")
	}
	if _, ok := q.PopMax(); ok {
		t.Fatal("PopMax on empty queue should report false")
	}
	validateConsistency(t, q)
}

func TestSamePriorityFIFO(t *testing.T) {
	q := New()
	for _, id := range []string{"a", "b", "c", "d"} {
		mustAdd(t, q, id, 4, nil)
	}
	it, _ := q.PopMin()
	if it.ID != "a" || it.Seq != 0 {
		t.Fatalf("PopMin = %+v, want a/seq0", it)
	}
	it, _ = q.PopMax()
	if it.ID != "b" || it.Seq != 1 {
		t.Fatalf("PopMax = %+v, want b/seq1 (FIFO on ties)", it)
	}
	it, _ = q.PopMin()
	if it.ID != "c" {
		t.Fatalf("PopMin = %+v, want c", it)
	}
	it, _ = q.PopMax()
	if it.ID != "d" {
		t.Fatalf("PopMax = %+v, want d", it)
	}
	validateConsistency(t, q)
}

func TestUpdateCrossesPriorities(t *testing.T) {
	q := New()
	mustAdd(t, q, "low", 1, nil)
	mustAdd(t, q, "mid", 5, nil)
	mustAdd(t, q, "high", 9, nil)

	if err := q.Update("low", 10); err != nil {
		t.Fatalf("Update: %v", err)
	}
	validateConsistency(t, q)

	it, _ := q.PopMax()
	if it.ID != "low" || it.Priority != 10 {
		t.Fatalf("PopMax = %+v, want low@10 after update", it)
	}
	if it.Seq != 0 {
		t.Fatalf("Update changed Seq: got %d, want 0", it.Seq)
	}
	if err := q.Update("high", 0); err != nil {
		t.Fatalf("Update: %v", err)
	}
	it, _ = q.PopMin()
	if it.ID != "high" || it.Priority != 0 {
		t.Fatalf("PopMin = %+v, want high@0 after update", it)
	}
	validateConsistency(t, q)
}

func TestRemoveKeepsHeapsConsistent(t *testing.T) {
	q := New()
	ids := []string{"a", "b", "c", "d", "e", "f"}
	for i, id := range ids {
		mustAdd(t, q, id, (i*7)%5, id) // priorities: 0,2,4,1,3,0
	}
	for _, id := range []string{"b", "e", "a"} {
		if err := q.Remove(id); err != nil {
			t.Fatalf("Remove(%q): %v", id, err)
		}
		validateConsistency(t, q)
	}
	if q.Len() != 3 {
		t.Fatalf("Len = %d, want 3", q.Len())
	}
	// Remaining c(4), d(1), f(0): PopMin must not return removed ids.
	var got []string
	for q.Len() > 0 {
		it, _ := q.PopMin()
		got = append(got, it.ID)
	}
	want := []string{"f", "d", "c"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("drain = %v, want %v", got, want)
	}
	validateConsistency(t, q)
}

func TestDuplicateAndUnknownID(t *testing.T) {
	q := New()
	if err := q.Add("", 1, nil); !errors.Is(err, ErrEmptyID) {
		t.Fatalf("Add empty id: %v, want ErrEmptyID", err)
	}
	mustAdd(t, q, "x", 1, nil)
	if err := q.Add("x", 2, nil); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("Add duplicate: %v, want ErrDuplicateID", err)
	}
	if err := q.Update("nope", 3); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update unknown: %v, want ErrNotFound", err)
	}
	if err := q.Remove("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Remove unknown: %v, want ErrNotFound", err)
	}
	if q.Len() != 1 {
		t.Fatalf("Len = %d, want 1 (failed ops must not mutate)", q.Len())
	}
	validateConsistency(t, q)
}

func TestSnapshot(t *testing.T) {
	q := New()
	mustAdd(t, q, "a", 5, "va")
	mustAdd(t, q, "b", 1, "vb")
	mustAdd(t, q, "c", 5, "vc")

	snap := q.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("Snapshot len = %d, want 3", len(snap))
	}
	// PopMin order: b(1), a(5,seq0), c(5,seq2).
	wantIDs := []string{"b", "a", "c"}
	for i, it := range snap {
		if it.ID != wantIDs[i] {
			t.Fatalf("Snapshot[%d].ID = %q, want %q", i, it.ID, wantIDs[i])
		}
	}
	if snap[1].Value != "va" || snap[1].Seq != 0 {
		t.Fatalf("Snapshot item = %+v, want a with value va seq 0", snap[1])
	}

	// Mutating the returned slice must not affect the queue.
	snap[0].ID = "corrupted"
	if q.Snapshot()[0].ID != "b" {
		t.Fatal("queue state affected by mutating snapshot slice")
	}
	// Snapshot must be independent of later queue mutations.
	_ = q.Update("b", 99)
	_ = q.Remove("a")
	if snap[0].Priority != 1 || len(snap) != 3 {
		t.Fatalf("Snapshot changed after queue mutation: %+v", snap)
	}
	validateConsistency(t, q)
}

func TestClose(t *testing.T) {
	q := New()
	mustAdd(t, q, "a", 1, nil)
	mustAdd(t, q, "b", 2, nil)

	q.Close()
	q.Close() // must be idempotent

	if err := q.Add("c", 3, nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("Add after Close: %v, want ErrClosed", err)
	}
	if err := q.Update("a", 9); !errors.Is(err, ErrClosed) {
		t.Fatalf("Update after Close: %v, want ErrClosed", err)
	}
	if err := q.Remove("a"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Remove after Close: %v, want ErrClosed", err)
	}
	// Reads and pops still work on remaining elements.
	if q.Len() != 2 || len(q.Snapshot()) != 2 {
		t.Fatalf("Len/Snapshot after Close wrong: %d", q.Len())
	}
	if it, ok := q.PopMin(); !ok || it.ID != "a" {
		t.Fatalf("PopMin after Close = %+v, %v; want a", it, ok)
	}
	if it, ok := q.PopMax(); !ok || it.ID != "b" {
		t.Fatalf("PopMax after Close = %+v, %v; want b", it, ok)
	}
	validateConsistency(t, q)
}

func TestConcurrentAccess(t *testing.T) {
	q := New()
	const workers = 4
	const perWorker = 25

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				id := fmt.Sprintf("w%d-%d", w, i)
				if err := q.Add(id, i%7, i); err != nil {
					t.Errorf("Add: %v", err)
				}
				_ = q.Update(id, i%3)
				_ = q.Snapshot()
				_ = q.Len()
			}
		}(w)
	}
	popped := make(chan string, workers*perWorker)
	for p := 0; p < 2; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for {
				var it Item
				var ok bool
				if p%2 == 0 {
					it, ok = q.PopMin()
				} else {
					it, ok = q.PopMax()
				}
				if !ok {
					return
				}
				popped <- it.ID
			}
		}(p)
	}
	wg.Wait()
	// Poppers stop as soon as the queue is momentarily empty, so some
	// elements may remain; drain them before verifying. The channel buffer
	// holds workers*perWorker ids, so these sends never block.
	for {
		it, ok := q.PopMin()
		if !ok {
			break
		}
		popped <- it.ID
	}
	close(popped)
	seen := make(map[string]bool)
	count := 0
	for id := range popped {
		if seen[id] {
			t.Fatalf("id %q popped more than once", id)
		}
		seen[id] = true
		count++
	}
	if count != workers*perWorker {
		t.Fatalf("popped %d elements, want %d", count, workers*perWorker)
	}
	validateConsistency(t, q)
}
