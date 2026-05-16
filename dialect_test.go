package genelet

import (
	"net/url"
	"strings"
	"testing"
)

func TestNormalizeDriver(t *testing.T) {
	tests := map[string]string{
		"mysql":      "mysql",
		"mariadb":    "mysql",
		"postgres":   "postgres",
		"postgresql": "postgres",
		"pg":         "postgres",
		"sqlite":     "sqlite3",
		"sqlite3":    "sqlite3",
	}
	for in, want := range tests {
		if got := NormalizeDriver(in); got != want {
			t.Fatalf("NormalizeDriver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRebindSQLForPostgres(t *testing.T) {
	sql := "SELECT * FROM users WHERE name=? AND note='literal ?' AND nick='it''s ?' AND id IN (?,?)"
	got := RebindSQL("postgres", sql)
	want := "SELECT * FROM users WHERE name=$1 AND note='literal ?' AND nick='it''s ?' AND id IN ($2,$3)"
	if got != want {
		t.Fatalf("postgres SQL = %q, want %q", got, want)
	}
	if got := RebindSQL("sqlite", sql); got != sql {
		t.Fatalf("sqlite SQL = %q, want original", got)
	}
	if got := RebindSQL("mysql", sql); got != sql {
		t.Fatalf("mysql SQL = %q, want original", got)
	}
}

func TestSQLiteCrudSmoke(t *testing.T) {
	c := &Config{ConnectArray: []string{"sqlite", ":memory:"}}
	db, err := c.OpenDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT, name TEXT)"); err != nil {
		t.Fatal(err)
	}
	crud := NewCrud(db, "users", nil)
	crud.SetDriver(c.DriverName())
	crud.LastIDColumn = "id"
	if err := crud.InsertHash(url.Values{"email": {"a@example.test"}, "name": {"Ann"}}); err != nil {
		t.Fatal(err)
	}
	if crud.LastID != 1 {
		t.Fatalf("LastID = %d, want 1", crud.LastID)
	}
	lists := make([]map[string]interface{}, 0)
	if err := crud.TopicsHash(&lists, []string{"id", "email", "name"}); err != nil {
		t.Fatal(err)
	}
	if len(lists) != 1 || lists[0]["email"] != "a@example.test" {
		t.Fatalf("lists = %#v", lists)
	}
}

func TestPostgresInsertUsesReturningAndDollarPlaceholders(t *testing.T) {
	crud := NewCrud(nil, "users", nil)
	crud.SetDriver("postgres")
	crud.LastIDColumn = "id"
	query := crud.sql("INSERT INTO users (email, name) VALUES (?,?) RETURNING id")
	if !strings.Contains(query, "$1") || !strings.Contains(query, "$2") || strings.Contains(query, "?") {
		t.Fatalf("postgres query was not rebound: %s", query)
	}
}

func TestSQLiteProceduresAreUnsupported(t *testing.T) {
	dbi := &DBI{Driver: "sqlite3"}
	err := dbi.DoProc(map[string]interface{}{}, []string{"id"}, "login_user", "u", "p")
	if gerr, ok := err.(Gerror); !ok || gerr.Code != 1175 {
		t.Fatalf("sqlite DoProc = %#v, want 1175 Gerror", err)
	}
}
