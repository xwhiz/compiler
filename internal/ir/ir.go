package ir

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xwhiz/compiler/internal/ast"
)

type Op string

const (
	OpDeclareLocal Op = "declare_local"
	OpInitString   Op = "init_string"
	OpPushInt      Op = "push_int"
	OpPushFloat    Op = "push_float"
	OpPushChar     Op = "push_char"
	OpPushString   Op = "push_string"
	OpPushLocalRef Op = "push_local_ref"
	OpLoadLocal    Op = "load_local"
	OpStoreLocal   Op = "store_local"
	OpLoadIndex    Op = "load_index"
	OpStoreIndex   Op = "store_index"
	OpAdd          Op = "add"
	OpSub          Op = "sub"
	OpMul          Op = "mul"
	OpDiv          Op = "div"
	OpMod          Op = "mod"
	OpNeg          Op = "neg"
	OpNot          Op = "not"
	OpLT           Op = "lt"
	OpLE           Op = "le"
	OpGT           Op = "gt"
	OpGE           Op = "ge"
	OpEQ           Op = "eq"
	OpNE           Op = "ne"
	OpAnd          Op = "and"
	OpOr           Op = "or"
	OpDup          Op = "dup"
	OpPop          Op = "pop"
	OpLabel        Op = "label"
	OpJump         Op = "jump"
	OpJumpIfZero   Op = "jump_if_zero"
	OpCallBuiltin  Op = "call_builtin"
	OpCallFunc     Op = "call_func"
	OpReturn       Op = "return"
)

type VarInfo struct {
	Name     string
	Type     ast.TypeName
	ArrayLen int
}

type Program struct {
	Functions []Function
}

type Function struct {
	Name         string
	ReturnType   ast.TypeName
	Params       []VarInfo
	Instructions []Instruction
}

type Instruction struct {
	Op          Op
	Name        string
	Type        ast.TypeName
	ArrayLen    int
	IntValue    int64
	FloatValue  float64
	StringValue string
	ArgCount    int
}

type funcInfo struct {
	ret     ast.TypeName
	builtin bool
}

type symbol struct {
	slot     string
	typ      ast.TypeName
	arrayLen int
}

type lowerer struct {
	funcs      map[string]funcInfo
	scopes     []map[string]symbol
	nextSlotID int
	nextLabel  int
}

func Lower(program *ast.Program) (*Program, error) {
	funcs := map[string]funcInfo{
		"print_int":     {ret: ast.TypeVoid, builtin: true},
		"print_float":   {ret: ast.TypeVoid, builtin: true},
		"print_char":    {ret: ast.TypeVoid, builtin: true},
		"print_str":     {ret: ast.TypeVoid, builtin: true},
		"print_newline": {ret: ast.TypeVoid, builtin: true},
	}
	for _, fn := range program.Functions {
		funcs[fn.Name] = funcInfo{ret: fn.ReturnType}
	}

	out := &Program{}
	for _, fn := range program.Functions {
		lowered, err := lowerFunc(fn, funcs)
		if err != nil {
			return nil, err
		}
		out.Functions = append(out.Functions, *lowered)
	}
	return out, nil
}

func Format(program *Program) string {
	var b strings.Builder
	for i, fn := range program.Functions {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "func %s return=%s\n", fn.Name, fn.ReturnType)
		if len(fn.Params) == 0 {
			b.WriteString("  params <empty>\n")
		} else {
			parts := make([]string, 0, len(fn.Params))
			for _, param := range fn.Params {
				parts = append(parts, fmt.Sprintf("%s:%s", param.Name, param.Type))
			}
			fmt.Fprintf(&b, "  params %s\n", strings.Join(parts, ", "))
		}
		for _, inst := range fn.Instructions {
			fmt.Fprintf(&b, "  %s\n", inst.String())
		}
		b.WriteString("end\n")
	}
	return b.String()
}

func (i Instruction) String() string {
	switch i.Op {
	case OpDeclareLocal:
		if i.ArrayLen > 0 {
			return fmt.Sprintf("%s %s %s[%d]", i.Op, i.Name, i.Type, i.ArrayLen)
		}
		return fmt.Sprintf("%s %s %s", i.Op, i.Name, i.Type)
	case OpInitString:
		return fmt.Sprintf("%s %s %q", i.Op, i.Name, i.StringValue)
	case OpPushInt:
		return fmt.Sprintf("%s %d", i.Op, i.IntValue)
	case OpPushFloat:
		return fmt.Sprintf("%s %g", i.Op, i.FloatValue)
	case OpPushChar:
		return fmt.Sprintf("%s %q", i.Op, rune(i.IntValue))
	case OpPushString:
		return fmt.Sprintf("%s %q", i.Op, i.StringValue)
	case OpPushLocalRef, OpLoadLocal, OpStoreLocal, OpLoadIndex, OpStoreIndex, OpLabel, OpJump, OpJumpIfZero:
		return fmt.Sprintf("%s %s", i.Op, i.Name)
	case OpCallBuiltin, OpCallFunc:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	default:
		return string(i.Op)
	}
}

