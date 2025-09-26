package payload

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
)

type Card struct {
	Holder   string `json:"holder"`
	PAN      string `json:"pan"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
	CVV      string `json:"cvv"`
	Note     string `json:"note,omitempty"`
}

func ValidateCard(b []byte) error {
	var c Card
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}

	if strings.TrimSpace(c.Holder) == "" {
		return errors.New("card holder is required")
	}

	if err := validatePAN(c.PAN); err != nil {
		return err
	}

	if c.ExpMonth < 1 || c.ExpMonth > 12 {
		return errors.New("exp month must be 1..12")
	}

	if c.ExpYear < 2000 || c.ExpYear > 2100 {
		return errors.New("exp year looks invalid")
	}

	if c.CVV == "" || len(c.CVV) < 3 || len(c.CVV) > 4 || !allDigits(c.CVV) {
		return errors.New("cvv must be 3-4 digits")
	}

	now := time.Now()
	if c.ExpYear < now.Year() || (c.ExpYear == now.Year() && c.ExpMonth < int(now.Month())) {
		return errors.New("card is expired")
	}

	return nil
}

func validatePAN(pan string) error {
	pan = sanitizeDigits(pan)
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
