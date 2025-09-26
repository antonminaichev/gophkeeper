package validate

import (
	"testing"

	"github.com/antonminaichev/gophkeeper/internal/domain"
)

func TestItemForCreate(t *testing.T) {
	tests := []struct {
		name    string
		item    domain.Item
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid item with alias",
			item: domain.Item{
				Type:  "LOGIN",
				Alias: "valid-alias",
			},
			wantErr: false,
		},
		{
			name: "valid item without alias",
			item: domain.Item{
				Type:  "TEXT",
				Alias: "",
			},
			wantErr: false,
		},
		{
			name: "valid item with whitespace alias",
			item: domain.Item{
				Type:  "CARD",
				Alias: "   ",
			},
			wantErr: false,
		},
		{
			name: "valid alias with numbers",
			item: domain.Item{
				Type:  "BINARY",
				Alias: "alias123",
			},
			wantErr: false,
		},
		{
			name: "valid alias with underscores",
			item: domain.Item{
				Type:  "LOGIN",
				Alias: "valid_alias",
			},
			wantErr: false,
		},
		{
			name: "valid alias with hyphens",
			item: domain.Item{
				Type:  "TEXT",
				Alias: "valid-alias",
			},
			wantErr: false,
		},
		{
			name: "valid alias with dots",
			item: domain.Item{
				Type:  "CARD",
				Alias: "valid.alias",
			},
			wantErr: false,
		},
		{
			name: "valid alias with mixed characters",
			item: domain.Item{
				Type:  "BINARY",
				Alias: "valid_alias-123.test",
			},
			wantErr: false,
		},
		{
			name: "valid alias at max length",
			item: domain.Item{
				Type:  "LOGIN",
				Alias: "a123456789012345678901234567890123456789012345678901234567890123", // 63 chars
			},
			wantErr: false,
		},

		// Error cases
		{
			name: "empty type",
			item: domain.Item{
				Type:  "",
				Alias: "valid-alias",
			},
			wantErr: true,
			errMsg:  "item type is required",
		},
		{
			name: "invalid alias with spaces",
			item: domain.Item{
				Type:  "LOGIN",
				Alias: "invalid alias",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with special characters",
			item: domain.Item{
				Type:  "TEXT",
				Alias: "invalid@alias",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with parentheses",
			item: domain.Item{
				Type:  "CARD",
				Alias: "invalid(alias)",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with brackets",
			item: domain.Item{
				Type:  "BINARY",
				Alias: "invalid[alias]",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with slashes",
			item: domain.Item{
				Type:  "LOGIN",
				Alias: "invalid/alias",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with backslashes",
			item: domain.Item{
				Type:  "TEXT",
				Alias: "invalid\\alias",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with quotes",
			item: domain.Item{
				Type:  "CARD",
				Alias: "invalid\"alias\"",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias too long",
			item: domain.Item{
				Type:  "BINARY",
				Alias: "a12345678901234567890123456789012345678901234567890123456789012345", // 65 chars
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
		{
			name: "invalid alias with unicode",
			item: domain.Item{
				Type:  "LOGIN",
				Alias: "invalid-алиас",
			},
			wantErr: true,
			errMsg:  "invalid alias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ItemForCreate(tt.item)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ItemForCreate() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ItemForCreate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ItemForCreate() unexpected error = %v", err)
				}
			}
		})
	}
}
