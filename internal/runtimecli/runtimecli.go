package runtimecli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xwhiz/compiler/internal/vm"
)

const (
	toolName    = "crun"
	exitOK      = 0
	exitUsage   = 2
	exitFailure = 1
)

type options struct {
	showHelp bool
	input    string
}

func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args)
	if err != nil {
		return printError(stderr, err)
	}
	if opts.showHelp {
		printUsage(stdout)
		return exitOK
	}
	data, err := os.ReadFile(opts.input)
	if err != nil {
		return printError(stderr, fmt.Errorf("read %s: %w", opts.input, err))
	}
	program, err := vm.ParseText(string(data))
	if err != nil {
		return printError(stderr, err)
	}
	_, err = vm.Execute(program, stdout)
	if err != nil {
		return printError(stderr, err)
	}
	return exitOK
}

func parseOptions(args []string) (options, error) {
	fs := flag.NewFlagSet(toolName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var help bool
	fs.BoolVar(&help, "help", false, "show usage")
	fs.BoolVar(&help, "h", false, "show usage")
	if err := fs.Parse(args); err != nil {
		return options{}, fmt.Errorf("parse flags: %w", err)
	}
	if help {
		return options{showHelp: true}, nil
	}
	positionals := fs.Args()
	if len(positionals) == 0 {
		return options{}, fmt.Errorf("missing object file")
	}
	if len(positionals) > 1 {
		return options{}, fmt.Errorf("expected one object file, got %d", len(positionals))
	}
	return options{input: positionals[0]}, nil
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, strings.TrimSpace(`Usage: crun <file.vmo>

crun loads VM object code emitted by cforge and executes it.
`)+"\n")
}

func printError(w io.Writer, err error) int {
	_, _ = fmt.Fprintf(w, "%s: %v\n", toolName, err)
	if strings.Contains(err.Error(), "missing object file") || strings.Contains(err.Error(), "expected one object file") || strings.Contains(err.Error(), "parse flags") {
		return exitUsage
	}
	return exitFailure
}
