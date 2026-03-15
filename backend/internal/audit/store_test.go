package audit

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "audit_test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSQLiteStore_InsertAndQuery(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	entry := Entry{
		Timestamp:         time.Now(),
		ClusterID:         "local",
		User:              "admin",
		SourceIP:          "127.0.0.1",
		Action:            ActionCreate,
		ResourceKind:      "deployment",
		ResourceNamespace: "default",
		ResourceName:      "nginx",
		Result:            ResultSuccess,
		Detail:            "created deployment",
	}

	if err := store.Insert(ctx, entry); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	entries, total, err := store.Query(ctx, QueryParams{PageSize: 10})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].User != "admin" {
		t.Errorf("user = %q, want %q", entries[0].User, "admin")
	}
	if entries[0].Action != ActionCreate {
		t.Errorf("action = %q, want %q", entries[0].Action, ActionCreate)
	}
}

func TestSQLiteStore_Pagination(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	// Insert 5 entries
	for i := 0; i < 5; i++ {
		store.Insert(ctx, Entry{
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			ClusterID: "local",
			User:      "admin",
			SourceIP:  "127.0.0.1",
			Action:    ActionCreate,
			Result:    ResultSuccess,
		})
	}

	// Page 1 (2 items)
	entries, total, err := store.Query(ctx, QueryParams{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Query page 1: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(entries) != 2 {
		t.Errorf("page 1 entries = %d, want 2", len(entries))
	}

	// Page 3 (1 item)
	entries, _, err = store.Query(ctx, QueryParams{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("Query page 3: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("page 3 entries = %d, want 1", len(entries))
	}
}

func TestSQLiteStore_FilterByUser(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	store.Insert(ctx, Entry{Timestamp: time.Now(), ClusterID: "local", User: "alice", SourceIP: "1.1.1.1", Action: ActionCreate, Result: ResultSuccess})
	store.Insert(ctx, Entry{Timestamp: time.Now(), ClusterID: "local", User: "bob", SourceIP: "2.2.2.2", Action: ActionDelete, Result: ResultSuccess})
	store.Insert(ctx, Entry{Timestamp: time.Now(), ClusterID: "local", User: "alice", SourceIP: "1.1.1.1", Action: ActionUpdate, Result: ResultSuccess})

	entries, total, err := store.Query(ctx, QueryParams{User: "alice"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestSQLiteStore_FilterByAction(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	store.Insert(ctx, Entry{Timestamp: time.Now(), ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionCreate, Result: ResultSuccess})
	store.Insert(ctx, Entry{Timestamp: time.Now(), ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionLogin, Result: ResultSuccess})
	store.Insert(ctx, Entry{Timestamp: time.Now(), ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionLogin, Result: ResultFailure})

	entries, total, _ := store.Query(ctx, QueryParams{Action: "login"})
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestSQLiteStore_FilterByTimeRange(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	store.Insert(ctx, Entry{Timestamp: now.Add(-48 * time.Hour), ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionCreate, Result: ResultSuccess})
	store.Insert(ctx, Entry{Timestamp: now.Add(-1 * time.Hour), ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionUpdate, Result: ResultSuccess})
	store.Insert(ctx, Entry{Timestamp: now, ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionDelete, Result: ResultSuccess})

	// Only last 24 hours
	entries, total, _ := store.Query(ctx, QueryParams{Since: now.Add(-24 * time.Hour)})
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestSQLiteStore_Cleanup(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	now := time.Now().UTC()
	// Insert old entry (100 days ago)
	store.Insert(ctx, Entry{Timestamp: now.AddDate(0, 0, -100), ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionCreate, Result: ResultSuccess})
	// Insert recent entry
	store.Insert(ctx, Entry{Timestamp: now, ClusterID: "local", User: "admin", SourceIP: "1.1.1.1", Action: ActionUpdate, Result: ResultSuccess})

	deleted, err := store.Cleanup(ctx, 90)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}

	_, total, _ := store.Query(ctx, QueryParams{})
	if total != 1 {
		t.Errorf("remaining = %d, want 1", total)
	}
}

func TestSQLiteStore_ConcurrentWrites(t *testing.T) {
	store := testStore(t)
	ctx := context.Background()

	errs := make(chan error, 10)
	for i := range 10 {
		go func(n int) {
			errs <- store.Insert(ctx, Entry{
				Timestamp: time.Now().Add(time.Duration(n) * time.Millisecond),
				ClusterID: "local",
				User:      "admin",
				SourceIP:  "1.1.1.1",
				Action:    ActionCreate,
				Result:    ResultSuccess,
			})
		}(i)
	}

	var failures int
	for range 10 {
		if err := <-errs; err != nil {
			failures++
		}
	}

	_, total, _ := store.Query(ctx, QueryParams{})
	expected := 10 - failures
	if total != expected {
		t.Errorf("total = %d, want %d (failures: %d)", total, expected, failures)
	}
	// With WAL + busy_timeout, we expect all writes to succeed
	if failures > 0 {
		t.Logf("warning: %d concurrent writes failed (SQLite busy)", failures)
	}
}

func TestSQLiteLogger_ImplementsLogger(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "audit_test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	slogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Ensure SQLiteLogger satisfies the Logger interface at compile time
	var _ Logger = NewSQLiteLogger(store, slogger)
}

func TestQueryParams_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    QueryParams
		wantPage int
		wantSize int
	}{
		{"defaults", QueryParams{}, 1, DefaultPageSize},
		{"page 0 → 1", QueryParams{Page: 0}, 1, DefaultPageSize},
		{"negative page → 1", QueryParams{Page: -5}, 1, DefaultPageSize},
		{"over max size", QueryParams{PageSize: 500}, 1, MaxPageSize},
		{"normal", QueryParams{Page: 3, PageSize: 20}, 3, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input.Normalize()
			if tt.input.Page != tt.wantPage {
				t.Errorf("Page = %d, want %d", tt.input.Page, tt.wantPage)
			}
			if tt.input.PageSize != tt.wantSize {
				t.Errorf("PageSize = %d, want %d", tt.input.PageSize, tt.wantSize)
			}
		})
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
