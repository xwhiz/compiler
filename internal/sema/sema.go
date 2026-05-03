package sema

import (
	"fmt"
	"maps"

	"github.com/xwhiz/compiler/internal/ast"
)

type paramSig struct {
	typ                ast.TypeName
	wantArray          bool
	allowStringLiteral bool
}

type funcSig struct {
	params  []paramSig
	ret     ast.TypeName
	builtin bool
}

type symbol struct {
	typ      ast.TypeName
	arrayLen int
}

type scope map[string]symbol

type exprInfo struct {
	typ             ast.TypeName
	arrayLen        int
	isLValue        bool
	isStringLiteral bool
}

type analyzer struct {
	globals map[string]symbol
	scopes  []scope
	funcs   map[string]funcSig
}

var builtins = map[string]funcSig{
	"print_int":     {params: []paramSig{{typ: ast.TypeInt}}, ret: ast.TypeVoid, builtin: true},
	"print_float":   {params: []paramSig{{typ: ast.TypeFloat}}, ret: ast.TypeVoid, builtin: true},
	"print_char":    {params: []paramSig{{typ: ast.TypeChar}}, ret: ast.TypeVoid, builtin: true},
	"print_str":     {params: []paramSig{{typ: ast.TypeChar, wantArray: true, allowStringLiteral: true}}, ret: ast.TypeVoid, builtin: true},
	"print_newline": {params: nil, ret: ast.TypeVoid, builtin: true},
}

func Analyze(program *ast.Program) error {
	if len(program.Decls) == 0 {
		return fmt.Errorf("semantic: missing top-level declarations")
	}
	a := &analyzer{globals: map[string]symbol{}, funcs: map[string]funcSig{}}
	maps.Copy(a.funcs, builtins)
	hasMain := false
	for _, decl := range program.Decls {
		switch node := decl.(type) {
		case *ast.VarDecl:
			if err := a.analyzeGlobalDecl(node); err != nil {
				return err
			}
		case *ast.FuncDecl:
			if node.Name == "main" {
				hasMain = true
			}
			if err := a.registerFunc(node); err != nil {
				return err
			}
			if err := a.analyzeFunc(node); err != nil {
				return err
			}
		default:
			return fmt.Errorf("semantic: unsupported top-level declaration %T", decl)
		}
	}
	if !hasMain {
		return fmt.Errorf("semantic: missing main function")
	}
	return nil
}

func (a *analyzer) analyzeGlobalDecl(node *ast.VarDecl) error {
	if node.Type == ast.TypeVoid {
		return fmt.Errorf("semantic: variable %q cannot have type void at %s", node.Name, node.Pos)
	}
	if !isSupportedType(node.Type) {
		return fmt.Errorf("semantic: global type %s not supported at %s", node.Type, node.Pos)
	}
	if node.ArrayLen < 0 {
		return fmt.Errorf("semantic: invalid array length for %q at %s", node.Name, node.Pos)
	}
	if _, exists := a.globals[node.Name]; exists {
		return fmt.Errorf("semantic: duplicate global %q at %s", node.Name, node.Pos)
	}
	if _, exists := a.funcs[node.Name]; exists {
		return fmt.Errorf("semantic: global %q conflicts with function at %s", node.Name, node.Pos)
	}
	a.globals[node.Name] = symbol{typ: node.Type, arrayLen: node.ArrayLen}
	if node.Init == nil {
		return nil
	}
	info, err := a.exprType(node.Init)
	if err != nil {
		return err
	}
	if node.ArrayLen > 0 {
		if node.Type == ast.TypeChar && info.isStringLiteral {
			if info.arrayLen > node.ArrayLen {
				return fmt.Errorf("semantic: string initializer too long for %s at %s", node.Name, node.Pos)
			}
			return nil
		}
		return fmt.Errorf("semantic: array initializer for %s at %s not supported", node.Name, node.Pos)
	}
	if !assignable(node.Type, info) {
		return fmt.Errorf("semantic: initializer type mismatch for %s at %s: want %s", node.Name, node.Pos, node.Type)
	}
	return nil
}

