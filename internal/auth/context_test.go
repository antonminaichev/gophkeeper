package auth

import (
	"context"
	"testing"
)

func TestUserIDFrom(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		wantID string
		wantOK bool
	}{
		{
			name:   "no user ID in context",
			ctx:    context.Background(),
			wantID: "",
			wantOK: false,
		},
		{
			name:   "empty user ID in context",
			ctx:    context.WithValue(context.Background(), userIDKey, ""),
			wantID: "",
			wantOK: false,
		},
		{
			name:   "valid user ID in context",
			ctx:    context.WithValue(context.Background(), userIDKey, "user123"),
			wantID: "user123",
			wantOK: true,
		},
		{
			name:   "non-string user ID in context",
			ctx:    context.WithValue(context.Background(), userIDKey, 123),
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := UserIDFrom(tt.ctx)
			if gotID != tt.wantID {
				t.Errorf("UserIDFrom() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotOK != tt.wantOK {
				t.Errorf("UserIDFrom() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestWithUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user123"

	ctxWithUser := WithUser(ctx, userID, "")

	gotID, gotOK := UserIDFrom(ctxWithUser)
	if !gotOK {
		t.Errorf("WithUserID() context should contain user ID")
	}
	if gotID != userID {
		t.Errorf("WithUserID() gotID = %v, want %v", gotID, userID)
	}
}

func TestUserEmailFrom(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		wantEmail string
		wantOK    bool
	}{
		{
			name:      "no user email in context",
			ctx:       context.Background(),
			wantEmail: "",
			wantOK:    false,
		},
		{
			name:      "empty user email in context",
			ctx:       context.WithValue(context.Background(), userEmailKey, ""),
			wantEmail: "",
			wantOK:    false,
		},
		{
			name:      "valid user email in context",
			ctx:       context.WithValue(context.Background(), userEmailKey, "test@example.com"),
			wantEmail: "test@example.com",
			wantOK:    true,
		},
		{
			name:      "non-string user email in context",
			ctx:       context.WithValue(context.Background(), userEmailKey, 123),
			wantEmail: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEmail, gotOK := UserEmailFrom(tt.ctx)
			if gotEmail != tt.wantEmail {
				t.Errorf("UserEmailFrom() gotEmail = %v, want %v", gotEmail, tt.wantEmail)
			}
			if gotOK != tt.wantOK {
				t.Errorf("UserEmailFrom() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestUserFrom(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		wantID    string
		wantEmail string
		wantOK    bool
	}{
		{
			name:      "no user in context",
			ctx:       context.Background(),
			wantID:    "",
			wantEmail: "",
			wantOK:    false,
		},
		{
			name:      "user ID only",
			ctx:       context.WithValue(context.Background(), userIDKey, "user123"),
			wantID:    "user123",
			wantEmail: "",
			wantOK:    true,
		},
		{
			name:      "user ID and email",
			ctx:       WithUser(context.Background(), "user123", "test@example.com"),
			wantID:    "user123",
			wantEmail: "test@example.com",
			wantOK:    true,
		},
		{
			name:      "empty user ID",
			ctx:       context.WithValue(context.Background(), userIDKey, ""),
			wantID:    "",
			wantEmail: "",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotEmail, gotOK := UserFrom(tt.ctx)
			if gotID != tt.wantID {
				t.Errorf("UserFrom() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotEmail != tt.wantEmail {
				t.Errorf("UserFrom() gotEmail = %v, want %v", gotEmail, tt.wantEmail)
			}
			if gotOK != tt.wantOK {
				t.Errorf("UserFrom() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

func TestWithUser(t *testing.T) {
	ctx := context.Background()
	userID := "user123"
	email := "test@example.com"

	// Test with both ID and email
	ctxWithUser := WithUser(ctx, userID, email)

	gotID, gotOK := UserIDFrom(ctxWithUser)
	if !gotOK || gotID != userID {
		t.Errorf("WithUser() user ID not set correctly")
	}

	gotEmail, gotOK := UserEmailFrom(ctxWithUser)
	if !gotOK || gotEmail != email {
		t.Errorf("WithUser() user email not set correctly")
	}

	// Test with empty email
	ctxWithUserOnlyID := WithUser(ctx, userID, "")

	gotID, gotOK = UserIDFrom(ctxWithUserOnlyID)
	if !gotOK || gotID != userID {
		t.Errorf("WithUser() user ID not set correctly with empty email")
	}

	gotEmail, gotOK = UserEmailFrom(ctxWithUserOnlyID)
	if gotOK {
		t.Errorf("WithUser() should not set email when empty")
	}
}

func TestUserIDFromContext(t *testing.T) {
	tests := []struct {
		name   string
		ctx    context.Context
		wantID string
		wantOK bool
	}{
		{
			name:   "no user ID in context",
			ctx:    context.Background(),
			wantID: "",
			wantOK: false,
		},
		{
			name:   "empty user ID in context",
			ctx:    context.WithValue(context.Background(), userIDKey, ""),
			wantID: "",
			wantOK: false,
		},
		{
			name:   "valid user ID in context",
			ctx:    context.WithValue(context.Background(), userIDKey, "user123"),
			wantID: "user123",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := UserIDFrom(tt.ctx)
			if gotID != tt.wantID {
				t.Errorf("UserIDFromContext() gotID = %v, want %v", gotID, tt.wantID)
			}
			if gotOK != tt.wantOK {
				t.Errorf("UserIDFromContext() gotOK = %v, want %v", gotOK, tt.wantOK)
			}
		})
	}
}

// Test that context values are properly isolated
func TestContextIsolation(t *testing.T) {
	ctx1 := WithUser(context.Background(), "user1", "")
	ctx2 := WithUser(context.Background(), "user2", "")

	id1, ok1 := UserIDFrom(ctx1)
	id2, ok2 := UserIDFrom(ctx2)

	if !ok1 || !ok2 {
		t.Errorf("Both contexts should contain user IDs")
	}

	if id1 == id2 {
		t.Errorf("Context values should be isolated: id1=%v, id2=%v", id1, id2)
	}

	if id1 != "user1" {
		t.Errorf("ctx1 should contain user1, got %v", id1)
	}

	if id2 != "user2" {
		t.Errorf("ctx2 should contain user2, got %v", id2)
	}
}

// Test context inheritance
func TestContextInheritance(t *testing.T) {
	baseCtx := context.Background()
	ctx1 := WithUser(baseCtx, "user1", "")
	ctx2 := context.WithValue(ctx1, "other_key", "other_value")

	// ctx2 should inherit user ID from ctx1
	id, ok := UserIDFrom(ctx2)
	if !ok {
		t.Errorf("Inherited context should contain user ID")
	}
	if id != "user1" {
		t.Errorf("Inherited context should contain user1, got %v", id)
	}

	// ctx2 should also contain the other value
	otherVal := ctx2.Value("other_key")
	if otherVal != "other_value" {
		t.Errorf("Inherited context should contain other value")
	}
}
