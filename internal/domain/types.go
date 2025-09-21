package domain

// ItemType represents the type of an item in the vault
type ItemType int

const (
	ItemLogin ItemType = iota
	ItemText
	ItemBinary
	ItemCard
)

// String returns the string representation of ItemType
func (t ItemType) String() string {
	switch t {
	case ItemLogin:
		return "LOGIN"
	case ItemText:
		return "TEXT"
	case ItemBinary:
		return "BINARY"
	case ItemCard:
		return "CARD"
	default:
		return "UNKNOWN"
	}
}

// Item represents a vault item
type Item struct {
	Type  string
	Alias string
}

// Meta represents metadata for an item
type Meta map[string]string
