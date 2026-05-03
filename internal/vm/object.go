package vm

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/ir"
)

func ParseText(input string) (*Program, error) {
	lines := strings.Split(input, "\n")
	program := &Program{}
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}
		if !strings.HasPrefix(line, "func ") {
			return nil, fmt.Errorf("vm object: line %d: expected function header", i+1)
		}

		fn, next, err := parseFunction(lines, i)
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
		i = next
	}
	return program, nil
}

func parseFunction(lines []string, start int) (Function, int, error) {
	header := strings.TrimSpace(lines[start])
	parts := strings.Fields(header)
	if len(parts) != 3 || parts[0] != "func" || !strings.HasPrefix(parts[2], "return=") {
		return Function{}, 0, fmt.Errorf("vm object: line %d: invalid function header", start+1)
	}
	fn := Function{Name: parts[1], ReturnType: ast.TypeName(strings.TrimPrefix(parts[2], "return="))}

	if start+1 >= len(lines) {
		return Function{}, 0, fmt.Errorf("vm object: line %d: missing params line", start+1)
	}
	paramsLine := strings.TrimSpace(lines[start+1])
	params, err := parseParamsLine(paramsLine, start+2)
	if err != nil {
		return Function{}, 0, err
	}
	fn.Params = params

	for i := start + 2; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if line == "end" {
			return fn, i + 1, nil
		}
		inst, err := parseInstruction(line, i+1)
		if err != nil {
			return Function{}, 0, err
		}
		fn.Instructions = append(fn.Instructions, inst)
	}

	return Function{}, 0, fmt.Errorf("vm object: line %d: missing end for function %s", start+1, fn.Name)
}

func parseParamsLine(line string, lineNo int) ([]ir.VarInfo, error) {
	if !strings.HasPrefix(line, "params ") {
		return nil, fmt.Errorf("vm object: line %d: expected params line", lineNo)
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, "params "))
	if rest == "<empty>" {
		return nil, nil
	}
	parts := strings.Split(rest, ",")
	params := make([]ir.VarInfo, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		pieces := strings.SplitN(part, ":", 2)
		if len(pieces) != 2 {
			return nil, fmt.Errorf("vm object: line %d: invalid param %q", lineNo, part)
		}
		name := pieces[0]
		typ, arrayLen, err := parseTypedSlot(pieces[1])
		if err != nil {
			return nil, fmt.Errorf("vm object: line %d: %w", lineNo, err)
		}
		params = append(params, ir.VarInfo{Name: name, Type: typ, ArrayLen: arrayLen})
	}
	return params, nil
}

func parseInstruction(line string, lineNo int) (Instruction, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return Instruction{}, fmt.Errorf("vm object: line %d: empty instruction", lineNo)
	}
	op := Op(parts[0])
	rest := strings.TrimSpace(strings.TrimPrefix(line, parts[0]))

	switch op {
	case OpDeclareLocal:
		fields := strings.Fields(rest)
		if len(fields) != 2 {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid DECLARE_LOCAL", lineNo)
		}
		typ, arrayLen, err := parseTypedSlot(fields[1])
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: %w", lineNo, err)
		}
		return Instruction{Op: op, Name: fields[0], Type: typ, ArrayLen: arrayLen}, nil
	case OpInitString:
		name, quoted, ok := strings.Cut(rest, " ")
		if !ok {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid INIT_STRING", lineNo)
		}
		text, err := strconv.Unquote(strings.TrimSpace(quoted))
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid string literal: %w", lineNo, err)
		}
		return Instruction{Op: op, Name: name, StringValue: text}, nil
	case OpPushInt:
		value, err := strconv.ParseInt(rest, 10, 64)
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid int: %w", lineNo, err)
		}
		return Instruction{Op: op, IntValue: value}, nil
	case OpPushFloat:
		value, err := strconv.ParseFloat(rest, 64)
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid float: %w", lineNo, err)
		}
		return Instruction{Op: op, FloatValue: value}, nil
	case OpPushChar:
		text, err := strconv.Unquote(rest)
		if err != nil || len(text) != 1 {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid char literal", lineNo)
		}
		return Instruction{Op: op, IntValue: int64(text[0])}, nil
	case OpPushString:
		text, err := strconv.Unquote(rest)
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid string literal: %w", lineNo, err)
		}
		return Instruction{Op: op, StringValue: text}, nil
	case OpPushLocalRef, OpLoadLocal, OpStoreLocal, OpLoadIndex, OpStoreIndex:
		return Instruction{Op: op, Name: rest}, nil
	case OpCallBuiltin, OpCallFunc:
		fields := strings.Fields(rest)
		if len(fields) != 2 {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid call instruction", lineNo)
		}
		argc, err := strconv.Atoi(fields[1])
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid arg count: %w", lineNo, err)
		}
		return Instruction{Op: op, Name: fields[0], ArgCount: argc}, nil
	case OpJump, OpJumpIfZero:
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid jump instruction", lineNo)
		}
		target, err := strconv.Atoi(fields[0])
		if err != nil {
			return Instruction{}, fmt.Errorf("vm object: line %d: invalid jump target: %w", lineNo, err)
		}
		name := ""
		if len(fields) > 1 {
			name = strings.Trim(strings.Join(fields[1:], " "), "()")
		}
		return Instruction{Op: op, Name: name, Target: target}, nil
	case OpAdd, OpSub, OpMul, OpDiv, OpMod, OpNeg, OpNot, OpLT, OpLE, OpGT, OpGE, OpEQ, OpNE, OpAnd, OpOr, OpDup, OpPop, OpRet:
		return Instruction{Op: op}, nil
	default:
		return Instruction{}, fmt.Errorf("vm object: line %d: unknown opcode %s", lineNo, op)
	}
}

func parseTypedSlot(spec string) (ast.TypeName, int, error) {
	if open := strings.IndexByte(spec, '['); open >= 0 {
		if !strings.HasSuffix(spec, "]") {
			return "", 0, fmt.Errorf("invalid type spec %q", spec)
		}
		typ := ast.TypeName(spec[:open])
		length, err := strconv.Atoi(spec[open+1 : len(spec)-1])
		if err != nil {
			return "", 0, fmt.Errorf("invalid array length in %q", spec)
		}
		return typ, length, nil
	}
	return ast.TypeName(spec), 0, nil
}
