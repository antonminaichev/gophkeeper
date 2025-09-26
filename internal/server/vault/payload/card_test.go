package payload

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateCard(t *testing.T) {
	// Создаем валидную карту для тестов
	validCard := Card{
		Holder:   "John Doe",
		PAN:      "4111111111111111", // Valid Visa test number
		ExpMonth: 12,
		ExpYear:  time.Now().Year() + 1,
		CVV:      "123",
		Note:     "Test card",
	}

	tests := []struct {
		name        string
		card        Card
		expectError bool
		errMsg      string
	}{
		{
			name:        "valid card",
			card:        validCard,
			expectError: false,
		},
		{
			name: "valid card with 4-digit CVV",
			card: Card{
				Holder:   "Jane Doe",
				PAN:      "5555555555554444", // Valid Mastercard test number
				ExpMonth: 6,
				ExpYear:  time.Now().Year() + 2,
				CVV:      "1234",
			},
			expectError: false,
		},
		{
			name: "valid card with minimal data",
			card: Card{
				Holder:   "Test User",
				PAN:      "378282246310005", // Valid Amex test number
				ExpMonth: 1,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: false,
		},

		// Error cases
		{
			name: "empty holder",
			card: Card{
				Holder:   "",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "card holder is required",
		},
		{
			name: "whitespace holder",
			card: Card{
				Holder:   "   ",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "card holder is required",
		},
		{
			name: "invalid PAN - too short",
			card: Card{
				Holder:   "John Doe",
				PAN:      "123456789",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "pan length must be 12..19 digits",
		},
		{
			name: "invalid PAN - too long",
			card: Card{
				Holder:   "John Doe",
				PAN:      "12345678901234567890",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "pan length must be 12..19 digits",
		},
		{
			name: "invalid PAN - non-digits",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111-1111-1111-1111", // После sanitizeDigits становится валидным
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: false, // Функция sanitizeDigits удаляет дефисы
		},
		{
			name: "invalid PAN - failed Luhn check",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111112", // Invalid Luhn
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "pan failed Luhn check",
		},
		{
			name: "invalid exp month - too low",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 0,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "exp month must be 1..12",
		},
		{
			name: "invalid exp month - too high",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 13,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "exp month must be 1..12",
		},
		{
			name: "invalid exp year - too low",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  1999,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "exp year looks invalid",
		},
		{
			name: "invalid exp year - too high",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  2101,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "exp year looks invalid",
		},
		{
			name: "invalid CVV - empty",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "",
			},
			expectError: true,
			errMsg:      "cvv must be 3-4 digits",
		},
		{
			name: "invalid CVV - too short",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "12",
			},
			expectError: true,
			errMsg:      "cvv must be 3-4 digits",
		},
		{
			name: "invalid CVV - too long",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "12345",
			},
			expectError: true,
			errMsg:      "cvv must be 3-4 digits",
		},
		{
			name: "invalid CVV - non-digits",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() + 1,
				CVV:      "abc",
			},
			expectError: true,
			errMsg:      "cvv must be 3-4 digits",
		},
		{
			name: "expired card - past year",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  time.Now().Year() - 1,
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "card is expired",
		},
		{
			name: "expired card - past month",
			card: Card{
				Holder:   "John Doe",
				PAN:      "4111111111111111",
				ExpMonth: int(time.Now().Month()) - 1,
				ExpYear:  time.Now().Year(),
				CVV:      "123",
			},
			expectError: true,
			errMsg:      "card is expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cardJSON, err := json.Marshal(tt.card)
			if err != nil {
				t.Fatalf("Failed to marshal card: %v", err)
			}

			err = ValidateCard(cardJSON)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateCard() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateCard() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCard() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateCardWithInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name:        "invalid JSON",
			jsonData:    `{"holder": "John Doe", "pan": "4111111111111111"`, // missing closing brace
			expectError: true,
		},
		{
			name:        "empty JSON",
			jsonData:    `{}`,
			expectError: true,
		},
		{
			name:        "non-object JSON",
			jsonData:    `"just a string"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCard([]byte(tt.jsonData))
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateCard() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCard() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestLuhnOK(t *testing.T) {
	tests := []struct {
		name     string
		pan      string
		expected bool
	}{
		{
			name:     "valid Visa",
			pan:      "4111111111111111",
			expected: true,
		},
		{
			name:     "valid Mastercard",
			pan:      "5555555555554444",
			expected: true,
		},
		{
			name:     "valid Amex",
			pan:      "378282246310005",
			expected: true,
		},
		{
			name:     "invalid Luhn",
			pan:      "4111111111111112",
			expected: false,
		},
		{
			name:     "empty string",
			pan:      "",
			expected: true, // 0 % 10 == 0
		},
		{
			name:     "single digit",
			pan:      "1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := luhnOK(tt.pan)
			if result != tt.expected {
				t.Errorf("luhnOK(%s) = %v, want %v", tt.pan, result, tt.expected)
			}
		})
	}
}

func TestSanitizeDigits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "digits only",
			input:    "1234567890",
			expected: "1234567890",
		},
		{
			name:     "with spaces",
			input:    "1234 5678 9012 3456",
			expected: "1234567890123456",
		},
		{
			name:     "with hyphens",
			input:    "1234-5678-9012-3456",
			expected: "1234567890123456",
		},
		{
			name:     "with letters",
			input:    "1234abc5678def9012",
			expected: "123456789012",
		},
		{
			name:     "mixed characters",
			input:    "1234-5678 9012.3456",
			expected: "1234567890123456",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no digits",
			input:    "abcdef",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeDigits(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeDigits(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAllDigits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "all digits",
			input:    "1234567890",
			expected: true,
		},
		{
			name:     "with letters",
			input:    "1234abc5678",
			expected: false,
		},
		{
			name:     "with spaces",
			input:    "1234 5678",
			expected: false,
		},
		{
			name:     "with hyphens",
			input:    "1234-5678",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "single digit",
			input:    "1",
			expected: true,
		},
		{
			name:     "zero",
			input:    "0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allDigits(tt.input)
			if result != tt.expected {
				t.Errorf("allDigits(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
