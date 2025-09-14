package handlers

import (
	"context"
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockItemsRepository is a mock implementation of storage.ItemsRepository interface
type mockItemsRepository struct {
	items       map[uuid.UUID]*storage.Item
	humanIDs    map[uuid.UUID]int64
	aliases     map[uuid.UUID]string
	nextHumanID int64
}

func newMockItemsRepository() *mockItemsRepository {
	return &mockItemsRepository{
		items:       make(map[uuid.UUID]*storage.Item),
		humanIDs:    make(map[uuid.UUID]int64),
		aliases:     make(map[uuid.UUID]string),
		nextHumanID: 1,
	}
}

func (m *mockItemsRepository) Create(ctx context.Context, owner uuid.UUID, typ int16, payload, metaJSON []byte, alias *string) (uuid.UUID, int64, error) {
	id := uuid.New()
	humanID := m.nextHumanID
	m.nextHumanID++

	item := &storage.Item{
		ID:        id,
		OwnerID:   owner,
		Type:      typ,
		Payload:   payload,
		MetaJSON:  metaJSON,
		Version:   1,
		HumanID:   &humanID,
		Alias:     alias,
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	m.items[id] = item
	m.humanIDs[owner] = humanID
	if alias != nil {
		m.aliases[owner] = *alias
	}

	return id, 1, nil
}

func (m *mockItemsRepository) Get(ctx context.Context, owner uuid.UUID, id uuid.UUID) (*storage.Item, error) {
	item, exists := m.items[id]
	if !exists {
		return nil, &mockError{code: "not found"}
	}
	if item.OwnerID != owner {
		return nil, &mockError{code: "not found"}
	}
	if item.DeletedAt != nil {
		return nil, &mockError{code: "not found"}
	}
	return item, nil
}

func (m *mockItemsRepository) GetByHuman(ctx context.Context, owner uuid.UUID, hid int64) (*storage.Item, error) {
	for _, item := range m.items {
		if item.OwnerID == owner && item.HumanID != nil && *item.HumanID == hid && item.DeletedAt == nil {
			return item, nil
		}
	}
	return nil, &mockError{code: "not found"}
}

func (m *mockItemsRepository) GetByAlias(ctx context.Context, owner uuid.UUID, alias string) (*storage.Item, error) {
	for _, item := range m.items {
		if item.OwnerID == owner && item.Alias != nil && *item.Alias == alias && item.DeletedAt == nil {
			return item, nil
		}
	}
	return nil, &mockError{code: "not found"}
}

func (m *mockItemsRepository) List(ctx context.Context, owner uuid.UUID, limit int32) ([]storage.Item, error) {
	var items []storage.Item
	for _, item := range m.items {
		if item.OwnerID == owner && item.DeletedAt == nil {
			items = append(items, *item)
			if int32(len(items)) >= limit {
				break
			}
		}
	}
	return items, nil
}

func (m *mockItemsRepository) ListChanges(ctx context.Context, owner uuid.UUID, after time.Time, afterID uuid.UUID, limit int32) ([]storage.Item, error) {
	var items []storage.Item
	for _, item := range m.items {
		if item.OwnerID == owner && item.UpdatedAt.After(after) {
			items = append(items, *item)
			if int32(len(items)) >= limit {
				break
			}
		}
	}
	return items, nil
}

func (m *mockItemsRepository) Update(ctx context.Context, owner uuid.UUID, id uuid.UUID, payload, metaJSON []byte, expectedVersion int64) (*storage.Item, error) {
	item, exists := m.items[id]
	if !exists {
		return nil, &mockError{code: "not found"}
	}
	if item.OwnerID != owner {
		return nil, &mockError{code: "not found"}
	}
	if item.Version != expectedVersion {
		return nil, &mockError{code: "version conflict"}
	}
	if item.DeletedAt != nil {
		return nil, &mockError{code: "not found"}
	}

	// Update the item
	item.Payload = payload
	item.MetaJSON = metaJSON
	item.Version++
	item.UpdatedAt = time.Now()

	return item, nil
}

func (m *mockItemsRepository) Delete(ctx context.Context, owner uuid.UUID, id uuid.UUID) error {
	item, exists := m.items[id]
	if !exists {
		return &mockError{code: "not found"}
	}
	if item.OwnerID != owner {
		return &mockError{code: "not found"}
	}
	if item.DeletedAt != nil {
		return &mockError{code: "not found"}
	}

	now := time.Now()
	item.DeletedAt = &now
	item.UpdatedAt = now
	return nil
}

// mockError simulates database error
type mockError struct {
	code string
}

func (e *mockError) Error() string {
	return "mock error: " + e.code
}

// TestVaultServer tests the VaultServer implementation
func TestVaultServer(t *testing.T) {
	// Create mock dependencies
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	// Create test context with user ID
	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	t.Run("CreateItem valid", func(t *testing.T) {
		req := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
			Alias: "test-alias",
		}

		resp, err := server.CreateItem(ctx, req)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}
		if resp == nil {
			t.Errorf("CreateItem() returned nil response")
		}
		if resp.Id == "" {
			t.Errorf("CreateItem() returned empty ID")
		}
	})

	t.Run("CreateItem without context", func(t *testing.T) {
		req := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
		}

		_, err := server.CreateItem(context.Background(), req)
		if err == nil {
			t.Errorf("CreateItem() without context should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("CreateItem() error should be gRPC status error")
		}
		if statusErr.Code() != codes.Unauthenticated {
			t.Errorf("CreateItem() error code = %v, want %v", statusErr.Code(), codes.Unauthenticated)
		}
	})

	t.Run("CreateItem invalid type", func(t *testing.T) {
		req := &pb.CreateItemRequest{
			Type:    pb.ItemType(999), // Invalid type
			Payload: []byte("test payload"),
		}

		_, err := server.CreateItem(ctx, req)
		if err == nil {
			t.Errorf("CreateItem() with invalid type should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("CreateItem() error should be gRPC status error")
		}
		if statusErr.Code() != codes.InvalidArgument {
			t.Errorf("CreateItem() error code = %v, want %v", statusErr.Code(), codes.InvalidArgument)
		}
	})

	t.Run("GetItemByRef by ID", func(t *testing.T) {
		// Create item first
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		// Get item by ID
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_Id{Id: createResp.Id},
		}

		resp, err := server.GetItemByRef(ctx, ref)
		if err != nil {
			t.Errorf("GetItemByRef() error = %v", err)
		}
		if resp == nil {
			t.Errorf("GetItemByRef() returned nil response")
		}
		if resp.Id != createResp.Id {
			t.Errorf("GetItemByRef() ID = %v, want %v", resp.Id, createResp.Id)
		}
	})

	t.Run("GetItemByRef by human ID", func(t *testing.T) {
		// Create item first
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
		}
		_, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		// Get item by human ID
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_HumanId{HumanId: 1},
		}

		resp, err := server.GetItemByRef(ctx, ref)
		if err != nil {
			t.Errorf("GetItemByRef() error = %v", err)
		}
		if resp == nil {
			t.Errorf("GetItemByRef() returned nil response")
		}
		if resp.HumanId != 1 {
			t.Errorf("GetItemByRef() HumanId = %v, want 1", resp.HumanId)
		}
	})

	t.Run("GetItemByRef by alias", func(t *testing.T) {
		// Create item first
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Alias:   "test-alias",
		}
		_, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		// Get item by alias
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_Alias{Alias: "test-alias"},
		}

		resp, err := server.GetItemByRef(ctx, ref)
		if err != nil {
			t.Errorf("GetItemByRef() error = %v", err)
		}
		if resp == nil {
			t.Errorf("GetItemByRef() returned nil response")
		}
		if resp.Alias != "test-alias" {
			t.Errorf("GetItemByRef() Alias = %v, want test-alias", resp.Alias)
		}
	})

	t.Run("ListItems", func(t *testing.T) {
		// Create multiple items
		for i := 0; i < 3; i++ {
			req := &pb.CreateItemRequest{
				Type:    pb.ItemType_TEXT,
				Payload: []byte("test payload"),
			}
			_, err := server.CreateItem(ctx, req)
			if err != nil {
				t.Fatalf("CreateItem() error = %v", err)
			}
		}

		// List items
		req := &pb.ListRequest{
			Limit: 10,
		}

		resp, err := server.ListItems(ctx, req)
		if err != nil {
			t.Errorf("ListItems() error = %v", err)
		}
		if resp == nil {
			t.Errorf("ListItems() returned nil response")
		}
		if len(resp.Items) < 3 {
			t.Errorf("ListItems() returned %d items, want at least 3", len(resp.Items))
		}
	})

	t.Run("UpdateItem valid", func(t *testing.T) {
		// Create item first
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("original payload"),
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		// Update item
		updateReq := &pb.UpdateItemRequest{
			Id:              createResp.Id,
			ExpectedVersion: 1,
			Payload:         []byte("updated payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Updated Item"},
			},
		}

		resp, err := server.UpdateItem(ctx, updateReq)
		if err != nil {
			t.Errorf("UpdateItem() error = %v", err)
		}
		if resp == nil {
			t.Errorf("UpdateItem() returned nil response")
		}
		if resp.Version != 2 {
			t.Errorf("UpdateItem() version = %v, want 2", resp.Version)
		}
	})

	t.Run("UpdateItem wrong version", func(t *testing.T) {
		// Create item first
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("original payload"),
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		// Update item with wrong version
		updateReq := &pb.UpdateItemRequest{
			Id:              createResp.Id,
			ExpectedVersion: 999,
			Payload:         []byte("updated payload"),
		}

		_, err = server.UpdateItem(ctx, updateReq)
		if err == nil {
			t.Errorf("UpdateItem() with wrong version should return error")
		}
	})

	t.Run("DeleteItem valid", func(t *testing.T) {
		// Create item first
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}

		// Delete item
		deleteReq := &pb.ItemID{Id: createResp.Id}

		resp, err := server.DeleteItem(ctx, deleteReq)
		if err != nil {
			t.Errorf("DeleteItem() error = %v", err)
		}
		if resp == nil {
			t.Errorf("DeleteItem() returned nil response")
		}

		// Try to get deleted item
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_Id{Id: createResp.Id},
		}
		_, err = server.GetItemByRef(ctx, ref)
		if err == nil {
			t.Errorf("GetItemByRef() deleted item should return error")
		}
	})

	t.Run("DeleteItem invalid ID", func(t *testing.T) {
		deleteReq := &pb.ItemID{Id: "invalid-uuid"}

		_, err := server.DeleteItem(ctx, deleteReq)
		if err == nil {
			t.Errorf("DeleteItem() with invalid ID should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("DeleteItem() error should be gRPC status error")
		}
		if statusErr.Code() != codes.InvalidArgument {
			t.Errorf("DeleteItem() error code = %v, want %v", statusErr.Code(), codes.InvalidArgument)
		}
	})
}
