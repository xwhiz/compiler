package sema

import (
	"fmt"

	"github.com/xwhiz/compiler/internal/ast"
)

type builtinSig struct {
	params []ast.TypeName
	ret    ast.TypeName
}

var builtins = map[string]builtinSig{
	"print_int":     {params: []ast.TypeName{ast.TypeInt}, ret: ast.TypeVoid},
	"print_newline": {params: nil, ret: ast.TypeVoid},
}

func Analyze(program *ast.Program) error {
	if len(program.Functions) == 0 {
		return fmt.Errorf("semantic: missing function definitions")
	}

	hasMain := false
	for _, fn := range program.Functions {
		if fn.Name == "main" {
			hasMain = true
		}
		if err := analyzeFunc(fn); err != nil {
			return err
		}
	}

	if !hasMain {
		return fmt.Errorf("semantic: missing main function")
	}

	return nil
}

func analyzeFunc(fn *ast.FuncDecl) error {
	for _, stmt := range fn.Body.Stmts {
		if err := analyzeStmt(stmt, fn.ReturnType); err != nil {
			return err
		}
	}
	return nil
}

func analyzeStmt(stmt ast.Stmt, returnType ast.TypeName) error {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		for _, inner := range node.Stmts {
			if err := analyzeStmt(inner, returnType); err != nil {
				return err
			}
		}
		return nil
	case *ast.ReturnStmt:
		if returnType == ast.TypeVoid {
			if node.Value != nil {
				return fmt.Errorf("semantic: void function cannot return value at %s", node.Pos)
			}
			return nil
		}
		if node.Value == nil {
			return fmt.Errorf("semantic: %s function must return value at %s", returnType, node.Pos)
		}
		typeName, err := exprType(node.Value)
		if err != nil {
			return err
		}
		if typeName != returnType {
			return fmt.Errorf("semantic: return type mismatch at %s: want %s, got %s", node.Pos, returnType, typeName)
		}
		return nil
	case *ast.ExprStmt:
		_, err := exprType(node.Expr)
		return err
	default:
		return fmt.Errorf("semantic: unsupported statement %T", stmt)
	}
}

func exprType(expr ast.Expr) (ast.TypeName, error) {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		return ast.TypeInt, nil
	case *ast.CallExpr:
		sig, ok := builtins[node.Callee]
		if !ok {
			return "", fmt.Errorf("semantic: call to unknown function %q at %s", node.Callee, node.Pos)
		}
		if len(node.Args) != len(sig.params) {
			return "", fmt.Errorf("semantic: wrong arg count for %s at %s: want %d, got %d", node.Callee, node.Pos, len(sig.params), len(node.Args))
		}
		for i, arg := range node.Args {
			argType, err := exprType(arg)
			if err != nil {
				return "", err
			}
			if argType != sig.params[i] {
				return "", fmt.Errorf("semantic: arg %d for %s at %s: want %s, got %s", i+1, node.Callee, node.Pos, sig.params[i], argType)
			}
		}
		return sig.ret, nil
	default:
		return "", fmt.Errorf("semantic: unsupported expression %T", expr)
	}
}
