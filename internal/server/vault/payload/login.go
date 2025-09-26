package payload

import (
	"encoding/json"
	"errors"

	"github.com/antonminaichev/gophkeeper/internal/domain"
)

type Login struct {
	URL      string `json:"url,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Note     string `json:"note,omitempty"`
}

func ValidateLogin(b []byte) error {
	var l Login
	if err := json.Unmarshal(b, &l); err != nil {
		return err
	}
	if l.Username == "" || l.Password == "" {
		return errors.New("username/password required")
	}
	return nil
}

func Validate(t domain.ItemType, payload []byte, meta domain.Meta) error {
	switch t {
	case domain.ItemLogin:
		return ValidateLogin(payload)
	case domain.ItemCard:
		return ValidateCard(payload)
	case domain.ItemText:
		return ValidateText(payload)
	case domain.ItemBinary:
		return ValidateBinary(payload, meta)
	default:
		return nil
	}
}
