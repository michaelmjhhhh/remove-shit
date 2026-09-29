package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/michaelmjhhhh/remove-shit/internal/clean"
	"github.com/michaelmjhhhh/remove-shit/internal/fileio"
	"github.com/michaelmjhhhh/remove-shit/internal/tui"
)

var version = "dev"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "remove-shit:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin *os.File, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("remove-shit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	format := fs.String("format", "auto", "input format: auto, text, markdown, html, rtf, docx")
	output := fs.String("output", "", "save UTF-8 text to a new file (never overwrite)")
	fs.StringVar(output, "o", "", "alias for --output")
	interactive := fs.Bool("interactive", false, "preview the input in the interactive app")
	fs.BoolVar(interactive, "i", false, "alias for --interactive")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "remove-shit — turn formatted content into readable plain text\n\nUsage:\n  remove-shit                         Interactive paste / file input\n  remove-shit [flags] FILE            Clean a file to stdout\n  cat FILE | remove-shit [flags]      Clean stdin to stdout\n  remove-shit --interactive FILE      Preview before saving\n\nFlags must precede FILE. Use - as FILE to read stdin.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if *showVersion {
		_, err := fmt.Fprintln(stdout, "remove-shit", version)
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("expected one input file; use --help for usage")
	}
	if !clean.ValidFormat(*format) {
		return fmt.Errorf("unknown input format %q", *format)
	}
	if *interactive && *output != "" {
		return fmt.Errorf("--interactive and --output cannot be combined; save from the preview with s")
	}
	var data []byte
	var err error
	name := fs.Arg(0)
	hasInput := name != "" || !term.IsTerminal(stdin.Fd())
	if hasInput {
		if name == "" || name == "-" {
			data, err = fileio.Read(stdin)
			name = ""
		} else {
			data, err = fileio.ReadFile(name)
		}
		if err != nil {
			return err
		}
		// An empty slice still signals an explicitly supplied input to the TUI.
		if data == nil {
			data = []byte{}
		}
	}
	if *interactive || !hasInput {
		if *output != "" {
			return fmt.Errorf("--output needs a file or piped input")
		}
		m := tui.New(*format, data, name)
		result, err := tea.NewProgram(m, tea.WithOutput(stderr)).Run()
		if err != nil {
			return err
		}
		final := result.(tui.Model)
		if final.Print {
			_, err = io.WriteString(stdout, fileio.Output(final.Text()))
		}
		return err
	}
	text, err := clean.Convert(data, name, *format)
	if err != nil {
		return err
	}
	if *output != "" {
		if err := fileio.Save(*output, text); err != nil {
			return err
		}
		_, err = fmt.Fprintln(stderr, "Saved:", *output)
		return err
	}
	_, err = io.WriteString(stdout, fileio.Output(text))
	return err
}
