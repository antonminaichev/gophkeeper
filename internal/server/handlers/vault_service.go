package handlers

// Vault gRPC service: CRUD for user items, with sync cursor support.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	gauth "github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type VaultServer struct {
	pb.UnimplementedVaultServiceServer
	items storage.ItemsRepository
}

func NewVaultServer(items storage.ItemsRepository) *VaultServer {
	return &VaultServer{items: items}
}

// CreateItem creates a new item owned by the authenticated user.
func (s *VaultServer) CreateItem(ctx context.Context, r *pb.CreateItemRequest) (*pb.ItemID, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	t := r.GetType()
	if t == 0 {
		t = pb.ItemType_LOGIN
	}
	switch t {
	case pb.ItemType_LOGIN, pb.ItemType_TEXT, pb.ItemType_BINARY, pb.ItemType_CARD:
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown item type")
	}

	metaJSON, err := metaToJSON(r.GetMeta())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meta")
	}
	var alias *string
	if a := strings.TrimSpace(r.GetAlias()); a != "" {
		alias = &a
	}

	id, _, err := s.items.Create(ctx, owner, int16(t), r.GetPayload(), metaJSON, alias)
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot create item")
	}
	return &pb.ItemID{Id: id.String()}, nil
}

// GetItemByRef fetches an item by UUID / human id / alias.
func (s *VaultServer) GetItemByRef(ctx context.Context, ref *pb.ItemRef) (*pb.Item, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	var it *storage.Item
	switch x := ref.GetRef().(type) {
	case *pb.ItemRef_Id:
		idStr := strings.TrimSpace(x.Id)
		itemID, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "bad id")
		}
		it, err = s.items.Get(ctx, owner, itemID)
	case *pb.ItemRef_HumanId:
		it, err = s.items.GetByHuman(ctx, owner, x.HumanId)
	case *pb.ItemRef_Alias:
		a := strings.TrimSpace(x.Alias)
		if a == "" {
			return nil, status.Error(codes.InvalidArgument, "empty alias")
		}
		it, err = s.items.GetByAlias(ctx, owner, a)
	default:
		return nil, status.Error(codes.InvalidArgument, "empty ref")
	}
	if err != nil {
		return nil, status.Error(codes.NotFound, "item not found")
	}
	return toPB(it), nil
}

func (s *VaultServer) ListItems(ctx context.Context, r *pb.ListRequest) (*pb.ListResponse, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	limit := r.GetLimit()
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	cursorStr := strings.TrimSpace(r.GetCursor())
	// Regular list (no sync cursor): show live, non-deleted items.
	if cursorStr == "" {
		items, err := s.items.List(ctx, owner, limit)
		if err != nil {
			return nil, status.Error(codes.Internal, "cannot list items")
		}
		out := make([]*pb.Item, 0, len(items))
		for i := range items {
			p := toPB(&items[i])
			// Keep list responses light: payload isn't needed here.
			p.Payload = nil
			out = append(out, p)
		}
		return &pb.ListResponse{Items: out}, nil
	}

	// Change feed mode: decode cursor and return changes after it.
	afterT, afterID, err := decodeCursor(cursorStr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad cursor")
	}

	changes, err := s.items.ListChanges(ctx, owner, afterT, afterID, limit)
	if err != nil {
		return nil, status.Error(codes.Internal, "cannot list changes")
	}

	resp := &pb.ListResponse{Items: make([]*pb.Item, 0, len(changes))}
	for i := range changes {
		p := toPB(&changes[i])
		// For sync we usually don't need payload; clients fetch by id when needed.
		p.Payload = nil
		resp.Items = append(resp.Items, p)
	}

	// Compute next cursor from the last row (if any).
	if n := len(changes); n > 0 {
		last := changes[n-1]
		resp.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return resp, nil
}

// UpdateItem applies optimistic update by version.
func (s *VaultServer) UpdateItem(ctx context.Context, r *pb.UpdateItemRequest) (*pb.Item, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(strings.TrimSpace(r.GetId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad id")
	}
	if r.GetExpectedVersion() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_version is required")
	}

	metaJSON, err := metaToJSON(r.GetMeta())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid meta")
	}

	it, err := s.items.Update(ctx, owner, id, r.GetPayload(), metaJSON, r.GetExpectedVersion())
	if err != nil {
		return nil, err
	}
	return toPB(it), nil
}

// DeleteItem marks the item as deleted and bumps updated_at (tombstone entry).
func (s *VaultServer) DeleteItem(ctx context.Context, r *pb.ItemID) (*emptypb.Empty, error) {
	owner, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(strings.TrimSpace(r.GetId()))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "bad id")
	}
	if err := s.items.Delete(ctx, owner, id); err != nil {
		return nil, status.Error(codes.Internal, "cannot delete item")
	}
	return &emptypb.Empty{}, nil
}

// userIDFromCtx extracts and validates owner id from context.
func userIDFromCtx(ctx context.Context) (uuid.UUID, error) {
	uid, ok := gauth.UserIDFrom(ctx)
	if !ok || strings.TrimSpace(uid) == "" {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing user")
	}
	id, err := uuid.Parse(uid)
	if err != nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "bad user id")
	}
	return id, nil
}

// toPB converts storage.Item to protobuf.
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

// Cursor helpers: stable, opaque base64(JSON{u,id}).
type listCursor struct {
	U int64  `json:"u"` // updated_at as unix seconds
	I string `json:"i"` // uuid string
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	raw, _ := json.Marshal(listCursor{U: t.Unix(), I: id.String()})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(s string) (time.Time, uuid.UUID, error) {
	data, err := base64.RawURLEncoding.DecodeString(s)
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
