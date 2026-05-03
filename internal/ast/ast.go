package ast

import (
	"fmt"
	"strings"

	"github.com/xwhiz/compiler/internal/token"
)

type TypeName string

const (
	TypeInt   TypeName = "int"
	TypeChar  TypeName = "char"
	TypeFloat TypeName = "float"
	TypeVoid  TypeName = "void"
)

type Program struct {
	Functions []*FuncDecl
}

type FuncDecl struct {
	ReturnType TypeName
	Name       string
	Pos        token.Position
	Body       *BlockStmt
}

type Stmt interface {
	stmtNode()
}

type Expr interface {
	exprNode()
}

type BlockStmt struct {
	Pos   token.Position
	Stmts []Stmt
}

func (*BlockStmt) stmtNode() {}

type ReturnStmt struct {
	Pos   token.Position
	Value Expr
}

func (*ReturnStmt) stmtNode() {}

type ExprStmt struct {
	Pos  token.Position
	Expr Expr
}

func (*ExprStmt) stmtNode() {}

type IntLiteral struct {
	Pos    token.Position
	Lexeme string
}

func (*IntLiteral) exprNode() {}

type CallExpr struct {
	Pos    token.Position
	Callee string
	Args   []Expr
}

func (*CallExpr) exprNode() {}

func FormatProgram(program *Program) string {
	var b strings.Builder
	writeLine(&b, 0, "Program")
	for _, fn := range program.Functions {
		writeFunc(&b, 1, fn)
	}
	return b.String()
}

func writeFunc(b *strings.Builder, level int, fn *FuncDecl) {
	writeLine(b, level, fmt.Sprintf("FuncDecl name=%s return=%s", fn.Name, fn.ReturnType))
	writeLine(b, level+1, "Params <empty>")
	writeLine(b, level+1, "Body")
	writeBlock(b, level+2, fn.Body)
}

func writeBlock(b *strings.Builder, level int, block *BlockStmt) {
	writeLine(b, level, "BlockStmt")
	for _, stmt := range block.Stmts {
		writeStmt(b, level+1, stmt)
	}
}

func writeStmt(b *strings.Builder, level int, stmt Stmt) {
	switch node := stmt.(type) {
	case *BlockStmt:
		writeBlock(b, level, node)
	case *ReturnStmt:
		writeLine(b, level, "ReturnStmt")
		if node.Value == nil {
			writeLine(b, level+1, "Value <nil>")
			return
		}
		writeExpr(b, level+1, node.Value)
	case *ExprStmt:
		writeLine(b, level, "ExprStmt")
		writeExpr(b, level+1, node.Expr)
	default:
		writeLine(b, level, fmt.Sprintf("<unknown stmt %T>", stmt))
	}
}

func writeExpr(b *strings.Builder, level int, expr Expr) {
	switch node := expr.(type) {
	case *IntLiteral:
		writeLine(b, level, fmt.Sprintf("IntLiteral value=%s", node.Lexeme))
	case *CallExpr:
		writeLine(b, level, fmt.Sprintf("CallExpr callee=%s", node.Callee))
		for _, arg := range node.Args {
			writeExpr(b, level+1, arg)
		}
	default:
		writeLine(b, level, fmt.Sprintf("<unknown expr %T>", expr))
	}
}

func writeLine(b *strings.Builder, level int, text string) {
	b.WriteString(strings.Repeat("    ", level))
	b.WriteString(text)
	b.WriteByte('\n')
}
