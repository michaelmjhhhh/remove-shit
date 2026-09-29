// Package clipboard copies plain text using the operating system's clipboard.
package clipboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func Write(text string) error {
	return write(text, runtime.GOOS, os.Getenv, exec.LookPath)
}

func write(text, platform string, getenv func(string) string, lookPath func(string) (string, error)) error {
	var candidates [][]string
	switch platform {
	case "darwin":
		candidates = [][]string{{"pbcopy"}}
	case "windows":
		candidates = [][]string{{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false); Set-Clipboard -Value ([Console]::In.ReadToEnd())"}}
	default:
		if getenv("WAYLAND_DISPLAY") != "" {
			candidates = append(candidates, []string{"wl-copy", "--type", "text/plain;charset=utf-8"})
		}
		if getenv("DISPLAY") != "" {
			candidates = append(candidates, []string{"xclip", "-selection", "clipboard"}, []string{"xsel", "--clipboard", "--input"})
		}
	}
	for _, candidate := range candidates {
		path, err := lookPath(candidate[0])
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, candidate[1:]...)
		cmd.Stdin = strings.NewReader(text)
		// Clipboard processes must not write into the terminal interface.
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s could not access the clipboard: %w", candidate[0], err)
		}
		return nil
	}
	return fmt.Errorf("no system clipboard available; save with s or print with p (Linux requires wl-copy, xclip or xsel and a desktop session)")
}
