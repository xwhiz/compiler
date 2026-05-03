package driver

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/lexer"
	"github.com/xwhiz/compiler/internal/parser"
)

type mode string

const (
	modeRun     mode = "run"
	modeTokens  mode = "tokens"
	modeAST     mode = "ast"
	modeSema    mode = "sema"
	modeIR      mode = "ir"
	modeCodegen mode = "codegen"
	toolName         = "mycc"
	exitOK           = 0
	exitUsage        = 2
	exitFailure      = 1
)

type options struct {
	showHelp bool
	mode     mode
	input    string
}

func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args, stdout)
	if err != nil {
		return printError(stderr, err)
	}

	if opts.showHelp {
		printUsage(stdout)
		return exitOK
	}

	source, err := os.ReadFile(opts.input)
	if err != nil {
		return printError(stderr, fmt.Errorf("read %s: %w", opts.input, err))
	}

	if err := dispatch(opts, source, stdout); err != nil {
		return printError(stderr, err)
	}

	return exitOK
}

func parseOptions(args []string, stdout io.Writer) (options, error) {
	fs := flag.NewFlagSet(toolName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		help    bool
		tokens  bool
		ast     bool
		sema    bool
		ir      bool
		codegen bool
	)

	fs.BoolVar(&help, "help", false, "show usage")
	fs.BoolVar(&help, "h", false, "show usage")
	fs.BoolVar(&tokens, "tokens", false, "print lexer output")
	fs.BoolVar(&ast, "ast", false, "print parser output")
	fs.BoolVar(&sema, "sema", false, "print semantic analysis output")
	fs.BoolVar(&ir, "ir", false, "print IR output")
	fs.BoolVar(&codegen, "codegen", false, "print VM code output")

	if err := fs.Parse(args); err != nil {
		return options{}, fmt.Errorf("parse flags: %w", err)
	}

	if help {
		return options{showHelp: true}, nil
	}

	selectedModes := 0
	selected := modeRun

	for _, candidate := range []struct {
		enabled bool
		mode    mode
	}{
		{enabled: tokens, mode: modeTokens},
		{enabled: ast, mode: modeAST},
		{enabled: sema, mode: modeSema},
		{enabled: ir, mode: modeIR},
		{enabled: codegen, mode: modeCodegen},
	} {
		if !candidate.enabled {
			continue
		}

		selectedModes++
		selected = candidate.mode
	}

	if selectedModes > 1 {
		return options{}, fmt.Errorf("choose only one phase flag")
	}

	positionals := fs.Args()
	if len(positionals) == 0 {
		return options{}, fmt.Errorf("missing input file")
	}
	if len(positionals) > 1 {
		return options{}, fmt.Errorf("expected one input file, got %d", len(positionals))
	}

	return options{
		mode:  selected,
		input: positionals[0],
	}, nil
}

func dispatch(opts options, source []byte, stdout io.Writer) error {
	inputName := filepath.Base(opts.input)
	message := ""

	switch opts.mode {
	case modeTokens:
		return printTokens(source, stdout)
	case modeAST:
		return printAST(source, stdout)
	case modeSema:
		message = "semantic analysis output"
	case modeIR:
		message = "IR output"
	case modeCodegen:
		message = "code generation output"
	case modeRun:
		message = "compile-and-run path"
	default:
		return fmt.Errorf("internal error: unknown mode %q", opts.mode)
	}

	_, err := fmt.Fprintf(stdout, "slice0: %s not implemented yet for %s (%d bytes loaded)\n", message, inputName, len(source))
	return err
}

func printTokens(source []byte, stdout io.Writer) error {
	tokens, err := lexer.Tokenize(string(source))
	if err != nil {
		return fmt.Errorf("lex: %w", err)
	}

	for _, tok := range tokens {
		if _, err := fmt.Fprintln(stdout, tok.String()); err != nil {
			return err
		}
	}

	return nil
}

func printAST(source []byte, stdout io.Writer) error {
	tokens, err := lexer.Tokenize(string(source))
	if err != nil {
		return fmt.Errorf("lex: %w", err)
	}

	program, err := parser.Parse(tokens)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	_, err = io.WriteString(stdout, ast.FormatProgram(program))
	return err
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, strings.TrimSpace(`Usage: mycc [phase flag] <file>

Phase flags:
  --tokens    print lexer output
  --ast       print parser output
  --sema      print semantic analysis output
  --ir        print IR output
  --codegen   print VM code output

Without phase flag, mycc will compile and run input program.
`)+"\n")
}

func printError(w io.Writer, err error) int {
	_, _ = fmt.Fprintf(w, "%s: %v\n", toolName, err)
	return classifyExitCode(err)
}

func classifyExitCode(err error) int {
	if err == nil {
		return exitOK
	}

	if strings.Contains(err.Error(), "missing input file") || strings.Contains(err.Error(), "choose only one phase flag") || strings.Contains(err.Error(), "expected one input file") || strings.Contains(err.Error(), "parse flags") {
		return exitUsage
	}

	return exitFailure
}
