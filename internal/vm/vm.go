package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/ir"
)

type Op string

const (
	OpDeclareLocal  Op = "DECLARE_LOCAL"
	OpInitString    Op = "INIT_STRING"
	OpPushInt       Op = "PUSH_INT"
	OpPushFloat     Op = "PUSH_FLOAT"
	OpPushChar      Op = "PUSH_CHAR"
	OpPushString    Op = "PUSH_STRING"
	OpPushLocalRef  Op = "PUSH_LOCAL_REF"
	OpPushGlobalRef Op = "PUSH_GLOBAL_REF"
	OpLoadLocal     Op = "LOAD_LOCAL"
	OpStoreLocal    Op = "STORE_LOCAL"
	OpLoadGlobal    Op = "LOAD_GLOBAL"
	OpStoreGlobal   Op = "STORE_GLOBAL"
	OpLoadIndex     Op = "LOAD_INDEX"
	OpStoreIndex    Op = "STORE_INDEX"
	OpLoadGIndex    Op = "LOAD_GINDEX"
	OpStoreGIndex   Op = "STORE_GINDEX"
	OpAdd           Op = "ADD"
	OpSub           Op = "SUB"
	OpMul           Op = "MUL"
	OpDiv           Op = "DIV"
	OpMod           Op = "MOD"
	OpNeg           Op = "NEG"
	OpNot           Op = "NOT"
	OpLT            Op = "LT"
	OpLE            Op = "LE"
	OpGT            Op = "GT"
	OpGE            Op = "GE"
	OpEQ            Op = "EQ"
	OpNE            Op = "NE"
	OpAnd           Op = "AND"
	OpOr            Op = "OR"
	OpDup           Op = "DUP"
	OpPop           Op = "POP"
	OpJump          Op = "JUMP"
	OpJumpIfZero    Op = "JUMP_IF_ZERO"
	OpCallBuiltin   Op = "CALL_BUILTIN"
	OpCallFunc      Op = "CALL_FUNC"
	OpRet           Op = "RET"
)

type ValueKind string

const (
	KindVoid           ValueKind = "void"
	KindInt            ValueKind = "int"
	KindFloat          ValueKind = "float"
	KindChar           ValueKind = "char"
	KindString         ValueKind = "string"
	KindLocalArrayRef  ValueKind = "local_array_ref"
	KindGlobalArrayRef ValueKind = "global_array_ref"
)

type Value struct {
	Kind  ValueKind
	Int   int64
	Float float64
	Text  string
}

type Cell struct {
	Type     ast.TypeName
	ArrayLen int
	Scalar   Value
	Array    []Value
}

type Program struct {
	Globals    []ir.VarInfo
	GlobalInit []Instruction
	Functions  []Function
}

type Function struct {
	Name         string
	ReturnType   ast.TypeName
	Params       []ir.VarInfo
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
	Target      int
}

type frame struct {
	locals map[string]Cell
}

