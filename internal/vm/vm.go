package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/xwhiz/compiler/internal/ir"
)

type Op string

const (
	OpDeclareLocal Op = "DECLARE_LOCAL"
	OpPushInt      Op = "PUSH_INT"
	OpLoadLocal    Op = "LOAD_LOCAL"
	OpStoreLocal   Op = "STORE_LOCAL"
	OpAddInt       Op = "ADD_INT"
	OpSubInt       Op = "SUB_INT"
	OpMulInt       Op = "MUL_INT"
	OpDivInt       Op = "DIV_INT"
	OpModInt       Op = "MOD_INT"
	OpNegInt       Op = "NEG_INT"
	OpNotInt       Op = "NOT_INT"
	OpLTInt        Op = "LT_INT"
	OpLEInt        Op = "LE_INT"
	OpGTInt        Op = "GT_INT"
	OpGEInt        Op = "GE_INT"
	OpEQInt        Op = "EQ_INT"
	OpNEInt        Op = "NE_INT"
	OpAndInt       Op = "AND_INT"
	OpOrInt        Op = "OR_INT"
	OpPop          Op = "POP"
	OpJump         Op = "JUMP"
	OpJumpIfZero   Op = "JUMP_IF_ZERO"
	OpCallBuiltin  Op = "CALL_BUILTIN"
	OpRet          Op = "RET"
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
	Target   int
}

type Value struct {
	Int int64
}

type frame struct {
	locals map[string]Value
}

func Compile(program *ir.Program) (*Program, error) {
	out := &Program{}
	for _, fn := range program.Functions {
		compiled, err := compileFunction(fn)
		if err != nil {
			return nil, err
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

	stack := make([]Value, 0, 16)
	fr := frame{locals: map[string]Value{}}
	for pc := 0; pc < len(mainFn.Instructions); pc++ {
		inst := mainFn.Instructions[pc]
		switch inst.Op {
		case OpDeclareLocal:
			fr.locals[inst.Name] = Value{}
		case OpPushInt:
			stack = append(stack, Value{Int: inst.IntValue})
		case OpLoadLocal:
			value, ok := fr.locals[inst.Name]
			if !ok {
				return 0, fmt.Errorf("vm: unknown local %q", inst.Name)
			}
			stack = append(stack, value)
		case OpStoreLocal:
			value, rest, err := popOne(stack)
			if err != nil {
				return 0, err
			}
			if _, ok := fr.locals[inst.Name]; !ok {
				return 0, fmt.Errorf("vm: unknown local %q", inst.Name)
			}
			fr.locals[inst.Name] = value
			stack = rest
		case OpAddInt:
			var err error
			stack, err = binaryIntOp(stack, func(a, b int64) (int64, error) { return a + b, nil })
			if err != nil {
				return 0, err
			}
		case OpSubInt:
			var err error
			stack, err = binaryIntOp(stack, func(a, b int64) (int64, error) { return a - b, nil })
			if err != nil {
				return 0, err
			}
		case OpMulInt:
			var err error
			stack, err = binaryIntOp(stack, func(a, b int64) (int64, error) { return a * b, nil })
			if err != nil {
				return 0, err
			}
		case OpDivInt:
			var err error
			stack, err = binaryIntOp(stack, func(a, b int64) (int64, error) {
				if b == 0 {
					return 0, fmt.Errorf("vm: division by zero")
				}
				return a / b, nil
			})
			if err != nil {
				return 0, err
			}
		case OpModInt:
			var err error
			stack, err = binaryIntOp(stack, func(a, b int64) (int64, error) {
				if b == 0 {
					return 0, fmt.Errorf("vm: modulo by zero")
				}
				return a % b, nil
			})
			if err != nil {
				return 0, err
			}
		case OpNegInt:
			value, rest, err := popOne(stack)
			if err != nil {
				return 0, err
			}
			stack = append(rest, Value{Int: -value.Int})
		case OpNotInt:
			value, rest, err := popOne(stack)
			if err != nil {
				return 0, err
			}
			stack = append(rest, truthyBool(value.Int == 0))
		case OpLTInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a < b })
			if err != nil {
				return 0, err
			}
		case OpLEInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a <= b })
			if err != nil {
				return 0, err
			}
		case OpGTInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a > b })
			if err != nil {
				return 0, err
			}
		case OpGEInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a >= b })
			if err != nil {
				return 0, err
			}
		case OpEQInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a == b })
			if err != nil {
				return 0, err
			}
		case OpNEInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a != b })
			if err != nil {
				return 0, err
			}
		case OpAndInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a != 0 && b != 0 })
			if err != nil {
				return 0, err
			}
		case OpOrInt:
			var err error
			stack, err = compareIntOp(stack, func(a, b int64) bool { return a != 0 || b != 0 })
			if err != nil {
				return 0, err
			}
		case OpPop:
			_, rest, err := popOne(stack)
			if err != nil {
				return 0, err
			}
			stack = rest
		case OpJump:
			pc = inst.Target - 1
		case OpJumpIfZero:
			value, rest, err := popOne(stack)
			if err != nil {
				return 0, err
			}
			stack = rest
			if value.Int == 0 {
				pc = inst.Target - 1
			}
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
	case OpDeclareLocal, OpLoadLocal, OpStoreLocal:
		return fmt.Sprintf("%s %s", i.Op, i.Name)
	case OpPushInt:
		return fmt.Sprintf("%s %d", i.Op, i.IntValue)
	case OpJump, OpJumpIfZero:
		return fmt.Sprintf("%s %d (%s)", i.Op, i.Target, i.Name)
	case OpCallBuiltin:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	default:
		return string(i.Op)
	}
}

