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

type BinaryOp string

const (
	BinaryAdd BinaryOp = "+"
	BinarySub BinaryOp = "-"
	BinaryMul BinaryOp = "*"
	BinaryDiv BinaryOp = "/"
	BinaryMod BinaryOp = "%"
	BinaryLT  BinaryOp = "<"
	BinaryLE  BinaryOp = "<="
	BinaryGT  BinaryOp = ">"
	BinaryGE  BinaryOp = ">="
	BinaryEQ  BinaryOp = "=="
	BinaryNE  BinaryOp = "!="
	BinaryAnd BinaryOp = "&&"
	BinaryOr  BinaryOp = "||"
)

type UnaryOp string

const (
	UnaryNeg UnaryOp = "-"
	UnaryNot UnaryOp = "!"
)

type Decl interface{ declNode() }
type Stmt interface{ stmtNode() }
type Expr interface{ exprNode() }

type Program struct {
	Decls     []Decl
	Globals   []*VarDecl
	Functions []*FuncDecl
}

type Param struct {
	Pos  token.Position
	Type TypeName
	Name string
}

type FuncDecl struct {
	ReturnType TypeName
	Name       string
	Pos        token.Position
	Params     []Param
	Body       *BlockStmt
}

func (*FuncDecl) declNode() {}

type BlockStmt struct {
	Pos   token.Position
	Stmts []Stmt
}

func (*BlockStmt) stmtNode() {}

type VarDecl struct {
	Pos      token.Position
	Type     TypeName
	Name     string
	ArrayLen int
	Init     Expr
}

func (*VarDecl) declNode() {}
func (*VarDecl) stmtNode() {}

type IfStmt struct {
	Pos  token.Position
	Cond Expr
	Then Stmt
	Else Stmt
}

func (*IfStmt) stmtNode() {}

type WhileStmt struct {
	Pos  token.Position
	Cond Expr
	Body Stmt
}

func (*WhileStmt) stmtNode() {}

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

type FloatLiteral struct {
	Pos    token.Position
	Lexeme string
}

func (*FloatLiteral) exprNode() {}

type CharLiteral struct {
	Pos    token.Position
	Lexeme string
}

func (*CharLiteral) exprNode() {}

type StringLiteral struct {
	Pos    token.Position
	Lexeme string
}

func (*StringLiteral) exprNode() {}

type IdentExpr struct {
	Pos  token.Position
	Name string
}

func (*IdentExpr) exprNode() {}

type IndexExpr struct {
	Pos   token.Position
	Name  string
	Index Expr
}

func (*IndexExpr) exprNode() {}

type CallExpr struct {
	Pos    token.Position
	Callee string
	Args   []Expr
}

func (*CallExpr) exprNode() {}

type AssignExpr struct {
	Pos    token.Position
	Target Expr
	Value  Expr
}

func (*AssignExpr) exprNode() {}

type BinaryExpr struct {
	Pos   token.Position
	Op    BinaryOp
	Left  Expr
	Right Expr
}

func (*BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Pos   token.Position
	Op    UnaryOp
	Value Expr
}

func (*UnaryExpr) exprNode() {}

func FormatProgram(program *Program) string {
	var b strings.Builder
	writeLine(&b, 0, "Program")
	for _, decl := range program.Decls {
		writeDecl(&b, 1, decl)
	}
	return b.String()
}

func writeDecl(b *strings.Builder, level int, decl Decl) {
	switch node := decl.(type) {
	case *FuncDecl:
		writeFunc(b, level, node)
	case *VarDecl:
		writeVarDecl(b, level, node, "GlobalVarDecl")
	default:
		writeLine(b, level, fmt.Sprintf("<unknown decl %T>", decl))
	}
}

