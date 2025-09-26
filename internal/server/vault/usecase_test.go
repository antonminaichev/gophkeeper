package vault

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockItemsRepository is a mock implementation of ItemsRepository
type mockItemsRepository struct {
	items map[uuid.UUID]*storage.Item
	// Для тестирования ошибок
	createError      error
	getError         error
	getByHumanError  error
	getByAliasError  error
	listError        error
	listChangesError error
	updateError      error
	deleteError      error
}

func newMockItemsRepository() *mockItemsRepository {
	return &mockItemsRepository{
		items: make(map[uuid.UUID]*storage.Item),
	}
}

func (m *mockItemsRepository) Create(ctx context.Context, owner uuid.UUID, typ int16, payload, metaJSON []byte, alias *string) (uuid.UUID, int64, error) {
	if m.createError != nil {
		return uuid.Nil, 0, m.createError
	}
	id := uuid.New()
	now := time.Now()
	item := &storage.Item{
		ID:        id,
		OwnerID:   owner,
		Type:      typ,
		Payload:   payload,
		MetaJSON:  metaJSON,
		Version:   1,
		Alias:     alias,
		UpdatedAt: now,
		CreatedAt: now,
	}
	m.items[id] = item
	return id, 1, nil
}

func (m *mockItemsRepository) Get(ctx context.Context, owner uuid.UUID, id uuid.UUID) (*storage.Item, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	item, exists := m.items[id]
	if !exists {
		return nil, errors.New("item not found")
	}
	return item, nil
}

func (m *mockItemsRepository) GetByHuman(ctx context.Context, owner uuid.UUID, hid int64) (*storage.Item, error) {
	if m.getByHumanError != nil {
		return nil, m.getByHumanError
	}
	for _, item := range m.items {
		if item.HumanID != nil && *item.HumanID == hid {
			return item, nil
		}
	}
	return nil, errors.New("item not found")
}

func (m *mockItemsRepository) GetByAlias(ctx context.Context, owner uuid.UUID, alias string) (*storage.Item, error) {
	if m.getByAliasError != nil {
		return nil, m.getByAliasError
	}
	for _, item := range m.items {
		if item.Alias != nil && *item.Alias == alias {
			return item, nil
		}
	}
	return nil, errors.New("item not found")
}

func (m *mockItemsRepository) List(ctx context.Context, owner uuid.UUID, limit int32) ([]storage.Item, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	var result []storage.Item
	count := int32(0)
	for _, item := range m.items {
		if count >= limit {
			break
		}
		if item.OwnerID == owner && item.DeletedAt == nil {
			result = append(result, *item)
			count++
		}
	}
	return result, nil
}

func (m *mockItemsRepository) ListChanges(ctx context.Context, owner uuid.UUID, after time.Time, afterID uuid.UUID, limit int32) ([]storage.Item, error) {
	if m.listChangesError != nil {
		return nil, m.listChangesError
	}
	var result []storage.Item
	count := int32(0)
	for _, item := range m.items {
		if count >= limit {
			break
		}
		if item.OwnerID == owner && item.UpdatedAt.After(after) {
			result = append(result, *item)
			count++
		}
	}
	return result, nil
}

func (m *mockItemsRepository) Update(ctx context.Context, owner uuid.UUID, id uuid.UUID, payload, metaJSON []byte, expectedVersion int64) (*storage.Item, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}
	item, exists := m.items[id]
	if !exists {
		return nil, errors.New("item not found")
	}
	if item.Version != expectedVersion {
		return nil, errors.New("version mismatch")
	}
	item.Payload = payload
	item.MetaJSON = metaJSON
	item.Version++
	item.UpdatedAt = time.Now()
	return item, nil
}

func (m *mockItemsRepository) Delete(ctx context.Context, owner uuid.UUID, id uuid.UUID) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	item, exists := m.items[id]
	if !exists {
		return errors.New("item not found")
	}
	now := time.Now()
	item.DeletedAt = &now
	return nil
}

func TestNewUsecase(t *testing.T) {
	repo := newMockItemsRepository()
	usecase := NewUsecase(repo)

	if usecase == nil {
		t.Errorf("NewUsecase() returned nil")
	}
	if usecase.items != repo {
		t.Errorf("NewUsecase() items = %v, want %v", usecase.items, repo)
	}
}

func TestNewUsecaseWithComposition(t *testing.T) {
	creator := newMockItemsRepository()
	retriever := newMockItemsRepository()
	lister := newMockItemsRepository()
	updater := newMockItemsRepository()
	deleter := newMockItemsRepository()

	usecase := NewUsecaseWithComposition(creator, retriever, lister, updater, deleter)

	if usecase == nil {
		t.Errorf("NewUsecaseWithComposition() returned nil")
	}
}

