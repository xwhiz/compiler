package vm

import (
	"bytes"
	"testing"

	"github.com/xwhiz/compiler/internal/ir"
)

func TestExecuteBuiltinPrints(t *testing.T) {
	irProgram := &ir.Program{
		Functions: []ir.Function{{
			Name: "main",
			Instructions: []ir.Instruction{
				{Op: ir.OpPushInt, IntValue: 123},
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
