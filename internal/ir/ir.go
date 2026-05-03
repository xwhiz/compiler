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
	OpPushInt      Op = "push_int"
	OpLoadLocal    Op = "load_local"
	OpStoreLocal   Op = "store_local"
	OpAddInt       Op = "add_int"
	OpSubInt       Op = "sub_int"
	OpMulInt       Op = "mul_int"
	OpDivInt       Op = "div_int"
	OpModInt       Op = "mod_int"
	OpNegInt       Op = "neg_int"
	OpNotInt       Op = "not_int"
	OpLTInt        Op = "lt_int"
	OpLEInt        Op = "le_int"
	OpGTInt        Op = "gt_int"
	OpGEInt        Op = "ge_int"
	OpEQInt        Op = "eq_int"
	OpNEInt        Op = "ne_int"
	OpAndInt       Op = "and_int"
	OpOrInt        Op = "or_int"
	OpPop          Op = "pop"
	OpLabel        Op = "label"
	OpJump         Op = "jump"
	OpJumpIfZero   Op = "jump_if_zero"
	OpCallBuiltin  Op = "call_builtin"
	OpReturn       Op = "return"
)

type Program struct {
	Functions []Function
}

type Function struct {
	Name         string
	Instructions []Instruction
}

type Instruction struct {
	Op       Op
	IntValue int64
	Name     string
	ArgCount int
}

type lowerer struct {
	scopes     []map[string]string
	nextSlotID int
	nextLabel  int
}

func Lower(program *ast.Program) (*Program, error) {
	out := &Program{}
	for _, fn := range program.Functions {
		lowered, err := lowerFunc(fn)
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
		fmt.Fprintf(&b, "func %s\n", fn.Name)
		for _, inst := range fn.Instructions {
			fmt.Fprintf(&b, "  %s\n", inst.String())
		}
		b.WriteString("end\n")
	}
	return b.String()
}

func (i Instruction) String() string {
	switch i.Op {
	case OpDeclareLocal, OpLoadLocal, OpStoreLocal, OpLabel, OpJump, OpJumpIfZero:
		return fmt.Sprintf("%s %s", i.Op, i.Name)
	case OpPushInt:
		return fmt.Sprintf("%s %d", i.Op, i.IntValue)
	case OpCallBuiltin:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	default:
		return string(i.Op)
	}
}

func lowerFunc(fn *ast.FuncDecl) (*Function, error) {
	out := &Function{Name: fn.Name}
	l := &lowerer{}
	if err := l.lowerBlock(fn.Body, out); err != nil {
		return nil, err
	}
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
		slot := l.declare(node.Name)
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpDeclareLocal, Name: slot})
		if node.Init != nil {
			if _, err := l.lowerExpr(node.Init, fn); err != nil {
				return err
			}
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpStoreLocal, Name: slot})
		}
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
	case *ast.IdentExpr:
		slot, ok := l.lookup(node.Name)
		if !ok {
			return false, fmt.Errorf("ir: unknown local %q", node.Name)
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpLoadLocal, Name: slot})
		return true, nil
	case *ast.CallExpr:
		for _, arg := range node.Args {
			if _, err := l.lowerExpr(arg, fn); err != nil {
				return false, err
			}
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpCallBuiltin, Name: node.Callee, ArgCount: len(node.Args)})
		return false, nil
	case *ast.AssignExpr:
		slot, ok := l.lookup(node.Name)
		if !ok {
			return false, fmt.Errorf("ir: unknown local %q", node.Name)
		}
		if _, err := l.lowerExpr(node.Value, fn); err != nil {
			return false, err
		}
		fn.Instructions = append(fn.Instructions,
			Instruction{Op: OpStoreLocal, Name: slot},
			Instruction{Op: OpLoadLocal, Name: slot},
		)
		return true, nil
	case *ast.BinaryExpr:
		if _, err := l.lowerExpr(node.Left, fn); err != nil {
			return false, err
		}
		if _, err := l.lowerExpr(node.Right, fn); err != nil {
			return false, err
		}
		switch node.Op {
		case ast.BinaryAdd:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpAddInt})
		case ast.BinarySub:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpSubInt})
		case ast.BinaryMul:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpMulInt})
		case ast.BinaryDiv:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpDivInt})
		case ast.BinaryMod:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpModInt})
		case ast.BinaryLT:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpLTInt})
		case ast.BinaryLE:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpLEInt})
		case ast.BinaryGT:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpGTInt})
		case ast.BinaryGE:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpGEInt})
		case ast.BinaryEQ:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpEQInt})
		case ast.BinaryNE:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNEInt})
		case ast.BinaryAnd:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpAndInt})
		case ast.BinaryOr:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpOrInt})
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
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNegInt})
		case ast.UnaryNot:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNotInt})
		default:
			return false, fmt.Errorf("ir: unsupported unary op %s", node.Op)
		}
		return true, nil
	default:
		return false, fmt.Errorf("ir: unsupported expression %T", expr)
	}
}

func (l *lowerer) pushScope() {
	l.scopes = append(l.scopes, map[string]string{})
}

func (l *lowerer) popScope() {
	l.scopes = l.scopes[:len(l.scopes)-1]
}

func (l *lowerer) declare(name string) string {
	slot := fmt.Sprintf("%s@%d", name, l.nextSlotID)
	l.nextSlotID++
	l.scopes[len(l.scopes)-1][name] = slot
	return slot
}

func (l *lowerer) lookup(name string) (string, bool) {
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if slot, ok := l.scopes[i][name]; ok {
			return slot, true
		}
	}
	return "", false
}

func (l *lowerer) newLabel(prefix string) string {
	label := fmt.Sprintf("%s_%d", prefix, l.nextLabel)
	l.nextLabel++
	return label
}
