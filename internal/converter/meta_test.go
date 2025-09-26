package pbconv

import (
	"testing"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

func TestMetaSliceToMap(t *testing.T) {
	tests := []struct {
		name     string
		input    []*pb.ItemMetaEntry
		expected map[string]string
	}{
		{
			name:     "empty slice",
			input:    []*pb.ItemMetaEntry{},
			expected: map[string]string{},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name: "single entry",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
			expected: map[string]string{
				"title": "Test Item",
			},
		},
		{
			name: "multiple entries",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
				{Key: "description", Value: "Test Description"},
				{Key: "category", Value: "test"},
			},
			expected: map[string]string{
				"title":       "Test Item",
				"description": "Test Description",
				"category":    "test",
			},
		},
		{
			name: "with nil entries",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
				nil,
				{Key: "description", Value: "Test Description"},
			},
			expected: map[string]string{
				"title":       "Test Item",
				"description": "Test Description",
			},
		},
		{
			name: "duplicate keys",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "First Title"},
				{Key: "title", Value: "Second Title"},
			},
			expected: map[string]string{
				"title": "Second Title", // последнее значение перезаписывает предыдущее
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MetaSliceToMap(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("MetaSliceToMap() length = %d, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("MetaSliceToMap()[%s] = %s, want %s", k, result[k], v)
				}
			}
		})
	}
}

func TestMapToMetaSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []*pb.ItemMetaEntry
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: nil,
		},
		{
			name:     "nil map",
			input:    nil,
			expected: nil,
		},
		{
			name: "single entry",
			input: map[string]string{
				"title": "Test Item",
			},
			expected: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
			},
		},
		{
			name: "multiple entries",
			input: map[string]string{
				"title":       "Test Item",
				"description": "Test Description",
				"category":    "test",
			},
			expected: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
				{Key: "description", Value: "Test Description"},
				{Key: "category", Value: "test"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapToMetaSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("MapToMetaSlice() length = %d, want %d", len(result), len(tt.expected))
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
					t.Errorf("MapToMetaSlice() missing entry: %s=%s", expected.Key, expected.Value)
				}
			}
		})
	}
}

func TestTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    []*pb.ItemMetaEntry
		expected string
	}{
		{
			name:     "empty slice",
			input:    []*pb.ItemMetaEntry{},
			expected: "",
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: "",
		},
		{
			name: "title present",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "Test Item"},
				{Key: "description", Value: "Test Description"},
			},
			expected: "Test Item",
		},
		{
			name: "title not present",
			input: []*pb.ItemMetaEntry{
				{Key: "description", Value: "Test Description"},
				{Key: "category", Value: "test"},
			},
			expected: "",
		},
		{
			name: "multiple titles",
			input: []*pb.ItemMetaEntry{
				{Key: "title", Value: "First Title"},
				{Key: "title", Value: "Second Title"},
			},
			expected: "First Title", // возвращает первое найденное
		},
		{
			name: "with nil entries",
			input: []*pb.ItemMetaEntry{
				nil,
				{Key: "title", Value: "Test Item"},
				nil,
			},
			expected: "Test Item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Title(tt.input)
			if result != tt.expected {
				t.Errorf("Title() = %s, want %s", result, tt.expected)
			}
		})
	}
}
