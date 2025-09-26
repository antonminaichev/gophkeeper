package vault

import (
	"testing"
	"time"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
	"github.com/antonminaichev/gophkeeper/internal/server/storage"
	"github.com/google/uuid"
)

func TestToPB(t *testing.T) {
	now := time.Now()
	ownerID := uuid.New()
	itemID := uuid.New()
	alias := "test-alias"
	humanID := int64(123)

	tests := []struct {
		name     string
		item     *storage.Item
		expected *pb.Item
	}{
		{
			name: "item with all fields",
			item: &storage.Item{
				ID:        itemID,
				OwnerID:   ownerID,
				Type:      1, // LOGIN
				Payload:   []byte("test payload"),
				MetaJSON:  []byte(`{"title": "Test Item"}`),
				Version:   1,
				HumanID:   &humanID,
				Alias:     &alias,
				DeletedAt: nil,
				UpdatedAt: now,
				CreatedAt: now,
			},
			expected: &pb.Item{
				Id:            itemID.String(),
				HumanId:       humanID,
				Alias:         alias,
				Type:          pb.ItemType(1), // LOGIN
				Payload:       []byte("test payload"),
				Meta:          []*pb.ItemMetaEntry{{Key: "title", Value: "Test Item"}},
				Version:       1,
				UpdatedAtUnix: now.Unix(),
				Deleted:       false,
			},
		},
		{
			name: "item without alias and human ID",
			item: &storage.Item{
				ID:        itemID,
				OwnerID:   ownerID,
				Type:      2, // TEXT
				Payload:   []byte("text content"),
				MetaJSON:  []byte(`{"title": "Text Item"}`),
				Version:   2,
				HumanID:   nil,
				Alias:     nil,
				DeletedAt: nil,
				UpdatedAt: now,
				CreatedAt: now,
			},
			expected: &pb.Item{
				Id:            itemID.String(),
				HumanId:       0,
				Alias:         "",
				Type:          pb.ItemType(2), // TEXT
				Payload:       []byte("text content"),
				Meta:          []*pb.ItemMetaEntry{{Key: "title", Value: "Text Item"}},
				Version:       2,
				UpdatedAtUnix: now.Unix(),
				Deleted:       false,
			},
		},
		{
			name: "deleted item",
			item: &storage.Item{
				ID:        itemID,
				OwnerID:   ownerID,
				Type:      3, // BINARY
				Payload:   []byte("binary content"),
				MetaJSON:  []byte(`{"title": "Binary Item"}`),
				Version:   3,
				HumanID:   &humanID,
				Alias:     &alias,
				DeletedAt: &now,
				UpdatedAt: now,
				CreatedAt: now,
			},
			expected: &pb.Item{
				Id:            itemID.String(),
				HumanId:       humanID,
				Alias:         alias,
				Type:          pb.ItemType(3), // BINARY
				Payload:       []byte("binary content"),
				Meta:          []*pb.ItemMetaEntry{{Key: "title", Value: "Binary Item"}},
				Version:       3,
				UpdatedAtUnix: now.Unix(),
				Deleted:       true,
			},
		},
		{
			name: "item with empty meta",
			item: &storage.Item{
				ID:        itemID,
				OwnerID:   ownerID,
				Type:      4, // CARD
				Payload:   []byte("card data"),
				MetaJSON:  []byte(`{}`),
				Version:   1,
				HumanID:   nil,
				Alias:     nil,
				DeletedAt: nil,
				UpdatedAt: now,
				CreatedAt: now,
			},
			expected: &pb.Item{
				Id:            itemID.String(),
				HumanId:       0,
				Alias:         "",
				Type:          pb.ItemType(4), // CARD
				Payload:       []byte("card data"),
				Meta:          nil,
				Version:       1,
				UpdatedAtUnix: now.Unix(),
				Deleted:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPB(tt.item)
			if result.Id != tt.expected.Id {
				t.Errorf("toPB().Id = %s, want %s", result.Id, tt.expected.Id)
			}
			if result.HumanId != tt.expected.HumanId {
				t.Errorf("toPB().HumanId = %d, want %d", result.HumanId, tt.expected.HumanId)
			}
			if result.Alias != tt.expected.Alias {
				t.Errorf("toPB().Alias = %s, want %s", result.Alias, tt.expected.Alias)
			}
			if result.Type != tt.expected.Type {
				t.Errorf("toPB().Type = %v, want %v", result.Type, tt.expected.Type)
			}
			if string(result.Payload) != string(tt.expected.Payload) {
				t.Errorf("toPB().Payload = %s, want %s", string(result.Payload), string(tt.expected.Payload))
			}
			if result.Version != tt.expected.Version {
				t.Errorf("toPB().Version = %d, want %d", result.Version, tt.expected.Version)
			}
			if result.UpdatedAtUnix != tt.expected.UpdatedAtUnix {
				t.Errorf("toPB().UpdatedAtUnix = %d, want %d", result.UpdatedAtUnix, tt.expected.UpdatedAtUnix)
			}
			if result.Deleted != tt.expected.Deleted {
				t.Errorf("toPB().Deleted = %v, want %v", result.Deleted, tt.expected.Deleted)
			}
			// Проверяем meta
			if len(result.Meta) != len(tt.expected.Meta) {
				t.Errorf("toPB().Meta length = %d, want %d", len(result.Meta), len(tt.expected.Meta))
			}
		})
	}
}