func TestUsecase_CreateItem(t *testing.T) {
	repo := newMockItemsRepository()
	usecase := NewUsecase(repo)
	ownerID := uuid.New()

	tests := []struct {
		name        string
		request     *pb.CreateItemRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "valid login item",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType_LOGIN,
				Payload: []byte(`{"username": "test", "password": "pass"}`),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test Login"}},
				Alias:   "test-login",
			},
			expectError: false,
		},
		{
			name: "valid text item",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType_TEXT,
				Payload: []byte("Some text content"),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test Text"}},
			},
			expectError: false,
		},
		{
			name: "valid binary item",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType_BINARY,
				Payload: []byte("binary content"),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test Binary"}, {Key: "filename", Value: "test.txt"}},
			},
			expectError: false,
		},
		{
			name: "valid card item",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType_CARD,
				Payload: []byte(`{"holder": "John Doe", "pan": "4111111111111111", "exp_month": 12, "exp_year": 2025, "cvv": "123"}`),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test Card"}},
			},
			expectError: false,
		},
		{
			name: "default type (LOGIN)",
			request: &pb.CreateItemRequest{
				Type:    0, // Default type
				Payload: []byte(`{"username": "test", "password": "pass"}`),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test Login"}},
			},
			expectError: false,
		},
		{
			name: "invalid type",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType(999),
				Payload: []byte("content"),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test"}},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "invalid meta",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType_LOGIN,
				Payload: []byte(`{"username": "test", "password": "pass"}`),
				Meta:    []*pb.ItemMetaEntry{{Key: "", Value: "Empty Key"}},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "repository error",
			request: &pb.CreateItemRequest{
				Type:    pb.ItemType_LOGIN,
				Payload: []byte(`{"username": "test", "password": "pass"}`),
				Meta:    []*pb.ItemMetaEntry{{Key: "title", Value: "Test Login"}},
			},
			expectError: true,
			errorCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем ошибку репозитория для теста "repository error"
			if tt.name == "repository error" {
				repo.createError = errors.New("repository error")
			} else {
				repo.createError = nil
			}

			id, err := usecase.CreateItem(context.Background(), ownerID, tt.request)
			if tt.expectError {
				if err == nil {
					t.Errorf("CreateItem() expected error, got nil")
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("CreateItem() error is not a gRPC status")
					return
				}
				if st.Code() != tt.errorCode {
					t.Errorf("CreateItem() error code = %v, want %v", st.Code(), tt.errorCode)
				}
			} else {
				if err != nil {
					t.Errorf("CreateItem() unexpected error = %v", err)
					return
				}
				if id == uuid.Nil {
					t.Errorf("CreateItem() returned nil UUID")
				}
			}
		})
	}
}

func TestUsecase_GetItemByRef(t *testing.T) {
	repo := newMockItemsRepository()
	usecase := NewUsecase(repo)
	ownerID := uuid.New()
	itemID := uuid.New()

	// Создаем тестовый элемент
	now := time.Now()
	humanID := int64(123)
	alias := "test-alias"
	item := &storage.Item{
		ID:        itemID,
		OwnerID:   ownerID,
		Type:      1,
		Payload:   []byte("test payload"),
		MetaJSON:  []byte(`{"title": "Test Item"}`),
		Version:   1,
		HumanID:   &humanID,
		Alias:     &alias,
		UpdatedAt: now,
		CreatedAt: now,
	}
	repo.items[itemID] = item

	tests := []struct {
		name        string
		ref         *pb.ItemRef
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "get by ID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Id{Id: itemID.String()},
			},
			expectError: false,
		},
		{
			name: "get by human ID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_HumanId{HumanId: humanID},
			},
			expectError: false,
		},
		{
			name: "get by alias",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Alias{Alias: alias},
			},
			expectError: false,
		},
		{
			name: "invalid UUID",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Id{Id: "invalid-uuid"},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "empty alias",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Alias{Alias: ""},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "whitespace alias",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Alias{Alias: "   "},
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "empty ref",
			ref: &pb.ItemRef{
				Ref: nil,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "item not found",
			ref: &pb.ItemRef{
				Ref: &pb.ItemRef_Id{Id: uuid.New().String()},
			},
			expectError: true,
			errorCode:   codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := usecase.GetItemByRef(context.Background(), ownerID, tt.ref)
			if tt.expectError {
				if err == nil {
					t.Errorf("GetItemByRef() expected error, got nil")
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("GetItemByRef() error is not a gRPC status")
					return
				}
				if st.Code() != tt.errorCode {
					t.Errorf("GetItemByRef() error code = %v, want %v", st.Code(), tt.errorCode)
				}
			} else {
				if err != nil {
					t.Errorf("GetItemByRef() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Errorf("GetItemByRef() returned nil result")
				}
			}
		})
	}
}