func writeFunc(b *strings.Builder, level int, fn *FuncDecl) {
	writeLine(b, level, fmt.Sprintf("FuncDecl name=%s return=%s", fn.Name, fn.ReturnType))
	if len(fn.Params) == 0 {
		writeLine(b, level+1, "Params <empty>")
	} else {
		writeLine(b, level+1, "Params")
		for _, param := range fn.Params {
			writeLine(b, level+2, fmt.Sprintf("Param name=%s type=%s", param.Name, param.Type))
		}
	}
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
	case *VarDecl:
		writeVarDecl(b, level, node, "VarDeclStmt")
	case *ReturnStmt:
		writeLine(b, level, "ReturnStmt")
		if node.Value == nil {
			writeLine(b, level+1, "Value <nil>")
			return
		}
		writeExpr(b, level+1, node.Value)
	case *IfStmt:
		writeLine(b, level, "IfStmt")
		writeLine(b, level+1, "Cond")
		writeExpr(b, level+2, node.Cond)
		writeLine(b, level+1, "Then")
		writeStmt(b, level+2, node.Then)
		if node.Else != nil {
			writeLine(b, level+1, "Else")
			writeStmt(b, level+2, node.Else)
		}
	case *WhileStmt:
		writeLine(b, level, "WhileStmt")
		writeLine(b, level+1, "Cond")
		writeExpr(b, level+2, node.Cond)
		writeLine(b, level+1, "Body")
		writeStmt(b, level+2, node.Body)
	case *ExprStmt:
		writeLine(b, level, "ExprStmt")
		writeExpr(b, level+1, node.Expr)
	default:
		writeLine(b, level, fmt.Sprintf("<unknown stmt %T>", stmt))
	}
}

func writeVarDecl(b *strings.Builder, level int, node *VarDecl, label string) {
	if node.ArrayLen > 0 {
		writeLine(b, level, fmt.Sprintf("%s name=%s type=%s array=%d", label, node.Name, node.Type, node.ArrayLen))
	} else {
		writeLine(b, level, fmt.Sprintf("%s name=%s type=%s", label, node.Name, node.Type))
	}
	if node.Init != nil {
		writeLine(b, level+1, "Init")
		writeExpr(b, level+2, node.Init)
	}
}

func writeExpr(b *strings.Builder, level int, expr Expr) {
	switch node := expr.(type) {
	case *IntLiteral:
		writeLine(b, level, fmt.Sprintf("IntLiteral value=%s", node.Lexeme))
	case *FloatLiteral:
		writeLine(b, level, fmt.Sprintf("FloatLiteral value=%s", node.Lexeme))
	case *CharLiteral:
		writeLine(b, level, fmt.Sprintf("CharLiteral value=%s", node.Lexeme))
	case *StringLiteral:
		writeLine(b, level, fmt.Sprintf("StringLiteral value=%s", node.Lexeme))
	case *IdentExpr:
		writeLine(b, level, fmt.Sprintf("IdentExpr name=%s", node.Name))
	case *IndexExpr:
		writeLine(b, level, fmt.Sprintf("IndexExpr name=%s", node.Name))
		writeExpr(b, level+1, node.Index)
	case *CallExpr:
		writeLine(b, level, fmt.Sprintf("CallExpr callee=%s", node.Callee))
		for _, arg := range node.Args {
			writeExpr(b, level+1, arg)
		}
	case *AssignExpr:
		writeLine(b, level, "AssignExpr")
		writeLine(b, level+1, "Target")
		writeExpr(b, level+2, node.Target)
		writeLine(b, level+1, "Value")
		writeExpr(b, level+2, node.Value)
	case *BinaryExpr:
		writeLine(b, level, fmt.Sprintf("BinaryExpr op=%q", string(node.Op)))
		writeExpr(b, level+1, node.Left)
		writeExpr(b, level+1, node.Right)
	case *UnaryExpr:
		writeLine(b, level, fmt.Sprintf("UnaryExpr op=%q", string(node.Op)))
		writeExpr(b, level+1, node.Value)
	default:
		writeLine(b, level, fmt.Sprintf("<unknown expr %T>", expr))
	}
}

func writeLine(b *strings.Builder, level int, text string) {
	b.WriteString(strings.Repeat("    ", level))
	b.WriteString(text)
	b.WriteByte('\n')
}
