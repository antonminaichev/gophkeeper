package handlers

import (
	"context"
	"testing"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestVaultServer_GetItemByRef(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	t.Run("GetItemByRef with UUID", func(t *testing.T) {
		// First create an item
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}

		// Then get it by UUID
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_Id{Id: createResp.Id},
		}
		item, err := server.GetItemByRef(ctx, ref)
		if err != nil {
			t.Errorf("GetItemByRef() error = %v", err)
		}
		if item == nil {
			t.Errorf("GetItemByRef() returned nil item")
		}
		if item.Id != createResp.Id {
			t.Errorf("GetItemByRef() returned wrong item ID")
		}
	})

	t.Run("GetItemByRef with human ID", func(t *testing.T) {
		// First create an item
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
		}
		_, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}

		// Then get it by human ID
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_HumanId{HumanId: 1},
		}
		item, err := server.GetItemByRef(ctx, ref)
		if err != nil {
			t.Errorf("GetItemByRef() error = %v", err)
		}
		if item == nil {
			t.Errorf("GetItemByRef() returned nil item")
		}
		// Note: The mock repository might return different IDs, so we just check that we got an item
		if item.Id == "" {
			t.Errorf("GetItemByRef() returned empty item ID")
		}
	})

	t.Run("GetItemByRef with alias", func(t *testing.T) {
		// First create an item with alias
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
			Alias: "test-alias",
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}

		// Then get it by alias
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_Alias{Alias: "test-alias"},
		}
		item, err := server.GetItemByRef(ctx, ref)
		if err != nil {
			t.Errorf("GetItemByRef() error = %v", err)
		}
		if item == nil {
			t.Errorf("GetItemByRef() returned nil item")
		}
		if item.Id != createResp.Id {
			t.Errorf("GetItemByRef() returned wrong item ID")
		}
	})

	t.Run("GetItemByRef without context", func(t *testing.T) {
		ref := &pb.ItemRef{
			Ref: &pb.ItemRef_Id{Id: "test-uuid"},
		}

		_, err := server.GetItemByRef(context.Background(), ref)
		if err == nil {
			t.Errorf("GetItemByRef() without context should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("GetItemByRef() error should be gRPC status error")
		}
		if statusErr.Code() != codes.Unauthenticated {
			t.Errorf("GetItemByRef() error code = %v, want %v", statusErr.Code(), codes.Unauthenticated)
		}
	})
}

func TestVaultServer_ListItems(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	t.Run("ListItems snapshot", func(t *testing.T) {
		// First create some items
		for i := 0; i < 3; i++ {
			createReq := &pb.CreateItemRequest{
				Type:    pb.ItemType_TEXT,
				Payload: []byte("test payload"),
				Meta: []*pb.ItemMetaEntry{
					{Key: "title", Value: "Test Item"},
				},
			}
			_, err := server.CreateItem(ctx, createReq)
			if err != nil {
				t.Errorf("CreateItem() error = %v", err)
			}
		}

		// Then list them
		listReq := &pb.ListRequest{
			Limit: 10,
		}
		resp, err := server.ListItems(ctx, listReq)
		if err != nil {
			t.Errorf("ListItems() error = %v", err)
		}
		if resp == nil {
			t.Errorf("ListItems() returned nil response")
		}
		if len(resp.Items) != 3 {
			t.Errorf("ListItems() returned %d items, want 3", len(resp.Items))
		}
	})

	t.Run("ListItems with cursor", func(t *testing.T) {
		// First create some items
		for i := 0; i < 3; i++ {
			createReq := &pb.CreateItemRequest{
				Type:    pb.ItemType_TEXT,
				Payload: []byte("test payload"),
				Meta: []*pb.ItemMetaEntry{
					{Key: "title", Value: "Test Item"},
				},
			}
			_, err := server.CreateItem(ctx, createReq)
			if err != nil {
				t.Errorf("CreateItem() error = %v", err)
			}
		}

		// Then list them with cursor (use empty cursor for change feed)
		listReq := &pb.ListRequest{
			Limit:  10,
			Cursor: "",
		}
		resp, err := server.ListItems(ctx, listReq)
		if err != nil {
			t.Errorf("ListItems() error = %v", err)
		}
		if resp == nil {
			t.Errorf("ListItems() returned nil response")
		}
	})

	t.Run("ListItems without context", func(t *testing.T) {
		listReq := &pb.ListRequest{
			Limit: 10,
		}

		_, err := server.ListItems(context.Background(), listReq)
		if err == nil {
			t.Errorf("ListItems() without context should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("ListItems() error should be gRPC status error")
		}
		if statusErr.Code() != codes.Unauthenticated {
			t.Errorf("ListItems() error code = %v, want %v", statusErr.Code(), codes.Unauthenticated)
		}
	})
}

func TestVaultServer_UpdateItem(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	t.Run("UpdateItem valid", func(t *testing.T) {
		// First create an item
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}

		// Then update it
		updateReq := &pb.UpdateItemRequest{
			Id:      createResp.Id,
			Payload: []byte("updated payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Updated Item"},
			},
			ExpectedVersion: 1,
		}
		item, err := server.UpdateItem(ctx, updateReq)
		if err != nil {
			t.Errorf("UpdateItem() error = %v", err)
		}
		if item == nil {
			t.Errorf("UpdateItem() returned nil item")
		}
		if item.Id != createResp.Id {
			t.Errorf("UpdateItem() returned wrong item ID")
		}
	})

	t.Run("UpdateItem without context", func(t *testing.T) {
		updateReq := &pb.UpdateItemRequest{
			Id:              "test-uuid",
			Payload:         []byte("updated payload"),
			ExpectedVersion: 1,
		}

		_, err := server.UpdateItem(context.Background(), updateReq)
		if err == nil {
			t.Errorf("UpdateItem() without context should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("UpdateItem() error should be gRPC status error")
		}
		if statusErr.Code() != codes.Unauthenticated {
			t.Errorf("UpdateItem() error code = %v, want %v", statusErr.Code(), codes.Unauthenticated)
		}
	})
}