func (a *analyzer) registerFunc(fn *ast.FuncDecl) error {
	if !isSupportedType(fn.ReturnType) && fn.ReturnType != ast.TypeVoid {
		return fmt.Errorf("semantic: function return type %s not supported at %s", fn.ReturnType, fn.Pos)
	}
	if _, exists := a.funcs[fn.Name]; exists {
		return fmt.Errorf("semantic: duplicate function %q at %s", fn.Name, fn.Pos)
	}
	if _, exists := a.globals[fn.Name]; exists {
		return fmt.Errorf("semantic: function %q conflicts with global at %s", fn.Name, fn.Pos)
	}
	params := make([]paramSig, 0, len(fn.Params))
	seen := map[string]struct{}{}
	for _, param := range fn.Params {
		if !isSupportedType(param.Type) {
			return fmt.Errorf("semantic: parameter type %s not supported yet at %s", param.Type, param.Pos)
		}
		if param.Type == ast.TypeVoid {
			return fmt.Errorf("semantic: parameter %q cannot have type void at %s", param.Name, param.Pos)
		}
		if _, ok := seen[param.Name]; ok {
			return fmt.Errorf("semantic: duplicate parameter %q at %s", param.Name, param.Pos)
		}
		seen[param.Name] = struct{}{}
		params = append(params, paramSig{typ: param.Type})
	}
	a.funcs[fn.Name] = funcSig{params: params, ret: fn.ReturnType}
	return nil
}

func (a *analyzer) analyzeFunc(fn *ast.FuncDecl) error {
	a.pushScope()
	defer a.popScope()
	for _, param := range fn.Params {
		if err := a.declareLocal(param.Name, symbol{typ: param.Type}, param.Pos); err != nil {
			return err
		}
	}
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
	case *ast.VarDecl:
		if node.Type == ast.TypeVoid {
			return fmt.Errorf("semantic: variable %q cannot have type void at %s", node.Name, node.Pos)
		}
		if !isSupportedType(node.Type) {
			return fmt.Errorf("semantic: local type %s not supported at %s", node.Type, node.Pos)
		}
		if node.ArrayLen < 0 {
			return fmt.Errorf("semantic: invalid array length for %q at %s", node.Name, node.Pos)
		}
		if err := a.declareLocal(node.Name, symbol{typ: node.Type, arrayLen: node.ArrayLen}, node.Pos); err != nil {
			return err
		}
		if node.Init == nil {
			return nil
		}
		info, err := a.exprType(node.Init)
		if err != nil {
			return err
		}
		if node.ArrayLen > 0 {
			if node.Type == ast.TypeChar && info.isStringLiteral {
				if info.arrayLen > node.ArrayLen {
					return fmt.Errorf("semantic: string initializer too long for %s at %s", node.Name, node.Pos)
				}
				return nil
			}
			return fmt.Errorf("semantic: array initializer for %s at %s not supported", node.Name, node.Pos)
		}
		if !assignable(node.Type, info) {
			return fmt.Errorf("semantic: initializer type mismatch for %s at %s: want %s", node.Name, node.Pos, node.Type)
		}
		return nil
	case *ast.IfStmt:
		info, err := a.exprType(node.Cond)
		if err != nil {
			return err
		}
		if !isNumericScalar(info) {
			return fmt.Errorf("semantic: if condition at %s must be numeric scalar", node.Pos)
		}
		if err := a.analyzeStmt(node.Then, returnType); err != nil {
			return err
		}
		if node.Else != nil {
			return a.analyzeStmt(node.Else, returnType)
		}
		return nil
	case *ast.WhileStmt:
		info, err := a.exprType(node.Cond)
		if err != nil {
			return err
		}
		if !isNumericScalar(info) {
			return fmt.Errorf("semantic: while condition at %s must be numeric scalar", node.Pos)
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
		info, err := a.exprType(node.Value)
		if err != nil {
			return err
		}
		if !assignable(returnType, info) {
			return fmt.Errorf("semantic: return type mismatch at %s: want %s", node.Pos, returnType)
		}
		return nil
	case *ast.ExprStmt:
		_, err := a.exprType(node.Expr)
		return err
	default:
		return fmt.Errorf("semantic: unsupported statement %T", stmt)
	}
}

