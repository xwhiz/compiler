package driver

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/ir"
	"github.com/xwhiz/compiler/internal/lexer"
	"github.com/xwhiz/compiler/internal/parser"
	"github.com/xwhiz/compiler/internal/sema"
	"github.com/xwhiz/compiler/internal/vm"
)

type mode string

const (
	modeCompile mode = "compile"
	modeTokens  mode = "tokens"
	modeAST     mode = "ast"
	modeSema    mode = "sema"
	modeIR      mode = "ir"
	modeCodegen mode = "codegen"
	toolName         = "cforge"
	exitOK           = 0
	exitUsage        = 2
	exitFailure      = 1
)

type options struct {
	showHelp bool
	mode     mode
	input    string
	output   string
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
		output  string
	)

	fs.BoolVar(&help, "help", false, "show usage")
	fs.BoolVar(&help, "h", false, "show usage")
	fs.BoolVar(&tokens, "tokens", false, "print lexer output")
	fs.BoolVar(&ast, "ast", false, "print parser output")
	fs.BoolVar(&sema, "sema", false, "print semantic analysis output")
	fs.BoolVar(&ir, "ir", false, "print IR output")
	fs.BoolVar(&codegen, "codegen", false, "print VM code output")
	fs.StringVar(&output, "o", "", "write object output file")

	if err := fs.Parse(args); err != nil {
		return options{}, fmt.Errorf("parse flags: %w", err)
	}

	if help {
		return options{showHelp: true}, nil
	}

	selectedModes := 0
	selected := modeCompile

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
	if selectedModes > 0 && output != "" {
		return options{}, fmt.Errorf("-o only allowed without phase flag")
	}

	positionals := fs.Args()
	if len(positionals) == 0 {
		return options{}, fmt.Errorf("missing input file")
	}
	if len(positionals) > 1 {
		return options{}, fmt.Errorf("expected one input file, got %d", len(positionals))
	}

	return options{
		mode:   selected,
		input:  positionals[0],
		output: output,
	}, nil
}

func dispatch(opts options, source []byte, stdout io.Writer) error {
	switch opts.mode {
	case modeTokens:
		return printTokens(source, stdout)
	}

	program, err := parseProgram(source)
	if err != nil {
		return err
	}

	switch opts.mode {
	case modeAST:
		_, err := io.WriteString(stdout, ast.FormatProgram(program))
		return err
	case modeSema:
		if err := sema.Analyze(program); err != nil {
			return err
		}
		_, err := io.WriteString(stdout, "semantic OK\n")
		return err
	case modeIR:
		lowered, err := compileIR(program)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, ir.Format(lowered))
		return err
	case modeCodegen:
		vmProgram, err := compileVM(program)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, vm.Format(vmProgram))
		return err
	case modeCompile:
		vmProgram, err := compileVM(program)
		if err != nil {
			return err
		}
		outputPath := opts.output
		if outputPath == "" {
			outputPath = defaultObjectPath(opts.input)
		}
		err = os.WriteFile(outputPath, []byte(vm.Format(vmProgram)), 0o644)
		if err != nil {
			return fmt.Errorf("write %s: %w", outputPath, err)
		}
		_, err = fmt.Fprintf(stdout, "wrote %s\n", outputPath)
		return err
	default:
		return fmt.Errorf("internal error: unknown mode %q", opts.mode)
	}
}

func defaultObjectPath(input string) string {
	ext := filepath.Ext(input)
	if ext == "" {
		return input + ".vmo"
	}
	return strings.TrimSuffix(input, ext) + ".vmo"
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

func parseProgram(source []byte) (*ast.Program, error) {
	tokens, err := lexer.Tokenize(string(source))
	if err != nil {
		return nil, fmt.Errorf("lex: %w", err)
	}

	program, err := parser.Parse(tokens)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return program, nil
}

func compileIR(program *ast.Program) (*ir.Program, error) {
	if err := sema.Analyze(program); err != nil {
		return nil, err
	}

	lowered, err := ir.Lower(program)
	if err != nil {
		return nil, err
	}

	return lowered, nil
}

func compileVM(program *ast.Program) (*vm.Program, error) {
	lowered, err := compileIR(program)
	if err != nil {
		return nil, err
	}

	compiled, err := vm.Compile(lowered)
	if err != nil {
		return nil, err
	}

	return compiled, nil
}

func printUsage(w io.Writer) {
	_, _ = io.WriteString(w, strings.TrimSpace(`Usage: cforge [phase flag] [-o output.vmo] <file>

Phase flags:
  --tokens    print lexer output
  --ast       print parser output
  --sema      print semantic analysis output
  --ir        print IR output
  --codegen   print VM code output

Without phase flag, cforge compiles input program to a .vmo object file.
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

	if strings.Contains(err.Error(), "missing input file") || strings.Contains(err.Error(), "choose only one phase flag") || strings.Contains(err.Error(), "expected one input file") || strings.Contains(err.Error(), "parse flags") || strings.Contains(err.Error(), "-o only allowed") {
		return exitUsage
	}

	return exitFailure
}
