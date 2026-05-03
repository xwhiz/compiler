package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/xwhiz/compiler/internal/ir"
)

type Op string

const (
	OpPushInt     Op = "PUSH_INT"
	OpCallBuiltin Op = "CALL_BUILTIN"
	OpRet         Op = "RET"
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

type Value struct {
	Int int64
}

func Compile(program *ir.Program) (*Program, error) {
	out := &Program{}
	for _, fn := range program.Functions {
		compiled := Function{Name: fn.Name}
		for _, inst := range fn.Instructions {
			switch inst.Op {
			case ir.OpPushInt:
				compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpPushInt, IntValue: inst.IntValue})
			case ir.OpCallBuiltin:
				compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpCallBuiltin, Name: inst.Name, ArgCount: inst.ArgCount})
			case ir.OpReturn:
				compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpRet})
			default:
				return nil, fmt.Errorf("vm: unsupported IR op %q", inst.Op)
			}
		}
		out.Functions = append(out.Functions, compiled)
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

func Execute(program *Program, stdout io.Writer) (int64, error) {
	mainFn, ok := findFunction(program, "main")
	if !ok {
		return 0, fmt.Errorf("vm: missing main function")
	}

	stack := make([]Value, 0, 8)
	for pc := 0; pc < len(mainFn.Instructions); pc++ {
		inst := mainFn.Instructions[pc]
		switch inst.Op {
		case OpPushInt:
			stack = append(stack, Value{Int: inst.IntValue})
		case OpCallBuiltin:
			args, rest, err := popArgs(stack, inst.ArgCount)
			if err != nil {
				return 0, err
			}
			stack = rest
			if err := callBuiltin(inst.Name, args, stdout); err != nil {
				return 0, err
			}
		case OpRet:
			if len(stack) == 0 {
				return 0, nil
			}
			return stack[len(stack)-1].Int, nil
		default:
			return 0, fmt.Errorf("vm: unsupported instruction %q", inst.Op)
		}
	}

	return 0, nil
}

func (i Instruction) String() string {
	switch i.Op {
	case OpPushInt:
		return fmt.Sprintf("%s %d", i.Op, i.IntValue)
	case OpCallBuiltin:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	case OpRet:
		return string(i.Op)
	default:
		return string(i.Op)
	}
}

func findFunction(program *Program, name string) (Function, bool) {
	for _, fn := range program.Functions {
		if fn.Name == name {
			return fn, true
		}
	}
	return Function{}, false
}

func popArgs(stack []Value, argc int) ([]Value, []Value, error) {
	if argc > len(stack) {
		return nil, nil, fmt.Errorf("vm: stack underflow: need %d args, have %d", argc, len(stack))
	}
	start := len(stack) - argc
	args := append([]Value(nil), stack[start:]...)
	return args, stack[:start], nil
}

func callBuiltin(name string, args []Value, stdout io.Writer) error {
	switch name {
	case "print_int":
		if len(args) != 1 {
			return fmt.Errorf("vm: print_int wants 1 arg, got %d", len(args))
		}
		_, err := fmt.Fprintf(stdout, "%d", args[0].Int)
		return err
	case "print_newline":
		if len(args) != 0 {
			return fmt.Errorf("vm: print_newline wants 0 args, got %d", len(args))
		}
		_, err := io.WriteString(stdout, "\n")
		return err
	default:
		return fmt.Errorf("vm: unknown builtin %q", name)
	}
}
