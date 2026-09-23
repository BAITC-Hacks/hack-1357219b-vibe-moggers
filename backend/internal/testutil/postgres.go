package testutil

import (
	"context"
	"database/sql"
	"github.com/lib/pq"
	"net/url"
	"os"
	"strings"
	"testing"
	"vibe-moggers/backend/internal/database"
	"vibe-moggers/backend/internal/httpapi"
)

// Postgres creates and drops only its own random test schema, never public data.
func Postgres(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for isolated PostgreSQL integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	admin, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	id, err := httpapi.NewID()
	if err != nil {
		t.Fatal(err)
	}
	schema := "qadam_test_" + strings.ReplaceAll(id, "-", "")
	if _, err = admin.Exec(`CREATE SCHEMA ` + pq.QuoteIdentifier(schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(`DROP SCHEMA ` + pq.QuoteIdentifier(schema) + ` CASCADE`); err != nil {
			t.Error(err)
		}
	})
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db, err := database.Open(u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for i := 0; i < 2; i++ {
		if err = database.Migrate(context.Background(), db); err != nil {
			t.Fatal(err)
		}
	}
	return db, u.String()
}
