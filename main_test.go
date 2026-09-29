package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBatchCLI(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.md")
	if err := os.WriteFile(source, []byte("# 标题\n\n**正文**"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{source}, os.Stdin, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "标题\n\n正文\n" || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	stdout.Reset()
	output := filepath.Join(dir, "clean.txt")
	args := []string{"--output", output, source}
	if err := run(args, os.Stdin, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 {
		t.Fatal("save polluted stdout")
	}
	if err := run(args, os.Stdin, &stdout, &stderr); !os.IsExist(err) {
		t.Fatalf("expected overwrite prevention, got %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "标题\n\n正文\n" {
		t.Fatalf("saved %q: %v", data, err)
	}
}

func TestPipedInput(t *testing.T) {
	p := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(p, []byte(`<p>Hello <b>world</b></p>`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out, diagnostic bytes.Buffer
	if err := run(nil, f, &out, &diagnostic); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello world\n" || diagnostic.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), diagnostic.String())
	}
}

func TestCLIInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"--format", "pdf"}, {"a", "b"}, {"--interactive", "--output", "x"}, {"/does/not/exist.md"}} {
		var out, diagnostic bytes.Buffer
		if err := run(args, os.Stdin, &out, &diagnostic); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if out.Len() != 0 {
			t.Fatal("error polluted stdout")
		}
	}
	var out, diagnostic bytes.Buffer
	if err := run([]string{"--help"}, os.Stdin, &out, &diagnostic); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diagnostic.String(), "Usage:") {
		t.Fatal("missing help")
	}
}

func TestVersion(t *testing.T) {
	var out, diagnostic bytes.Buffer
	if err := run([]string{"--version"}, os.Stdin, &out, &diagnostic); err != nil {
		t.Fatal(err)
	}
	if out.String() != "remove-shit "+version+"\n" || diagnostic.Len() != 0 {
		t.Fatalf("unexpected version output: %q / %q", out.String(), diagnostic.String())
	}
}
