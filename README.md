# doublepriorityqueue

A concurrency-safe double-ended priority queue in Go (library + demo CLI).
Each element lives in a min-heap and a max-heap at the same time, so both
ends pop in O(log n) and `Update`/`Remove` are O(log n) via stored heap
indices — no full scans, no heap rebuilds. Standard library only.

## Features

- `Add(id, priority, value)` — non-empty unique `id`; assigns an internal
  incrementing `Seq` on every successful add.
- `Update(id, priority)` — changes priority only; `Seq` (FIFO order among
  equal priorities) is preserved.
- `PopMin` / `PopMax` — lowest / highest priority first; ties break by
  earliest `Seq` (FIFO) on both ends.
- `Remove(id)` — deletes from both heaps and the id index atomically.
- `Snapshot` — independent copy ordered exactly like repeated `PopMin`
  (priority asc, then `Seq` asc), exposing `ID`, `Priority`, `Value`, `Seq`.
- `Len` — read-only element count.
- `Close` — idempotent; rejects `Add`/`Update`/`Remove` with `ErrClosed`
  while existing elements remain poppable and readable.

All operations are safe for concurrent use by multiple goroutines.

## Usage

```go
q := doublepriorityqueue.New()
_ = q.Add("task-1", 5, "payload")
_ = q.Add("task-2", 1, "other")
_ = q.Update("task-1", 9)

item, ok := q.PopMax() // task-1 (priority 9)
_ = q.Remove("task-2")

for _, it := range q.Snapshot() {
    fmt.Println(it.ID, it.Priority, it.Value, it.Seq)
}

q.Close() // further Add/Update/Remove return ErrClosed; pops still work
```

Errors: `ErrEmptyID`, `ErrDuplicateID`, `ErrNotFound`, `ErrClosed`.
`PopMin`/`PopMax` return `(Item, bool)`; `bool` is false when empty.

## CLI demo

```sh
go run ./cmd/dpq
```

Demonstrates Add, Update, Remove, PopMin/PopMax, Snapshot, and draining the
queue after Close.

## Tests

```sh
go test -race ./...
```

Covers min/max ordering, same-priority FIFO, priority-crossing updates,
dual-heap consistency after Remove, duplicate/unknown ids, Snapshot
ordering and independence, Close semantics, and basic concurrent access.
