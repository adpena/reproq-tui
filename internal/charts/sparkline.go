package charts

import (
	"math"
)

var sparkChars = []rune("▁▂▃▄▅▆▇█")

func Sparkline(values []float64, width int) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return pad(width)
	}
	if len(values) > width {
		values = downsample(values, width)
	}
	min, max := minMax(values)
	if math.IsNaN(min) || math.IsNaN(max) || min == max {
		return pad(width, sparkChars[0])
	}
	out := make([]rune, 0, width)
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			out = append(out, sparkChars[0])
			continue
		}
		norm := (value - min) / (max - min)
		idx := int(math.Round(norm * float64(len(sparkChars)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		out = append(out, sparkChars[idx])
	}
	if len(out) < width {
		out = append(out, []rune(pad(width-len(out), sparkChars[0]))...)
	}
	return string(out)
}

func downsample(values []float64, width int) []float64 {
	if width <= 0 {
		return nil
	}
	if len(values) <= width {
		return values
	}
	step := float64(len(values)) / float64(width)
	out := make([]float64, 0, width)
	for i := 0; i < width; i++ {
		start := int(math.Floor(float64(i) * step))
		end := int(math.Floor(float64(i+1) * step))
		if end <= start {
			end = start + 1
		}
		if end > len(values) {
			end = len(values)
		}
		sum := 0.0
		count := 0
		for _, v := range values[start:end] {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			sum += v
			count++
		}
		if count == 0 {
			out = append(out, math.NaN())
		} else {
			out = append(out, sum/float64(count))
		}
	}
	return out
}

func minMax(values []float64) (float64, float64) {
	min := math.Inf(1)
	max := math.Inf(-1)
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min == math.Inf(1) || max == math.Inf(-1) {
		return math.NaN(), math.NaN()
	}
	return min, max
}

func pad(width int, ch ...rune) string {
	if width <= 0 {
		return ""
	}
	fill := ' '
	if len(ch) > 0 {
		fill = ch[0]
	}
	out := make([]rune, width)
	for i := range out {
		out[i] = fill
	}
	return string(out)
}
