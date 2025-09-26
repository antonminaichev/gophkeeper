package domain

import "testing"

func TestItemType_String(t *testing.T) {
	tests := []struct {
		name     string
		itemType ItemType
		expected string
	}{
		{
			name:     "LOGIN type",
			itemType: ItemLogin,
			expected: "LOGIN",
		},
		{
			name:     "TEXT type",
			itemType: ItemText,
			expected: "TEXT",
		},
		{
			name:     "BINARY type",
			itemType: ItemBinary,
			expected: "BINARY",
		},
		{
			name:     "CARD type",
			itemType: ItemCard,
			expected: "CARD",
		},
		{
			name:     "unknown type",
			itemType: ItemType(999),
			expected: "UNKNOWN",
		},
		{
			name:     "negative type",
			itemType: ItemType(-1),
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.itemType.String()
			if result != tt.expected {
				t.Errorf("ItemType.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestItemType_Constants(t *testing.T) {
	// Проверяем, что константы имеют ожидаемые значения
	if ItemLogin != 0 {
		t.Errorf("ItemLogin = %d, want 0", ItemLogin)
	}
	if ItemText != 1 {
		t.Errorf("ItemText = %d, want 1", ItemText)
	}
	if ItemBinary != 2 {
		t.Errorf("ItemBinary = %d, want 2", ItemBinary)
	}
	if ItemCard != 3 {
		t.Errorf("ItemCard = %d, want 3", ItemCard)
	}
}

func TestMeta(t *testing.T) {
	// Тестируем, что Meta является map[string]string
	meta := Meta{
		"title":       "Test Item",
		"description": "Test Description",
		"category":    "test",
	}

	if len(meta) != 3 {
		t.Errorf("Meta length = %d, want 3", len(meta))
	}

	if meta["title"] != "Test Item" {
		t.Errorf("Meta[\"title\"] = %s, want \"Test Item\"", meta["title"])
	}

	if meta["description"] != "Test Description" {
		t.Errorf("Meta[\"description\"] = %s, want \"Test Description\"", meta["description"])
	}

	if meta["category"] != "test" {
		t.Errorf("Meta[\"category\"] = %s, want \"test\"", meta["category"])
	}

	// Тестируем добавление нового элемента
	meta["new_key"] = "new_value"
	if meta["new_key"] != "new_value" {
		t.Errorf("Meta[\"new_key\"] = %s, want \"new_value\"", meta["new_key"])
	}

	// Тестируем удаление элемента
	delete(meta, "category")
	if _, exists := meta["category"]; exists {
		t.Errorf("Meta[\"category\"] should not exist after deletion")
	}
}

func TestItem(t *testing.T) {
	item := Item{
		Type:  "LOGIN",
		Alias: "test-alias",
	}

	if item.Type != "LOGIN" {
		t.Errorf("Item.Type = %s, want \"LOGIN\"", item.Type)
	}

	if item.Alias != "test-alias" {
		t.Errorf("Item.Alias = %s, want \"test-alias\"", item.Alias)
	}

	// Тестируем изменение полей
	item.Type = "TEXT"
	item.Alias = "new-alias"

	if item.Type != "TEXT" {
		t.Errorf("Item.Type after change = %s, want \"TEXT\"", item.Type)
	}

	if item.Alias != "new-alias" {
		t.Errorf("Item.Alias after change = %s, want \"new-alias\"", item.Alias)
	}
}
