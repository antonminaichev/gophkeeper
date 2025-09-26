package validate

import (
	"errors"
	"regexp"
	"strings"

	"github.com/antonminaichev/gophkeeper/internal/domain"
)

var aliasRe = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]{1,64}$`)

func ItemForCreate(it domain.Item) error {
	if it.Type == "" {
		return errors.New("item type is required")
	}
	if strings.TrimSpace(it.Alias) != "" && !aliasRe.MatchString(it.Alias) {
		return errors.New("invalid alias")
	}
	// Здесь можно добавить лимиты по размеру payload/meta
	return nil
}
