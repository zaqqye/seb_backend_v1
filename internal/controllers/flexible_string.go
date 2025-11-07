package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// FlexibleString allows JSON fields to be provided as string or number.
type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	if fs == nil {
		return fmt.Errorf("FlexibleString: nil receiver")
	}

	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	var asString string
	if err := json.Unmarshal(trimmed, &asString); err == nil {
		*fs = FlexibleString(strings.TrimSpace(asString))
		return nil
	}

	var asNumber json.Number
	if err := json.Unmarshal(trimmed, &asNumber); err == nil {
		*fs = FlexibleString(asNumber.String())
		return nil
	}

	return fmt.Errorf("FlexibleString: expected string or number, got %s", string(data))
}

func (fs FlexibleString) String() string {
	return string(fs)
}
