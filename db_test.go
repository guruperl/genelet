package genelet

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	configPath := os.Getenv("SUMMER")
	if configPath == "" {
		t.Skip("SUMMER is unset; set SUMMER=$PWD/etc/summer.local.json to run DB-backed Genelet tests")
	}
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("SUMMER config %s is missing; run ./scripts/aofei-local.sh up", configPath)
		}
		t.Fatal(err)
	}
	c, err := NewConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open(c.ConnectArray[0], c.ConnectArray[1])
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("configured DB is unavailable: %v", err)
	}
	return db
}