func lowerFunc(fn *ast.FuncDecl, funcs map[string]funcInfo) (*Function, error) {
	out := &Function{Name: fn.Name, ReturnType: fn.ReturnType}
	l := &lowerer{funcs: funcs}
	l.pushScope()
	for _, param := range fn.Params {
		slot := l.declare(param.Name, param.Type, 0)
		out.Params = append(out.Params, VarInfo{Name: slot, Type: param.Type})
	}
	if err := l.lowerBlock(fn.Body, out); err != nil {
		return nil, err
	}
	l.popScope()
	return out, nil
}

func (l *lowerer) lowerBlock(block *ast.BlockStmt, fn *Function) error {
	l.pushScope()
	defer l.popScope()
	for _, stmt := range block.Stmts {
		if err := l.lowerStmt(stmt, fn); err != nil {
			return err
		}
	}
	return nil
}

func (l *lowerer) lowerStmt(stmt ast.Stmt, fn *Function) error {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		return l.lowerBlock(node, fn)
	case *ast.VarDeclStmt:
		slot := l.declare(node.Name, node.Type, node.ArrayLen)
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpDeclareLocal, Name: slot, Type: node.Type, ArrayLen: node.ArrayLen})
		if node.Init == nil {
			return nil
		}
		if node.ArrayLen > 0 {
			if lit, ok := node.Init.(*ast.StringLiteral); ok {
				fn.Instructions = append(fn.Instructions, Instruction{Op: OpInitString, Name: slot, StringValue: stripStringQuotes(lit.Lexeme)})
				return nil
			}
			return fmt.Errorf("ir: unsupported array initializer for %s", node.Name)
		}
		if _, err := l.lowerExpr(node.Init, fn); err != nil {
			return err
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpStoreLocal, Name: slot})
		return nil
	case *ast.IfStmt:
		if _, err := l.lowerExpr(node.Cond, fn); err != nil {
			return err
		}
		elseLabel := l.newLabel("if_else")
		endLabel := l.newLabel("if_end")
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpJumpIfZero, Name: elseLabel})
		if err := l.lowerStmt(node.Then, fn); err != nil {
			return err
		}
		if node.Else != nil {
			fn.Instructions = append(fn.Instructions,
				Instruction{Op: OpJump, Name: endLabel},
				Instruction{Op: OpLabel, Name: elseLabel},
			)
			if err := l.lowerStmt(node.Else, fn); err != nil {
				return err
			}
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpLabel, Name: endLabel})
			return nil
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpLabel, Name: elseLabel})
		return nil
	case *ast.WhileStmt:
		startLabel := l.newLabel("while_start")
		endLabel := l.newLabel("while_end")
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpLabel, Name: startLabel})
		if _, err := l.lowerExpr(node.Cond, fn); err != nil {
			return err
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpJumpIfZero, Name: endLabel})
		if err := l.lowerStmt(node.Body, fn); err != nil {
			return err
		}
		fn.Instructions = append(fn.Instructions,
			Instruction{Op: OpJump, Name: startLabel},
			Instruction{Op: OpLabel, Name: endLabel},
		)
		return nil
	case *ast.ExprStmt:
		pushesValue, err := l.lowerExpr(node.Expr, fn)
		if err != nil {
			return err
		}
		if pushesValue {
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpPop})
		}
		return nil
	case *ast.ReturnStmt:
		if node.Value != nil {
			if _, err := l.lowerExpr(node.Value, fn); err != nil {
				return err
			}
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpReturn})
		return nil
	default:
		return fmt.Errorf("ir: unsupported statement %T", stmt)
	}
}

