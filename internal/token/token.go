package token

import "fmt"

type Type string

const (
	EOF Type = "EOF"

	Identifier Type = "IDENT"
	IntLiteral Type = "INT_LIT"

	KeywordInt    Type = "KW_INT"
	KeywordChar   Type = "KW_CHAR"
	KeywordFloat  Type = "KW_FLOAT"
	KeywordVoid   Type = "KW_VOID"
	KeywordIf     Type = "KW_IF"
	KeywordElse   Type = "KW_ELSE"
	KeywordWhile  Type = "KW_WHILE"
	KeywordReturn Type = "KW_RETURN"

	LParen    Type = "LPAREN"
	RParen    Type = "RPAREN"
	LBrace    Type = "LBRACE"
	RBrace    Type = "RBRACE"
	LBracket  Type = "LBRACKET"
	RBracket  Type = "RBRACKET"
	Comma     Type = "COMMA"
	Semicolon Type = "SEMICOLON"

	Plus      Type = "PLUS"
	Minus     Type = "MINUS"
	Star      Type = "STAR"
	Slash     Type = "SLASH"
	Percent   Type = "PERCENT"
	Assign    Type = "ASSIGN"
	Equal     Type = "EQ"
	Not       Type = "NOT"
	NotEqual  Type = "NEQ"
	Less      Type = "LT"
	LessEq    Type = "LE"
	Greater   Type = "GT"
	GreaterEq Type = "GE"
	AndAnd    Type = "AND_AND"
	OrOr      Type = "OR_OR"
)

type Position struct {
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

type Token struct {
	Type   Type
	Lexeme string
	Pos    Position
}

func (t Token) String() string {
	return fmt.Sprintf("%d:%d %-10s %q", t.Pos.Line, t.Pos.Column, t.Type, t.Lexeme)
}

var Keywords = map[string]Type{
	"int":    KeywordInt,
	"char":   KeywordChar,
	"float":  KeywordFloat,
	"void":   KeywordVoid,
	"if":     KeywordIf,
	"else":   KeywordElse,
	"while":  KeywordWhile,
	"return": KeywordReturn,
}