func Compile(program *ir.Program) (*Program, error) {
	out := &Program{Globals: append([]ir.VarInfo(nil), program.Globals...)}
	initInsts, err := compileInstructions(program.GlobalInit)
	if err != nil {
		return nil, err
	}
	out.GlobalInit = initInsts
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
	b.WriteString("globals\n")
	if len(program.Globals) == 0 {
		b.WriteString("  <empty>\n")
	} else {
		for _, global := range program.Globals {
			if global.ArrayLen > 0 {
				fmt.Fprintf(&b, "  %s:%s[%d]\n", global.Name, global.Type, global.ArrayLen)
			} else {
				fmt.Fprintf(&b, "  %s:%s\n", global.Name, global.Type)
			}
		}
	}
	b.WriteString("endglobals\n")
	b.WriteString("init\n")
	for _, inst := range program.GlobalInit {
		fmt.Fprintf(&b, "  %s\n", inst.String())
	}
	b.WriteString("endinit\n")
	for _, fn := range program.Functions {
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

func Execute(program *Program, stdout io.Writer) (int64, error) {
	globals := map[string]Cell{}
	for _, global := range program.Globals {
		globals[global.Name] = newCell(global.Type, global.ArrayLen)
	}
	if _, err := executeInstructions(program, program.GlobalInit, ast.TypeVoid, globals, nil, stdout); err != nil {
		return 0, err
	}
	mainFn, ok := findFunction(program, "main")
	if !ok {
		return 0, fmt.Errorf("vm: missing main function")
	}
	ret, err := executeFunction(program, mainFn, nil, globals, stdout)
	if err != nil {
		return 0, err
	}
	if ret.Kind == KindFloat {
		return int64(ret.Float), nil
	}
	return ret.Int, nil
}

func executeFunction(program *Program, fn Function, args []Value, globals map[string]Cell, stdout io.Writer) (Value, error) {
	if len(args) != len(fn.Params) {
		return Value{}, fmt.Errorf("vm: function %s wants %d args, got %d", fn.Name, len(fn.Params), len(args))
	}
	fr := &frame{locals: map[string]Cell{}}
	for i, param := range fn.Params {
		casted, err := castValue(args[i], param.Type)
		if err != nil {
			return Value{}, err
		}
		fr.locals[param.Name] = Cell{Type: param.Type, Scalar: casted}
	}
	return executeInstructions(program, fn.Instructions, fn.ReturnType, globals, fr, stdout)
}

func executeInstructions(program *Program, insts []Instruction, returnType ast.TypeName, globals map[string]Cell, fr *frame, stdout io.Writer) (Value, error) {
	stack := make([]Value, 0, 32)
	for pc := 0; pc < len(insts); pc++ {
		inst := insts[pc]
		switch inst.Op {
		case OpDeclareLocal:
			if fr == nil {
				return Value{}, fmt.Errorf("vm: DECLARE_LOCAL outside function")
			}
			fr.locals[inst.Name] = newCell(inst.Type, inst.ArrayLen)
		case OpInitString:
			if err := initString(globals, fr, inst.Name, inst.StringValue); err != nil {
				return Value{}, err
			}
		case OpPushInt:
			stack = append(stack, Value{Kind: KindInt, Int: inst.IntValue})
		case OpPushFloat:
			stack = append(stack, Value{Kind: KindFloat, Float: inst.FloatValue})
		case OpPushChar:
			stack = append(stack, Value{Kind: KindChar, Int: inst.IntValue})
		case OpPushString:
			stack = append(stack, Value{Kind: KindString, Text: inst.StringValue})
		case OpPushLocalRef:
			stack = append(stack, Value{Kind: KindLocalArrayRef, Text: inst.Name})
		case OpPushGlobalRef:
			stack = append(stack, Value{Kind: KindGlobalArrayRef, Text: inst.Name})
		case OpLoadLocal:
			cell, ok := resolveLocalCell(fr, inst.Name)
			if !ok {
				return Value{}, fmt.Errorf("vm: unknown local %q", inst.Name)
			}
			stack = append(stack, cell.Scalar)
		case OpStoreLocal:
			value, rest, err := popOne(stack)
			if err != nil {
				return Value{}, err
			}
			cell, ok := resolveLocalCell(fr, inst.Name)
			if !ok {
				return Value{}, fmt.Errorf("vm: unknown local %q", inst.Name)
			}
			casted, err := castValue(value, cell.Type)
			if err != nil {
				return Value{}, err
			}
			cell.Scalar = casted
			fr.locals[inst.Name] = cell
			stack = rest
		case OpLoadGlobal:
			cell, ok := globals[inst.Name]
			if !ok {
				return Value{}, fmt.Errorf("vm: unknown global %q", inst.Name)
			}
			stack = append(stack, cell.Scalar)
		case OpStoreGlobal:
			value, rest, err := popOne(stack)
			if err != nil {
				return Value{}, err
			}
			cell, ok := globals[inst.Name]
			if !ok {
				return Value{}, fmt.Errorf("vm: unknown global %q", inst.Name)
			}
			casted, err := castValue(value, cell.Type)
			if err != nil {
				return Value{}, err
			}
			cell.Scalar = casted
			globals[inst.Name] = cell
			stack = rest
		case OpLoadIndex:
			cell, ok := resolveLocalArray(fr, inst.Name)
			newStack, err := loadIndex(stack, cell, ok)
			if err != nil {
				return Value{}, err
			}
			stack = newStack
		case OpStoreIndex:
			cell, ok := resolveLocalArray(fr, inst.Name)
			newStack, err := storeIndex(stack, cell, ok, func(cell Cell) { fr.locals[inst.Name] = cell })
			if err != nil {
				return Value{}, err
			}
			stack = newStack
		case OpLoadGIndex:
			cell, ok := resolveGlobalArray(globals, inst.Name)
			newStack, err := loadIndex(stack, cell, ok)
			if err != nil {
				return Value{}, err
			}
			stack = newStack
		case OpStoreGIndex:
			cell, ok := resolveGlobalArray(globals, inst.Name)
			newStack, err := storeIndex(stack, cell, ok, func(cell Cell) { globals[inst.Name] = cell })
			if err != nil {
				return Value{}, err
			}
			stack = newStack
		case OpAdd, OpSub, OpMul, OpDiv, OpMod, OpLT, OpLE, OpGT, OpGE, OpEQ, OpNE, OpAnd, OpOr:
			newStack, err := evalBinary(stack, inst.Op)
			if err != nil {
				return Value{}, err
			}
			stack = newStack
		case OpNeg, OpNot:
			newStack, err := evalUnary(stack, inst.Op)
			if err != nil {
				return Value{}, err
			}
			stack = newStack
		case OpDup:
			value, _, err := popOne(stack)
			if err != nil {
				return Value{}, err
			}
			stack = append(stack, value)
		case OpPop:
			_, rest, err := popOne(stack)
			if err != nil {
				return Value{}, err
			}
			stack = rest
		case OpJump:
			pc = inst.Target - 1
		case OpJumpIfZero:
			value, rest, err := popOne(stack)
			if err != nil {
				return Value{}, err
			}
			stack = rest
			truth, err := truthy(value)
			if err != nil {
				return Value{}, err
			}
			if !truth {
				pc = inst.Target - 1
			}
		case OpCallBuiltin:
			callArgs, rest, err := popArgs(stack, inst.ArgCount)
			if err != nil {
				return Value{}, err
			}
			stack = rest
			if err := callBuiltin(globals, fr, inst.Name, callArgs, stdout); err != nil {
				return Value{}, err
			}
		case OpCallFunc:
			callArgs, rest, err := popArgs(stack, inst.ArgCount)
			if err != nil {
				return Value{}, err
			}
			stack = rest
			callee, ok := findFunction(program, inst.Name)
			if !ok {
				return Value{}, fmt.Errorf("vm: unknown function %q", inst.Name)
			}
			ret, err := executeFunction(program, callee, callArgs, globals, stdout)
			if err != nil {
				return Value{}, err
			}
			if callee.ReturnType != ast.TypeVoid {
				stack = append(stack, ret)
			}
		case OpRet:
			if returnType == ast.TypeVoid {
				return zeroValue(ast.TypeVoid), nil
			}
			if len(stack) == 0 {
				return zeroValue(returnType), nil
			}
			value, _, err := popOne(stack)
			if err != nil {
				return Value{}, err
			}
			return castValue(value, returnType)
		default:
			return Value{}, fmt.Errorf("vm: unsupported instruction %q", inst.Op)
		}
	}
	return zeroValue(returnType), nil
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
	case OpPushLocalRef, OpPushGlobalRef, OpLoadLocal, OpStoreLocal, OpLoadGlobal, OpStoreGlobal, OpLoadIndex, OpStoreIndex, OpLoadGIndex, OpStoreGIndex:
		return fmt.Sprintf("%s %s", i.Op, i.Name)
	case OpJump, OpJumpIfZero:
		return fmt.Sprintf("%s %d (%s)", i.Op, i.Target, i.Name)
	case OpCallBuiltin, OpCallFunc:
		return fmt.Sprintf("%s %s %d", i.Op, i.Name, i.ArgCount)
	default:
		return string(i.Op)
	}
}

func compileFunction(fn ir.Function) (Function, error) {
	compiled := Function{Name: fn.Name, ReturnType: fn.ReturnType, Params: append([]ir.VarInfo(nil), fn.Params...)}
	insts, err := compileInstructions(fn.Instructions)
	if err != nil {
		return Function{}, err
	}
	compiled.Instructions = insts
	return compiled, nil
}

func compileInstructions(irInsts []ir.Instruction) ([]Instruction, error) {
	labels := map[string]int{}
	pc := 0
	for _, inst := range irInsts {
		if inst.Op == ir.OpLabel {
			labels[inst.Name] = pc
			continue
		}
		pc++
	}
	compiled := make([]Instruction, 0, len(irInsts))
	for _, inst := range irInsts {
		switch inst.Op {
		case ir.OpLabel:
			continue
		case ir.OpDeclareLocal:
			compiled = append(compiled, Instruction{Op: OpDeclareLocal, Name: inst.Name, Type: inst.Type, ArrayLen: inst.ArrayLen})
		case ir.OpInitString:
			compiled = append(compiled, Instruction{Op: OpInitString, Name: inst.Name, StringValue: inst.StringValue})
		case ir.OpPushInt:
			compiled = append(compiled, Instruction{Op: OpPushInt, IntValue: inst.IntValue})
		case ir.OpPushFloat:
			compiled = append(compiled, Instruction{Op: OpPushFloat, FloatValue: inst.FloatValue})
		case ir.OpPushChar:
			compiled = append(compiled, Instruction{Op: OpPushChar, IntValue: inst.IntValue})
		case ir.OpPushString:
			compiled = append(compiled, Instruction{Op: OpPushString, StringValue: inst.StringValue})
		case ir.OpPushLocalRef:
			compiled = append(compiled, Instruction{Op: OpPushLocalRef, Name: inst.Name})
		case ir.OpPushGlobalRef:
			compiled = append(compiled, Instruction{Op: OpPushGlobalRef, Name: inst.Name})
		case ir.OpLoadLocal:
			compiled = append(compiled, Instruction{Op: OpLoadLocal, Name: inst.Name})
		case ir.OpStoreLocal:
			compiled = append(compiled, Instruction{Op: OpStoreLocal, Name: inst.Name})
		case ir.OpLoadGlobal:
			compiled = append(compiled, Instruction{Op: OpLoadGlobal, Name: inst.Name})
		case ir.OpStoreGlobal:
			compiled = append(compiled, Instruction{Op: OpStoreGlobal, Name: inst.Name})
		case ir.OpLoadIndex:
			compiled = append(compiled, Instruction{Op: OpLoadIndex, Name: inst.Name})
		case ir.OpStoreIndex:
			compiled = append(compiled, Instruction{Op: OpStoreIndex, Name: inst.Name})
		case ir.OpLoadGIndex:
			compiled = append(compiled, Instruction{Op: OpLoadGIndex, Name: inst.Name})
		case ir.OpStoreGIndex:
			compiled = append(compiled, Instruction{Op: OpStoreGIndex, Name: inst.Name})
		case ir.OpAdd:
			compiled = append(compiled, Instruction{Op: OpAdd})
		case ir.OpSub:
			compiled = append(compiled, Instruction{Op: OpSub})
		case ir.OpMul:
			compiled = append(compiled, Instruction{Op: OpMul})
		case ir.OpDiv:
			compiled = append(compiled, Instruction{Op: OpDiv})
		case ir.OpMod:
			compiled = append(compiled, Instruction{Op: OpMod})
		case ir.OpNeg:
			compiled = append(compiled, Instruction{Op: OpNeg})
		case ir.OpNot:
			compiled = append(compiled, Instruction{Op: OpNot})
		case ir.OpLT:
			compiled = append(compiled, Instruction{Op: OpLT})
		case ir.OpLE:
			compiled = append(compiled, Instruction{Op: OpLE})
		case ir.OpGT:
			compiled = append(compiled, Instruction{Op: OpGT})
		case ir.OpGE:
			compiled = append(compiled, Instruction{Op: OpGE})
		case ir.OpEQ:
			compiled = append(compiled, Instruction{Op: OpEQ})
		case ir.OpNE:
			compiled = append(compiled, Instruction{Op: OpNE})
		case ir.OpAnd:
			compiled = append(compiled, Instruction{Op: OpAnd})
		case ir.OpOr:
			compiled = append(compiled, Instruction{Op: OpOr})
		case ir.OpDup:
			compiled = append(compiled, Instruction{Op: OpDup})
		case ir.OpPop:
			compiled = append(compiled, Instruction{Op: OpPop})
		case ir.OpJump:
			target, ok := labels[inst.Name]
			if !ok {
				return nil, fmt.Errorf("vm: unknown jump label %q", inst.Name)
			}
			compiled = append(compiled, Instruction{Op: OpJump, Name: inst.Name, Target: target})
		case ir.OpJumpIfZero:
			target, ok := labels[inst.Name]
			if !ok {
				return nil, fmt.Errorf("vm: unknown jump label %q", inst.Name)
			}
			compiled = append(compiled, Instruction{Op: OpJumpIfZero, Name: inst.Name, Target: target})
		case ir.OpCallBuiltin:
			compiled = append(compiled, Instruction{Op: OpCallBuiltin, Name: inst.Name, ArgCount: inst.ArgCount})
		case ir.OpCallFunc:
			compiled = append(compiled, Instruction{Op: OpCallFunc, Name: inst.Name, ArgCount: inst.ArgCount})
		case ir.OpReturn:
			compiled = append(compiled, Instruction{Op: OpRet})
		default:
			return nil, fmt.Errorf("vm: unsupported IR op %q", inst.Op)
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

func newCell(typ ast.TypeName, arrayLen int) Cell {
	cell := Cell{Type: typ, ArrayLen: arrayLen}
	if arrayLen > 0 {
		cell.Array = make([]Value, arrayLen)
		for i := range cell.Array {
			cell.Array[i] = zeroValue(typ)
		}
		return cell
	}
	cell.Scalar = zeroValue(typ)
	return cell
}

func zeroValue(typ ast.TypeName) Value {
	switch typ {
	case ast.TypeInt:
		return Value{Kind: KindInt}
	case ast.TypeFloat:
		return Value{Kind: KindFloat}
	case ast.TypeChar:
		return Value{Kind: KindChar}
	default:
		return Value{Kind: KindVoid}
	}
}

func castValue(v Value, target ast.TypeName) (Value, error) {
	switch target {
	case ast.TypeInt:
		switch v.Kind {
		case KindInt:
			return v, nil
		case KindChar:
			return Value{Kind: KindInt, Int: v.Int}, nil
		default:
			return Value{}, fmt.Errorf("vm: cannot cast %s to int", v.Kind)
		}
	case ast.TypeFloat:
		switch v.Kind {
		case KindFloat:
			return v, nil
		case KindInt, KindChar:
			return Value{Kind: KindFloat, Float: float64(v.Int)}, nil
		default:
			return Value{}, fmt.Errorf("vm: cannot cast %s to float", v.Kind)
		}
	case ast.TypeChar:
		if v.Kind == KindChar {
			return v, nil
		}
		return Value{}, fmt.Errorf("vm: cannot cast %s to char", v.Kind)
	case ast.TypeVoid:
		return Value{Kind: KindVoid}, nil
	default:
		return Value{}, fmt.Errorf("vm: unknown target type %s", target)
	}
}

func indexAsInt(v Value) (int, error) {
	switch v.Kind {
	case KindInt, KindChar:
		return int(v.Int), nil
	default:
		return 0, fmt.Errorf("vm: array index must be int-like, got %s", v.Kind)
	}
}

func truthy(v Value) (bool, error) {
	switch v.Kind {
	case KindInt, KindChar:
		return v.Int != 0, nil
	case KindFloat:
		return v.Float != 0, nil
	default:
		return false, fmt.Errorf("vm: truthiness undefined for %s", v.Kind)
	}
}

func evalUnary(stack []Value, op Op) ([]Value, error) {
	value, rest, err := popOne(stack)
	if err != nil {
		return nil, err
	}
	switch op {
	case OpNeg:
		switch value.Kind {
		case KindFloat:
			return append(rest, Value{Kind: KindFloat, Float: -value.Float}), nil
		case KindInt, KindChar:
			return append(rest, Value{Kind: KindInt, Int: -value.Int}), nil
		default:
			return nil, fmt.Errorf("vm: unary - needs numeric value, got %s", value.Kind)
		}
	case OpNot:
		truth, err := truthy(value)
		if err != nil {
			return nil, err
		}
		if truth {
			return append(rest, Value{Kind: KindInt, Int: 0}), nil
		}
		return append(rest, Value{Kind: KindInt, Int: 1}), nil
	default:
		return nil, fmt.Errorf("vm: unsupported unary op %s", op)
	}
}

func evalBinary(stack []Value, op Op) ([]Value, error) {
	right, rest, err := popOne(stack)
	if err != nil {
		return nil, err
	}
	left, rest, err := popOne(rest)
	if err != nil {
		return nil, err
	}
	result, err := binaryResult(left, right, op)
	if err != nil {
		return nil, err
	}
	return append(rest, result), nil
}

func binaryResult(left, right Value, op Op) (Value, error) {
	if op == OpAnd || op == OpOr {
		lb, err := truthy(left)
		if err != nil {
			return Value{}, err
		}
		rb, err := truthy(right)
		if err != nil {
			return Value{}, err
		}
		if op == OpAnd {
			return boolInt(lb && rb), nil
		}
		return boolInt(lb || rb), nil
	}
	if left.Kind == KindFloat || right.Kind == KindFloat {
		lf, err := asFloat(left)
		if err != nil {
			return Value{}, err
		}
		rf, err := asFloat(right)
		if err != nil {
			return Value{}, err
		}
		switch op {
		case OpAdd:
			return Value{Kind: KindFloat, Float: lf + rf}, nil
		case OpSub:
			return Value{Kind: KindFloat, Float: lf - rf}, nil
		case OpMul:
			return Value{Kind: KindFloat, Float: lf * rf}, nil
		case OpDiv:
			if rf == 0 {
				return Value{}, fmt.Errorf("vm: division by zero")
			}
			return Value{Kind: KindFloat, Float: lf / rf}, nil
		case OpLT:
			return boolInt(lf < rf), nil
		case OpLE:
			return boolInt(lf <= rf), nil
		case OpGT:
			return boolInt(lf > rf), nil
		case OpGE:
			return boolInt(lf >= rf), nil
		case OpEQ:
			return boolInt(lf == rf), nil
		case OpNE:
			return boolInt(lf != rf), nil
		case OpMod:
			return Value{}, fmt.Errorf("vm: modulo not defined for float")
		default:
			return Value{}, fmt.Errorf("vm: unsupported binary op %s", op)
		}
	}
	li, err := asIntLike(left)
	if err != nil {
		return Value{}, err
	}
	ri, err := asIntLike(right)
	if err != nil {
		return Value{}, err
	}
	switch op {
	case OpAdd:
		return Value{Kind: KindInt, Int: li + ri}, nil
	case OpSub:
		return Value{Kind: KindInt, Int: li - ri}, nil
	case OpMul:
		return Value{Kind: KindInt, Int: li * ri}, nil
	case OpDiv:
		if ri == 0 {
			return Value{}, fmt.Errorf("vm: division by zero")
		}
		return Value{Kind: KindInt, Int: li / ri}, nil
	case OpMod:
		if ri == 0 {
			return Value{}, fmt.Errorf("vm: modulo by zero")
		}
		return Value{Kind: KindInt, Int: li % ri}, nil
	case OpLT:
		return boolInt(li < ri), nil
	case OpLE:
		return boolInt(li <= ri), nil
	case OpGT:
		return boolInt(li > ri), nil
	case OpGE:
		return boolInt(li >= ri), nil
	case OpEQ:
		return boolInt(li == ri), nil
	case OpNE:
		return boolInt(li != ri), nil
	default:
		return Value{}, fmt.Errorf("vm: unsupported binary op %s", op)
	}
}

func asFloat(v Value) (float64, error) {
	switch v.Kind {
	case KindFloat:
		return v.Float, nil
	case KindInt, KindChar:
		return float64(v.Int), nil
	default:
		return 0, fmt.Errorf("vm: expected numeric value, got %s", v.Kind)
	}
}

func asIntLike(v Value) (int64, error) {
	switch v.Kind {
	case KindInt, KindChar:
		return v.Int, nil
	default:
		return 0, fmt.Errorf("vm: expected int-like value, got %s", v.Kind)
	}
}

func boolInt(v bool) Value {
	if v {
		return Value{Kind: KindInt, Int: 1}
	}
	return Value{Kind: KindInt, Int: 0}
}

func resolveLocalCell(fr *frame, name string) (Cell, bool) {
	if fr == nil {
		return Cell{}, false
	}
	cell, ok := fr.locals[name]
	return cell, ok
}

func resolveLocalArray(fr *frame, name string) (Cell, bool) {
	cell, ok := resolveLocalCell(fr, name)
	if !ok || cell.ArrayLen == 0 {
		return Cell{}, false
	}
	return cell, true
}

func resolveGlobalArray(globals map[string]Cell, name string) (Cell, bool) {
	cell, ok := globals[name]
	if !ok || cell.ArrayLen == 0 {
		return Cell{}, false
	}
	return cell, true
}

func loadIndex(stack []Value, cell Cell, ok bool) ([]Value, error) {
	if !ok {
		return nil, fmt.Errorf("vm: unknown array")
	}
	indexVal, rest, err := popOne(stack)
	if err != nil {
		return nil, err
	}
	idx, err := indexAsInt(indexVal)
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= cell.ArrayLen {
		return nil, fmt.Errorf("vm: array index out of bounds: %d", idx)
	}
	return append(rest, cell.Array[idx]), nil
}

func storeIndex(stack []Value, cell Cell, ok bool, save func(Cell)) ([]Value, error) {
	if !ok {
		return nil, fmt.Errorf("vm: unknown array")
	}
	indexVal, rest, err := popOne(stack)
	if err != nil {
		return nil, err
	}
	value, rest, err := popOne(rest)
	if err != nil {
		return nil, err
	}
	idx, err := indexAsInt(indexVal)
	if err != nil {
		return nil, err
	}
	if idx < 0 || idx >= cell.ArrayLen {
		return nil, fmt.Errorf("vm: array index out of bounds: %d", idx)
	}
	casted, err := castValue(value, cell.Type)
	if err != nil {
		return nil, err
	}
	cell.Array[idx] = casted
	save(cell)
	return rest, nil
}

func initString(globals map[string]Cell, fr *frame, name, text string) error {
	if fr != nil {
		if cell, ok := fr.locals[name]; ok {
			if cell.ArrayLen == 0 || cell.Type != ast.TypeChar {
				return fmt.Errorf("vm: cannot init string into %q", name)
			}
			for i := range cell.Array {
				cell.Array[i] = zeroValue(ast.TypeChar)
			}
			for i := 0; i < len(text) && i < cell.ArrayLen-1; i++ {
				cell.Array[i] = Value{Kind: KindChar, Int: int64(text[i])}
			}
			fr.locals[name] = cell
			return nil
		}
	}
	cell, ok := globals[name]
	if !ok || cell.ArrayLen == 0 || cell.Type != ast.TypeChar {
		return fmt.Errorf("vm: cannot init string into %q", name)
	}
	for i := range cell.Array {
		cell.Array[i] = zeroValue(ast.TypeChar)
	}
	for i := 0; i < len(text) && i < cell.ArrayLen-1; i++ {
		cell.Array[i] = Value{Kind: KindChar, Int: int64(text[i])}
	}
	globals[name] = cell
	return nil
}

func callBuiltin(globals map[string]Cell, fr *frame, name string, args []Value, stdout io.Writer) error {
	switch name {
	case "print_int":
		if len(args) != 1 {
			return fmt.Errorf("vm: print_int wants 1 arg, got %d", len(args))
		}
		value, err := asIntLike(args[0])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%d", value)
		return err
	case "print_float":
		if len(args) != 1 {
			return fmt.Errorf("vm: print_float wants 1 arg, got %d", len(args))
		}
		value, err := asFloat(args[0])
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(stdout, "%g", value)
		return err
	case "print_char":
		if len(args) != 1 {
			return fmt.Errorf("vm: print_char wants 1 arg, got %d", len(args))
		}
		if args[0].Kind != KindChar {
			return fmt.Errorf("vm: print_char wants char, got %s", args[0].Kind)
		}
		_, err := fmt.Fprintf(stdout, "%c", rune(args[0].Int))
		return err
	case "print_str":
		if len(args) != 1 {
			return fmt.Errorf("vm: print_str wants 1 arg, got %d", len(args))
		}
		switch args[0].Kind {
		case KindString:
			_, err := io.WriteString(stdout, args[0].Text)
			return err
		case KindLocalArrayRef:
			cell, ok := resolveLocalArray(fr, args[0].Text)
			if !ok || cell.Type != ast.TypeChar {
				return fmt.Errorf("vm: print_str wants local char array ref, got %q", args[0].Text)
			}
			return printCharArray(stdout, cell)
		case KindGlobalArrayRef:
			cell, ok := resolveGlobalArray(globals, args[0].Text)
			if !ok || cell.Type != ast.TypeChar {
				return fmt.Errorf("vm: print_str wants global char array ref, got %q", args[0].Text)
			}
			return printCharArray(stdout, cell)
		default:
			return fmt.Errorf("vm: print_str wants string or char array ref, got %s", args[0].Kind)
		}
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

func printCharArray(stdout io.Writer, cell Cell) error {
	var b strings.Builder
	for _, v := range cell.Array {
		if v.Kind == KindChar && v.Int == 0 {
			break
		}
		if v.Kind != KindChar {
			return fmt.Errorf("vm: print_str found non-char array element")
		}
		b.WriteByte(byte(v.Int))
	}
	_, err := io.WriteString(stdout, b.String())
	return err
}
