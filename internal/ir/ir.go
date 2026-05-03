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
	OpPop          Op = "pop"
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
	case OpDeclareLocal, OpLoadLocal, OpStoreLocal:
		return fmt.Sprintf("%s %s", i.Op, i.Name)
	case OpPushInt:
		return fmt.Sprintf("%s %d", i.Op, i.IntValue)
	case OpCallBuiltin:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	case OpReturn, OpAddInt, OpSubInt, OpMulInt, OpDivInt, OpModInt, OpNegInt, OpPop:
		return string(i.Op)
	default:
		return string(i.Op)
	}
}

func lowerFunc(fn *ast.FuncDecl) (*Function, error) {
	out := &Function{Name: fn.Name}
	for _, stmt := range fn.Body.Stmts {
		if err := lowerStmt(stmt, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func lowerStmt(stmt ast.Stmt, fn *Function) error {
	switch node := stmt.(type) {
	case *ast.BlockStmt:
		for _, inner := range node.Stmts {
			if err := lowerStmt(inner, fn); err != nil {
				return err
			}
		}
		return nil
	case *ast.VarDeclStmt:
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpDeclareLocal, Name: node.Name})
		if node.Init != nil {
			if _, err := lowerExpr(node.Init, fn); err != nil {
				return err
			}
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpStoreLocal, Name: node.Name})
		}
		return nil
	case *ast.ExprStmt:
		pushesValue, err := lowerExpr(node.Expr, fn)
		if err != nil {
			return err
		}
		if pushesValue {
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpPop})
		}
		return nil
	case *ast.ReturnStmt:
		if node.Value != nil {
			if _, err := lowerExpr(node.Value, fn); err != nil {
				return err
			}
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpReturn})
		return nil
	default:
		return fmt.Errorf("ir: unsupported statement %T", stmt)
	}
}

func lowerExpr(expr ast.Expr, fn *Function) (bool, error) {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		value, err := strconv.ParseInt(node.Lexeme, 10, 64)
		if err != nil {
			return false, fmt.Errorf("ir: parse int literal %q: %w", node.Lexeme, err)
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushInt, IntValue: value})
		return true, nil
	case *ast.IdentExpr:
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpLoadLocal, Name: node.Name})
		return true, nil
	case *ast.CallExpr:
		for _, arg := range node.Args {
			if _, err := lowerExpr(arg, fn); err != nil {
				return false, err
			}
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpCallBuiltin, Name: node.Callee, ArgCount: len(node.Args)})
		return false, nil
	case *ast.AssignExpr:
		if _, err := lowerExpr(node.Value, fn); err != nil {
			return false, err
		}
		fn.Instructions = append(fn.Instructions,
			Instruction{Op: OpStoreLocal, Name: node.Name},
			Instruction{Op: OpLoadLocal, Name: node.Name},
		)
		return true, nil
	case *ast.BinaryExpr:
		if _, err := lowerExpr(node.Left, fn); err != nil {
			return false, err
		}
		if _, err := lowerExpr(node.Right, fn); err != nil {
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
		default:
			return false, fmt.Errorf("ir: unsupported binary op %s", node.Op)
		}
		return true, nil
	case *ast.UnaryExpr:
		if _, err := lowerExpr(node.Value, fn); err != nil {
			return false, err
		}
		switch node.Op {
		case ast.UnaryNeg:
			fn.Instructions = append(fn.Instructions, Instruction{Op: OpNegInt})
		default:
			return false, fmt.Errorf("ir: unsupported unary op %s", node.Op)
		}
		return true, nil
	default:
		return false, fmt.Errorf("ir: unsupported expression %T", expr)
	}
}
