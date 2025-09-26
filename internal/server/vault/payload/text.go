package payload

import (
	"errors"
	"strings"
)

func ValidateText(b []byte) error {
	text := strings.TrimSpace(string(b))
	if text == "" {
		return errors.New("text content is required")
	}
	if len(text) > 1024*1024 { // 1MB limit
		return errors.New("text content too large")
	}
	return nil
}
