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

func TestParseIfWhile(t *testing.T) {
	source := "int main() { int i = 0; while (i < 3) { if (i == 1) print_int(i); i = i + 1; } return i; }"
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	stmts := program.Functions[0].Body.Stmts
	if len(stmts) < 2 {
		t.Fatalf("len(stmts) = %d, want at least 2", len(stmts))
	}
	if _, ok := stmts[1].(*ast.WhileStmt); !ok {
		t.Fatalf("stmt type = %T, want *ast.WhileStmt", stmts[1])
	}
}

func TestParseFunctionParams(t *testing.T) {
	source := "int add(int a, int b) { return a + b; } int main(void) { return add(2, 3); }"
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(program.Functions) != 2 {
		t.Fatalf("len(program.Functions) = %d, want 2", len(program.Functions))
	}
	add := program.Functions[0]
	if len(add.Params) != 2 {
		t.Fatalf("len(add.Params) = %d, want 2", len(add.Params))
	}
	if add.Params[0].Name != "a" || add.Params[1].Name != "b" {
		t.Fatalf("param names = %q, %q; want a, b", add.Params[0].Name, add.Params[1].Name)
	}
}

func TestParseArrayStringAndFloat(t *testing.T) {
	source := "float twice(float x) { return x + x; } int main() { char s[6] = \"hello\"; int a[3]; a[0] = 4; return 0; }"
	tokens, err := lexer.Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	program, err := Parse(tokens)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(program.Functions) != 2 {
		t.Fatalf("len(program.Functions) = %d, want 2", len(program.Functions))
	}
	mainFn := program.Functions[1]
	decl, ok := mainFn.Body.Stmts[0].(*ast.VarDeclStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.VarDeclStmt", mainFn.Body.Stmts[0])
	}
	if decl.ArrayLen != 6 || decl.Type != ast.TypeChar {
		t.Fatalf("decl = %#v, want char[6]", decl)
	}
	if _, ok := decl.Init.(*ast.StringLiteral); !ok {
		t.Fatalf("decl.Init type = %T, want *ast.StringLiteral", decl.Init)
	}
	assignStmt, ok := mainFn.Body.Stmts[2].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.ExprStmt", mainFn.Body.Stmts[2])
	}
	assign, ok := assignStmt.Expr.(*ast.AssignExpr)
	if !ok {
		t.Fatalf("expr type = %T, want *ast.AssignExpr", assignStmt.Expr)
	}
	if _, ok := assign.Target.(*ast.IndexExpr); !ok {
		t.Fatalf("assign target type = %T, want *ast.IndexExpr", assign.Target)
	}
}
