package parser

import (
	"testing"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/lexer"
)

func TestParseMinimalMain(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { return 0; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(program.Functions) != 1 {
		t.Fatalf("len(program.Functions) = %d, want 1", len(program.Functions))
	}

	fn := program.Functions[0]
	if fn.Name != "main" {
		t.Fatalf("fn.Name = %q, want %q", fn.Name, "main")
	}
	if fn.ReturnType != ast.TypeInt {
		t.Fatalf("fn.ReturnType = %q, want %q", fn.ReturnType, ast.TypeInt)
	}
	if len(fn.Body.Stmts) != 1 {
		t.Fatalf("len(fn.Body.Stmts) = %d, want 1", len(fn.Body.Stmts))
	}

	ret, ok := fn.Body.Stmts[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.ReturnStmt", fn.Body.Stmts[0])
	}

	lit, ok := ret.Value.(*ast.IntLiteral)
	if !ok {
		t.Fatalf("ret.Value type = %T, want *ast.IntLiteral", ret.Value)
	}
	if lit.Lexeme != "0" {
		t.Fatalf("lit.Lexeme = %q, want %q", lit.Lexeme, "0")
	}
}

func TestParseReportsStatementError(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { 42; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	_, err = Parse(tokens)
	if err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}

	if got, want := err.Error(), "1:14: expected statement, got INT_LIT \"42\""; got != want {
		t.Fatalf("Parse() error = %q, want %q", got, want)
	}
}
