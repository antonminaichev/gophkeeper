package pbconv

import pb "github.com/antonminaichev/gophkeeper/api/proto"

// MetaSliceToMap: []*ItemMetaEntry -> map[string]string.
func MetaSliceToMap(meta []*pb.ItemMetaEntry) map[string]string {
	if len(meta) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(meta))
	for _, kv := range meta {
		if kv == nil {
			continue
		}
		out[kv.GetKey()] = kv.GetValue()
	}
	return out
}

// MapToMetaSlice: map[string]string -> []*ItemMetaEntry.
func MapToMetaSlice(m map[string]string) []*pb.ItemMetaEntry {
	if len(m) == 0 {
		return nil
	}
	out := make([]*pb.ItemMetaEntry, 0, len(m))
	for k, v := range m {
		out = append(out, &pb.ItemMetaEntry{Key: k, Value: v})
	}
	return out
}

// Title returns meta["title"].
func Title(meta []*pb.ItemMetaEntry) string {
	for _, kv := range meta {
		if kv.GetKey() == "title" {
			return kv.GetValue()
		}
	}
	return ""
}
