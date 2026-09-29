# remove-shit

[![Release](https://img.shields.io/github/v/release/michaelmjhhhh/remove-shit?style=flat-square)](https://github.com/michaelmjhhhh/remove-shit/releases/latest)
[![CI](https://github.com/michaelmjhhhh/remove-shit/actions/workflows/ci.yml/badge.svg)](https://github.com/michaelmjhhhh/remove-shit/actions/workflows/ci.yml)
[![macOS](https://img.shields.io/badge/macOS-Apple_Silicon_%26_Intel-black?style=flat-square&logo=apple)](#install)

A macOS CLI that converts pasted text and TXT, Markdown, HTML, RTF, and DOCX files to plain text. Built with Go and Charmbracelet. All processing is local.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/michaelmjhhhh/remove-shit/main/install.sh | sh
```

Installs to `/usr/local/bin` after verifying the release checksum. Supports Apple Silicon and Intel; Go is not required. Run again to update.

For a custom directory, append `INSTALL_DIR="$HOME/.local/bin"` before `sh` and add that directory to your `PATH`. To uninstall the default installation:

```sh
sudo rm /usr/local/bin/remove-shit
```

## Usage

```sh
remove-shit                              # Interactive mode
remove-shit -o clean.txt document.docx    # Save to a new file
remove-shit -i notes.rtf                  # Preview a file
cat page.html | remove-shit > clean.txt   # Pipe input
remove-shit --format markdown notes.txt  # Override input format
remove-shit --help
```

In interactive mode, choose **Paste text** or **Open a file**. Paste with **Cmd+V**, then press **Ctrl+D** to clean. For files, enter the path and press **Enter**.

| Key | Action |
| --- | --- |
| `C` | Copy |
| `S` | Save |
| `P` | Print and exit |
| `Ctrl+F` | Change input format |
| `↑` / `↓` | Scroll |
| `N` | New conversion |
| `Esc` | Back |
| `Ctrl+C` | Quit |

Flags must precede the filename. Output preserves paragraphs, Unicode, and link addresses, using UTF-8 without a BOM. Preview wrapping is not included in copied or saved text. Existing files are never overwritten.

## Limits

- Text input: UTF-8 or BOM-marked UTF-16, up to 32 MiB.
- TXT mode preserves literal `#` and `**`; choose Markdown mode to remove markup.
- DOCX extracts the main document only. RTF supports common body-text formatting.
- Table alignment and code indentation are removed. HTML scripts and CSS are not evaluated.
- PDF, legacy `.doc`, and images are unsupported.

## Development

Requires Go 1.26.1 or later.

```sh
go build -o remove-shit .
go test -race ./...
go vet ./...
python3 scripts/test-install.py
```

Build macOS release archives with `sh scripts/build-release.sh v0.1.0`.
