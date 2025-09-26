package payload

import (
	"testing"
)

func TestValidateText(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		expectError bool
		errMsg      string
	}{
		{
			name:        "valid text",
			payload:     []byte("This is valid text content"),
			expectError: false,
		},
		{
			name:        "valid text with newlines",
			payload:     []byte("Line 1\nLine 2\nLine 3"),
			expectError: false,
		},
		{
			name:        "valid text with tabs",
			payload:     []byte("Column1\tColumn2\tColumn3"),
			expectError: false,
		},
		{
			name:        "valid text with special characters",
			payload:     []byte("Text with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"),
			expectError: false,
		},
		{
			name:        "valid text with unicode",
			payload:     []byte("Текст на русском языке"),
			expectError: false,
		},
		{
			name:        "valid text with emoji",
			payload:     []byte("Text with emoji 🚀🎉✨"),
			expectError: false,
		},
		{
			name:        "valid text at size limit",
			payload:     make([]byte, 1024*1024), // 1MB
			expectError: false,
		},

		// Error cases
		{
			name:        "empty payload",
			payload:     []byte(""),
			expectError: true,
			errMsg:      "text content is required",
		},
		{
			name:        "nil payload",
			payload:     nil,
			expectError: true,
			errMsg:      "text content is required",
		},
		{
			name:        "whitespace only",
			payload:     []byte("   "),
			expectError: true,
			errMsg:      "text content is required",
		},
		{
			name:        "newlines only",
			payload:     []byte("\n\n\n"),
			expectError: true,
			errMsg:      "text content is required",
		},
		{
			name:        "tabs only",
			payload:     []byte("\t\t\t"),
			expectError: true,
			errMsg:      "text content is required",
		},
		{
			name:        "mixed whitespace only",
			payload:     []byte(" \n\t \n\t "),
			expectError: true,
			errMsg:      "text content is required",
		},
		{
			name:        "payload too large",
			payload:     make([]byte, 1024*1024+1), // 1MB + 1 byte
			expectError: true,
			errMsg:      "text content too large",
		},
		{
			name:        "very large payload",
			payload:     make([]byte, 10*1024*1024), // 10MB
			expectError: true,
			errMsg:      "text content too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateText(tt.payload)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateText() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateText() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateText() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateTextEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		expectError bool
	}{
		{
			name:        "single character",
			payload:     []byte("a"),
			expectError: false,
		},
		{
			name:        "single newline",
			payload:     []byte("\n"),
			expectError: true,
		},
		{
			name:        "single space",
			payload:     []byte(" "),
			expectError: true,
		},
		{
			name:        "single tab",
			payload:     []byte("\t"),
			expectError: true,
		},
		{
			name:        "text with leading whitespace",
			payload:     []byte("   Hello World"),
			expectError: false,
		},
		{
			name:        "text with trailing whitespace",
			payload:     []byte("Hello World   "),
			expectError: false,
		},
		{
			name:        "text with both leading and trailing whitespace",
			payload:     []byte("   Hello World   "),
			expectError: false,
		},
		{
			name:        "text with internal whitespace",
			payload:     []byte("Hello   World"),
			expectError: false,
		},
		{
			name:        "text with mixed whitespace",
			payload:     []byte("  Hello\n\tWorld  "),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateText(tt.payload)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateText() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateText() unexpected error = %v", err)
				}
			}
		})
	}
}
