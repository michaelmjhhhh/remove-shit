package fileio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/michaelmjhhhh/remove-shit/internal/clean"
)

func Expand(path string) string {
	path = strings.TrimSpace(path)
	if len(path) >= 2 && (path[0] == '"' && path[len(path)-1] == '"' || path[0] == '\'' && path[len(path)-1] == '\'') {
		path = path[1 : len(path)-1]
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func Read(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, clean.MaxBytes+1))
	if err == nil && len(b) > clean.MaxBytes {
		err = fmt.Errorf("input exceeds the 32 MiB limit")
	}
	return b, err
}

func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(Expand(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("input must be a regular file")
	}
	return Read(f)
}

func Output(text string) string {
	if text == "" {
		return ""
	}
	return text + "\n"
}

// Save never overwrites an existing file. A failed write removes the partial file.
func Save(path, text string) error {
	path = Expand(path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, Output(text))
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}
