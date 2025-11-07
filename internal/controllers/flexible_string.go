package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// FlexibleString allows JSON fields to be provided as string or number
type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	if fs == nil {
		return fmt.Errorf("FlexibleString: nil receiver")
	}
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	var s string
	if err := json.Unmarshal(trimmed, &s); err == nil {
		*fs = FlexibleString(strings.TrimSpace(s))
		return nil
	}

	var num json.Number
	if err := json.Unmarshal(trimmed, &num); err == nil {
		*fs = FlexibleString(num.String())
		return nil
	}

	return fmt.Errorf("FlexibleString: expected string or number, got %s", string(data))
}

func (fs FlexibleString) String() string {
	return string(fs)
}
