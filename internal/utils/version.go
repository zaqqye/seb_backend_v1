package utils

import (
	"strconv"
	"strings"
)

// CompareVersions compares two semantic-ish version strings (e.g. "1.2.3").
// Returns 1 if current > target, 0 if equal, -1 if current < target.
func CompareVersions(current, target string) int {
	curParts := splitVersion(current)
	tgtParts := splitVersion(target)
	maxLen := len(curParts)
	if len(tgtParts) > maxLen {
		maxLen = len(tgtParts)
	}
	for i := 0; i < maxLen; i++ {
		var curVal, tgtVal int
		if i < len(curParts) {
			curVal = curParts[i]
		}
		if i < len(tgtParts) {
			tgtVal = tgtParts[i]
		}
		if curVal > tgtVal {
			return 1
		}
		if curVal < tgtVal {
			return -1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	v = strings.TrimSpace(v)
	if v == "" {
		return []int{0}
	}
	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			result = append(result, 0)
			continue
		}
		if val, err := strconv.Atoi(p); err == nil {
			result = append(result, val)
		} else {
			// fallback: ignore non-numeric characters
			clean := trimNonDigits(p)
			if clean == "" {
				result = append(result, 0)
				continue
			}
			if val, err := strconv.Atoi(clean); err == nil {
				result = append(result, val)
			} else {
				result = append(result, 0)
			}
		}
	}
	return result
}

func trimNonDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return b.String()
}