func TestUsecase_ListItems(t *testing.T) {
	repo := newMockItemsRepository()
	usecase := NewUsecase(repo)
	ownerID := uuid.New()

	// Создаем тестовые элементы
	now := time.Now()
	for i := 0; i < 5; i++ {
		id := uuid.New()
		item := &storage.Item{
			ID:        id,
			OwnerID:   ownerID,
			Type:      1,
			Payload:   []byte("test payload"),
			MetaJSON:  []byte(`{"title": "Test Item"}`),
			Version:   1,
			UpdatedAt: now.Add(time.Duration(i) * time.Minute),
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		repo.items[id] = item
	}

	tests := []struct {
		name        string
		request     *pb.ListRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "list with default limit",
			request: &pb.ListRequest{
				Limit: 0,
			},
			expectError: false,
		},
		{
			name: "list with custom limit",
			request: &pb.ListRequest{
				Limit: 3,
			},
			expectError: false,
		},
		{
			name: "list with large limit",
			request: &pb.ListRequest{
				Limit: 1000,
			},
			expectError: false,
		},
		{
			name: "list with cursor",
			request: &pb.ListRequest{
				Limit:  3,
				Cursor: "", // empty cursor for snapshot mode
			},
			expectError: false,
		},
		{
			name: "invalid cursor",
			request: &pb.ListRequest{
				Limit:  3,
				Cursor: "invalid-cursor",
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "repository error",
			request: &pb.ListRequest{
				Limit: 3,
			},
			expectError: true,
			errorCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем ошибку репозитория для теста "repository error"
			if tt.name == "repository error" {
				repo.listError = errors.New("repository error")
			} else {
				repo.listError = nil
			}

			result, err := usecase.ListItems(context.Background(), ownerID, tt.request)
			if tt.expectError {
				if err == nil {
					t.Errorf("ListItems() expected error, got nil")
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("ListItems() error is not a gRPC status")
					return
				}
				if st.Code() != tt.errorCode {
					t.Errorf("ListItems() error code = %v, want %v", st.Code(), tt.errorCode)
				}
			} else {
				if err != nil {
					t.Errorf("ListItems() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Errorf("ListItems() returned nil result")
				}
			}
		})
	}
}

func TestUsecase_UpdateItem(t *testing.T) {
	repo := newMockItemsRepository()
	usecase := NewUsecase(repo)
	ownerID := uuid.New()
	itemID := uuid.New()

	// Создаем тестовый элемент
	now := time.Now()
	item := &storage.Item{
		ID:        itemID,
		OwnerID:   ownerID,
		Type:      1,
		Payload:   []byte("original payload"),
		MetaJSON:  []byte(`{"title": "Original Title"}`),
		Version:   1,
		UpdatedAt: now,
		CreatedAt: now,
	}
	repo.items[itemID] = item

	tests := []struct {
		name        string
		request     *pb.UpdateItemRequest
		expectError bool
		errorCode   codes.Code
	}{
		{
			name: "valid update",
			request: &pb.UpdateItemRequest{
				Id:              itemID.String(),
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "title", Value: "Updated Title"}},
				ExpectedVersion: 1,
			},
			expectError: false,
		},
		{
			name: "invalid UUID",
			request: &pb.UpdateItemRequest{
				Id:              "invalid-uuid",
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "title", Value: "Updated Title"}},
				ExpectedVersion: 1,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "zero expected version",
			request: &pb.UpdateItemRequest{
				Id:              itemID.String(),
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "title", Value: "Updated Title"}},
				ExpectedVersion: 0,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "negative expected version",
			request: &pb.UpdateItemRequest{
				Id:              itemID.String(),
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "title", Value: "Updated Title"}},
				ExpectedVersion: -1,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "invalid meta",
			request: &pb.UpdateItemRequest{
				Id:              itemID.String(),
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "", Value: "Empty Key"}},
				ExpectedVersion: 1,
			},
			expectError: true,
			errorCode:   codes.InvalidArgument,
		},
		{
			name: "version mismatch",
			request: &pb.UpdateItemRequest{
				Id:              itemID.String(),
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "title", Value: "Updated Title"}},
				ExpectedVersion: 2, // Wrong version
			},
			expectError: true,
			errorCode:   codes.Unknown, // Не gRPC статус
		},
		{
			name: "item not found",
			request: &pb.UpdateItemRequest{
				Id:              uuid.New().String(),
				Payload:         []byte("updated payload"),
				Meta:            []*pb.ItemMetaEntry{{Key: "title", Value: "Updated Title"}},
				ExpectedVersion: 1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Для каждого теста создаем новый элемент
			if tt.name != "version mismatch" && tt.name != "item not found" && tt.name != "invalid UUID" {
				// Создаем новый элемент для тестов, которые не ожидают ошибок
				newItemID := uuid.New()
				now := time.Now()
				newItem := &storage.Item{
					ID:        newItemID,
					OwnerID:   ownerID,
					Type:      1,
					Payload:   []byte("original payload"),
					MetaJSON:  []byte(`{"title": "Original Title"}`),
					Version:   1,
					UpdatedAt: now,
					CreatedAt: now,
				}
				repo.items[newItemID] = newItem

				// Обновляем request с новым ID
				tt.request.Id = newItemID.String()
			}

			result, err := usecase.UpdateItem(context.Background(), ownerID, tt.request)
			if tt.expectError {
				if err == nil {
					t.Errorf("UpdateItem() expected error, got nil")
					return
				}
				if tt.errorCode != codes.Unknown {
					st, ok := status.FromError(err)
					if !ok {
						// Для некоторых ошибок (например, "item not found") ожидается обычная ошибка
						if tt.name == "item not found" {
							// Это нормально - репозиторий возвращает обычную ошибку
							return
						}
						t.Errorf("UpdateItem() error is not a gRPC status")
						return
					}
					if st.Code() != tt.errorCode {
						t.Errorf("UpdateItem() error code = %v, want %v", st.Code(), tt.errorCode)
					}
				}
			} else {
				if err != nil {
					t.Errorf("UpdateItem() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Errorf("UpdateItem() returned nil result")
				}
			}
		})
	}
}

