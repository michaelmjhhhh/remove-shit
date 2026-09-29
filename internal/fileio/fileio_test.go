package fileio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveNeverOverwrites(t *testing.T) {
	p := filepath.Join(t.TempDir(), "output.txt")
	if err := Save(p, "中文\n\nHello"); err != nil {
		t.Fatal(err)
	}
	if err := Save(p, "replacement"); !os.IsExist(err) {
		t.Fatalf("expected exists error, got %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "中文\n\nHello\n" {
		t.Fatalf("unexpected file content: %q", b)
	}
}

func TestReadRejectsDirectory(t *testing.T) {
	if _, err := ReadFile(t.TempDir()); err == nil {
		t.Fatal("directory accepted")
	}
}
