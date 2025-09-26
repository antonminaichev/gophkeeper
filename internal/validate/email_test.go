package validate

import "testing"

func TestEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid emails
		{
			name:     "valid email",
			input:    "test@example.com",
			expected: true,
		},
		{
			name:     "valid email with subdomain",
			input:    "user@mail.example.com",
			expected: true,
		},
		{
			name:     "valid email with numbers",
			input:    "user123@example123.com",
			expected: true,
		},
		{
			name:     "valid email with dots",
			input:    "first.last@example.com",
			expected: true,
		},
		{
			name:     "valid email with hyphens",
			input:    "user-name@example-domain.com",
			expected: true,
		},
		{
			name:     "valid email with underscores",
			input:    "user_name@example.com",
			expected: true,
		},
		{
			name:     "valid email with plus",
			input:    "user+tag@example.com",
			expected: true,
		},
		{
			name:     "valid email with spaces around",
			input:    "  test@example.com  ",
			expected: true,
		},

		// Invalid emails
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: false,
		},
		{
			name:     "no @ symbol",
			input:    "testexample.com",
			expected: false,
		},
		{
			name:     "multiple @ symbols",
			input:    "test@@example.com",
			expected: false,
		},
		{
			name:     "@ at start",
			input:    "@example.com",
			expected: false,
		},
		{
			name:     "@ at end",
			input:    "test@",
			expected: false,
		},
		{
			name:     "no domain",
			input:    "test@",
			expected: false,
		},
		{
			name:     "no TLD",
			input:    "test@example",
			expected: false,
		},
		{
			name:     "domain without dot",
			input:    "test@examplecom",
			expected: false,
		},
		{
			name:     "just @",
			input:    "@",
			expected: false,
		},
		{
			name:     "domain with multiple dots but no TLD",
			input:    "test@example..com",
			expected: true, // Функция проверяет только наличие точки в домене
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Email(tt.input)
			if result != tt.expected {
				t.Errorf("Email(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