func (l *lowerer) lowerExpr(expr ast.Expr, fn *Function) (bool, error) {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		value, err := strconv.ParseInt(node.Lexeme, 10, 64)
		if err != nil {
			return false, fmt.Errorf("ir: parse int literal %q: %w", node.Lexeme, err)
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushInt, IntValue: value})
		return true, nil
	case *ast.FloatLiteral:
		value, err := strconv.ParseFloat(node.Lexeme, 64)
		if err != nil {
			return false, fmt.Errorf("ir: parse float literal %q: %w", node.Lexeme, err)
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushFloat, FloatValue: value})
		return true, nil
	case *ast.CharLiteral:
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushChar, IntValue: int64(stripCharQuotes(node.Lexeme))})
		return true, nil
	case *ast.StringLiteral:
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushString, StringValue: stripStringQuotes(node.Lexeme)})
		return true, nil
	case *ast.IdentExpr:
		sym, ok := l.lookup(node.Name)
		if !ok {
			return false, fmt.Errorf("ir: unknown local %q", node.Name)
		}
		if sym.arrayLen > 0 {
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushLocalRef, Name: sym.slot})
			return true, nil
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpLoadLocal, Name: sym.slot})
		return true, nil
	case *ast.IndexExpr:
		sym, ok := l.lookup(node.Name)
		if !ok {
			return false, fmt.Errorf("ir: unknown local %q", node.Name)
		}
		if _, err := l.lowerExpr(node.Index, fn); err != nil {
			return false, err
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpLoadIndex, Name: sym.slot})
		return true, nil
	case *ast.CallExpr:
		for _, arg := range node.Args {
			if _, err := l.lowerExpr(arg, fn); err != nil {
				return false, err
			}
		}
		info, ok := l.funcs[node.Callee]
		if !ok {
			return false, fmt.Errorf("ir: unknown function %q", node.Callee)
		}
		if info.builtin {
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpCallBuiltin, Name: node.Callee, ArgCount: len(node.Args)})
		} else {
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpCallFunc, Name: node.Callee, ArgCount: len(node.Args)})
		}
		return info.ret != ast.TypeVoid, nil
	case *ast.AssignExpr:
		switch target := node.Target.(type) {
		case *ast.IdentExpr:
			sym, ok := l.lookup(target.Name)
			if !ok {
				return false, fmt.Errorf("ir: unknown local %q", target.Name)
			}
			if _, err := l.lowerExpr(node.Value, fn); err != nil {
				return false, err
			}
			fn.Instructions = append(fn.Instructions,
				Instruction{Op: OpDup},
				Instruction{Op: OpStoreLocal, Name: sym.slot},
			)
			return true, nil
		case *ast.IndexExpr:
			sym, ok := l.lookup(target.Name)
			if !ok {
				return false, fmt.Errorf("ir: unknown local %q", target.Name)
			}
			if _, err := l.lowerExpr(node.Value, fn); err != nil {
				return false, err
			}
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpDup})
			if _, err := l.lowerExpr(target.Index, fn); err != nil {
				return false, err
			}
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpStoreIndex, Name: sym.slot})
			return true, nil
		default:
			return false, fmt.Errorf("ir: unsupported assignment target %T", node.Target)
		}
	case *ast.BinaryExpr:
		if _, err := l.lowerExpr(node.Left, fn); err != nil {
			return false, err
		}
		if _, err := l.lowerExpr(node.Right, fn); err != nil {
			return false, err
		}
		switch node.Op {
		case ast.BinaryAdd:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpAdd})
		case ast.BinarySub:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpSub})
		case ast.BinaryMul:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpMul})
		case ast.BinaryDiv:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpDiv})
		case ast.BinaryMod:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpMod})
		case ast.BinaryLT:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpLT})
		case ast.BinaryLE:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpLE})
		case ast.BinaryGT:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpGT})
		case ast.BinaryGE:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpGE})
		case ast.BinaryEQ:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpEQ})
		case ast.BinaryNE:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNE})
		case ast.BinaryAnd:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpAnd})
		case ast.BinaryOr:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpOr})
		default:
			return false, fmt.Errorf("ir: unsupported binary op %s", node.Op)
		}
		return true, nil
	case *ast.UnaryExpr:
		if _, err := l.lowerExpr(node.Value, fn); err != nil {
			return false, err
		}
		switch node.Op {
		case ast.UnaryNeg:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNeg})
		case ast.UnaryNot:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNot})
		default:
			return false, fmt.Errorf("ir: unsupported unary op %s", node.Op)
		}
		return true, nil
	default:
		return false, fmt.Errorf("ir: unsupported expression %T", expr)
	}
}

func (l *lowerer) pushScope() { l.scopes = append(l.scopes, map[string]symbol{}) }

func (l *lowerer) popScope() { l.scopes = l.scopes[:len(l.scopes)-1] }

func (l *lowerer) declare(name string, typ ast.TypeName, arrayLen int) string {
	slot := fmt.Sprintf("%s@%d", name, l.nextSlotID)
	l.nextSlotID++
	l.scopes[len(l.scopes)-1][name] = symbol{slot: slot, typ: typ, arrayLen: arrayLen}
	return slot
}

func (l *lowerer) lookup(name string) (symbol, bool) {
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if sym, ok := l.scopes[i][name]; ok {
			return sym, true
		}
	}
	return symbol{}, false
}

func (l *lowerer) newLabel(prefix string) string {
	label := fmt.Sprintf("%s_%d", prefix, l.nextLabel)
	l.nextLabel++
	return label
}

func stripStringQuotes(s string) string {
	if len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return s
}

func stripCharQuotes(s string) byte {
	if len(s) >= 3 {
		return s[1]
	}
	return 0
}
