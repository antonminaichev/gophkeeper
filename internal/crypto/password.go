package crypto

// Password hashing helpers based on Argon2id.
// Comments are short on purpose.

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/argon2"
)

// ArgonParams describes tunable parameters for Argon2id.
type ArgonParams struct {
	Time     uint32 // iterations
	MemoryMB uint32 // memory in megabytes
	Parallel uint8  // number of lanes
	KeyLen   uint32 // output length in bytes
	SaltLen  uint32 // salt length in bytes
}

// Sensible defaults for a CLI/GRPC service on typical hardware.
var DefaultArgon = ArgonParams{
	Time:     1,
	MemoryMB: 64,
	Parallel: 1,
	KeyLen:   32,
	SaltLen:  16,
}

// ErrWeakPassword is returned by HashPassword when basic checks fail.
var ErrWeakPassword = errors.New("weak password")

// NewSalt returns random salt.
func NewSalt(n uint32) ([]byte, error) {
	if n == 0 {
		n = 16
	}
	s := make([]byte, n)
	_, err := rand.Read(s)
	return s, err
}

// HashPassword derives a hash using provided params and a fresh salt.
func HashPassword(p ArgonParams, password []byte) (salt []byte, hash []byte, err error) {
	if len(password) < 8 {
		return nil, nil, ErrWeakPassword
	}
	if p.SaltLen == 0 {
		p.SaltLen = 16
	}
	salt, err = NewSalt(p.SaltLen)
	if err != nil {
		return nil, nil, err
	}
	hash = argon2.IDKey(password, salt, p.Time, p.MemoryMB*1024, p.Parallel, p.KeyLen)
	return salt, hash, nil
}

// VerifyPassword re-derives the hash and compares in constant time.
func VerifyPassword(p ArgonParams, password, salt, expected []byte) bool {
	if len(salt) == 0 || len(expected) == 0 || len(password) == 0 {
		return false
	}
	h := argon2.IDKey(password, salt, p.Time, p.MemoryMB*1024, p.Parallel, p.KeyLen)
	if len(h) != len(expected) {
		return false
	}
	// Constant-time comparison
	var v byte
	for i := range h {
		v |= h[i] ^ expected[i]
	}
	return v == 0
}

// NeedsRehash lets you detect whether stored hashes should be upgraded
func NeedsRehash(p ArgonParams, currentKeyLen uint32) bool {
	return p.KeyLen != currentKeyLen
}