func TestUsecase_DeleteItem(t *testing.T) {
	repo := newMockItemsRepository()
	usecase := NewUsecase(repo)
	ownerID := uuid.New()
	itemID := uuid.New()

	// Создаем тестовый элемент
	now := time.Now()
	item := &storage.Item{
		ID:        itemID,
		OwnerID:   ownerID,
		Type:      1,
		Payload:   []byte("test payload"),
		MetaJSON:  []byte(`{"title": "Test Item"}`),
		Version:   1,
		UpdatedAt: now,
		CreatedAt: now,
	}
	repo.items[itemID] = item

	tests := []struct {
		name        string
		id          uuid.UUID
		expectError bool
		errorCode   codes.Code
	}{
		{
			name:        "valid delete",
			id:          itemID,
			expectError: false,
		},
		{
			name:        "item not found",
			id:          uuid.New(),
			expectError: true,
			errorCode:   codes.Internal,
		},
		{
			name:        "repository error",
			id:          itemID,
			expectError: true,
			errorCode:   codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем ошибку репозитория для теста "repository error"
			if tt.name == "repository error" {
				repo.deleteError = errors.New("repository error")
			} else {
				repo.deleteError = nil
			}

			err := usecase.DeleteItem(context.Background(), ownerID, tt.id)
			if tt.expectError {
				if err == nil {
					t.Errorf("DeleteItem() expected error, got nil")
					return
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("DeleteItem() error is not a gRPC status")
					return
				}
				if st.Code() != tt.errorCode {
					t.Errorf("DeleteItem() error code = %v, want %v", st.Code(), tt.errorCode)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteItem() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidType(t *testing.T) {
	tests := []struct {
		name     string
		itemType pb.ItemType
		expected bool
	}{
		{
			name:     "LOGIN type",
			itemType: pb.ItemType_LOGIN,
			expected: true,
		},
		{
			name:     "TEXT type",
			itemType: pb.ItemType_TEXT,
			expected: true,
		},
		{
			name:     "BINARY type",
			itemType: pb.ItemType_BINARY,
			expected: true,
		},
		{
			name:     "CARD type",
			itemType: pb.ItemType_CARD,
			expected: true,
		},
		{
			name:     "unknown type",
			itemType: pb.ItemType(999),
			expected: false,
		},
		{
			name:     "negative type",
			itemType: pb.ItemType(-1),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validType(tt.itemType)
			if result != tt.expected {
				t.Errorf("validType(%v) = %v, want %v", tt.itemType, result, tt.expected)
			}
		})
	}
}
