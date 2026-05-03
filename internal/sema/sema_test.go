package sema

import (
	"testing"

	"github.com/xwhiz/compiler/internal/lexer"
	"github.com/xwhiz/compiler/internal/parser"
)

func TestAnalyzeRejectsUndeclaredVariable(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { print_int(x); return 0; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	err = Analyze(program)
	if err == nil {
		t.Fatal("Analyze() error = nil, want non-nil")
	}

	if got, want := err.Error(), "semantic: unknown variable \"x\" at 1:24"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}
