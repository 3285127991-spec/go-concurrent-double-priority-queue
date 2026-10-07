// Command dpq demonstrates the doublepriorityqueue library: Add, Update,
// Remove, PopMin/PopMax, Snapshot, and draining the queue after Close.
package main

import (
	"fmt"

	dpq "doublepriorityqueue"
)

func main() {
	q := dpq.New()

	fmt.Println("== Add ==")
	_ = q.Add("alpha", 5, "first")
	_ = q.Add("bravo", 1, "low-priority")
	_ = q.Add("charlie", 9, "urgent")
	_ = q.Add("delta", 1, "also-low")
	_ = q.Add("echo", 5, "fifth")
	printSnapshot(q)

	fmt.Println("== Update: bravo 1 -> 7 ==")
	_ = q.Update("bravo", 7)
	printSnapshot(q)

	fmt.Println("== Remove: echo ==")
	_ = q.Remove("echo")
	printSnapshot(q)

	fmt.Println("== Pop both ends ==")
	if it, ok := q.PopMin(); ok {
		fmt.Printf("PopMin -> %s (p=%d, seq=%d, v=%v)\n", it.ID, it.Priority, it.Seq, it.Value)
	}
	if it, ok := q.PopMax(); ok {
		fmt.Printf("PopMax -> %s (p=%d, seq=%d, v=%v)\n", it.ID, it.Priority, it.Seq, it.Value)
	}
	printSnapshot(q)

	fmt.Println("== Close ==")
	q.Close()
	q.Close() // idempotent
	if err := q.Add("fox", 3, "rejected"); err != nil {
		fmt.Println("Add after Close:", err)
	}
	if err := q.Update("alpha", 0); err != nil {
		fmt.Println("Update after Close:", err)
	}
	if err := q.Remove("alpha"); err != nil {
		fmt.Println("Remove after Close:", err)
	}

	fmt.Println("== Drain after Close (pops still allowed) ==")
	for q.Len() > 0 {
		it, _ := q.PopMin()
		fmt.Printf("PopMin -> %s (p=%d, seq=%d, v=%v)\n", it.ID, it.Priority, it.Seq, it.Value)
	}
	fmt.Println("Len:", q.Len())
}

func printSnapshot(q *dpq.Queue) {
	fmt.Printf("Len=%d Snapshot(PopMin order):\n", q.Len())
	for _, it := range q.Snapshot() {
		fmt.Printf("  %-8s p=%d seq=%d v=%v\n", it.ID, it.Priority, it.Seq, it.Value)
	}
}
