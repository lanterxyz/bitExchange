package coreapi_test

import (
	"os"
	"path/filepath"
	"testing"

	"bitExchange/internal/coreapi"
)

func TestAtomicWritePreservesOldFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`{"v":"old"}`), 0o600); err != nil {
		t.Fatalf("seed old file: %v", err)
	}

	badPath := filepath.Join(dir, "missing", "config.json")
	err := coreapi.AtomicWrite(badPath, []byte(`{"v":"new"}`))
	if err == nil {
		t.Fatalf("AtomicWrite should have failed for missing dir")
	}

	data, _ := os.ReadFile(path)
	if string(data) != `{"v":"old"}` {
		t.Fatalf("old file corrupted: %s", data)
	}
}

func TestAtomicWriteReplacesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := coreapi.AtomicWrite(path, []byte(`{"v":"new"}`)); err != nil {
		t.Fatalf("AtomicWrite: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != `{"v":"new"}` {
		t.Fatalf("file = %s", data)
	}
}
