package client

import (
	"testing"

	pb "github.com/antonminaichev/gophkeeper/api/proto"
)

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid UUID",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: true,
		},
		{
			name:     "valid UUID with spaces",
			input:    "  550e8400-e29b-41d4-a716-446655440000  ",
			expected: true,
		},
		{
			name:     "invalid UUID - too short",
			input:    "550e8400-e29b-41d4-a716-44665544000",
			expected: false,
		},
		{
			name:     "invalid UUID - too long",
			input:    "550e8400-e29b-41d4-a716-4466554400000",
			expected: false,
		},
		{
			name:     "invalid UUID - wrong format",
			input:    "550e8400-e29b-41d4-a716-44665544000-",
			expected: false,
		},
		{
			name:     "invalid UUID - no dashes",
			input:    "550e8400e29b41d4a716446655440000",
			expected: false,
		},
		{
			name:     "invalid UUID - too many dashes",
			input:    "550e8400-e29b-41d4-a716-446655440000-",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: false,
		},
		{
			name:     "not a UUID",
			input:    "not-a-uuid",
			expected: false,
		},
		{
			name:     "number",
			input:    "123456789",
			expected: false,
		},
		{
			name:     "UUID with tabs",
			input:    "\t550e8400-e29b-41d4-a716-446655440000\t",
			expected: true,
		},
		{
			name:     "UUID with newlines",
			input:    "\n550e8400-e29b-41d4-a716-446655440000\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LooksLikeUUID(tt.input)
			if result != tt.expected {
				t.Errorf("LooksLikeUUID(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCLI_buildItemRef(t *testing.T) {
	cli := NewCLI("1.0.0", "abc123", "2023-01-01")

	tests := []struct {
		name     string
		selector string
		expected *pb.ItemRef
	}{
		{
			name:     "UUID selector",
			selector: "550e8400-e29b-41d4-a716-446655440000",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "550e8400-e29b-41d4-a716-446655440000"}},
		},
		{
			name:     "UUID with spaces",
			selector: "  550e8400-e29b-41d4-a716-446655440000  ",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "550e8400-e29b-41d4-a716-446655440000"}},
		},
		{
			name:     "alias selector",
			selector: "@test-alias",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Alias{Alias: "test-alias"}},
		},
		{
			name:     "alias with spaces",
			selector: "  @test-alias  ",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Alias{Alias: "test-alias"}},
		},
		{
			name:     "human ID selector",
			selector: "123",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_HumanId{HumanId: 123}},
		},
		{
			name:     "human ID with spaces",
			selector: "  123  ",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_HumanId{HumanId: 123}},
		},
		{
			name:     "zero human ID",
			selector: "0",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "0"}},
		},
		{
			name:     "negative human ID",
			selector: "-1",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "-1"}},
		},
		{
			name:     "non-numeric string",
			selector: "not-a-number",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "not-a-number"}},
		},
		{
			name:     "empty string",
			selector: "",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: ""}},
		},
		{
			name:     "only spaces",
			selector: "   ",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: ""}},
		},
		{
			name:     "large number",
			selector: "999999999999999999",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_HumanId{HumanId: 999999999999999999}},
		},
		{
			name:     "decimal number",
			selector: "123.45",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "123.45"}},
		},
		{
			name:     "alias without @",
			selector: "test-alias",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "test-alias"}},
		},
		{
			name:     "empty alias",
			selector: "@",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Alias{Alias: ""}},
		},
		{
			name:     "alias with @ in middle",
			selector: "test@alias",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Id{Id: "test@alias"}},
		},
		{
			name:     "multiple @ symbols",
			selector: "@@test",
			expected: &pb.ItemRef{Ref: &pb.ItemRef_Alias{Alias: "@test"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cli.buildItemRef(tt.selector)
			if result == nil {
				t.Errorf("buildItemRef(%q) returned nil", tt.selector)
				return
			}
			if result.Ref == nil {
				t.Errorf("buildItemRef(%q) returned nil Ref", tt.selector)
				return
			}

			// Check the type of reference
			switch expectedRef := tt.expected.Ref.(type) {
			case *pb.ItemRef_Id:
				if actualRef, ok := result.Ref.(*pb.ItemRef_Id); !ok {
					t.Errorf("buildItemRef(%q) expected ID ref, got %T", tt.selector, result.Ref)
				} else if actualRef.Id != expectedRef.Id {
					t.Errorf("buildItemRef(%q) ID = %q, want %q", tt.selector, actualRef.Id, expectedRef.Id)
				}
			case *pb.ItemRef_Alias:
				if actualRef, ok := result.Ref.(*pb.ItemRef_Alias); !ok {
					t.Errorf("buildItemRef(%q) expected Alias ref, got %T", tt.selector, result.Ref)
				} else if actualRef.Alias != expectedRef.Alias {
					t.Errorf("buildItemRef(%q) Alias = %q, want %q", tt.selector, actualRef.Alias, expectedRef.Alias)
				}
			case *pb.ItemRef_HumanId:
				if actualRef, ok := result.Ref.(*pb.ItemRef_HumanId); !ok {
					t.Errorf("buildItemRef(%q) expected HumanId ref, got %T", tt.selector, result.Ref)
				} else if actualRef.HumanId != expectedRef.HumanId {
					t.Errorf("buildItemRef(%q) HumanId = %d, want %d", tt.selector, actualRef.HumanId, expectedRef.HumanId)
				}
			}
		})
	}
}

func TestCLI_buildItemRefWithCache(t *testing.T) {
	cli := NewCLI("1.0.0", "abc123", "2023-01-01")

	// Test with cache lookup
	t.Run("cache lookup", func(t *testing.T) {
		// This test would require setting up a mock cache
		// For now, we'll test the basic functionality
		selector := "test-selector"
		result := cli.buildItemRef(selector)
		if result == nil {
			t.Errorf("buildItemRef(%q) returned nil", selector)
		}
	})
}

func TestCLI_buildItemRefEdgeCases(t *testing.T) {
	cli := NewCLI("1.0.0", "abc123", "2023-01-01")

	tests := []struct {
		name     string
		selector string
	}{
		{
			name:     "very long string",
			selector: "a" + string(make([]byte, 1000)),
		},
		{
			name:     "unicode string",
			selector: "тест-селектор",
		},
		{
			name:     "special characters",
			selector: "!@#$%^&*()",
		},
		{
			name:     "newlines and tabs",
			selector: "\n\t\r test \t\n\r",
		},
		{
			name:     "mixed case alias",
			selector: "@Test-Alias",
		},
		{
			name:     "alias with numbers",
			selector: "@test123",
		},
		{
			name:     "alias with special chars",
			selector: "@test_alias-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cli.buildItemRef(tt.selector)
			if result == nil {
				t.Errorf("buildItemRef(%q) returned nil", tt.selector)
			}
			if result.Ref == nil {
				t.Errorf("buildItemRef(%q) returned nil Ref", tt.selector)
			}
		})
	}
}
