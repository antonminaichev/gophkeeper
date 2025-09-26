package payload

import (
	"errors"

	"github.com/antonminaichev/gophkeeper/internal/domain"
)

func ValidateBinary(b []byte, meta domain.Meta) error {
	if len(b) == 0 {
		return errors.New("binary content is required")
	}

	// Check size limit (10MB)
	if len(b) > 10*1024*1024 {
		return errors.New("binary content too large")
	}

	// Check if filename is provided in meta
	if filename, ok := meta["filename"]; !ok || filename == "" {
		return errors.New("filename is required for binary items")
	}

	return nil
}
