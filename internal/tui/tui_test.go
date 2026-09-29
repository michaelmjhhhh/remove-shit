package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}
func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func TestPastePreviewSaveAndPrint(t *testing.T) {
	m := New("auto", nil, "")
	m, _ = update(m, key('1'))
	source := "# 标题\n\n" + strings.Repeat("**中文** 👩‍💻\n", 150)
	m, _ = update(m, tea.PasteMsg{Content: source})
	if m.area.Value() != source {
		t.Fatal("multiline paste was truncated or changed")
	}
	m, cmd := update(m, tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("clean command missing")
	}
	m, _ = update(m, cmd())
	if m.screen != preview || strings.Contains(m.Text(), "**") || strings.Count(m.Text(), "中文") != 150 {
		t.Fatalf("bad preview: %v %q", m.screen, m.Text())
	}
	m, _ = update(m, tea.WindowSizeMsg{Width: 40, Height: 16})
	if !strings.Contains(m.View().Content, "Clean text") {
		t.Fatal("resized preview missing")
	}
	m, _ = update(m, key('s'))
	path := filepath.Join(t.TempDir(), "saved.txt")
	m.path.SetValue(path)
	m, cmd = update(m, key(tea.KeyEnter))
	m, _ = update(m, cmd())
	if m.screen != preview || m.err != nil {
		t.Fatalf("save failed: %v", m.err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != m.Text()+"\n" {
		t.Fatalf("saved data differs: %v", err)
	}
	m, _ = update(m, key('s'))
	m.path.SetValue(path)
	m, cmd = update(m, key(tea.KeyEnter))
	m, _ = update(m, cmd())
	if m.screen != save || !os.IsExist(m.err) {
		t.Fatal("overwrite did not stay on save screen with an error")
	}
	m, _ = update(m, key(tea.KeyEscape))
	m, cmd = update(m, key('p'))
	if !m.Print || cmd == nil {
		t.Fatal("print & quit failed")
	}
}

func TestFileInputAndRecoverableError(t *testing.T) {
	m := New("auto", nil, "")
	m, _ = update(m, key('2'))
	m.path.SetValue(filepath.Join(t.TempDir(), "missing.md"))
	m, cmd := update(m, key(tea.KeyEnter))
	m, _ = update(m, cmd())
	if m.screen != file || m.err == nil {
		t.Fatal("file error is not recoverable")
	}
	source := filepath.Join(t.TempDir(), "source with spaces.html")
	if err := os.WriteFile(source, []byte("<p>Hello <b>world</b></p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	m.path.SetValue(source)
	m, cmd = update(m, key(tea.KeyEnter))
	m, _ = update(m, cmd())
	if m.screen != preview || m.Text() != "Hello world" {
		t.Fatal("file preview failed")
	}
	m, _ = update(m, key('s'))
	m, _ = update(m, key(tea.KeyEscape))
	m, _ = update(m, key(tea.KeyEscape))
	if m.screen != file || m.path.Value() != source {
		t.Fatal("returning from save lost source path")
	}
}

func TestPreviewWrapsWordsWithoutChangingOutput(t *testing.T) {
	text := "The purpose of travel is not simply the distance covered, but whether the scenery before you helps you see your inner self more clearly.\n\nHold onto your passions, embrace the journey, and stay curious—you will invariably meet wonderful things along the way."
	m := New("text", nil, "")
	m, _ = update(m, resultMsg{text: text})
	for _, width := range []int{114, 40, 80, 160} {
		m, _ = update(m, tea.WindowSizeMsg{Width: width, Height: 24})
		wrapped := m.view.GetContent()
		if strings.Join(strings.Fields(wrapped), " ") != strings.Join(strings.Fields(text), " ") {
			t.Fatalf("words split at width %d: %q", width, wrapped)
		}
		if !strings.Contains(wrapped, "\n\n") {
			t.Fatal("paragraph break lost")
		}
		for _, line := range strings.Split(wrapped, "\n") {
			if line != strings.TrimSpace(line) {
				t.Fatalf("padding leaked into wrapped line: %q", line)
			}
			if ansi.StringWidth(line) > width-4 {
				t.Fatalf("line exceeds viewport: %q", line)
			}
		}
		if m.Text() != text {
			t.Fatal("display wrapping changed output")
		}
	}
	var copied string
	m.copyText = func(s string) error { copied = s; return nil }
	m, cmd := update(m, key('c'))
	if m.status == "Copied to clipboard" {
		t.Fatal("success shown before clipboard write")
	}
	m, _ = update(m, cmd())
	if copied != text {
		t.Fatalf("clipboard contains display breaks or padding: %q", copied)
	}
	if m.status != "Copied to clipboard" || m.err != nil {
		t.Fatalf("copy failed: %v", m.err)
	}
	path := filepath.Join(t.TempDir(), "unwrapped.txt")
	m, _ = update(m, key('s'))
	m.path.SetValue(path)
	m, cmd = update(m, key(tea.KeyEnter))
	m, _ = update(m, cmd())
	b, err := os.ReadFile(path)
	if err != nil || string(b) != text+"\n" {
		t.Fatalf("saved output changed: %q, %v", b, err)
	}
}

func TestCopyFailureAndPreviewFitsTerminal(t *testing.T) {
	m := New("text", nil, "")
	m, _ = update(m, resultMsg{text: strings.Repeat("A full paragraph with several words.\n\n", 50)})
	for _, width := range []int{40, 80, 160} {
		m, _ = update(m, tea.WindowSizeMsg{Width: width, Height: 24})
		m.copyText = func(string) error { return fmt.Errorf("clipboard unavailable") }
		m, cmd := update(m, key('c'))
		m, _ = update(m, cmd())
		if m.err == nil || m.status != "" {
			t.Fatal("copy failure reported as success")
		}
		if m.screen != preview {
			t.Fatal("copy failure left preview")
		}
		for _, message := range []tea.Msg{copiedMsg{}, savedMsg{path: "/tmp/clean.txt"}} {
			m, _ = update(m, message)
			m.view.GotoBottom()
			view := m.View().Content
			if lipgloss.Height(view) > m.height {
				t.Fatalf("view overflows height at width %d: %d > %d", width, lipgloss.Height(view), m.height)
			}
			if lipgloss.Width(view) > m.width {
				t.Fatalf("view overflows width: %d > %d", lipgloss.Width(view), m.width)
			}
			if !strings.Contains(ansi.Strip(view), "REMOVE SHIT") {
				t.Fatal("header lost")
			}
		}
	}
}

func TestPreviewWrapsUnicodeAndLongURLs(t *testing.T) {
	text := "中文段落 👩‍💻 café\n\nhttps://example.com/" + strings.Repeat("x", 120)
	m := New("text", nil, "")
	m, _ = update(m, tea.WindowSizeMsg{Width: 40, Height: 24})
	m, _ = update(m, resultMsg{text: text})
	for _, line := range strings.Split(m.view.GetContent(), "\n") {
		if ansi.StringWidth(line) > m.view.Width() {
			t.Fatalf("line too wide: %q", line)
		}
	}
	if !strings.Contains(m.view.GetContent(), "👩‍💻") || m.Text() != text {
		t.Fatal("Unicode or original text changed")
	}
}
