// Package clean extracts readable text without terminal or document formatting.
package clean

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/unicode/norm"
)

const MaxBytes = 32 << 20

var Formats = []string{"auto", "text", "markdown", "html", "rtf", "docx"}
var htmlTag = regexp.MustCompile(`(?i)^<(?:!doctype\s+html|/?(?:html|body|p|div|span|h[1-6]|ul|ol|li|a|strong|em|b|i|br|table|pre|script|style)(?:\s[^>]*|/?)>)`)

func ValidFormat(format string) bool {
	for _, f := range Formats {
		if f == format {
			return true
		}
	}
	return false
}

func Detect(data []byte, name string) string {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) || strings.EqualFold(filepath.Ext(name), ".docx") {
		return "docx"
	}
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, `{\rtf`) {
		return "rtf"
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".text":
		return "text"
	case ".md", ".markdown", ".mdown":
		return "markdown"
	case ".html", ".htm":
		return "html"
	case ".rtf":
		return "rtf"
	}
	if htmlTag.MatchString(s) {
		return "html"
	}
	return "markdown"
}

func Convert(data []byte, name, format string) (string, error) {
	if len(data) > MaxBytes {
		return "", fmt.Errorf("input exceeds the 32 MiB limit")
	}
	if !ValidFormat(format) {
		return "", fmt.Errorf("unknown format %q (use %s)", format, strings.Join(Formats, ", "))
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if bytes.HasPrefix(data, []byte("%PDF-")) || bytes.HasPrefix(data, []byte{0xd0, 0xcf, 0x11, 0xe0}) {
		return "", fmt.Errorf("PDF and legacy Office files are unsupported; export as TXT, HTML, RTF or DOCX")
	}
	if format == "auto" {
		format = Detect(data, name)
	}
	var s string
	var err error
	if format == "docx" {
		s, err = docxText(data)
	} else if format == "rtf" {
		s, err = rtfText(data)
	} else {
		data, err = decodeText(data)
		if err != nil {
			return "", err
		}
		s = ansi.Strip(string(data))
		switch format {
		case "html":
			s, err = htmlText(s)
		case "markdown":
			var rendered bytes.Buffer
			md := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(html.WithUnsafe(), html.WithHardWraps()))
			if err = md.Convert([]byte(s), &rendered); err == nil {
				s, err = htmlText(rendered.String())
			}
		}
	}
	if err != nil {
		return "", err
	}
	return Normalize(s), nil
}

func decodeText(data []byte) ([]byte, error) {
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) || bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		var err error
		data, err = textunicode.UTF16(textunicode.LittleEndian, textunicode.ExpectBOM).NewDecoder().Bytes(data)
		if err != nil {
			return nil, fmt.Errorf("decode UTF-16: %w", err)
		}
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("input is not UTF-8 or BOM-marked UTF-16; export the file as UTF-8 text")
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return nil, fmt.Errorf("binary input is unsupported; use TXT, Markdown, HTML, RTF or DOCX")
	}
	return data, nil
}

// Normalize preserves paragraphs and meaningful Unicode (including emoji joiners).
func Normalize(s string) string {
	s = ansi.Strip(s)
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = strings.Map(func(r rune) rune {
		switch r {
		case '\u2028', '\u2029':
			return '\n'
		case '\ufeff', '\u200b', '\u2060', '\u00ad', '\u200e', '\u200f', '\u061c':
			return -1
		}
		if r >= '\u202a' && r <= '\u202e' || r >= '\u2066' && r <= '\u2069' {
			return -1
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
	var out []string
	blank := false
	for _, line := range strings.Split(s, "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			if len(out) > 0 {
				blank = true
			}
			continue
		}
		if blank {
			out = append(out, "")
			blank = false
		}
		out = append(out, line)
	}
	return norm.NFC.String(strings.Join(out, "\n"))
}
