package charts

import "math"

func Gauge(value, max float64, width int) string {
	if width <= 0 {
		return ""
	}
	if max <= 0 || math.IsNaN(value) || math.IsNaN(max) {
		return pad(width)
	}
	ratio := value / max
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(math.Round(ratio * float64(width)))
	if filled > width {
		filled = width
	}
	return repeat("█", filled) + repeat("░", width-filled)
}

func repeat(ch string, count int) string {
	if count <= 0 {
		return ""
	}
	out := make([]byte, 0, len(ch)*count)
	for i := 0; i < count; i++ {
		out = append(out, ch...)
	}
	return string(out)
}
