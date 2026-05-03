package parser

import (
	"fmt"

	"github.com/xwhiz/compiler/internal/ast"
	"github.com/xwhiz/compiler/internal/token"
)

type Parser struct {
	tokens []token.Token
	index  int
}

func Parse(tokens []token.Token) (*ast.Program, error) {
	p := &Parser{tokens: tokens}
	return p.parseProgram()
}

func (p *Parser) parseProgram() (*ast.Program, error) {
	program := &ast.Program{}
	for !p.check(token.EOF) {
		fn, err := p.parseFuncDecl()
		if err != nil {
			return nil, err
		}
		program.Functions = append(program.Functions, fn)
	}

	if _, err := p.consume(token.EOF, "expected end of file"); err != nil {
		return nil, err
	}

	return program, nil
}

func (p *Parser) parseFuncDecl() (*ast.FuncDecl, error) {
	retType, pos, err := p.parseTypeName()
	if err != nil {
		return nil, err
	}

	nameTok, err := p.consume(token.Identifier, "expected function name")
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(token.LParen, "expected '(' after function name"); err != nil {
		return nil, err
	}
	if _, err := p.consume(token.RParen, "expected ')' after parameter list"); err != nil {
		return nil, err
	}

	body, err := p.parseBlockStmt()
	if err != nil {
		return nil, err
	}

	return &ast.FuncDecl{
		ReturnType: retType,
		Name:       nameTok.Lexeme,
		Pos:        pos,
		Body:       body,
	}, nil
}

func (p *Parser) parseTypeName() (ast.TypeName, token.Position, error) {
	tok := p.peek()
	switch tok.Type {
	case token.KeywordInt:
		p.advance()
		return ast.TypeInt, tok.Pos, nil
	case token.KeywordChar:
		p.advance()
		return ast.TypeChar, tok.Pos, nil
	case token.KeywordFloat:
		p.advance()
		return ast.TypeFloat, tok.Pos, nil
	case token.KeywordVoid:
		p.advance()
		return ast.TypeVoid, tok.Pos, nil
	default:
		return "", tok.Pos, p.errorAt(tok, "expected type specifier")
	}
}

func (p *Parser) parseBlockStmt() (*ast.BlockStmt, error) {
	start, err := p.consume(token.LBrace, "expected '{' to start block")
	if err != nil {
		return nil, err
	}

	block := &ast.BlockStmt{Pos: start.Pos}
	for !p.check(token.RBrace) && !p.check(token.EOF) {
		stmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		block.Stmts = append(block.Stmts, stmt)
	}

	if _, err := p.consume(token.RBrace, "expected '}' to close block"); err != nil {
		return nil, err
	}

	return block, nil
}

func (p *Parser) parseStmt() (ast.Stmt, error) {
	switch p.peek().Type {
	case token.LBrace:
		return p.parseBlockStmt()
	case token.KeywordReturn:
		return p.parseReturnStmt()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseReturnStmt() (ast.Stmt, error) {
	retTok, err := p.consume(token.KeywordReturn, "expected 'return'")
	if err != nil {
		return nil, err
	}

	stmt := &ast.ReturnStmt{Pos: retTok.Pos}
	if !p.check(token.Semicolon) {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		stmt.Value = expr
	}

	if _, err := p.consume(token.Semicolon, "expected ';' after return statement"); err != nil {
		return nil, err
	}

	return stmt, nil
}

func (p *Parser) parseExpr() (ast.Expr, error) {
	tok := p.peek()
	switch tok.Type {
	case token.IntLiteral:
		p.advance()
		return &ast.IntLiteral{Pos: tok.Pos, Lexeme: tok.Lexeme}, nil
	case token.Identifier:
		return p.parseCallExpr()
	default:
		return nil, p.errorAt(tok, "expected expression")
	}
}

func (p *Parser) parseExprStmt() (ast.Stmt, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(token.Semicolon, "expected ';' after expression statement"); err != nil {
		return nil, err
	}

	return &ast.ExprStmt{Pos: exprPos(expr), Expr: expr}, nil
}

func (p *Parser) parseCallExpr() (ast.Expr, error) {
	nameTok, err := p.consume(token.Identifier, "expected function name")
	if err != nil {
		return nil, err
	}

	if _, err := p.consume(token.LParen, "expected '(' after function name"); err != nil {
		return nil, err
	}

	call := &ast.CallExpr{Pos: nameTok.Pos, Callee: nameTok.Lexeme}
	if !p.check(token.RParen) {
		for {
			arg, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			call.Args = append(call.Args, arg)

			if !p.match(token.Comma) {
				break
			}
		}
	}

	if _, err := p.consume(token.RParen, "expected ')' after argument list"); err != nil {
		return nil, err
	}

	return call, nil
}

func (p *Parser) consume(want token.Type, message string) (token.Token, error) {
	tok := p.peek()
	if tok.Type != want {
		return token.Token{}, p.errorAt(tok, message)
	}
	p.advance()
	return tok, nil
}

func (p *Parser) check(want token.Type) bool {
	return p.peek().Type == want
}

func (p *Parser) match(want token.Type) bool {
	if !p.check(want) {
		return false
	}
	p.advance()
	return true
}

func (p *Parser) peek() token.Token {
	if p.index >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[p.index]
}

func (p *Parser) advance() {
	if p.index < len(p.tokens) {
		p.index++
	}
}

func (p *Parser) errorAt(tok token.Token, message string) error {
	return fmt.Errorf("%d:%d: %s, got %s %q", tok.Pos.Line, tok.Pos.Column, message, tok.Type, tok.Lexeme)
}

func exprPos(expr ast.Expr) token.Position {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		return node.Pos
	case *ast.CallExpr:
		return node.Pos
	default:
		return token.Position{}
	}
}
