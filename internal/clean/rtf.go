package clean

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
)

type rtfState struct {
	skip        bool
	uc          int
	instruction bool
	result      bool
	link        string
	linkStart   int
}

var hyperlink = regexp.MustCompile(`(?i)HYPERLINK\s+"([^"]+)"`)

// rtfText handles grouped RTF controls, code pages, Unicode fallback characters,
// and field hyperlinks. Embedded objects and non-body destinations are skipped.
func rtfText(data []byte) (string, error) {
	data = bytes.TrimSpace(data)
	if !bytes.HasPrefix(data, []byte(`{\rtf`)) {
		return "", fmt.Errorf("invalid RTF header")
	}
	state := rtfState{uc: 1}
	var stack []rtfState
	var out []rune
	var raw []byte
	var instruction strings.Builder
	var enc encoding.Encoding = charmap.Windows1252
	fallback := 0
	pendingLink := ""
	flush := func() {
		if len(raw) == 0 {
			return
		}
		decoded, _ := enc.NewDecoder().Bytes(raw)
		if state.instruction {
			instruction.Write(decoded)
		} else if !state.skip {
			out = append(out, []rune(string(decoded))...)
		}
		raw = raw[:0]
	}
	emit := func(r rune) {
		flush()
		if state.instruction {
			instruction.WriteRune(r)
		} else if !state.skip {
			out = append(out, r)
		}
	}
	for i := 0; i < len(data); {
		c := data[i]
		i++
		switch c {
		case '{':
			flush()
			stack = append(stack, state)
		case '}':
			flush()
			if len(stack) == 0 {
				return "", fmt.Errorf("unbalanced RTF groups")
			}
			parent := stack[len(stack)-1]
			if state.result && !parent.result && state.link != "" && !state.skip {
				if strings.TrimSpace(string(out[state.linkStart:])) != state.link {
					out = append(out, []rune(" ("+state.link+")")...)
				}
			}
			if state.instruction && !parent.instruction {
				if match := hyperlink.FindStringSubmatch(instruction.String()); len(match) > 1 {
					pendingLink = match[1]
				}
				instruction.Reset()
			}
			state = parent
			stack = stack[:len(stack)-1]
		case '\r', '\n': // Source line breaks aren't visible RTF content.
		case '\\':
			if i >= len(data) {
				return "", fmt.Errorf("truncated RTF escape")
			}
			next := data[i]
			i++
			if next == '\\' || next == '{' || next == '}' {
				if fallback > 0 {
					fallback--
				} else {
					raw = append(raw, next)
				}
				continue
			}
			if next == '\'' {
				if i+2 > len(data) {
					return "", fmt.Errorf("truncated RTF hex escape")
				}
				n, err := strconv.ParseUint(string(data[i:i+2]), 16, 8)
				if err != nil {
					return "", fmt.Errorf("invalid RTF hex escape")
				}
				i += 2
				if fallback > 0 {
					fallback--
				} else {
					raw = append(raw, byte(n))
				}
				continue
			}
			flush()
			if next == '*' {
				state.skip = true
				continue
			}
			if next == '~' {
				if fallback > 0 {
					fallback--
				} else {
					emit(' ')
				}
				continue
			}
			if next == '_' {
				if fallback > 0 {
					fallback--
				} else {
					emit('-')
				}
				continue
			}
			if next == '-' {
				if fallback > 0 {
					fallback--
				}
				continue
			}
			if !asciiLetter(next) {
				continue
			}
			start := i - 1
			for i < len(data) && asciiLetter(data[i]) {
				i++
			}
			word := string(data[start:i])
			start = i
			if i < len(data) && data[i] == '-' {
				i++
			}
			for i < len(data) && data[i] >= '0' && data[i] <= '9' {
				i++
			}
			n := 0
			if i > start {
				var err error
				n, err = strconv.Atoi(string(data[start:i]))
				if err != nil {
					return "", fmt.Errorf("invalid RTF parameter")
				}
			}
			if i < len(data) && data[i] == ' ' {
				i++
			}
			switch word {
			case "fonttbl", "colortbl", "stylesheet", "info", "pict", "object", "header", "headerl", "headerr", "footer", "footerl", "footerr", "filetbl", "listtable", "listoverridetable", "generator", "datastore", "themedata", "colorschememapping":
				state.skip = true
			case "fldinst":
				state.instruction = true
				instruction.Reset()
			case "fldrslt":
				state.result, state.link, state.linkStart = true, pendingLink, len(out)
				pendingLink = ""
			case "uc":
				if n < 0 || n > 32 {
					return "", fmt.Errorf("invalid RTF Unicode fallback length")
				}
				state.uc = n
			case "u":
				emit(rune(uint16(n)))
				fallback = state.uc
			case "ansicpg":
				var err error
				enc, err = ianaindex.MIME.Encoding(fmt.Sprintf("windows-%d", n))
				if err != nil || enc == nil {
					return "", fmt.Errorf("unsupported RTF code page %d; export as UTF-8 text", n)
				}
			case "bin":
				if n < 0 || n > len(data)-i {
					return "", fmt.Errorf("invalid RTF binary length")
				}
				i += n
			case "par", "line", "row":
				emit('\n')
				if word == "par" {
					emit('\n')
				}
			case "tab", "cell":
				emit('\t')
			case "emdash":
				emit('—')
			case "endash":
				emit('–')
			case "lquote", "rquote":
				emit('\'')
			case "ldblquote", "rdblquote":
				emit('"')
			case "bullet":
				emit('•')
			}
		default:
			if fallback > 0 {
				fallback--
			} else {
				raw = append(raw, c)
			}
		}
	}
	flush()
	if len(stack) != 0 {
		return "", fmt.Errorf("unbalanced RTF groups")
	}
	if pendingLink != "" {
		out = append(out, []rune(" ("+pendingLink+")")...)
	}
	// RTF represents supplementary Unicode as paired signed UTF-16 controls.
	var result []rune
	for i := 0; i < len(out); i++ {
		r := out[i]
		if r >= 0xd800 && r <= 0xdbff && i+1 < len(out) && out[i+1] >= 0xdc00 && out[i+1] <= 0xdfff {
			r = utf16.DecodeRune(r, out[i+1])
			i++
		}
		if utf16.IsSurrogate(r) {
			r = '\ufffd'
		}
		result = append(result, r)
	}
	return string(result), nil
}

func asciiLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }
