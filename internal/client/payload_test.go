package client

import (
	"testing"
)

func TestEncodeLogin(t *testing.T) {
	tests := []struct {
		name    string
		payload LoginPayload
		wantErr bool
	}{
		{
			name: "valid login payload",
			payload: LoginPayload{
				Username: "testuser",
				Password: "password123",
				URL:      "https://example.com",
				Note:     "Test note",
			},
			wantErr: false,
		},
		{
			name: "minimal login payload",
			payload: LoginPayload{
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "empty username",
			payload: LoginPayload{
				Username: "",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "empty password",
			payload: LoginPayload{
				Username: "testuser",
				Password: "",
			},
			wantErr: true,
		},
		{
			name: "whitespace username",
			payload: LoginPayload{
				Username: "   ",
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "whitespace password",
			payload: LoginPayload{
				Username: "testuser",
				Password: "   ",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := EncodeLogin(tt.payload)

			if tt.wantErr {
				if err == nil {
					t.Errorf("EncodeLogin() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("EncodeLogin() unexpected error = %v", err)
				return
			}

			if len(data) == 0 {
				t.Errorf("EncodeLogin() returned empty data")
			}
		})
	}
}

func TestDecodeLogin(t *testing.T) {
	originalPayload := LoginPayload{
		Username: "testuser",
		Password: "password123",
		URL:      "https://example.com",
		Note:     "Test note",
	}

	encoded, err := EncodeLogin(originalPayload)
	if err != nil {
		t.Fatalf("EncodeLogin() failed: %v", err)
	}

	decoded, err := DecodeLogin(encoded)
	if err != nil {
		t.Fatalf("DecodeLogin() failed: %v", err)
	}

	if decoded.Username != originalPayload.Username {
		t.Errorf("DecodeLogin() username = %v, want %v", decoded.Username, originalPayload.Username)
	}
	if decoded.Password != originalPayload.Password {
		t.Errorf("DecodeLogin() password = %v, want %v", decoded.Password, originalPayload.Password)
	}
	if decoded.URL != originalPayload.URL {
		t.Errorf("DecodeLogin() URL = %v, want %v", decoded.URL, originalPayload.URL)
	}
	if decoded.Note != originalPayload.Note {
		t.Errorf("DecodeLogin() note = %v, want %v", decoded.Note, originalPayload.Note)
	}
}

func TestDecodeLoginErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte(""),
		},
		{
			name: "invalid JSON",
			data: []byte("{invalid json}"),
		},
		{
			name: "null data",
			data: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeLogin(tt.data)
			if err == nil {
				t.Errorf("DecodeLogin() expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestEncodeCard(t *testing.T) {
	tests := []struct {
		name    string
		payload CardPayload
		wantErr bool
	}{
		{
			name: "valid card payload",
			payload: CardPayload{
				Holder:   "JOHN DOE",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  2025,
				CVV:      "123",
				Note:     "Test card",
			},
			wantErr: false,
		},
		{
			name: "minimal card payload",
			payload: CardPayload{
				Holder:   "JANE DOE",
				PAN:      "5555555555554444",
				ExpMonth: 1,
				ExpYear:  2026,
				CVV:      "1234",
			},
			wantErr: false,
		},
		{
			name: "empty holder",
			payload: CardPayload{
				Holder:   "",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  2025,
				CVV:      "123",
			},
			wantErr: true,
		},
		{
			name: "invalid PAN",
			payload: CardPayload{
				Holder:   "JOHN DOE",
				PAN:      "1234567890123456", // invalid Luhn
				ExpMonth: 12,
				ExpYear:  2025,
				CVV:      "123",
			},
			wantErr: true,
		},
		{
			name: "invalid expiration month",
			payload: CardPayload{
				Holder:   "JOHN DOE",
				PAN:      "4111111111111111",
				ExpMonth: 13,
				ExpYear:  2025,
				CVV:      "123",
			},
			wantErr: true,
		},
		{
			name: "invalid expiration year",
			payload: CardPayload{
				Holder:   "JOHN DOE",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  1999,
				CVV:      "123",
			},
			wantErr: true,
		},
		{
			name: "invalid CVV",
			payload: CardPayload{
				Holder:   "JOHN DOE",
				PAN:      "4111111111111111",
				ExpMonth: 12,
				ExpYear:  2025,
				CVV:      "12", // too short
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := EncodeCard(tt.payload)

			if tt.wantErr {
				if err == nil {
					t.Errorf("EncodeCard() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("EncodeCard() unexpected error = %v", err)
				return
			}

			if len(data) == 0 {
				t.Errorf("EncodeCard() returned empty data")
			}
		})
	}
}

func TestDecodeCard(t *testing.T) {
	originalPayload := CardPayload{
		Holder:   "JOHN DOE",
		PAN:      "4111111111111111",
		ExpMonth: 12,
		ExpYear:  2025,
		CVV:      "123",
		Note:     "Test card",
	}

	encoded, err := EncodeCard(originalPayload)
	if err != nil {
		t.Fatalf("EncodeCard() failed: %v", err)
	}

	decoded, err := DecodeCard(encoded)
	if err != nil {
		t.Fatalf("DecodeCard() failed: %v", err)
	}

	if decoded.Holder != originalPayload.Holder {
		t.Errorf("DecodeCard() holder = %v, want %v", decoded.Holder, originalPayload.Holder)
	}
	if decoded.PAN != originalPayload.PAN {
		t.Errorf("DecodeCard() PAN = %v, want %v", decoded.PAN, originalPayload.PAN)
	}
	if decoded.ExpMonth != originalPayload.ExpMonth {
		t.Errorf("DecodeCard() ExpMonth = %v, want %v", decoded.ExpMonth, originalPayload.ExpMonth)
	}
	if decoded.ExpYear != originalPayload.ExpYear {
		t.Errorf("DecodeCard() ExpYear = %v, want %v", decoded.ExpYear, originalPayload.ExpYear)
	}
	if decoded.CVV != originalPayload.CVV {
		t.Errorf("DecodeCard() CVV = %v, want %v", decoded.CVV, originalPayload.CVV)
	}
	if decoded.Note != originalPayload.Note {
		t.Errorf("DecodeCard() note = %v, want %v", decoded.Note, originalPayload.Note)
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     string
	}{
		{
			name:     "normal password",
			password: "password123",
			want:     "******",
		},
		{
			name:     "empty password",
			password: "",
			want:     "******",
		},
		{
			name:     "long password",
			password: "verylongpassword123456789",
			want:     "******",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskPassword(tt.password)
			if got != tt.want {
				t.Errorf("MaskPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskCVV(t *testing.T) {
	tests := []struct {
		name string
		cvv  string
		want string
	}{
		{
			name: "3 digit CVV",
			cvv:  "123",
			want: "***",
		},
		{
			name: "4 digit CVV",
			cvv:  "1234",
			want: "****",
		},
		{
			name: "longer CVV",
			cvv:  "12345",
			want: "****",
		},
		{
			name: "empty CVV",
			cvv:  "",
			want: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskCVV(tt.cvv)
			if got != tt.want {
				t.Errorf("MaskCVV() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMaskPAN(t *testing.T) {
	tests := []struct {
		name string
		pan  string
		want string
	}{
		{
			name: "16 digit PAN",
			pan:  "4111111111111111",
			want: "•••• •••• •••• 1111",
		},
		{
			name: "PAN with spaces",
			pan:  "4111 1111 1111 1111",
			want: "•••• •••• •••• 1111",
		},
		{
			name: "short PAN",
			pan:  "1234",
			want: "1234",
		},
		{
			name: "empty PAN",
			pan:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskPAN(tt.pan)
			if got != tt.want {
				t.Errorf("MaskPAN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePAN(t *testing.T) {
	tests := []struct {
		name    string
		pan     string
		wantErr bool
	}{
		{
			name:    "valid Visa PAN",
			pan:     "4111111111111111",
			wantErr: false,
		},
		{
			name:    "valid Mastercard PAN",
			pan:     "5555555555554444",
			wantErr: false,
		},
		{
			name:    "valid Amex PAN",
			pan:     "378282246310005",
			wantErr: false,
		},
		{
			name:    "invalid Luhn",
			pan:     "4111111111111112",
			wantErr: true,
		},
		{
			name:    "too short",
			pan:     "12345678901",
			wantErr: true,
		},
		{
			name:    "too long",
			pan:     "12345678901234567890",
			wantErr: true,
		},
		{
			name:    "non-digits",
			pan:     "4111-1111-1111-1111",
			wantErr: true,
		},
		{
			name:    "empty PAN",
			pan:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePAN(tt.pan)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePAN() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validatePAN() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGuessMIME(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "text file",
			path: "/path/to/file.txt",
			want: "text/plain",
		},
		{
			name: "json file",
			path: "/path/to/config.json",
			want: "application/json",
		},
		{
			name: "image file",
			path: "/path/to/image.jpg",
			want: "image/jpeg",
		},
		{
			name: "no extension",
			path: "/path/to/file",
			want: "",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "unknown extension",
			path: "/path/to/file.unknown",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GuessMIME(tt.path)
			if got != tt.want {
				t.Errorf("GuessMIME() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{
			name: "bytes",
			n:    512,
			want: "512 B",
		},
		{
			name: "kilobytes",
			n:    1024,
			want: "1 KB",
		},
		{
			name: "megabytes",
			n:    1024 * 1024,
			want: "1 MB",
		},
		{
			name: "gigabytes",
			n:    1024 * 1024 * 1024,
			want: "1 GB",
		},
		{
			name: "fractional",
			n:    1536,
			want: "1.5 KB",
		},
		{
			name: "zero",
			n:    0,
			want: "0 B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HumanSize(tt.n)
			if got != tt.want {
				t.Errorf("HumanSize() = %v, want %v", got, tt.want)
			}
		})
	}
}