func TestVaultServer_DeleteItem(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	t.Run("DeleteItem valid", func(t *testing.T) {
		// First create an item
		createReq := &pb.CreateItemRequest{
			Type:    pb.ItemType_TEXT,
			Payload: []byte("test payload"),
			Meta: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
		}
		createResp, err := server.CreateItem(ctx, createReq)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}

		// Then delete it
		deleteReq := &pb.ItemID{
			Id: createResp.Id,
		}
		resp, err := server.DeleteItem(ctx, deleteReq)
		if err != nil {
			t.Errorf("DeleteItem() error = %v", err)
		}
		if resp == nil {
			t.Errorf("DeleteItem() returned nil response")
		}
	})

	t.Run("DeleteItem with invalid UUID", func(t *testing.T) {
		deleteReq := &pb.ItemID{
			Id: "invalid-uuid",
		}

		_, err := server.DeleteItem(ctx, deleteReq)
		if err == nil {
			t.Errorf("DeleteItem() with invalid UUID should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("DeleteItem() error should be gRPC status error")
		}
		if statusErr.Code() != codes.InvalidArgument {
			t.Errorf("DeleteItem() error code = %v, want %v", statusErr.Code(), codes.InvalidArgument)
		}
	})

	t.Run("DeleteItem without context", func(t *testing.T) {
		deleteReq := &pb.ItemID{
			Id: "test-uuid",
		}

		_, err := server.DeleteItem(context.Background(), deleteReq)
		if err == nil {
			t.Errorf("DeleteItem() without context should return error")
		}

		statusErr, ok := status.FromError(err)
		if !ok {
			t.Errorf("DeleteItem() error should be gRPC status error")
		}
		if statusErr.Code() != codes.Unauthenticated {
			t.Errorf("DeleteItem() error code = %v, want %v", statusErr.Code(), codes.Unauthenticated)
		}
	})
}

func TestNewVaultServer(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	if server == nil {
		t.Errorf("NewVaultServer() returned nil")
	}
	if server.uc == nil {
		t.Errorf("NewVaultServer() usecase not set")
	}
}

func TestNewVaultServerWithComposition(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServerWithComposition(
		itemsRepo,
		itemsRepo,
		itemsRepo,
		itemsRepo,
		itemsRepo,
	)

	if server == nil {
		t.Errorf("NewVaultServerWithComposition() returned nil")
	}
	if server.uc == nil {
		t.Errorf("NewVaultServerWithComposition() usecase not set")
	}
}

