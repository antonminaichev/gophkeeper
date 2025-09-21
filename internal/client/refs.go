package client

import (
	"strconv"
	"strings"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

// LooksLikeUUID checks UUID.
func LooksLikeUUID(s string) bool {
	s = strings.TrimSpace(s)
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// buildItemRef строит универсальный ItemRef из селектора.
func (cli *CLI) buildItemRef(selector string) *pb.ItemRef {
	selector = strings.TrimSpace(selector)

	// Если есть локальный кэш — пробуем резолвнуть селектор в UUID.
	if cache, err := LoadCache(); err == nil && cache != nil {
		if id, ok := cache.LookupID(selector); ok && LooksLikeUUID(id) {
			return &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: id}}
		}
	}

	switch {
	case LooksLikeUUID(selector):
		return &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: selector}}
	case strings.HasPrefix(selector, "@"):
		return &pb.ItemRef{Ref: &pb.ItemRef_Alias{Alias: strings.TrimPrefix(selector, "@")}}
	default:
		if n, err := strconv.ParseInt(selector, 10, 64); err == nil && n > 0 {
			return &pb.ItemRef{Ref: &pb.ItemRef_HumanId{HumanId: n}}
		}
	}
	return &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: selector}}
}
