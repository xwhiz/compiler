package sema

import (
	"fmt"

	"github.com/xwhiz/compiler/internal/ast"
)

type builtinSig struct {
	params []ast.TypeName
	ret    ast.TypeName
}

type scope map[string]ast.TypeName

type analyzer struct {
	builtins []scope
	scopes   []scope
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
	a := &analyzer{}
	for _, fn := range program.Functions {
		if fn.Name == "main" {
			hasMain = true
		}
		if err := a.analyzeFunc(fn); err != nil {
			return err
		}
	}

	if !hasMain {
		return fmt.Errorf("semantic: missing main function")
	}

	return nil
}

func (a *analyzer) analyzeFunc(fn *ast.FuncDecl) error {
	a.pushScope()
	defer a.popScope()
	return a.analyzeBlock(fn.Body, fn.ReturnType)
}

func (a *analyzer) analyzeBlock(block *ast.BlockStmt, returnType ast.TypeName) error {
	a.pushScope()
	defer a.popScope()

	for _, stmt := range block.Stmts {
		if err := a.analyzeStmt(stmt, returnType); err != nil {
			return err
		}
	}
	return nil
}

func (a *analyzer) analyzeStmt(stmt ast.Stmt, returnType ast.TypeName) error {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		return a.analyzeBlock(node, returnType)
	case *ast.VarDeclStmt:
		if node.Type != ast.TypeInt {
			return fmt.Errorf("semantic: local type %s not supported yet at %s", node.Type, node.Pos)
		}
		if node.Type == ast.TypeVoid {
			return fmt.Errorf("semantic: variable %q cannot have type void at %s", node.Name, node.Pos)
		}
		if err := a.declare(node.Name, node.Type, node.Pos); err != nil {
			return err
		}
		if node.Init == nil {
			return nil
		}
		initType, err := a.exprType(node.Init)
		if err != nil {
			return err
		}
		if initType != node.Type {
			return fmt.Errorf("semantic: initializer type mismatch for %s at %s: want %s, got %s", node.Name, node.Pos, node.Type, initType)
		}
		return nil
	case *ast.IfStmt:
		condType, err := a.exprType(node.Cond)
		if err != nil {
			return err
		}
		if condType != ast.TypeInt {
			return fmt.Errorf("semantic: if condition at %s must be int", node.Pos)
		}
		if err := a.analyzeStmt(node.Then, returnType); err != nil {
			return err
		}
		if node.Else != nil {
			if err := a.analyzeStmt(node.Else, returnType); err != nil {
				return err
			}
		}
		return nil
	case *ast.WhileStmt:
		condType, err := a.exprType(node.Cond)
		if err != nil {
			return err
		}
		if condType != ast.TypeInt {
			return fmt.Errorf("semantic: while condition at %s must be int", node.Pos)
		}
		return a.analyzeStmt(node.Body, returnType)
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
		typeName, err := a.exprType(node.Value)
		if err != nil {
			return err
		}
		if typeName != returnType {
			return fmt.Errorf("semantic: return type mismatch at %s: want %s, got %s", node.Pos, returnType, typeName)
		}
		return nil
	case *ast.ExprStmt:
		_, err := a.exprType(node.Expr)
		return err
	default:
		return fmt.Errorf("semantic: unsupported statement %T", stmt)
	}
}

func (a *analyzer) exprType(expr ast.Expr) (ast.TypeName, error) {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		return ast.TypeInt, nil
	case *ast.IdentExpr:
		typ, ok := a.lookup(node.Name)
		if !ok {
			return "", fmt.Errorf("semantic: unknown variable %q at %s", node.Name, node.Pos)
		}
		return typ, nil
	case *ast.CallExpr:
		sig, ok := builtins[node.Callee]
		if !ok {
			return "", fmt.Errorf("semantic: call to unknown function %q at %s", node.Callee, node.Pos)
		}
		if len(node.Args) != len(sig.params) {
			return "", fmt.Errorf("semantic: wrong arg count for %s at %s: want %d, got %d", node.Callee, node.Pos, len(sig.params), len(node.Args))
		}
		for i, arg := range node.Args {
			argType, err := a.exprType(arg)
			if err != nil {
				return "", err
			}
			if argType != sig.params[i] {
				return "", fmt.Errorf("semantic: arg %d for %s at %s: want %s, got %s", i+1, node.Callee, node.Pos, sig.params[i], argType)
			}
		}
		return sig.ret, nil
	case *ast.AssignExpr:
		lhsType, ok := a.lookup(node.Name)
		if !ok {
			return "", fmt.Errorf("semantic: unknown variable %q at %s", node.Name, node.Pos)
		}
		rhsType, err := a.exprType(node.Value)
		if err != nil {
			return "", err
		}
		if lhsType != rhsType {
			return "", fmt.Errorf("semantic: assignment type mismatch for %s at %s: want %s, got %s", node.Name, node.Pos, lhsType, rhsType)
		}
		return lhsType, nil
	case *ast.BinaryExpr:
		leftType, err := a.exprType(node.Left)
		if err != nil {
			return "", err
		}
		rightType, err := a.exprType(node.Right)
		if err != nil {
			return "", err
		}
		if leftType != ast.TypeInt || rightType != ast.TypeInt {
			return "", fmt.Errorf("semantic: binary op %s at %s needs int operands", node.Op, node.Pos)
		}
		return ast.TypeInt, nil
	case *ast.UnaryExpr:
		valueType, err := a.exprType(node.Value)
		if err != nil {
			return "", err
		}
		if valueType != ast.TypeInt {
			return "", fmt.Errorf("semantic: unary op %s at %s needs int operand", node.Op, node.Pos)
		}
		return ast.TypeInt, nil
	default:
		return "", fmt.Errorf("semantic: unsupported expression %T", expr)
	}
}

func (a *analyzer) pushScope() {
	a.scopes = append(a.scopes, scope{})
}

func (a *analyzer) popScope() {
	if len(a.scopes) == 0 {
		return
	}
	a.scopes = a.scopes[:len(a.scopes)-1]
}

func (a *analyzer) declare(name string, typ ast.TypeName, pos interface{ String() string }) error {
	current := a.scopes[len(a.scopes)-1]
	if _, exists := current[name]; exists {
		return fmt.Errorf("semantic: duplicate declaration of %q at %s", name, pos.String())
	}
	current[name] = typ
	return nil
}

func (a *analyzer) lookup(name string) (ast.TypeName, bool) {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if typ, ok := a.scopes[i][name]; ok {
			return typ, true
		}
	}
	return "", false
}
