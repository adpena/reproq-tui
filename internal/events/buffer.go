package events

import "github.com/adpena/reproq-tui/pkg/models"

type Buffer struct {
	items []models.Event
	size  int
}

func NewBuffer(size int) *Buffer {
	if size < 1 {
		size = 1
	}
	return &Buffer{
		items: make([]models.Event, 0, size),
		size:  size,
	}
}

func (b *Buffer) Add(event models.Event) {
	if len(b.items) < b.size {
		b.items = append(b.items, event)
		return
	}
	copy(b.items, b.items[1:])
	b.items[len(b.items)-1] = event
}

func (b *Buffer) Items() []models.Event {
	out := make([]models.Event, len(b.items))
	copy(out, b.items)
	return out
}

func (b *Buffer) Clear() {
	b.items = b.items[:0]
}
