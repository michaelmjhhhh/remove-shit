package clean

import (
	"strings"

	"golang.org/x/net/html"
)

func htmlText(source string) (string, error) {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, pre bool) {
		if n.Type == html.TextNode {
			s := n.Data
			if !pre {
				// Collapse HTML source whitespace without joining adjacent inline words.
				s = strings.Map(func(r rune) rune {
					if r == '\n' || r == '\r' || r == '\t' {
						return ' '
					}
					return r
				}, s)
			}
			b.WriteString(s)
			return
		}
		tag := n.Data
		if n.Type == html.ElementNode {
			switch tag {
			case "script", "style", "head", "template", "noscript":
				return
			}
			for _, a := range n.Attr {
				if a.Key == "hidden" || a.Key == "aria-hidden" && a.Val == "true" {
					return
				}
			}
			if tag == "br" {
				b.WriteByte('\n')
				return
			}
			if tag == "img" {
				for _, a := range n.Attr {
					if a.Key == "alt" {
						b.WriteString(a.Val)
					}
				}
				return
			}
			if tag == "input" {
				return
			}
		}
		block := isBlock(tag)
		if block {
			b.WriteString("\n\n")
		}
		start := b.Len()
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, pre || tag == "pre")
		}
		if tag == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" && a.Val != "" && !strings.HasPrefix(a.Val, "#") {
					label := strings.TrimSpace(b.String()[start:])
					if label != a.Val {
						b.WriteString(" (" + a.Val + ")")
					}
				}
			}
		}
		if tag == "td" || tag == "th" {
			b.WriteString("\t")
		}
		if block {
			b.WriteString("\n\n")
		}
	}
	walk(doc, false)
	return b.String(), nil
}

func isBlock(tag string) bool {
	switch tag {
	case "p", "div", "section", "article", "header", "footer", "main", "aside", "nav", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre", "ul", "ol", "li", "dl", "dt", "dd", "table", "tr", "hr":
		return true
	}
	return false
}
