// Command dpq-cli demonstrates the dpq library: Add, Update, Remove,
// PopMin/PopMax, and draining the queue after Close.
package main

import (
	"fmt"

	"doublepriorityqueue/dpq"
)

func main() {
	q := dpq.New()

	fmt.Println("-- Add 5 tasks --")
	q.Add("compile", 3, "build the binary")
	q.Add("test", 2, "run unit tests")
	q.Add("lint", 2, "run static checks")
	q.Add("docs", 5, "write README")
	q.Add("release", 9, "tag and ship")
	printSnapshot(q)

	fmt.Println("-- Update(docs, 1) and Remove(lint) --")
	must(q.Update("docs", 1))
	must(q.Remove("lint"))
	printSnapshot(q)

	fmt.Println("-- Pop both ends --")
	fmt.Println("PopMin:", popMin(q))
	fmt.Println("PopMax:", popMax(q))

	fmt.Println("-- Close, then drain --")
	q.Close()
	if err := q.Add("late", 0, nil); err != nil {
		fmt.Println("Add after Close rejected:", err)
	}
	for q.Len() > 0 {
		fmt.Println("PopMin:", popMin(q))
	}
	fmt.Println("queue empty, Len =", q.Len())
}

func printSnapshot(q *dpq.Queue) {
	for _, item := range q.Snapshot() {
		fmt.Printf("  seq=%d pri=%-3d id=%-8s value=%v\n",
			item.Seq, item.Priority, item.ID, item.Value)
	}
}

func popMin(q *dpq.Queue) string {
	item, _ := q.PopMin()
	return fmt.Sprintf("%s (pri=%d)", item.ID, item.Priority)
}

func popMax(q *dpq.Queue) string {
	item, _ := q.PopMax()
	return fmt.Sprintf("%s (pri=%d)", item.ID, item.Priority)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
