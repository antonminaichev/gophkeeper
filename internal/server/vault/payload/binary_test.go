package payload

import (
	"testing"

	"github.com/antonminaichev/gophkeeper/internal/domain"
)

func TestValidateBinary(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		meta        domain.Meta
		expectError bool
		errMsg      string
	}{
		{
			name:    "valid binary with filename",
			payload: []byte("binary content"),
			meta: domain.Meta{
				"filename": "test.txt",
				"title":    "Test File",
			},
			expectError: false,
		},
		{
			name:    "valid binary with large content",
			payload: make([]byte, 5*1024*1024), // 5MB
			meta: domain.Meta{
				"filename": "large.bin",
			},
			expectError: false,
		},
		{
			name:    "valid binary at size limit",
			payload: make([]byte, 10*1024*1024), // 10MB
			meta: domain.Meta{
				"filename": "max.bin",
			},
			expectError: false,
		},

		// Error cases
		{
			name:    "empty payload",
			payload: []byte{},
			meta: domain.Meta{
				"filename": "test.txt",
			},
			expectError: true,
			errMsg:      "binary content is required",
		},
		{
			name:    "nil payload",
			payload: nil,
			meta: domain.Meta{
				"filename": "test.txt",
			},
			expectError: true,
			errMsg:      "binary content is required",
		},
		{
			name:    "payload too large",
			payload: make([]byte, 11*1024*1024), // 11MB
			meta: domain.Meta{
				"filename": "too-large.bin",
			},
			expectError: true,
			errMsg:      "binary content too large",
		},
		{
			name:    "missing filename",
			payload: []byte("binary content"),
			meta: domain.Meta{
				"title": "Test File",
			},
			expectError: true,
			errMsg:      "filename is required for binary items",
		},
		{
			name:    "empty filename",
			payload: []byte("binary content"),
			meta: domain.Meta{
				"filename": "",
				"title":    "Test File",
			},
			expectError: true,
			errMsg:      "filename is required for binary items",
		},
		{
			name:    "filename not in meta",
			payload: []byte("binary content"),
			meta: domain.Meta{
				"title": "Test File",
			},
			expectError: true,
			errMsg:      "filename is required for binary items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBinary(tt.payload, tt.meta)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateBinary() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateBinary() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateBinary() unexpected error = %v", err)
				}
			}
		})
	}
}
