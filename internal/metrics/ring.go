package metrics

import (
	"time"

	"github.com/adpena/reproq-tui/pkg/models"
)

type RingBuffer struct {
	samples []models.Sample
	start   int
	count   int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		capacity = 1
	}
	return &RingBuffer{
		samples: make([]models.Sample, capacity),
	}
}

func (r *RingBuffer) Capacity() int {
	return len(r.samples)
}

func (r *RingBuffer) Len() int {
	return r.count
}

func (r *RingBuffer) Add(sample models.Sample) {
	if len(r.samples) == 0 {
		return
	}
	idx := (r.start + r.count) % len(r.samples)
	r.samples[idx] = sample
	if r.count < len(r.samples) {
		r.count++
		return
	}
	r.start = (r.start + 1) % len(r.samples)
}

func (r *RingBuffer) Latest() (models.Sample, bool) {
	if r.count == 0 {
		return models.Sample{}, false
	}
	idx := (r.start + r.count - 1) % len(r.samples)
	return r.samples[idx], true
}

func (r *RingBuffer) Values() []models.Sample {
	return r.ValuesSince(time.Time{})
}

func (r *RingBuffer) ValuesSince(cutoff time.Time) []models.Sample {
	if r.count == 0 {
		return nil
	}
	out := make([]models.Sample, 0, r.count)
	for i := 0; i < r.count; i++ {
		idx := (r.start + i) % len(r.samples)
		sample := r.samples[idx]
		if !cutoff.IsZero() && sample.Timestamp.Before(cutoff) {
			continue
		}
		out = append(out, sample)
	}
	return out
}
