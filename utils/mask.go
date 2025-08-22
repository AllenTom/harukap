package util

import "fmt"

// MaskKeepHeadTail returns a masked version of s keeping the first keepHead
// and last keepTail characters visible. If s is too short, it returns s as-is.
func MaskKeepHeadTail(s string, keepHead int, keepTail int) string {
	if s == "" {
		return s
	}
	if keepHead < 0 {
		keepHead = 0
	}
	if keepTail < 0 {
		keepTail = 0
	}
	if len(s) <= keepHead+keepTail {
		return s
	}
	head := s[:keepHead]
	tail := s[len(s)-keepTail:]
	return fmt.Sprintf("%s***%s", head, tail)
}
