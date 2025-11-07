package controllers

import (
	"strings"

	"github.com/google/uuid"
)

// toUUIDSlice converts a slice of string UUIDs into []uuid.UUID.
func toUUIDSlice(ids []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ids))
	for _, raw := range ids {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		val, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, val)
	}
	return out, nil
}
