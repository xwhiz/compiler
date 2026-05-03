package vm

import (
	"bytes"
	"testing"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/ir"
)

func TestExecuteBuiltinPrints(t *testing.T) {
	irProgram := &ir.Program{
		Functions: []ir.Function{{
			Name:       "main",
			ReturnType: ast.TypeInt,
			Instructions: []ir.Instruction{
				{Op: ir.OpDeclareLocal, Name: "x@0", Type: ast.TypeInt},
				{Op: ir.OpPushInt, IntValue: 123},
				{Op: ir.OpStoreLocal, Name: "x@0"},
				{Op: ir.OpLoadLocal, Name: "x@0"},
				{Op: ir.OpCallBuiltin, Name: "print_int", ArgCount: 1},
				{Op: ir.OpCallBuiltin, Name: "print_newline", ArgCount: 0},
				{Op: ir.OpPushInt, IntValue: 0},
				{Op: ir.OpReturn},
			},
		}},
	}

	program, err := Compile(irProgram)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var out bytes.Buffer
	ret, err := Execute(program, &out)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if ret != 0 {
		t.Fatalf("Execute() return = %d, want 0", ret)
	}
	if out.String() != "123\n" {
		t.Fatalf("Execute() output = %q, want %q", out.String(), "123\n")
	}
}

func TestExecuteLoopAndJump(t *testing.T) {
	irProgram := &ir.Program{
		Functions: []ir.Function{{
			Name:       "main",
			ReturnType: ast.TypeInt,
			Instructions: []ir.Instruction{
				{Op: ir.OpDeclareLocal, Name: "i@0", Type: ast.TypeInt},
				{Op: ir.OpPushInt, IntValue: 0},
				{Op: ir.OpStoreLocal, Name: "i@0"},
				{Op: ir.OpLabel, Name: "while_start_0"},
				{Op: ir.OpLoadLocal, Name: "i@0"},
				{Op: ir.OpPushInt, IntValue: 3},
				{Op: ir.OpLT},
				{Op: ir.OpJumpIfZero, Name: "while_end_1"},
				{Op: ir.OpLoadLocal, Name: "i@0"},
				{Op: ir.OpCallBuiltin, Name: "print_int", ArgCount: 1},
				{Op: ir.OpCallBuiltin, Name: "print_newline", ArgCount: 0},
				{Op: ir.OpLoadLocal, Name: "i@0"},
				{Op: ir.OpPushInt, IntValue: 1},
				{Op: ir.OpAdd},
				{Op: ir.OpStoreLocal, Name: "i@0"},
				{Op: ir.OpJump, Name: "while_start_0"},
				{Op: ir.OpLabel, Name: "while_end_1"},
				{Op: ir.OpPushInt, IntValue: 0},
				{Op: ir.OpReturn},
			},
		}},
	}

	program, err := Compile(irProgram)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var out bytes.Buffer
	_, err = Execute(program, &out)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out.String() != "0\n1\n2\n" {
		t.Fatalf("Execute() output = %q, want %q", out.String(), "0\n1\n2\n")
	}
}

func TestExecuteUserFunctionCall(t *testing.T) {
	irProgram := &ir.Program{
		Functions: []ir.Function{
			{
				Name:       "add",
				ReturnType: ast.TypeInt,
				Params:     []ir.VarInfo{{Name: "a@0", Type: ast.TypeInt}, {Name: "b@1", Type: ast.TypeInt}},
				Instructions: []ir.Instruction{
					{Op: ir.OpLoadLocal, Name: "a@0"},
					{Op: ir.OpLoadLocal, Name: "b@1"},
					{Op: ir.OpAdd},
					{Op: ir.OpReturn},
				},
			},
			{
				Name:       "main",
				ReturnType: ast.TypeInt,
				Instructions: []ir.Instruction{
					{Op: ir.OpPushInt, IntValue: 7},
					{Op: ir.OpPushInt, IntValue: 8},
					{Op: ir.OpCallFunc, Name: "add", ArgCount: 2},
					{Op: ir.OpCallBuiltin, Name: "print_int", ArgCount: 1},
					{Op: ir.OpCallBuiltin, Name: "print_newline", ArgCount: 0},
					{Op: ir.OpPushInt, IntValue: 0},
					{Op: ir.OpReturn},
				},
			},
		},
	}

	program, err := Compile(irProgram)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var out bytes.Buffer
	ret, err := Execute(program, &out)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if ret != 0 {
		t.Fatalf("Execute() return = %d, want 0", ret)
	}
	if out.String() != "15\n" {
		t.Fatalf("Execute() output = %q, want %q", out.String(), "15\n")
	}
}

func TestParseTextRoundTrip(t *testing.T) {
	irProgram := &ir.Program{
		Functions: []ir.Function{{
			Name:       "main",
			ReturnType: ast.TypeInt,
			Instructions: []ir.Instruction{
				{Op: ir.OpPushInt, IntValue: 7},
				{Op: ir.OpCallBuiltin, Name: "print_int", ArgCount: 1},
				{Op: ir.OpCallBuiltin, Name: "print_newline", ArgCount: 0},
				{Op: ir.OpPushInt, IntValue: 0},
				{Op: ir.OpReturn},
			},
		}},
	}
	compiled, err := Compile(irProgram)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	text := Format(compiled)
	loaded, err := ParseText(text)
	if err != nil {
		t.Fatalf("ParseText() error = %v", err)
	}
	var out bytes.Buffer
	ret, err := Execute(loaded, &out)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if ret != 0 {
		t.Fatalf("Execute() return = %d, want 0", ret)
	}
	if out.String() != "7\n" {
		t.Fatalf("Execute() output = %q, want %q", out.String(), "7\n")
	}
}

func TestExecuteGlobalScalarAndArray(t *testing.T) {
	irProgram := &ir.Program{
		Globals: []ir.VarInfo{{Name: "g", Type: ast.TypeInt}, {Name: "msg", Type: ast.TypeChar, ArrayLen: 6}},
		GlobalInit: []ir.Instruction{
			{Op: ir.OpPushInt, IntValue: 10},
			{Op: ir.OpStoreGlobal, Name: "g"},
			{Op: ir.OpInitString, Name: "msg", StringValue: "hello"},
		},
		Functions: []ir.Function{{
			Name:       "main",
			ReturnType: ast.TypeInt,
			Instructions: []ir.Instruction{
				{Op: ir.OpPushGlobalRef, Name: "msg"},
				{Op: ir.OpCallBuiltin, Name: "print_str", ArgCount: 1},
				{Op: ir.OpCallBuiltin, Name: "print_newline", ArgCount: 0},
				{Op: ir.OpLoadGlobal, Name: "g"},
				{Op: ir.OpCallBuiltin, Name: "print_int", ArgCount: 1},
				{Op: ir.OpCallBuiltin, Name: "print_newline", ArgCount: 0},
				{Op: ir.OpLoadGlobal, Name: "g"},
				{Op: ir.OpReturn},
			},
		}},
	}

	program, err := Compile(irProgram)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	var out bytes.Buffer
	ret, err := Execute(program, &out)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if ret != 10 {
		t.Fatalf("Execute() return = %d, want 10", ret)
	}
	if out.String() != "hello\n10\n" {
		t.Fatalf("Execute() output = %q, want %q", out.String(), "hello\n10\n")
	}
}