func compileFunction(fn ir.Function) (Function, error) {
	compiled := Function{Name: fn.Name}
	labels := map[string]int{}
	pc := 0
	for _, inst := range fn.Instructions {
		if inst.Op == ir.OpLabel {
			labels[inst.Name] = pc
			continue
		}
		pc++
	}

	for _, inst := range fn.Instructions {
		switch inst.Op {
		case ir.OpLabel:
			continue
		case ir.OpDeclareLocal:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpDeclareLocal, Name: inst.Name})
		case ir.OpPushInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpPushInt, IntValue: inst.IntValue})
		case ir.OpLoadLocal:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpLoadLocal, Name: inst.Name})
		case ir.OpStoreLocal:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpStoreLocal, Name: inst.Name})
		case ir.OpAddInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpAddInt})
		case ir.OpSubInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpSubInt})
		case ir.OpMulInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpMulInt})
		case ir.OpDivInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpDivInt})
		case ir.OpModInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpModInt})
		case ir.OpNegInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpNegInt})
		case ir.OpNotInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpNotInt})
		case ir.OpLTInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpLTInt})
		case ir.OpLEInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpLEInt})
		case ir.OpGTInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpGTInt})
		case ir.OpGEInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpGEInt})
		case ir.OpEQInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpEQInt})
		case ir.OpNEInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpNEInt})
		case ir.OpAndInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpAndInt})
		case ir.OpOrInt:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpOrInt})
		case ir.OpPop:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpPop})
		case ir.OpJump:
			target, ok := labels[inst.Name]
			if !ok {
				return Function{}, fmt.Errorf("vm: unknown jump label %q", inst.Name)
			}
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpJump, Name: inst.Name, Target: target})
		case ir.OpJumpIfZero:
			target, ok := labels[inst.Name]
			if !ok {
				return Function{}, fmt.Errorf("vm: unknown jump label %q", inst.Name)
			}
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpJumpIfZero, Name: inst.Name, Target: target})
		case ir.OpCallBuiltin:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpCallBuiltin, Name: inst.Name, ArgCount: inst.ArgCount})
		case ir.OpReturn:
			compiled.Instructions = append(compiled.Instructions, Instruction{Op: OpRet})
		default:
			return Function{}, fmt.Errorf("vm: unsupported IR op %q", inst.Op)
		}
	}

	return compiled, nil
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

func popOne(stack []Value) (Value, []Value, error) {
	if len(stack) == 0 {
		return Value{}, nil, fmt.Errorf("vm: stack underflow")
	}
	return stack[len(stack)-1], stack[:len(stack)-1], nil
}

func binaryIntOp(stack []Value, op func(int64, int64) (int64, error)) ([]Value, error) {
	if len(stack) < 2 {
		return nil, fmt.Errorf("vm: stack underflow: need 2 values, have %d", len(stack))
	}
	left := stack[len(stack)-2]
	right := stack[len(stack)-1]
	result, err := op(left.Int, right.Int)
	if err != nil {
		return nil, err
	}
	stack = stack[:len(stack)-2]
	stack = append(stack, Value{Int: result})
	return stack, nil
}

func compareIntOp(stack []Value, pred func(int64, int64) bool) ([]Value, error) {
	return binaryIntOp(stack, func(a, b int64) (int64, error) {
		if pred(a, b) {
			return 1, nil
		}
		return 0, nil
	})
}

func truthyBool(v bool) Value {
	if v {
		return Value{Int: 1}
	}
	return Value{Int: 0}
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
