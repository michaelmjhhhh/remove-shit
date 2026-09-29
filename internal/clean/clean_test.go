package clean

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	textunicode "golang.org/x/text/encoding/unicode"
)

func TestConvert(t *testing.T) {
	tests := []struct{ name, input, filename, format, want string }{
		{"plain Unicode", "\ufeff  中文\u00a0  内容\r\n\r\n\r\n\t👩‍💻 café\u200b\x1b[31m!\x1b[0m\x00", "", "text", ""},
		{"paragraphs", "\ufeff  中文\u00a0  内容\r\n\r\n\r\n\t👩‍💻 cafe\u0301\u200b\x1b[31m!\x1b[0m", "", "text", "中文 内容\n\n👩‍💻 café!"},
		{"Markdown", "# 标题\n\n**粗体** 和 _斜体_ ~~删除线~~\n\n- 第一项\n- 第二项\n\n[官网](https://example.com)\n\n```go\nfmt.Println(\"hi\")\n```", "x.md", "auto", "标题\n\n粗体 和 斜体 删除线\n\n第一项\n\n第二项\n\n官网 (https://example.com)\n\nfmt.Println(\"hi\")"},
		{"Markdown line breaks", "first\nsecond\n\nthird", "", "markdown", "first\nsecond\n\nthird"},
		{"BOM Markdown", "\ufeff# 标题\n\n**正文**", "x.md", "auto", "标题\n\n正文"},
		{"mixed Markdown and HTML", "# Title\n\nUse <b>bold</b> and **strong**", "", "auto", "Title\n\nUse bold and strong"},
		{"HTML", `<html><head><title>Private</title><style>p{color:red}</style></head><body><h1>你好</h1><p>Hello <strong>world</strong>&nbsp;&amp; friends<br>Next line</p><script>alert(1)</script><p hidden>hidden</p><a href="https://example.com">Website</a><img alt="猫" src="cat.jpg"></body></html>`, "page.html", "auto", "你好\n\nHello world & friends\nNext line\n\nWebsite (https://example.com)猫"},
		{"HTML table", `<table><tr><th>Name</th><th>Value</th></tr><tr><td>A</td><td>1</td></tr></table>`, "", "html", "Name Value\n\nA 1"},
		{"text is literal", "# title\n**keep this**\n2 * 3 < 9\nfoo_bar_baz", "notes.txt", "auto", "# title\n**keep this**\n2 * 3 < 9\nfoo_bar_baz"},
		{"OSC hyperlink", "\x1b]8;;https://example.com\x1b\\hello\x1b]8;;\x1b\\", "", "text", "hello"},
		{"RTF", `{\rtf1\ansi\ansicpg1252{\fonttbl{\f0 Arial;}}\b Bold\b0  and caf\'e9\par \uc1\u20320?\u22909? \u-10179?\u-8704?}`, "a.rtf", "auto", "Bold and café\n\n你好 😀"},
		{"RTF groups", `{\rtf1 A {\b bold {\i inner}} end {\*\unknown omitted}{\pict\bin4 abcd}\par next}`, "", "rtf", "A bold inner end\n\nnext"},
		{"RTF links", `{\rtf1 {\field{\*\fldinst HYPERLINK "https://one.test"}{\fldrslt One}} and {\field{\*\fldinst HYPERLINK "https://two.test"}{\fldrslt Two}}.}`, "", "rtf", "One (https://one.test) and Two (https://two.test)."},
		{"HTML entities in code", "`<b>literal</b>` and &amp;", "a.md", "auto", "<b>literal</b> and &"},
		{"empty", " \n\t", "", "auto", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert([]byte(tt.input), tt.filename, tt.format)
			if tt.name == "plain Unicode" {
				if err == nil {
					t.Fatal("expected binary input error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestDOCX(t *testing.T) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	parts := map[string]string{
		"word/document.xml":            `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>你好 </w:t></w:r><w:hyperlink r:id="rId1"><w:r><w:t>网站</w:t></w:r></w:hyperlink><w:del><w:r><w:delText>Deleted</w:delText></w:r></w:del></w:p><w:p><w:r><w:t>下一段</w:t><w:br/><w:t>下一行</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`,
		"word/_rels/document.xml.rels": `<Relationships><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com"/></Relationships>`,
	}
	for name, content := range parts {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := Convert(b.Bytes(), "test.docx", "auto")
	if err != nil {
		t.Fatal(err)
	}
	want := "你好 网站 (https://example.com)\n\n下一段\n下一行\n\nA B"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUTF16(t *testing.T) {
	for _, endian := range []textunicode.Endianness{textunicode.LittleEndian, textunicode.BigEndian} {
		data, err := textunicode.UTF16(endian, textunicode.UseBOM).NewEncoder().Bytes([]byte("中文\r\n\r\nText"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Convert(data, "x.txt", "auto")
		if err != nil {
			t.Fatal(err)
		}
		if got != "中文\n\nText" {
			t.Fatalf("got %q", got)
		}
	}
}

func TestInvalidInput(t *testing.T) {
	for _, tt := range []struct{ data, format string }{
		{"bad", "unknown"}, {"\xff", "text"}, {"not a docx", "docx"}, {"not rtf", "rtf"}, {`{\rtf1 missing close`, "rtf"}, {`{\rtf1\'xz}`, "rtf"}, {`{\rtf1\bin999 x}`, "rtf"}, {strings.Repeat("x", MaxBytes+1), "text"},
	} {
		if _, err := Convert([]byte(tt.data), "", tt.format); err == nil {
			t.Fatalf("expected error for %.20q / %s", tt.data, tt.format)
		}
	}
}

func FuzzConvert(f *testing.F) {
	for _, s := range []string{"hello", "# title", "<p>中文</p>", `{\rtf1\uc1\u20320?}`, "\x1b[31mred", "PK\x03\x04"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 4096 {
			t.Skip()
		}
		_, _ = Convert(b, "", "auto")
	})
}
