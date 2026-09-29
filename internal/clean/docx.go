package clean

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

func docxText(data []byte) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("invalid DOCX: %w", err)
	}
	readPart := func(name string) ([]byte, error) {
		for _, f := range z.File {
			if f.Name != name {
				continue
			}
			if f.UncompressedSize64 > MaxBytes {
				return nil, fmt.Errorf("DOCX part exceeds 32 MiB")
			}
			r, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer r.Close()
			b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
			if len(b) > MaxBytes {
				return nil, fmt.Errorf("DOCX part exceeds 32 MiB")
			}
			return b, err
		}
		return nil, nil
	}
	links := make(map[string]string)
	rels, err := readPart("word/_rels/document.xml.rels")
	if err != nil {
		return "", err
	}
	if len(rels) > 0 {
		var relationships struct {
			Items []struct {
				ID     string `xml:"Id,attr"`
				Target string `xml:"Target,attr"`
				Type   string `xml:"Type,attr"`
			} `xml:"Relationship"`
		}
		if err := xml.Unmarshal(rels, &relationships); err != nil {
			return "", fmt.Errorf("invalid DOCX relationships: %w", err)
		}
		for _, r := range relationships.Items {
			if strings.HasSuffix(r.Type, "/hyperlink") {
				links[r.ID] = r.Target
			}
		}
	}
	document, err := readPart("word/document.xml")
	if err != nil {
		return "", err
	}
	if len(document) == 0 {
		return "", fmt.Errorf("file is not a DOCX document (missing word/document.xml)")
	}
	d := xml.NewDecoder(bytes.NewReader(document))
	var b strings.Builder
	var textDepth, skipDepth, tableDepth int
	var href string
	var linkStart int
	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("invalid DOCX document: %w", err)
		}
		switch v := t.(type) {
		case xml.StartElement:
			if skipDepth > 0 {
				skipDepth++
				continue
			}
			if v.Name.Local == "del" {
				skipDepth = 1
				continue
			}
			switch v.Name.Local {
			case "t":
				textDepth++
			case "tbl":
				tableDepth++
			case "tab":
				b.WriteByte('\t')
			case "br", "cr":
				b.WriteByte('\n')
			case "hyperlink":
				href = ""
				linkStart = b.Len()
				for _, a := range v.Attr {
					if a.Name.Local == "id" {
						href = links[a.Value]
					}
				}
			}
		case xml.EndElement:
			if skipDepth > 0 {
				skipDepth--
				continue
			}
			switch v.Name.Local {
			case "t":
				textDepth--
			case "tbl":
				tableDepth--
				b.WriteString("\n\n")
			case "p":
				if tableDepth > 0 {
					b.WriteByte(' ')
				} else {
					b.WriteString("\n\n")
				}
			case "tr":
				b.WriteByte('\n')
			case "tc":
				b.WriteByte('\t')
			case "hyperlink":
				if href != "" && strings.TrimSpace(b.String()[linkStart:]) != href {
					b.WriteString(" (" + href + ")")
				}
				href = ""
			}
		case xml.CharData:
			if textDepth > 0 && skipDepth == 0 {
				b.Write(v)
			}
		}
	}
	return b.String(), nil
}
