package genelet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadComponentReturnsErrors(t *testing.T) {
	if _, err := LoadComponent(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("LoadComponent missing file returned nil error")
	}

	dir := t.TempDir()
	badJSON := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSON, []byte(`{`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadComponent(badJSON); err == nil {
		t.Fatal("LoadComponent malformed JSON returned nil error")
	}

	unsafe := filepath.Join(dir, "unsafe.json")
	if err := os.WriteFile(unsafe, []byte(`{"current_table":"users; DROP TABLE users"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadComponent(unsafe); err == nil {
		t.Fatal("LoadComponent unsafe table returned nil error")
	}
}
