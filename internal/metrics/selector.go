package metrics

import (
	"strconv"
	"strings"
)

type Selector struct {
	Name   string
	Labels map[string]string
}

func compileSelectors(mapping map[string]string) map[string]Selector {
	out := make(map[string]Selector, len(mapping))
	for key, raw := range mapping {
		out[key] = ParseSelector(raw)
	}
	return out
}

func ParseSelector(raw string) Selector {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Selector{}
	}
	name := trimmed
	labels := map[string]string{}
	open := strings.Index(trimmed, "{")
	if open == -1 {
		return Selector{Name: name}
	}
	close := strings.LastIndex(trimmed, "}")
	if close == -1 || close < open {
		close = len(trimmed)
	}
	name = strings.TrimSpace(trimmed[:open])
	content := strings.TrimSpace(trimmed[open+1 : close])
	if content == "" {
		return Selector{Name: name}
	}
	for _, part := range splitLabelPairs(content) {
		key, val, ok := parseLabelPair(part)
		if !ok {
			continue
		}
		labels[key] = val
	}
	return Selector{Name: name, Labels: labels}
}

func splitLabelPairs(raw string) []string {
	var parts []string
	var buf strings.Builder
	inQuotes := false
	escape := false
	for _, r := range raw {
		switch {
		case escape:
			buf.WriteRune(r)
			escape = false
		case r == '\\':
			buf.WriteRune(r)
			escape = true
		case r == '"':
			buf.WriteRune(r)
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			parts = append(parts, buf.String())
			buf.Reset()
		default:
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		parts = append(parts, buf.String())
	}
	return parts
}

func parseLabelPair(raw string) (string, string, bool) {
	part := strings.TrimSpace(raw)
	if part == "" {
		return "", "", false
	}
	segments := strings.SplitN(part, "=", 2)
	if len(segments) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(segments[0])
	value := strings.TrimSpace(segments[1])
	if key == "" || value == "" {
		return "", "", false
	}
	if strings.HasPrefix(value, "\"") {
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		} else {
			value = strings.Trim(value, "\"")
		}
	}
	return key, value, true
}
