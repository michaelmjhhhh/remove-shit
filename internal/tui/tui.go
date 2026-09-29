package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/michaelmjhhhh/remove-shit/internal/clean"
	"github.com/michaelmjhhhh/remove-shit/internal/clipboard"
	"github.com/michaelmjhhhh/remove-shit/internal/fileio"
)

type screen int

const (
	home screen = iota
	paste
	file
	working
	preview
	save
)

type resultMsg struct {
	text, name string
	err        error
}
type savedMsg struct {
	path string
	err  error
}
type copiedMsg struct{ err error }

type Model struct {
	screen, previous                    screen
	selection, width, height            int
	area                                textarea.Model
	path                                textinput.Model
	view                                viewport.Model
	format, cleaned, sourceName, status string
	err                                 error
	Print                               bool
	initial                             []byte
	initialName                         string
	copyText                            func(string) error
}

var accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#7DD3FC")).Bold(true)
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
var bad = lipgloss.NewStyle().Foreground(lipgloss.Color("#FCA5A5"))

func New(format string, initial []byte, name string) Model {
	a := textarea.New()
	a.Placeholder = "Paste your text here…"
	a.Prompt = "│ "
	a.ShowLineNumbers = false
	a.CharLimit = 0
	a.MaxHeight = 0
	a.MaxWidth = 0
	a.SetVirtualCursor(true)
	p := textinput.New()
	p.CharLimit = 0
	p.SetVirtualCursor(true)
	v := viewport.New()
	// Pre-wrap at word boundaries; viewport soft wrapping splits words.
	v.SoftWrap = false
	m := Model{area: a, path: p, view: v, format: format, width: 80, height: 24, initial: initial, initialName: name, copyText: clipboard.Write}
	if initial != nil {
		m.screen = working
		m.previous = home
	}
	m.resize()
	return m
}

func (m Model) Init() tea.Cmd {
	if m.initial != nil {
		return convert(m.initial, m.initialName, m.format)
	}
	return nil
}

func convert(data []byte, name, format string) tea.Cmd {
	return func() tea.Msg { text, err := clean.Convert(data, name, format); return resultMsg{text, name, err} }
}

func (m *Model) resize() {
	w := max(1, m.width-4)
	h := max(3, m.height-12)
	m.area.SetWidth(w)
	m.area.SetHeight(h)
	m.path.SetWidth(max(1, w-4))
	oldWidth := m.view.Width()
	m.view.SetWidth(w)
	if oldWidth != w {
		m.wrapPreview()
	}
	header, footer := m.previewParts()
	// Two blank separators and the frame's top/bottom padding.
	m.view.SetHeight(max(1, m.height-lipgloss.Height(header)-lipgloss.Height(footer)-4))
	m.view.SetYOffset(m.view.YOffset())
}

