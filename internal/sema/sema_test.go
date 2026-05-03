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

func TestAnalyzeRejectsDuplicateDeclSameScope(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { int x = 1; int x = 2; return x; }")
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

	if got, want := err.Error(), "semantic: duplicate declaration of \"x\" at 1:25"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}

func TestAnalyzeRejectsForwardFunctionCall(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { return add(1, 2); } int add(int a, int b) { return a + b; }")
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

	if got, want := err.Error(), "semantic: call to unknown function \"add\" at 1:21"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}

func TestAnalyzeAcceptsDefinedFunctionCall(t *testing.T) {
	tokens, err := lexer.Tokenize("int add(int a, int b) { return a + b; } int main() { return add(1, 2); }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if err := Analyze(program); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
}

func TestAnalyzeAcceptsGlobalShadowing(t *testing.T) {
	tokens, err := lexer.Tokenize("int g = 10; int main() { int g = 3; return g; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}
	program, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err := Analyze(program); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
}

func TestAnalyzeRejectsDuplicateGlobal(t *testing.T) {
	tokens, err := lexer.Tokenize("int g = 1; int g = 2; int main() { return g; }")
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
	if got, want := err.Error(), "semantic: duplicate global \"g\" at 1:12"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}

func TestAnalyzeRejectsGlobalFunctionCollision(t *testing.T) {
	tokens, err := lexer.Tokenize("int g = 1; int g() { return 0; }")
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
	if got, want := err.Error(), "semantic: function \"g\" conflicts with global at 1:12"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}

func TestAnalyzeAcceptsStringAndArrayUsage(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { char s[6] = \"hello\"; int a[2]; a[0] = 4; print_str(s); print_int(a[0]); return 0; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}
	program, err := parser.Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if err := Analyze(program); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
}

func TestAnalyzeRejectsFloatToIntInitializer(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { int x = 2.5; return x; }")
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
	if got, want := err.Error(), "semantic: initializer type mismatch for x at 1:14: want int"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}

func TestAnalyzeRejectsStringTooLong(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { char s[4] = \"hello\"; return 0; }")
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
	if got, want := err.Error(), "semantic: string initializer too long for s at 1:14"; got != want {
		t.Fatalf("Analyze() error = %q, want %q", got, want)
	}
}
