package output

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jmorrison-juniper/misthelper-go/internal/api"
)

func TestSQLitePersistenceAndCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writer, err := NewWriter(api.Config{OutputFormat: "sqlite"}, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Error(err)
		}
	})
	name := "Lab's \"west\", wing\n第二"
	record := []map[string]any{{"id": "fixture-id", "name": name, "location": map[string]any{"floor": "2"}}}
	if err := writer.Write(context.Background(), "listOrgSites", record); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := writer.Write(ctx, "listOrgSites", record); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled write error = %v; want context.Canceled", err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "mist_data.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	var gotName, floor string
	if err := db.QueryRow(`SELECT name, location_floor FROM listOrgSites WHERE id = ?`, "fixture-id").Scan(&gotName, &floor); err != nil {
		t.Fatal(err)
	}
	if gotName != name || floor != "2" {
		t.Fatalf("persisted fields = %q, %q; want %q, 2", gotName, floor, name)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM listOrgSites").Scan(&count); err != nil || count != 1 {
		t.Fatalf("cancelled write changed rows: count = %d, error = %v", count, err)
	}
}
