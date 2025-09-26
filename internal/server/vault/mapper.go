package vault

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
)

func toPB(it *storage.Item) *pb.Item {
	var alias string
	if it.Alias != nil {
		alias = *it.Alias
	}
	hid := int64(0)
	if it.HumanID != nil {
		hid = *it.HumanID
	}
	return &pb.Item{
		Id:            it.ID.String(),
		HumanId:       hid,
		Alias:         alias,
		Type:          pb.ItemType(it.Type),
		Payload:       it.Payload,
		Meta:          mapToMeta(it.MetaJSON),
		Version:       it.Version,
		UpdatedAtUnix: it.UpdatedAt.Unix(),
		Deleted:       it.DeletedAt != nil,
	}
}

// mapToMeta decodes JSON {"k":"v",...} into protobuf entries.
func mapToMeta(b []byte) []*pb.ItemMetaEntry {
	if len(b) == 0 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	out := make([]*pb.ItemMetaEntry, 0, len(m))
	for k, v := range m {
		out = append(out, &pb.ItemMetaEntry{Key: k, Value: v})
	}
	return out
}

// metaToJSON encodes protobuf entries into a compact JSON map.
func metaToJSON(entries []*pb.ItemMetaEntry) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(entries))
	for _, kv := range entries {
		k := strings.TrimSpace(kv.GetKey())
		if k == "" {
			return nil, errors.New("empty meta key")
		}
		m[k] = kv.GetValue()
	}
	return json.Marshal(m)
}

// --- cursor helpers: stable, opaque base64(JSON{u,id}) ---

type listCursor struct {
	U int64  `json:"u"` // updated_at as unix seconds
	I string `json:"i"` // uuid string
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	raw, _ := json.Marshal(listCursor{U: t.Unix(), I: id.String()})
	return base64RawURLEncodingEncodeToString(raw)
}

func decodeCursor(s string) (time.Time, uuid.UUID, error) {
	data, err := base64RawURLEncodingDecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	var c listCursor
	if err := json.Unmarshal(data, &c); err != nil {
		return time.Time{}, uuid.Nil, err
	}
	t := time.Unix(c.U, 0)
	uid, err := uuid.Parse(c.I)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return t, uid, nil
}

func base64RawURLEncodingEncodeToString(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func base64RawURLEncodingDecodeString(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
