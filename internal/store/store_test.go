package store

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no está definida")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateCreaTablasYEsIdempotente(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)

	for i := 0; i < 2; i++ {
		if err := Migrate(ctx, db); err != nil {
			t.Fatalf("migración %d: %v", i+1, err)
		}
	}

	for _, tabla := range []string{"alerts", "alert_events"} {
		var existe bool
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)`, tabla).Scan(&existe)
		if err != nil {
			t.Fatal(err)
		}
		if !existe {
			t.Fatalf("falta la tabla %s", tabla)
		}
	}
}

func nuevoStoreDeTest(t *testing.T, ventana time.Duration) *Store {
	t.Helper()
	ctx := context.Background()
	db := testDB(t)
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `TRUNCATE alert_events, alerts`); err != nil {
		t.Fatal(err)
	}
	return &Store{db: db, dedupWindow: ventana}
}
