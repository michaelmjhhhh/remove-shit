package clipboard

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNativeClipboardGetsExactText(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper requires a POSIX shell")
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "copied.txt")
	helper := filepath.Join(dir, "clipboard-helper")
	t.Setenv("CLIPBOARD_TEST_OUTPUT", output)
	if err := os.WriteFile(helper, []byte("#!/bin/sh\ncat > \"$CLIPBOARD_TEST_OUTPUT\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	text := "Hello 中文 👩‍💻\n\nA paragraph without display wrapping. $(echo untouched)"
	for _, platform := range []string{"darwin", "linux", "windows"} {
		var command string
		err := write(text, platform, func(string) string { return "1" }, func(name string) (string, error) { command = name; return helper, nil })
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"darwin": "pbcopy", "linux": "wl-copy", "windows": "powershell.exe"}[platform]
		if command != want {
			t.Fatalf("selected %q, want %q", command, want)
		}
		got, err := os.ReadFile(output)
		if err != nil || string(got) != text {
			t.Fatalf("clipboard content changed: %q (%v)", got, err)
		}
	}
}

func TestMissingOrFailedClipboardReturnsError(t *testing.T) {
	if err := write("hello", "darwin", os.Getenv, func(string) (string, error) { return "", exec.ErrNotFound }); err == nil {
		t.Fatal("missing clipboard reported success")
	}
	if runtime.GOOS == "windows" {
		return
	}
	path, err := exec.LookPath("false")
	if err != nil {
		t.Fatal(err)
	}
	if err := write("hello", "darwin", os.Getenv, func(string) (string, error) { return path, nil }); err == nil {
		t.Fatal("failed clipboard reported success")
	}
}
