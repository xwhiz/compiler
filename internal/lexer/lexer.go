package lexer

import (
	"fmt"

	"github.com/xwhiz/compiler/internal/token"
)

type Lexer struct {
	input  string
	index  int
	line   int
	column int
}

func Tokenize(input string) ([]token.Token, error) {
	lx := &Lexer{
		input:  input,
		line:   1,
		column: 1,
	}

	var tokens []token.Token
	for {
		if err := lx.skipIgnored(); err != nil {
			return nil, err
		}

		pos := lx.position()
		ch, ok := lx.peek()
		if !ok {
			tokens = append(tokens, token.Token{Type: token.EOF, Lexeme: "", Pos: pos})
			return tokens, nil
		}

		switch {
		case isIdentStart(ch):
			tokens = append(tokens, lx.scanIdentifier())
		case isDigit(ch):
			tokens = append(tokens, lx.scanIntLiteral())
		default:
			tok, err := lx.scanPunctuationOrOperator()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, tok)
		}
	}
}

func (lx *Lexer) skipIgnored() error {
	for {
		ch, ok := lx.peek()
		if !ok {
			return nil
		}

		switch ch {
		case ' ', '\t', '\r', '\n':
			lx.advance()
		case '/':
			next, ok := lx.peekNext()
			if !ok || next != '/' {
				return nil
			}

			lx.advance()
			lx.advance()
			for {
				commentCh, ok := lx.peek()
				if !ok || commentCh == '\n' {
					break
				}
				lx.advance()
			}
		default:
			return nil
		}
	}
}

func (lx *Lexer) scanIdentifier() token.Token {
	pos := lx.position()
	start := lx.index
	for {
		ch, ok := lx.peek()
		if !ok || !isIdentPart(ch) {
			break
		}
		lx.advance()
	}

	lexeme := lx.input[start:lx.index]
	typeName := token.Identifier
	if keywordType, ok := token.Keywords[lexeme]; ok {
		typeName = keywordType
	}

	return token.Token{Type: typeName, Lexeme: lexeme, Pos: pos}
}

func (lx *Lexer) scanIntLiteral() token.Token {
	pos := lx.position()
	start := lx.index
	for {
		ch, ok := lx.peek()
		if !ok || !isDigit(ch) {
			break
		}
		lx.advance()
	}

	return token.Token{Type: token.IntLiteral, Lexeme: lx.input[start:lx.index], Pos: pos}
}

func (lx *Lexer) scanPunctuationOrOperator() (token.Token, error) {
	pos := lx.position()
	ch, _ := lx.peek()

	switch ch {
	case '(':
		lx.advance()
		return token.Token{Type: token.LParen, Lexeme: "(", Pos: pos}, nil
	case ')':
		lx.advance()
		return token.Token{Type: token.RParen, Lexeme: ")", Pos: pos}, nil
	case '{':
		lx.advance()
		return token.Token{Type: token.LBrace, Lexeme: "{", Pos: pos}, nil
	case '}':
		lx.advance()
		return token.Token{Type: token.RBrace, Lexeme: "}", Pos: pos}, nil
	case '[':
		lx.advance()
		return token.Token{Type: token.LBracket, Lexeme: "[", Pos: pos}, nil
	case ']':
		lx.advance()
		return token.Token{Type: token.RBracket, Lexeme: "]", Pos: pos}, nil
	case ',':
		lx.advance()
		return token.Token{Type: token.Comma, Lexeme: ",", Pos: pos}, nil
	case ';':
		lx.advance()
		return token.Token{Type: token.Semicolon, Lexeme: ";", Pos: pos}, nil
	case '+':
		lx.advance()
		return token.Token{Type: token.Plus, Lexeme: "+", Pos: pos}, nil
	case '-':
		lx.advance()
		return token.Token{Type: token.Minus, Lexeme: "-", Pos: pos}, nil
	case '*':
		lx.advance()
		return token.Token{Type: token.Star, Lexeme: "*", Pos: pos}, nil
	case '/':
		lx.advance()
		return token.Token{Type: token.Slash, Lexeme: "/", Pos: pos}, nil
	case '%':
		lx.advance()
		return token.Token{Type: token.Percent, Lexeme: "%", Pos: pos}, nil
	case '=':
		lx.advance()
		if lx.match('=') {
			return token.Token{Type: token.Equal, Lexeme: "==", Pos: pos}, nil
		}
		return token.Token{Type: token.Assign, Lexeme: "=", Pos: pos}, nil
	case '!':
		lx.advance()
		if lx.match('=') {
			return token.Token{Type: token.NotEqual, Lexeme: "!=", Pos: pos}, nil
		}
		return token.Token{Type: token.Not, Lexeme: "!", Pos: pos}, nil
	case '<':
		lx.advance()
		if lx.match('=') {
			return token.Token{Type: token.LessEq, Lexeme: "<=", Pos: pos}, nil
		}
		return token.Token{Type: token.Less, Lexeme: "<", Pos: pos}, nil
	case '>':
		lx.advance()
		if lx.match('=') {
			return token.Token{Type: token.GreaterEq, Lexeme: ">=", Pos: pos}, nil
		}
		return token.Token{Type: token.Greater, Lexeme: ">", Pos: pos}, nil
	case '&':
		lx.advance()
		if lx.match('&') {
			return token.Token{Type: token.AndAnd, Lexeme: "&&", Pos: pos}, nil
		}
	case '|':
		lx.advance()
		if lx.match('|') {
			return token.Token{Type: token.OrOr, Lexeme: "||", Pos: pos}, nil
		}
	}

	return token.Token{}, lx.errorf(pos, "unexpected character %q", ch)
}

func (lx *Lexer) match(want byte) bool {
	ch, ok := lx.peek()
	if !ok || ch != want {
		return false
	}
	if ch == want {
		lx.advance()
		return true
	}
	return false
}

func (lx *Lexer) peek() (byte, bool) {
	if lx.index >= len(lx.input) {
		return 0, false
	}
	return lx.input[lx.index], true
}

func (lx *Lexer) peekNext() (byte, bool) {
	nextIndex := lx.index + 1
	if nextIndex >= len(lx.input) {
		return 0, false
	}
	return lx.input[nextIndex], true
}

func (lx *Lexer) advance() {
	if lx.index >= len(lx.input) {
		return
	}

	ch := lx.input[lx.index]
	lx.index++
	if ch == '\n' {
		lx.line++
		lx.column = 1
		return
	}
	lx.column++
}

func (lx *Lexer) position() token.Position {
	return token.Position{Line: lx.line, Column: lx.column}
}

func (lx *Lexer) errorf(pos token.Position, format string, args ...any) error {
	return fmt.Errorf("%d:%d: %s", pos.Line, pos.Column, fmt.Sprintf(format, args...))
}

func isIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
