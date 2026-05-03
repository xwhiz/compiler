package ir

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xwhiz/compiler/internal/ast"
)

type Op string

const (
	OpPushInt     Op = "push_int"
	OpCallBuiltin Op = "call_builtin"
	OpReturn      Op = "return"
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
	case OpPushInt:
		return fmt.Sprintf("%s %d", i.Op, i.IntValue)
	case OpCallBuiltin:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	case OpReturn:
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
	case *ast.ExprStmt:
		return lowerExpr(node.Expr, fn)
	case *ast.ReturnStmt:
		if node.Value != nil {
			if err := lowerExpr(node.Value, fn); err != nil {
				return err
			}
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpReturn})
		return nil
	default:
		return fmt.Errorf("ir: unsupported statement %T", stmt)
	}
}

func lowerExpr(expr ast.Expr, fn *Function) error {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		value, err := strconv.ParseInt(node.Lexeme, 10, 64)
		if err != nil {
			return fmt.Errorf("ir: parse int literal %q: %w", node.Lexeme, err)
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpPushInt, IntValue: value})
		return nil
	case *ast.CallExpr:
		for _, arg := range node.Args {
			if err := lowerExpr(arg, fn); err != nil {
				return err
			}
		}
		fn.Instructions = append(fn.Instructions, Instruction{Op: OpCallBuiltin, Name: node.Callee, ArgCount: len(node.Args)})
		return nil
	default:
		return fmt.Errorf("ir: unsupported expression %T", expr)
	}
}
