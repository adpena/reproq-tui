package ui

import (
	"fmt"
	"math"
	"time"
)

func formatNumber(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "-"
	}
	abs := math.Abs(value)
	switch {
	case abs >= 1_000_000:
		return fmt.Sprintf("%.1fM", value/1_000_000)
	case abs >= 1_000:
		return fmt.Sprintf("%.1fk", value/1_000)
	case abs >= 100:
		return fmt.Sprintf("%.0f", value)
	case abs >= 10:
		return fmt.Sprintf("%.1f", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

func formatCount(value int64) string {
	if value < 0 {
		return "-"
	}
	switch {
	case value >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.1fk", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func formatRate(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "-"
	}
	return fmt.Sprintf("%s/s", formatNumber(value))
}

func formatPercent(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", value*100)
}

func formatBytes(value float64) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "-"
	}
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%.0f B", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", value/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Truncate(time.Second).String()
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05")
}

func formatRelative(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	ago := time.Since(t)
	if ago < time.Second {
		return "just now"
	}
	if ago < time.Minute {
		return fmt.Sprintf("%ds ago", int(ago.Seconds()))
	}
	if ago < time.Hour {
		return fmt.Sprintf("%dm ago", int(ago.Minutes()))
	}
	return fmt.Sprintf("%dh ago", int(ago.Hours()))
}

func formatScheduleAt(target, now time.Time) string {
	if target.IsZero() {
		return "-"
	}
	delta := target.Sub(now)
	if delta >= 0 {
		if delta < time.Second {
			return "now"
		}
		if delta < time.Minute {
			return fmt.Sprintf("in %ds", int(delta.Seconds()))
		}
		if delta < time.Hour {
			return fmt.Sprintf("in %dm", int(delta.Minutes()))
		}
		if delta < 24*time.Hour {
			return fmt.Sprintf("in %dh", int(delta.Hours()))
		}
		return target.Format("Jan 2")
	}
	overdue := -delta
	if overdue < time.Second {
		return "now"
	}
	if overdue < time.Minute {
		return fmt.Sprintf("%ds overdue", int(overdue.Seconds()))
	}
	if overdue < time.Hour {
		return fmt.Sprintf("%dm overdue", int(overdue.Minutes()))
	}
	if overdue < 24*time.Hour {
		return fmt.Sprintf("%dh overdue", int(overdue.Hours()))
	}
	return target.Format("Jan 2")
}

func formatScheduleDetailed(target, now time.Time) string {
	if target.IsZero() {
		return "-"
	}
	abs := formatTimestamp(target)
	rel := formatScheduleAt(target, now)
	if rel == "-" {
		return abs
	}
	return fmt.Sprintf("%s (%s)", abs, rel)
}
