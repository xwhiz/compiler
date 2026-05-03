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

func TestParseReportsMissingSemicolon(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { print_int(42) }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	_, err = Parse(tokens)
	if err == nil {
		t.Fatal("Parse() error = nil, want non-nil")
	}

	if got, want := err.Error(), "1:28: expected ';' after expression statement, got RBRACE \"}\""; got != want {
		t.Fatalf("Parse() error = %q, want %q", got, want)
	}
}

func TestParseBuiltinCallStatement(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { print_int(123); return 0; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stmts := program.Functions[0].Body.Stmts
	if len(stmts) != 2 {
		t.Fatalf("len(stmts) = %d, want 2", len(stmts))
	}

	exprStmt, ok := stmts[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.ExprStmt", stmts[0])
	}

	call, ok := exprStmt.Expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expr type = %T, want *ast.CallExpr", exprStmt.Expr)
	}

	if call.Callee != "print_int" {
		t.Fatalf("call.Callee = %q, want %q", call.Callee, "print_int")
	}
	if len(call.Args) != 1 {
		t.Fatalf("len(call.Args) = %d, want 1", len(call.Args))
	}
}

func TestParseLocalDeclAndArithmetic(t *testing.T) {
	tokens, err := lexer.Tokenize("int main() { int x = 10; int y = 2; return x * y + 3; }")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stmts := program.Functions[0].Body.Stmts
	if len(stmts) != 3 {
		t.Fatalf("len(stmts) = %d, want 3", len(stmts))
	}

	decl, ok := stmts[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.VarDeclStmt", stmts[0])
	}
	if decl.Name != "x" || decl.Type != ast.TypeInt {
		t.Fatalf("decl = %#v, want int x", decl)
	}

	ret, ok := stmts[2].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.ReturnStmt", stmts[2])
	}
	if _, ok := ret.Value.(*ast.BinaryExpr); !ok {
		t.Fatalf("ret.Value type = %T, want *ast.BinaryExpr", ret.Value)
	}
}
