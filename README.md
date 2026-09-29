<div align="center">

# remove-shit

**Keep the words. Lose the formatting.**

A small, local-first macOS CLI that turns messy content into clean plain text.

[![Release](https://img.shields.io/github/v/release/michaelmjhhhh/remove-shit?style=flat-square&color=7dd3fc)](https://github.com/michaelmjhhhh/remove-shit/releases/latest)
[![CI](https://github.com/michaelmjhhhh/remove-shit/actions/workflows/ci.yml/badge.svg)](https://github.com/michaelmjhhhh/remove-shit/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/michaelmjhhhh/remove-shit?style=flat-square&logo=go&logoColor=white)](go.mod)
[![macOS](https://img.shields.io/badge/macOS-Apple_Silicon_%26_Intel-111827?style=flat-square&logo=apple&logoColor=white)](#install)
[![Built with Charm](https://img.shields.io/badge/Built_with-Charm-ff90e8?style=flat-square)](https://charm.sh)

[Install](#install) · [Usage](#usage) · [Formats](#formats) · [Releases](https://github.com/michaelmjhhhh/remove-shit/releases)

</div>

---

Paste a wall of text. Open a document. Get just the words.

- **An interactive terminal UI** — paste, preview, copy, and save without leaving your terminal.
- **Five input formats** — TXT, Markdown, HTML, RTF, and DOCX.
- **Readable output** — keep paragraphs, Unicode, emoji, and link addresses.
- **Native Mac clipboard** — copy with `C`, powered by `pbcopy`.
- **No cloud, no account** — conversion happens entirely on your Mac.
- **Friendly to scripts** — clean stdout, pipe support, and no accidental file overwrites.

```text
Before                              After
──────────────────────────────      ──────────────────────────────
# A little less noise               A little less noise

**Keep** the words.                  Keep the words.
[Visit](https://example.com)         Visit (https://example.com)
```

## Install

**macOS only for now.** Apple Silicon and Intel builds are available. No Go installation required.

```sh
curl -fsSL https://raw.githubusercontent.com/michaelmjhhhh/remove-shit/main/install.sh | sh
```

The installer downloads the latest release, verifies its SHA-256 checksum, and installs `remove-shit` into `/usr/local/bin`. It asks for administrator access only if that directory requires it.

```sh
remove-shit --version
remove-shit
```

<details>
<summary>Custom install location, pinned version, and uninstall</summary>

Install without administrator access:

```sh
curl -fsSL https://raw.githubusercontent.com/michaelmjhhhh/remove-shit/main/install.sh \
  | INSTALL_DIR="$HOME/.local/bin" sh
```

Add that directory to your shell's `PATH` if it is not already present.

Install a specific release:

```sh
curl -fsSL https://raw.githubusercontent.com/michaelmjhhhh/remove-shit/main/install.sh \
  | REMOVE_SHIT_VERSION=v0.1.0 sh
```

Run the install command again to upgrade. To uninstall a default installation:

```sh
sudo rm /usr/local/bin/remove-shit
```

For a custom installation, remove `remove-shit` from the directory you chose.

</details>

## Usage

### Interactive

```sh
remove-shit
```

1. Choose **Paste text** or **Open a file**.
2. Paste your text and press **Ctrl+D**, or enter a file path and press **Enter**.
3. Review the result. Press **C** to copy or **S** to save a `.txt` file.

Use your terminal's usual paste shortcut, typically **Cmd+V**. Enter inserts a newline in the text editor. File paths support spaces, surrounding quotes, and `~/`; no shell escaping is needed.

| Key | Action |
| :--- | :--- |
| `Ctrl+D` | Clean pasted text |
| `Ctrl+F` | Cycle the input format |
| `↑` / `↓` | Scroll through the preview |
| `C` | Copy the clean text |
| `S` | Save a new text file |
| `P` | Print to stdout and exit |
| `N` | Start another conversion |
| `Esc` | Go back |
| `Q` / `Ctrl+C` | Quit (`Q` works on home and preview screens) |

The preview wraps at word boundaries and fits your terminal. Display wrapping and padding never enter copied or saved text.

### Files and pipes

```sh
# Clean a document into a new text file
remove-shit -o clean.txt document.docx

# Preview a file before copying or saving
remove-shit -i notes.rtf

# Send clean text to stdout
remove-shit notes.md

# Work with a pipe
cat page.html | remove-shit > clean.txt

# Treat a .txt file as Markdown
remove-shit --format markdown notes.txt

# Preserve literal #, *, and underscores
remove-shit --format text notes.txt
```

Flags go **before** the filename. `-o` means `--output`, `-i` means `--interactive`, and a filename of `-` reads stdin. The UI writes to stderr; batch stdout contains only the cleaned text. Run `remove-shit --help` for all options.

## Formats

| Input | What gets cleaned |
| :--- | :--- |
| **TXT** | ANSI styling, control characters, and excess whitespace |
| **Markdown** | Heading and list markers, emphasis, strikethrough, and code fences |
| **HTML** | Tags, scripts, styles, and metadata; HTML entities are decoded |
| **RTF** | Common rich-text controls, font tables, and embedded objects |
| **DOCX** | Main-document formatting; paragraph text, table text, and hyperlinks are extracted |

Links become `label (URL)`. Code content and image alt text remain. Repeated spaces and excess blank lines disappear, while meaningful line breaks and paragraphs stay.

Output is **UTF-8, without a BOM, using LF line endings**. Nonempty saved files end with one newline. Existing files are never overwritten.

<details>
<summary>Detection, limits, and supported content</summary>

- `auto` recognizes file extensions and detects RTF / HTML content. Other pasted content defaults to Markdown. Use `--format` or `Ctrl+F` to choose explicitly.
- TXT mode keeps literal formatting-like characters such as `**` and `#`. Choose Markdown mode if you want these interpreted as markup.
- Text input supports UTF-8 or BOM-marked UTF-16. Input is limited to 32 MiB, as is each decompressed DOCX part that is read.
- Unicode is normalized to NFC. BOMs, zero-width spaces, and directional controls are removed; joiners needed by emoji and some languages remain.
- Tables become plain text. Alignment, borders, and code indentation are removed.
- DOCX extraction covers the main document, excluding headers, footers, comments, footnotes, and deleted revision content.
- RTF supports common body-text formatting, with limited support for complex layouts and legacy encodings.
- HTML conversion does not execute JavaScript or evaluate CSS visibility. Elements marked `hidden` or `aria-hidden="true"` are skipped.
- PDF, legacy `.doc`, images, and scanned documents are not supported.
- Clipboard copying requires access to your Mac's clipboard. In a remote or headless session, save or print the result instead.

Terminal paste already removes fonts and colors in most cases. This tool additionally removes text markup and excess whitespace.

</details>

## Development

Built with Go and [Charmbracelet](https://charm.sh)'s Bubble Tea, Bubbles, and Lip Gloss v2. Requires **Go 1.26.1 or later** to build from source.

```sh
git clone https://github.com/michaelmjhhhh/remove-shit.git
cd remove-shit
go build -o remove-shit .
./remove-shit
```

```sh
go test -race ./...
go vet ./...
python3 scripts/test-install.py
```

Build macOS release archives and checksums:

```sh
sh scripts/build-release.sh v0.1.0
```

`internal/clean` handles extraction, `internal/tui` handles the interface, and `internal/fileio` handles file operations. Release artifacts and local text exports are excluded from Git.

---

Found something that should be cleaner? [Open an issue](https://github.com/michaelmjhhhh/remove-shit/issues).