func TestUserIDFromCtx(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		expectError bool
	}{
		{
			name:        "valid user ID",
			ctx:         auth.WithUser(context.Background(), "550e8400-e29b-41d4-a716-446655440000", "test@example.com"),
			expectError: false,
		},
		{
			name:        "no user ID",
			ctx:         context.Background(),
			expectError: true,
		},
		{
			name:        "empty user ID",
			ctx:         auth.WithUser(context.Background(), "", "test@example.com"),
			expectError: true,
		},
		{
			name:        "whitespace user ID",
			ctx:         auth.WithUser(context.Background(), "   ", "test@example.com"),
			expectError: true,
		},
		{
			name:        "invalid UUID",
			ctx:         auth.WithUser(context.Background(), "invalid-uuid", "test@example.com"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := userIDFromCtx(tt.ctx)

			if tt.expectError {
				if err == nil {
					t.Errorf("userIDFromCtx() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("userIDFromCtx() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestVaultServer_CreateItemExtended(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	tests := []struct {
		name         string
		itemType     pb.ItemType
		payload      []byte
		alias        string
		expectError  bool
		expectedCode codes.Code
	}{
		{
			name:        "create text item",
			itemType:    pb.ItemType_TEXT,
			payload:     []byte("test text content"),
			alias:       "text-alias",
			expectError: false,
		},
		{
			name:        "create login item",
			itemType:    pb.ItemType_LOGIN,
			payload:     []byte(`{"username":"user","password":"pass"}`),
			alias:       "login-alias",
			expectError: false,
		},
		{
			name:        "create card item",
			itemType:    pb.ItemType_CARD,
			payload:     []byte(`{"pan":"4111111111111111","holder":"John Doe"}`),
			alias:       "card-alias",
			expectError: false,
		},
		{
			name:        "create binary item",
			itemType:    pb.ItemType_BINARY,
			payload:     []byte("binary data"),
			alias:       "binary-alias",
			expectError: false,
		},
		{
			name:         "create with invalid type",
			itemType:     pb.ItemType(999),
			payload:      []byte("test content"),
			alias:        "invalid-alias",
			expectError:  true,
			expectedCode: codes.InvalidArgument,
		},
		{
			name:        "create with empty payload",
			itemType:    pb.ItemType_TEXT,
			payload:     []byte(""),
			alias:       "empty-alias",
			expectError: false,
		},
		{
			name:        "create with empty alias",
			itemType:    pb.ItemType_TEXT,
			payload:     []byte("test content"),
			alias:       "",
			expectError: false,
		},
		{
			name:        "create with very long payload",
			itemType:    pb.ItemType_TEXT,
			payload:     make([]byte, 10000),
			alias:       "large-alias",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.CreateItemRequest{
				Type:    tt.itemType,
				Payload: tt.payload,
				Meta: []*pb.ItemMetaEntry{
					{Key: "title", Value: "Test Item"},
				},
				Alias: tt.alias,
			}

			resp, err := server.CreateItem(ctx, req)

			if tt.expectError {
				if err == nil {
					t.Errorf("CreateItem() expected error, got nil")
				}
				if tt.expectedCode != 0 {
					statusErr, ok := status.FromError(err)
					if !ok {
						t.Errorf("CreateItem() error should be gRPC status error")
					} else if statusErr.Code() != tt.expectedCode {
						t.Errorf("CreateItem() error code = %v, want %v", statusErr.Code(), tt.expectedCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("CreateItem() unexpected error = %v", err)
				}
				if resp == nil {
					t.Errorf("CreateItem() returned nil response")
				} else if resp.Id == "" {
					t.Errorf("CreateItem() returned empty ID")
				}
			}
		})
	}
}

func TestVaultServer_GetItemByRefExtended(t *testing.T) {
	itemsRepo := newMockItemsRepository()
	server := NewVaultServer(itemsRepo)

	userID := uuid.New()
	ctx := auth.WithUser(context.Background(), userID.String(), "test@example.com")

	// Create test items
	createReq := &pb.CreateItemRequest{
		Type:    pb.ItemType_TEXT,
		Payload: []byte("test payload"),
		Meta: []*pb.ItemMetaEntry{
			{Key: "title", Value: "Test Item"},
		},
		Alias: "test-alias",
	}
	createResp, err := server.CreateItem(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateItem() error = %v", err)
	}

	tests := []struct {
		name        string
		ref         *pb.ItemRef
		expectError bool
	}{
		{
			name: "get by valid UUID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Id{Id: createResp.Id},
			},
			expectError: false,
		},
		{
			name: "get by invalid UUID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Id{Id: "invalid-uuid"},
			},
			expectError: true,
		},
		{
			name: "get by non-existent UUID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Id{Id: "550e8400-e29b-41d4-a716-446655440000"},
			},
			expectError: true,
		},
		{
			name: "get by alias",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Alias{Alias: "test-alias"},
			},
			expectError: false,
		},
		{
			name: "get by non-existent alias",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Alias{Alias: "non-existent-alias"},
			},
			expectError: true,
		},
		{
			name: "get by human ID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_HumanId{HumanId: 1},
			},
			expectError: false,
		},
		{
			name: "get by non-existent human ID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_HumanId{HumanId: 999},
			},
			expectError: true,
		},
		{
			name: "get by zero human ID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_HumanId{HumanId: 0},
			},
			expectError: true,
		},
		{
			name: "get by negative human ID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_HumanId{HumanId: -1},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := server.GetItemByRef(ctx, tt.ref)

			if tt.expectError {
				if err == nil {
					t.Errorf("GetItemByRef() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetItemByRef() unexpected error = %v", err)
				}
				if item == nil {
					t.Errorf("GetItemByRef() returned nil item")
				}
			}
		})
	}
}