func (m *Model) wrapPreview() {
	lines := strings.Split(ansi.Wrap(m.cleaned, m.view.Width(), ""), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	m.view.SetContent(strings.Join(lines, "\n"))
}

func (m Model) Text() string { return m.cleaned }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	updated := next.(Model)
	updated.resize()
	return updated, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case resultMsg:
		m.initial = nil
		if msg.err != nil {
			m.err = msg.err
			m.screen = m.previous
			return m, nil
		}
		m.cleaned, m.sourceName, m.err, m.status = msg.text, msg.name, nil, ""
		m.screen = preview
		if m.cleaned == "" {
			m.status = "No visible text remains"
		}
		m.wrapPreview()
		m.view.GotoTop()
		return m, nil
	case savedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.screen = save
			return m, nil
		}
		m.screen, m.err, m.status = preview, nil, "Saved: "+msg.path
		return m, nil
	case copiedMsg:
		m.status, m.err = "", nil
		if msg.err != nil {
			m.err = fmt.Errorf("copy failed: %w", msg.err)
		} else {
			m.status = "Copied to clipboard"
		}
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if key == "esc" && m.screen != working {
			m.err = nil
			m.status = ""
			switch m.screen {
			case save:
				m.screen = preview
			case preview:
				m.screen = m.previous
			default:
				m.screen = home
			}
			if m.screen == paste {
				return m, m.area.Focus()
			}
			if m.screen == file {
				if m.sourceName != "" {
					m.path.SetValue(m.sourceName)
				}
				return m, m.path.Focus()
			}
			return m, nil
		}
		switch m.screen {
		case home:
			switch key {
			case "q":
				return m, tea.Quit
			case "up", "down", "tab", "j", "k":
				m.selection = 1 - m.selection
			case "1", "2", "enter":
				if key == "1" {
					m.selection = 0
				}
				if key == "2" {
					m.selection = 1
				}
				m.err, m.status = nil, ""
				if m.selection == 0 {
					m.screen = paste
					return m, m.area.Focus()
				}
				m.screen = file
				m.path.SetValue("")
				m.path.Placeholder = "/path/to/document.docx"
				return m, m.path.Focus()
			}
		case paste, file:
			if key == "ctrl+f" {
				// DOCX is binary and only meaningful for a file input.
				formats := clean.Formats
				if m.screen == paste {
					formats = formats[:len(formats)-1]
				}
				for i, f := range formats {
					if f == m.format {
						m.format = formats[(i+1)%len(formats)]
						break
					}
				}
				if m.screen == paste && m.format == "docx" {
					m.format = "auto"
				}
				return m, nil
			}
			if m.screen == paste && key == "ctrl+d" {
				m.previous, m.screen, m.err = paste, working, nil
				return m, convert([]byte(m.area.Value()), "", m.format)
			}
			if m.screen == file && key == "enter" {
				path, format := fileio.Expand(m.path.Value()), m.format
				if path == "" {
					m.err = fmt.Errorf("enter a file path")
					return m, nil
				}
				m.previous, m.screen, m.err = file, working, nil
				return m, func() tea.Msg {
					data, err := fileio.ReadFile(path)
					if err != nil {
						return resultMsg{err: err}
					}
					text, err := clean.Convert(data, path, format)
					return resultMsg{text, path, err}
				}
			}
		case preview:
			switch key {
			case "q":
				return m, tea.Quit
			case "p":
				m.Print = true
				return m, tea.Quit
			case "c":
				m.status, m.err = "Copying…", nil
				text, write := m.cleaned, m.copyText
				return m, func() tea.Msg { return copiedMsg{write(text)} }
			case "n":
				m.screen = home
				m.status = ""
				return m, nil
			case "s":
				m.screen = save
				m.status = ""
				m.err = nil
				name := "cleaned.txt"
				if m.sourceName != "" {
					name = strings.TrimSuffix(m.sourceName, filepath.Ext(m.sourceName)) + ".clean.txt"
				}
				m.path.SetValue(name)
				m.path.Placeholder = "cleaned.txt"
				return m, m.path.Focus()
			}
		case save:
			if key == "enter" {
				path := fileio.Expand(m.path.Value())
				if path == "" {
					m.err = fmt.Errorf("enter an output path")
					return m, nil
				}
				if filepath.Ext(path) == "" {
					path += ".txt"
				}
				m.screen = working
				m.err = nil
				text := m.cleaned
				return m, func() tea.Msg { return savedMsg{path, fileio.Save(path, text)} }
			}
		}
	}
	var cmd tea.Cmd
	switch m.screen {
	case paste:
		m.area, cmd = m.area.Update(msg)
	case file, save:
		m.path, cmd = m.path.Update(msg)
	case preview:
		m.view, cmd = m.view.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if m.width < 24 || m.height < 12 {
		v := tea.NewView(ansi.Wrap("Enlarge the terminal to at least 24 × 12. Ctrl+C to quit.", max(1, m.width), ""))
		v.AltScreen = true
		return v
	}
	if m.screen == preview {
		header, footer := m.previewParts()
		if lipgloss.Height(header)+lipgloss.Height(footer)+5 > m.height {
			v := tea.NewView(ansi.Wrap("Enlarge the terminal to view the preview. Ctrl+C to quit.", max(1, m.width), ""))
			v.AltScreen = true
			return v
		}
		v := tea.NewView(lipgloss.NewStyle().Padding(1, 2).Render(header + "\n\n" + m.view.View() + "\n\n" + footer))
		v.AltScreen = true
		return v
	}
	var body, help string
	title := accent.Render("REMOVE SHIT") + "  " + muted.Render("→ plain text")
	switch m.screen {
	case home:
		items := []string{"Paste text", "Open a file"}
		body = "Keep the words. Clear the formatting.\n\n"
		for i, item := range items {
			if i == m.selection {
				body += accent.Render("› " + item)
			} else {
				body += "  " + item
			}
			body += "\n"
		}
		body += "\n" + muted.Render("TXT · Markdown · HTML · RTF · DOCX\nAll processing stays on your device")
		help = "↑/↓ select · enter open · 1/2 shortcut · q quit"
	case paste:
		body = "Paste text  ·  format: " + m.format + "\n\n" + m.area.View()
		help = "ctrl+d clean · ctrl+f format · enter newline · esc back"
	case file:
		body = "Open a file  ·  format: " + m.format + "\n\n" + m.path.View() + "\n\nPaste a file path; spaces and ~/ are supported."
		help = "enter load & clean · ctrl+f format · esc back"
	case working:
		body = "Processing locally…"
		help = "ctrl+c quit"
	case save:
		body = "Save plain text\n\n" + m.path.View() + "\n\nUTF-8 · .txt · existing files are never overwritten"
		help = "enter save · esc preview"
	}
	if m.err != nil {
		body += "\n\n" + bad.Render(clean.Normalize(m.err.Error()))
	}
	if m.status != "" {
		body += "\n\n" + accent.Render(clean.Normalize(m.status))
	}
	content := title + "\n\n" + body + "\n\n" + muted.Render(help)
	v := tea.NewView(lipgloss.NewStyle().Padding(1, 2).Width(max(20, m.width-4)).Render(content))
	v.AltScreen = true
	return v
}

func (m Model) previewParts() (string, string) {
	w := max(1, m.width-4)
	header := accent.Render("REMOVE SHIT") + "  " + muted.Render("→ plain text") + "\n\n" +
		fmt.Sprintf("Clean text  ·  %d characters  ·  %.0f%%", utf8.RuneCountInString(m.cleaned), m.view.ScrollPercent()*100)
	footer := muted.Render("s save .txt · c copy · p print & exit\n↑/↓ scroll · n new · esc back · q quit")
	if m.status != "" {
		footer = accent.Render(clean.Normalize(m.status)) + "\n\n" + footer
	}
	if m.err != nil {
		footer = bad.Render(clean.Normalize(m.err.Error())) + "\n\n" + footer
	}
	return ansi.Wrap(header, w, ""), ansi.Wrap(footer, w, "")
}
