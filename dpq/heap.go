package dpq

// heap is a binary heap over *element. Each element's position in this heap
// is tracked via setIndex so any element can be fixed or removed in O(log n).
type heap struct {
	items    []*element
	less     func(a, b *element) bool
	setIndex func(e *element, i int)
}

func (h *heap) len() int { return len(h.items) }

func (h *heap) push(e *element) {
	h.items = append(h.items, e)
	h.setIndex(e, len(h.items)-1)
	h.up(len(h.items) - 1)
}

func (h *heap) pop() *element {
	n := len(h.items)
	top := h.items[0]
	last := h.items[n-1]
	h.items = h.items[:n-1]
	if n > 1 {
		h.items[0] = last
		h.setIndex(last, 0)
		h.down(0)
	}
	h.setIndex(top, -1)
	return top
}

func (h *heap) removeAt(i int) {
	n := len(h.items)
	e := h.items[i]
	last := h.items[n-1]
	h.items = h.items[:n-1]
	if i != n-1 {
		h.items[i] = last
		h.setIndex(last, i)
		h.fix(i)
	}
	h.setIndex(e, -1)
}

func (h *heap) fix(i int) {
	if !h.down(i) {
		h.up(i)
	}
}

func (h *heap) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !h.less(h.items[i], h.items[parent]) {
			break
		}
		h.swap(i, parent)
		i = parent
	}
}

func (h *heap) down(i int) bool {
	moved := false
	n := len(h.items)
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		child := left
		if right := left + 1; right < n && h.less(h.items[right], h.items[left]) {
			child = right
		}
		if !h.less(h.items[child], h.items[i]) {
			break
		}
		h.swap(i, child)
		i = child
		moved = true
	}
	return moved
}

func (h *heap) swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.setIndex(h.items[i], i)
	h.setIndex(h.items[j], j)
}
