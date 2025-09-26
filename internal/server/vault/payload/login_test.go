package payload

import (
	"encoding/json"
	"testing"

	"github.com/antonminaichev/gophkeeper/internal/domain"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name        string
		login       Login
		expectError bool
		errMsg      string
	}{
		{
			name: "valid login",
			login: Login{
				URL:      "https://example.com",
				Username: "testuser",
				Password: "testpass",
				Note:     "Test login",
			},
			expectError: false,
		},
		{
			name: "minimal valid login",
			login: Login{
				Username: "testuser",
				Password: "testpass",
			},
			expectError: false,
		},
		{
			name: "login with empty URL",
			login: Login{
				URL:      "",
				Username: "testuser",
				Password: "testpass",
			},
			expectError: false,
		},
		{
			name: "login with empty note",
			login: Login{
				Username: "testuser",
				Password: "testpass",
				Note:     "",
			},
			expectError: false,
		},

		// Error cases
		{
			name: "empty username",
			login: Login{
				Username: "",
				Password: "testpass",
			},
			expectError: true,
			errMsg:      "username/password required",
		},
		{
			name: "empty password",
			login: Login{
				Username: "testuser",
				Password: "",
			},
			expectError: true,
			errMsg:      "username/password required",
		},
		{
			name: "both username and password empty",
			login: Login{
				Username: "",
				Password: "",
			},
			expectError: true,
			errMsg:      "username/password required",
		},
		{
			name: "whitespace username",
			login: Login{
				Username: "   ",
				Password: "testpass",
			},
			expectError: false, // Функция не проверяет пробелы
		},
		{
			name: "whitespace password",
			login: Login{
				Username: "testuser",
				Password: "   ",
			},
			expectError: false, // Функция не проверяет пробелы
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loginJSON, err := json.Marshal(tt.login)
			if err != nil {
				t.Fatalf("Failed to marshal login: %v", err)
			}

			err = ValidateLogin(loginJSON)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateLogin() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateLogin() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateLogin() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateLoginWithInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name:        "invalid JSON",
			jsonData:    `{"username": "testuser", "password": "testpass"`, // missing closing brace
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
			err := ValidateLogin([]byte(tt.jsonData))
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateLogin() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateLogin() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		itemType    domain.ItemType
		payload     []byte
		meta        domain.Meta
		expectError bool
	}{
		{
			name:        "valid login",
			itemType:    domain.ItemLogin,
			payload:     []byte(`{"username": "testuser", "password": "testpass"}`),
			meta:        domain.Meta{},
			expectError: false,
		},
		{
			name:        "valid card",
			itemType:    domain.ItemCard,
			payload:     []byte(`{"holder": "John Doe", "pan": "4111111111111111", "exp_month": 12, "exp_year": 2025, "cvv": "123"}`),
			meta:        domain.Meta{},
			expectError: false,
		},
		{
			name:        "valid text",
			itemType:    domain.ItemText,
			payload:     []byte("Some text content"),
			meta:        domain.Meta{},
			expectError: false,
		},
		{
			name:     "valid binary",
			itemType: domain.ItemBinary,
			payload:  []byte("binary content"),
			meta: domain.Meta{
				"filename": "test.txt",
			},
			expectError: false,
		},
		{
			name:        "unknown type",
			itemType:    domain.ItemType(999),
			payload:     []byte("some content"),
			meta:        domain.Meta{},
			expectError: false, // unknown types are allowed
		},

		// Error cases
		{
			name:        "invalid login",
			itemType:    domain.ItemLogin,
			payload:     []byte(`{"username": "", "password": "testpass"}`),
			meta:        domain.Meta{},
			expectError: true,
		},
		{
			name:        "invalid card",
			itemType:    domain.ItemCard,
			payload:     []byte(`{"holder": "", "pan": "4111111111111111", "exp_month": 12, "exp_year": 2025, "cvv": "123"}`),
			meta:        domain.Meta{},
			expectError: true,
		},
		{
			name:        "invalid text",
			itemType:    domain.ItemText,
			payload:     []byte("   "), // whitespace only
			meta:        domain.Meta{},
			expectError: true,
		},
		{
			name:        "invalid binary - missing filename",
			itemType:    domain.ItemBinary,
			payload:     []byte("binary content"),
			meta:        domain.Meta{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.itemType, tt.payload, tt.meta)
			if tt.expectError {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}
