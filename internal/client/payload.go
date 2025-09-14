package client

import (
	"encoding/json"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url,omitempty"`
	Note     string `json:"note,omitempty"`
}

func EncodeLogin(p LoginPayload) ([]byte, error) {
	if strings.TrimSpace(p.Username) == "" {
		return nil, errors.New("username is required")
	}
	if strings.TrimSpace(p.Password) == "" {
		return nil, errors.New("password is required")
	}
	return json.Marshal(p)
}

func DecodeLogin(b []byte) (LoginPayload, error) {
	var p LoginPayload
	if len(b) == 0 {
		return p, errors.New("empty login payload")
	}
	err := json.Unmarshal(b, &p)
	return p, err
}

type CardPayload struct {
	Holder   string `json:"holder"`         // Cardholder name
	PAN      string `json:"pan"`            // Digits only
	ExpMonth int    `json:"exp_month"`      // 1..12
	ExpYear  int    `json:"exp_year"`       // full year, e.g., 2027
	CVV      string `json:"cvv"`            // 3-4 digits
	Note     string `json:"note,omitempty"` // optional
}

// EncodeCard validates & normalizes fields and returns JSON.
func EncodeCard(p CardPayload) ([]byte, error) {
	p.PAN = sanitizeDigits(p.PAN)
	p.CVV = sanitizeDigits(p.CVV)

	if strings.TrimSpace(p.Holder) == "" {
		return nil, errors.New("card holder is required")
	}
	if err := validatePAN(p.PAN); err != nil {
		return nil, err
	}
	if p.ExpMonth < 1 || p.ExpMonth > 12 {
		return nil, errors.New("exp month must be 1..12")
	}
	if p.ExpYear < 2000 || p.ExpYear > 2100 {
		return nil, errors.New("exp year looks invalid")
	}
	if p.CVV == "" || len(p.CVV) < 3 || len(p.CVV) > 4 || !allDigits(p.CVV) {
		return nil, errors.New("cvv must be 3-4 digits")
	}
	now := time.Now()
	if p.ExpYear < now.Year() || (p.ExpYear == now.Year() && p.ExpMonth < int(now.Month())) {
	}

	return json.Marshal(p)
}

func DecodeCard(b []byte) (CardPayload, error) {
	var p CardPayload
	if len(b) == 0 {
		return p, errors.New("empty card payload")
	}
	err := json.Unmarshal(b, &p)
	return p, err
}

// MaskPassword returns "******" preserving nothing.
func MaskPassword(_ string) string { return "******" }

// MaskCVV returns "***" or "****".
func MaskCVV(cvv string) string {
	if len(cvv) >= 4 {
		return "****"
	}
	return "***"
}

// MaskPAN keeps last 4 digits, groups by 4: "•••• •••• •••• 1234".
func MaskPAN(pan string) string {
	d := sanitizeDigits(pan)
	if len(d) <= 4 {
		return d
	}
	last4 := d[len(d)-4:]
	hidden := strings.Repeat("•", len(d)-4)
	return groupBy4(hidden+last4, ' ')
}

func groupBy4(s string, sep rune) string {
	// Group from left to right every 4 runes.
	var b strings.Builder
	for i, r := range s {
		if i > 0 && i%4 == 0 {
			b.WriteRune(sep)
		}
		b.WriteRune(r)
	}
	return b.String()
}

func sanitizeDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func allDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return s != ""
}

// Luhn check for PAN (very common for payment cards).
func validatePAN(pan string) error {
	if !allDigits(pan) {
		return errors.New("pan must contain digits only")
	}
	if n := len(pan); n < 12 || n > 19 {
		return errors.New("pan length must be 12..19 digits")
	}
	if !luhnOK(pan) {
		return errors.New("pan failed Luhn check")
	}
	return nil
}

func luhnOK(s string) bool {
	sum := 0
	double := false
	// from right to left
	for i := len(s) - 1; i >= 0; i-- {
		d := int(s[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// GuessMIME tries to guess mime by file extension; returns empty string if unknown.
func GuessMIME(path string) string {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(path)))
	if ext == "" {
		return ""
	}
	mt := mime.TypeByExtension(ext)
	// Strip params like "; charset=utf-8"
	if i := strings.IndexByte(mt, ';'); i >= 0 {
		mt = mt[:i]
	}
	return strings.TrimSpace(mt)
}

// HumanSize returns a friendly representation like "1.2 MB".
func HumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	value := float64(n) / float64(div)
	suffix := "KMGTPE"[exp : exp+1]
	return strings.TrimSuffix(strconv.FormatFloat(value, 'f', 1, 64), ".0") + " " + suffix + "B"
}

// LoadBinary reads file content and suggests meta (filename/mime).
type BinaryLoad struct {
	Data     []byte
	FileName string
	MIME     string
}

func LoadBinary(path string) (BinaryLoad, error) {
	var out BinaryLoad
	b, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	out.Data = b
	out.FileName = filepath.Base(path)
	out.MIME = GuessMIME(path)
	return out, nil
}