func (a *analyzer) exprType(expr ast.Expr) (exprInfo, error) {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		return exprInfo{typ: ast.TypeInt}, nil
	case *ast.FloatLiteral:
		return exprInfo{typ: ast.TypeFloat}, nil
	case *ast.CharLiteral:
		return exprInfo{typ: ast.TypeChar}, nil
	case *ast.StringLiteral:
		return exprInfo{typ: ast.TypeChar, arrayLen: len(node.Lexeme) - 1, isStringLiteral: true}, nil
	case *ast.IdentExpr:
		sym, ok := a.lookup(node.Name)
		if !ok {
			return exprInfo{}, fmt.Errorf("semantic: unknown variable %q at %s", node.Name, node.Pos)
		}
		return exprInfo{typ: sym.typ, arrayLen: sym.arrayLen, isLValue: sym.arrayLen == 0}, nil
	case *ast.IndexExpr:
		sym, ok := a.lookup(node.Name)
		if !ok {
			return exprInfo{}, fmt.Errorf("semantic: unknown variable %q at %s", node.Name, node.Pos)
		}
		if sym.arrayLen == 0 {
			return exprInfo{}, fmt.Errorf("semantic: %q is not an array at %s", node.Name, node.Pos)
		}
		indexInfo, err := a.exprType(node.Index)
		if err != nil {
			return exprInfo{}, err
		}
		if !isIntLikeScalar(indexInfo) {
			return exprInfo{}, fmt.Errorf("semantic: array index for %s at %s must be int-like", node.Name, node.Pos)
		}
		return exprInfo{typ: sym.typ, isLValue: true}, nil
	case *ast.CallExpr:
		sig, ok := a.funcs[node.Callee]
		if !ok {
			return exprInfo{}, fmt.Errorf("semantic: call to unknown function %q at %s", node.Callee, node.Pos)
		}
		if len(node.Args) != len(sig.params) {
			return exprInfo{}, fmt.Errorf("semantic: wrong arg count for %s at %s: want %d, got %d", node.Callee, node.Pos, len(sig.params), len(node.Args))
		}
		for i, arg := range node.Args {
			info, err := a.exprType(arg)
			if err != nil {
				return exprInfo{}, err
			}
			if !paramAccepts(sig.params[i], info) {
				return exprInfo{}, fmt.Errorf("semantic: arg %d for %s at %s has wrong type", i+1, node.Callee, node.Pos)
			}
		}
		return exprInfo{typ: sig.ret}, nil
	case *ast.AssignExpr:
		target, err := a.exprType(node.Target)
		if err != nil {
			return exprInfo{}, err
		}
		if !target.isLValue || target.arrayLen > 0 {
			return exprInfo{}, fmt.Errorf("semantic: assignment target at %s is not assignable", node.Pos)
		}
		value, err := a.exprType(node.Value)
		if err != nil {
			return exprInfo{}, err
		}
		if !assignable(target.typ, value) {
			return exprInfo{}, fmt.Errorf("semantic: assignment type mismatch at %s", node.Pos)
		}
		return exprInfo{typ: target.typ}, nil
	case *ast.BinaryExpr:
		left, err := a.exprType(node.Left)
		if err != nil {
			return exprInfo{}, err
		}
		right, err := a.exprType(node.Right)
		if err != nil {
			return exprInfo{}, err
		}
		switch node.Op {
		case ast.BinaryAdd, ast.BinarySub, ast.BinaryMul, ast.BinaryDiv:
			if !isNumericScalar(left) || !isNumericScalar(right) {
				return exprInfo{}, fmt.Errorf("semantic: binary op %s at %s needs numeric operands", node.Op, node.Pos)
			}
			return exprInfo{typ: promotedNumericType(left.typ, right.typ)}, nil
		case ast.BinaryMod:
			if !isIntLikeScalar(left) || !isIntLikeScalar(right) {
				return exprInfo{}, fmt.Errorf("semantic: binary op %% at %s needs int-like operands", node.Pos)
			}
			return exprInfo{typ: ast.TypeInt}, nil
		case ast.BinaryLT, ast.BinaryLE, ast.BinaryGT, ast.BinaryGE, ast.BinaryEQ, ast.BinaryNE:
			if !isNumericScalar(left) || !isNumericScalar(right) {
				return exprInfo{}, fmt.Errorf("semantic: comparison at %s needs numeric operands", node.Pos)
			}
			return exprInfo{typ: ast.TypeInt}, nil
		case ast.BinaryAnd, ast.BinaryOr:
			if !isNumericScalar(left) || !isNumericScalar(right) {
				return exprInfo{}, fmt.Errorf("semantic: logical op at %s needs numeric operands", node.Pos)
			}
			return exprInfo{typ: ast.TypeInt}, nil
		default:
			return exprInfo{}, fmt.Errorf("semantic: unsupported binary op %s at %s", node.Op, node.Pos)
		}
	case *ast.UnaryExpr:
		value, err := a.exprType(node.Value)
		if err != nil {
			return exprInfo{}, err
		}
		switch node.Op {
		case ast.UnaryNeg:
			if !isNumericScalar(value) {
				return exprInfo{}, fmt.Errorf("semantic: unary - at %s needs numeric operand", node.Pos)
			}
			return exprInfo{typ: unaryNumericType(value.typ)}, nil
		case ast.UnaryNot:
			if !isNumericScalar(value) {
				return exprInfo{}, fmt.Errorf("semantic: unary ! at %s needs numeric operand", node.Pos)
			}
			return exprInfo{typ: ast.TypeInt}, nil
		default:
			return exprInfo{}, fmt.Errorf("semantic: unsupported unary op %s at %s", node.Op, node.Pos)
		}
	default:
		return exprInfo{}, fmt.Errorf("semantic: unsupported expression %T", expr)
	}
}