func TestMapToMeta(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []*pb.ItemMetaEntry
	}{
		{
			name:     "empty JSON",
			input:    []byte("{}"),
			expected: nil,
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty input",
			input:    []byte(""),
			expected: nil,
		},
		{
			name:  "valid JSON with single entry",
			input: []byte(`{"title": "Test Item"}`),
			expected: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
		},
		{
			name:  "valid JSON with multiple entries",
			input: []byte(`{"title": "Test Item", "description": "Test Description", "category": "test"}`),
			expected: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
				{Key: "description", Value: "Test Description"},
				{Key: "category", Value: "test"},
			},
		},
		{
			name:     "invalid JSON",
			input:    []byte(`{"title": "Test Item"`),
			expected: nil,
		},
		{
			name:     "non-object JSON",
			input:    []byte(`"just a string"`),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapToMeta(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("mapToMeta() length = %d, want %d", len(result), len(tt.expected))
				return
			}
			// Проверяем, что все ожидаемые записи присутствуют
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual.Key == expected.Key && actual.Value == expected.Value {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("mapToMeta() missing entry: %s=%s", expected.Key, expected.Value)
				}
			}
		})
	}
}

func TestMetaToJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       []*pb.ItemMetaEntry
		expected    []byte
		expectError bool
	}{
		{
			name:        "empty entries",
			input:       []*pb.ItemMetaEntry{},
			expected:    nil,
			expectError: false,
		},
		{
			name:        "nil entries",
			input:       nil,
			expected:    nil,
			expectError: false,
		},
		{
			name: "single entry",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
			expected:    []byte(`{"title":"Test Item"}`),
			expectError: false,
		},
		{
			name: "multiple entries",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
				{Key: "description", Value: "Test Description"},
				{Key: "category", Value: "test"},
			},
			expected:    []byte(`{"category":"test","description":"Test Description","title":"Test Item"}`),
			expectError: false,
		},
		{
			name: "entry with empty key",
			input: []*pb.ItemMetaEntry{
				{Key: "", Value: "Test Value"},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "entry with whitespace key",
			input: []*pb.ItemMetaEntry{
				{Key: "   ", Value: "Test Value"},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := metaToJSON(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("metaToJSON() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("metaToJSON() unexpected error = %v", err)
					return
				}
				if string(result) != string(tt.expected) {
					t.Errorf("metaToJSON() = %s, want %s", string(result), string(tt.expected))
				}
			}
		})
	}
}

func TestEncodeCursor(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	result := encodeCursor(now, id)

	// Проверяем, что результат не пустой
	if result == "" {
		t.Errorf("encodeCursor() returned empty string")
	}

	// Проверяем, что результат можно декодировать
	decodedTime, decodedID, err := decodeCursor(result)
	if err != nil {
		t.Errorf("decodeCursor() error = %v", err)
		return
	}

	// Проверяем время с точностью до секунд (так как в JSON сохраняется Unix timestamp)
	if decodedTime.Unix() != now.Unix() {
		t.Errorf("decodeCursor() time = %v, want %v", decodedTime.Unix(), now.Unix())
	}

	if decodedID != id {
		t.Errorf("decodeCursor() id = %v, want %v", decodedID, id)
	}
}

func TestDecodeCursor(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	// Создаем валидный курсор
	validCursor := encodeCursor(now, id)

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid cursor",
			input:       validCursor,
			expectError: false,
		},
		{
			name:        "empty cursor",
			input:       "",
			expectError: true,
		},
		{
			name:        "invalid base64",
			input:       "invalid-base64!",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			input:       "aW52YWxpZC1qc29u", // base64 of "invalid-json"
			expectError: true,
		},
		{
			name:        "invalid UUID",
			input:       "eyJ1IjoxNjAwMDAwMDAwLCJpIjoiaW52YWxpZC11dWlkIn0=", // base64 of {"u":1600000000,"i":"invalid-uuid"}
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeCursor(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("decodeCursor() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("decodeCursor() unexpected error = %v", err)
				}
			}
		})
	}
}
