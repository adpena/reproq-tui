package charts

import "math"

func Bar(values []float64, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(values) == 0 {
		return pad(width)
	}
	if len(values) > width {
		values = downsample(values, width)
	}
	_, max := minMax(values)
	if math.IsNaN(max) || max <= 0 {
		return pad(width)
	}
	cols := make([]int, len(values))
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
			cols[i] = 0
			continue
		}
		cols[i] = int(math.Round((v / max) * float64(height)))
	}

	lines := make([]string, 0, height)
	for row := height; row >= 1; row-- {
		line := make([]rune, len(values))
		for i, h := range cols {
			if h >= row {
				line[i] = '█'
			} else {
				line[i] = ' '
			}
		}
		lines = append(lines, string(line))
	}
	return joinLines(lines)
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}