func (a *analyzer) pushScope() { a.scopes = append(a.scopes, scope{}) }

func (a *analyzer) popScope() {
	if len(a.scopes) > 0 {
		a.scopes = a.scopes[:len(a.scopes)-1]
	}
}

func (a *analyzer) declareLocal(name string, sym symbol, pos interface{ String() string }) error {
	current := a.scopes[len(a.scopes)-1]
	if _, exists := current[name]; exists {
		return fmt.Errorf("semantic: duplicate declaration of %q at %s", name, pos.String())
	}
	current[name] = sym
	return nil
}

func (a *analyzer) lookup(name string) (symbol, bool) {
	for i := len(a.scopes) - 1; i >= 0; i-- {
		if sym, ok := a.scopes[i][name]; ok {
			return sym, true
		}
	}
	sym, ok := a.globals[name]
	return sym, ok
}

func isSupportedType(typ ast.TypeName) bool {
	return typ == ast.TypeInt || typ == ast.TypeChar || typ == ast.TypeFloat
}

func isNumericScalar(info exprInfo) bool {
	return info.arrayLen == 0 && (info.typ == ast.TypeInt || info.typ == ast.TypeChar || info.typ == ast.TypeFloat)
}

func isIntLikeScalar(info exprInfo) bool {
	return info.arrayLen == 0 && (info.typ == ast.TypeInt || info.typ == ast.TypeChar)
}

func promotedNumericType(left, right ast.TypeName) ast.TypeName {
	if left == ast.TypeFloat || right == ast.TypeFloat {
		return ast.TypeFloat
	}
	return ast.TypeInt
}

func unaryNumericType(typ ast.TypeName) ast.TypeName {
	if typ == ast.TypeFloat {
		return ast.TypeFloat
	}
	return ast.TypeInt
}

func assignable(target ast.TypeName, src exprInfo) bool {
	if src.arrayLen > 0 {
		return false
	}
	if target == src.typ {
		return true
	}
	switch target {
	case ast.TypeInt:
		return src.typ == ast.TypeChar
	case ast.TypeFloat:
		return src.typ == ast.TypeInt || src.typ == ast.TypeChar
	default:
		return false
	}
}

func paramAccepts(param paramSig, src exprInfo) bool {
	if param.wantArray {
		if src.arrayLen > 0 && src.typ == param.typ {
			return true
		}
		return param.allowStringLiteral && src.isStringLiteral && src.typ == param.typ
	}
	return assignable(param.typ, src)
}
