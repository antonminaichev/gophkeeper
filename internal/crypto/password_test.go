package crypto

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "testpassword123",
			wantErr:  false,
		},
		{
			name:     "short password",
			password: "short",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "exactly 8 characters",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "7 characters",
			password: "1234567",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			salt, hash, err := HashPassword(DefaultArgon, []byte(tt.password))

			if tt.wantErr {
				if err == nil {
					t.Errorf("HashPassword() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("HashPassword() unexpected error = %v", err)
				return
			}

			if len(salt) == 0 {
				t.Errorf("HashPassword() salt is empty")
			}

			if len(hash) == 0 {
				t.Errorf("HashPassword() hash is empty")
			}

			if len(salt) != int(DefaultArgon.SaltLen) {
				t.Errorf("HashPassword() salt length = %d, want %d", len(salt), DefaultArgon.SaltLen)
			}

			if len(hash) != int(DefaultArgon.KeyLen) {
				t.Errorf("HashPassword() hash length = %d, want %d", len(hash), DefaultArgon.KeyLen)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := []byte("testpassword123")
	salt, hash, err := HashPassword(DefaultArgon, password)
	if err != nil {
		t.Fatalf("HashPassword() failed: %v", err)
	}

	tests := []struct {
		name     string
		password []byte
		salt     []byte
		hash     []byte
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			salt:     salt,
			hash:     hash,
			want:     true,
		},
		{
			name:     "wrong password",
			password: []byte("wrongpassword"),
			salt:     salt,
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: []byte(""),
			salt:     salt,
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty salt",
			password: password,
			salt:     []byte(""),
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty hash",
			password: password,
			salt:     salt,
			hash:     []byte(""),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyPassword(DefaultArgon, tt.password, tt.salt, tt.hash)
			if got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSalt(t *testing.T) {
	tests := []struct {
		name string
		n    uint32
	}{
		{
			name: "default length",
			n:    0,
		},
		{
			name: "custom length",
			n:    32,
		},
		{
			name: "small length",
			n:    8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedLen := tt.n
			if expectedLen == 0 {
				expectedLen = 16 // default
			}

			salt, err := NewSalt(tt.n)
			if err != nil {
				t.Errorf("NewSalt() error = %v", err)
				return
			}

			if len(salt) != int(expectedLen) {
				t.Errorf("NewSalt() length = %d, want %d", len(salt), expectedLen)
			}

			// Check that salt is not all zeros
			allZero := true
			for _, b := range salt {
				if b != 0 {
					allZero = false
					break
				}
			}
			if allZero {
				t.Errorf("NewSalt() returned all zeros")
			}
		})
	}
}

func TestNeedsRehash(t *testing.T) {
	tests := []struct {
		name          string
		params        ArgonParams
		currentKeyLen uint32
		want          bool
	}{
		{
			name: "same key length",
			params: ArgonParams{
				KeyLen: 32,
			},
			currentKeyLen: 32,
			want:          false,
		},
		{
			name: "different key length",
			params: ArgonParams{
				KeyLen: 64,
			},
			currentKeyLen: 32,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NeedsRehash(tt.params, tt.currentKeyLen)
			if got != tt.want {
				t.Errorf("NeedsRehash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultArgonParams(t *testing.T) {
	if DefaultArgon.Time == 0 {
		t.Errorf("DefaultArgon.Time should not be 0")
	}
	if DefaultArgon.MemoryMB == 0 {
		t.Errorf("DefaultArgon.MemoryMB should not be 0")
	}
	if DefaultArgon.Parallel == 0 {
		t.Errorf("DefaultArgon.Parallel should not be 0")
	}
	if DefaultArgon.KeyLen == 0 {
		t.Errorf("DefaultArgon.KeyLen should not be 0")
	}
	if DefaultArgon.SaltLen == 0 {
		t.Errorf("DefaultArgon.SaltLen should not be 0")
	}
}

// Benchmark tests
func BenchmarkHashPassword(b *testing.B) {
	password := []byte("testpassword123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := HashPassword(DefaultArgon, password)
		if err != nil {
			b.Fatalf("HashPassword() failed: %v", err)
		}
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := []byte("testpassword123")
	salt, hash, err := HashPassword(DefaultArgon, password)
	if err != nil {
		b.Fatalf("HashPassword() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword(DefaultArgon, password, salt, hash)
	}
}

