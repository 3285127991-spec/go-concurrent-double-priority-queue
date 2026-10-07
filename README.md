# doublepriorityqueue

A concurrent double-ended priority queue for Go. Library + tiny demo CLI,
standard library only, no third-party dependencies.

Each element lives in a min-heap and a max-heap at the same time, plus an
id index, so both ends pop in O(log n) and updates/removals never scan or
rebuild the heaps.

## Features

- `Add` / `Update` / `Remove` / `PopMin` / `PopMax` all O(log n)
- Same priority pops FIFO: every `Add` gets an internal increasing `Seq`
- `Update` changes only the priority; the original `Seq` is preserved
- Safe for concurrent use from multiple goroutines
- `Snapshot` returns an independent copy in `PopMin` order
- `Close` rejects mutations but lets you drain and read; idempotent

## API

```go
q := dpq.New()

err := q.Add("id-1", 5, anyValue) // id non-empty, must not exist
err  = q.Update("id-1", 9)        // priority only, Seq kept
err  = q.Remove("id-1")           // removes from both heaps
item, ok := q.PopMin()            // lowest priority, FIFO on ties
item, ok  = q.PopMax()            // highest priority, FIFO on ties
items := q.Snapshot()             // []dpq.Item in PopMin order
n     := q.Len()                  // read-only
q.Close()                         // reject Add/Update/Remove, idempotent
```

`dpq.Item` exposes `ID`, `Priority`, `Value`, `Seq`.
Errors: `ErrEmptyID`, `ErrDuplicateID`, `ErrUnknownID`, `ErrClosed`.

## Run

```sh
go test -race ./...     # run the tests with the race detector
go run ./cmd/dpq-cli    # demo: Add/Update/Remove, pop both ends, Close+drain
```

## Layout

- `dpq/dpq.go` - public Queue API, error values, concurrency control
- `dpq/heap.go` - internal binary heap with per-element index tracking
- `dpq/dpq_test.go` - ordering, FIFO, update, consistency, close, concurrency
- `cmd/dpq-cli/main.go` - minimal demonstration CLI
