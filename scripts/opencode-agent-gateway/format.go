package main

import (
	"encoding/json"
	"fmt"
)

// validateFormat checks that the format field, if present, is valid JSON.
func validateFormat(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return fmt.Errorf("invalid format: %w", err)
	}
	return nil
}
