package parser

import (
	"fmt"
	"strconv"

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
		decl, err := p.parseTopLevelDecl()
		if err != nil {
			return nil, err
		}
		program.Decls = append(program.Decls, decl)
		switch node := decl.(type) {
		case *ast.VarDecl:
			program.Globals = append(program.Globals, node)
		case *ast.FuncDecl:
			program.Functions = append(program.Functions, node)
		}
	}
	if _, err := p.consume(token.EOF, "expected end of file"); err != nil {
		return nil, err
	}
	return program, nil
}

func (p *Parser) parseTopLevelDecl() (ast.Decl, error) {
	retType, pos, err := p.parseTypeName()
	if err != nil {
		return nil, err
	}
	nameTok, err := p.consume(token.Identifier, "expected declaration name")
	if err != nil {
		return nil, err
	}
	if p.match(token.LParen) {
		params, err := p.parseParams()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(token.RParen, "expected ')' after parameter list"); err != nil {
			return nil, err
		}
		body, err := p.parseBlockStmt()
		if err != nil {
			return nil, err
		}
		return &ast.FuncDecl{ReturnType: retType, Name: nameTok.Lexeme, Pos: pos, Params: params, Body: body}, nil
	}
	return p.parseVarDeclTail(retType, pos, nameTok)
}

func (p *Parser) parseParams() ([]ast.Param, error) {
	if p.check(token.RParen) {
		return nil, nil
	}
	if p.peek().Type == token.KeywordVoid {
		typeName, pos, err := p.parseTypeName()
		if err != nil {
			return nil, err
		}
		if p.check(token.RParen) {
			if typeName != ast.TypeVoid {
				return nil, p.errorAtTokenPos(pos, "internal error: invalid void parameter list")
			}
			return nil, nil
		}
		nameTok, err := p.consume(token.Identifier, "expected parameter name")
		if err != nil {
			return nil, err
		}
		params := []ast.Param{{Pos: pos, Type: typeName, Name: nameTok.Lexeme}}
		for p.match(token.Comma) {
			param, err := p.parseParam()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
		}
		return params, nil
	}

	param, err := p.parseParam()
	if err != nil {
		return nil, err
	}
	params := []ast.Param{param}
	for p.match(token.Comma) {
		param, err := p.parseParam()
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	return params, nil
}

func (p *Parser) parseParam() (ast.Param, error) {
	typeName, pos, err := p.parseTypeName()
	if err != nil {
		return ast.Param{}, err
	}
	nameTok, err := p.consume(token.Identifier, "expected parameter name")
	if err != nil {
		return ast.Param{}, err
	}
	return ast.Param{Pos: pos, Type: typeName, Name: nameTok.Lexeme}, nil
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
	for isTypeToken(p.peek().Type) {
		decl, err := p.parseLocalVarDecl()
		if err != nil {
			return nil, err
		}
		block.Stmts = append(block.Stmts, decl)
	}
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

func (p *Parser) parseLocalVarDecl() (ast.Stmt, error) {
	typeName, pos, err := p.parseTypeName()
	if err != nil {
		return nil, err
	}
	nameTok, err := p.consume(token.Identifier, "expected variable name")
	if err != nil {
		return nil, err
	}
	decl, err := p.parseVarDeclTail(typeName, pos, nameTok)
	if err != nil {
		return nil, err
	}
	return decl.(*ast.VarDecl), nil
}

func (p *Parser) parseVarDeclTail(typeName ast.TypeName, pos token.Position, nameTok token.Token) (ast.Decl, error) {
	decl := &ast.VarDecl{Pos: pos, Type: typeName, Name: nameTok.Lexeme}
	if p.match(token.LBracket) {
		lenTok, err := p.consume(token.IntLiteral, "expected array length")
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(lenTok.Lexeme)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid array length %q", lenTok.Pos, lenTok.Lexeme)
		}
		decl.ArrayLen = length
		if _, err := p.consume(token.RBracket, "expected ']' after array length"); err != nil {
			return nil, err
		}
	}
	if p.match(token.Assign) {
		init, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		decl.Init = init
	}
	if _, err := p.consume(token.Semicolon, "expected ';' after declaration"); err != nil {
		return nil, err
	}
	return decl, nil
}

func (p *Parser) parseStmt() (ast.Stmt, error) {
	switch p.peek().Type {
	case token.LBrace:
		return p.parseBlockStmt()
	case token.KeywordIf:
		return p.parseIfStmt()
	case token.KeywordWhile:
		return p.parseWhileStmt()
	case token.KeywordReturn:
		return p.parseReturnStmt()
	default:
		return p.parseExprStmt()
	}
}

func (p *Parser) parseIfStmt() (ast.Stmt, error) {
	ifTok, err := p.consume(token.KeywordIf, "expected 'if'")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(token.LParen, "expected '(' after if"); err != nil {
		return nil, err
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(token.RParen, "expected ')' after if condition"); err != nil {
		return nil, err
	}
	thenStmt, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	stmt := &ast.IfStmt{Pos: ifTok.Pos, Cond: cond, Then: thenStmt}
	if p.match(token.KeywordElse) {
		elseStmt, err := p.parseStmt()
		if err != nil {
			return nil, err
		}
		stmt.Else = elseStmt
	}
	return stmt, nil
}

func (p *Parser) parseWhileStmt() (ast.Stmt, error) {
	whileTok, err := p.consume(token.KeywordWhile, "expected 'while'")
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(token.LParen, "expected '(' after while"); err != nil {
		return nil, err
	}
	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if _, err := p.consume(token.RParen, "expected ')' after while condition"); err != nil {
		return nil, err
	}
	body, err := p.parseStmt()
	if err != nil {
		return nil, err
	}
	return &ast.WhileStmt{Pos: whileTok.Pos, Cond: cond, Body: body}, nil
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

func (p *Parser) parseExpr() (ast.Expr, error) { return p.parseAssignment() }

func (p *Parser) parseAssignment() (ast.Expr, error) {
	left, err := p.parseLogicalOr()
	if err != nil {
		return nil, err
	}
	if !p.match(token.Assign) {
		return left, nil
	}
	switch left.(type) {
	case *ast.IdentExpr, *ast.IndexExpr:
	default:
		return nil, p.errorAtTokenPos(exprPos(left), "expected identifier or array element on left side of assignment")
	}
	right, err := p.parseAssignment()
	if err != nil {
		return nil, err
	}
	return &ast.AssignExpr{Pos: exprPos(left), Target: left, Value: right}, nil
}

func (p *Parser) parseLogicalOr() (ast.Expr, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	for p.match(token.OrOr) {
		tok := p.previous()
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Pos: tok.Pos, Op: ast.BinaryOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseLogicalAnd() (ast.Expr, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	for p.match(token.AndAnd) {
		tok := p.previous()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Pos: tok.Pos, Op: ast.BinaryAnd, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseEquality() (ast.Expr, error) {
	left, err := p.parseRelational()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		var op ast.BinaryOp
		switch tok.Type {
		case token.Equal:
			op = ast.BinaryEQ
		case token.NotEqual:
			op = ast.BinaryNE
		default:
			return left, nil
		}
		p.advance()
		right, err := p.parseRelational()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Pos: tok.Pos, Op: op, Left: left, Right: right}
	}
}

func (p *Parser) parseRelational() (ast.Expr, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		var op ast.BinaryOp
		switch tok.Type {
		case token.Less:
			op = ast.BinaryLT
		case token.LessEq:
			op = ast.BinaryLE
		case token.Greater:
			op = ast.BinaryGT
		case token.GreaterEq:
			op = ast.BinaryGE
		default:
			return left, nil
		}
		p.advance()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Pos: tok.Pos, Op: op, Left: left, Right: right}
	}
}

func (p *Parser) parseAdditive() (ast.Expr, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		var op ast.BinaryOp
		switch tok.Type {
		case token.Plus:
			op = ast.BinaryAdd
		case token.Minus:
			op = ast.BinarySub
		default:
			return left, nil
		}
		p.advance()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Pos: tok.Pos, Op: op, Left: left, Right: right}
	}
}

func (p *Parser) parseMultiplicative() (ast.Expr, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		var op ast.BinaryOp
		switch tok.Type {
		case token.Star:
			op = ast.BinaryMul
		case token.Slash:
			op = ast.BinaryDiv
		case token.Percent:
			op = ast.BinaryMod
		default:
			return left, nil
		}
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpr{Pos: tok.Pos, Op: op, Left: left, Right: right}
	}
}

func (p *Parser) parseUnary() (ast.Expr, error) {
	if p.match(token.Minus) {
		pos := p.previous().Pos
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Pos: pos, Op: ast.UnaryNeg, Value: value}, nil
	}
	if p.match(token.Not) {
		pos := p.previous().Pos
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpr{Pos: pos, Op: ast.UnaryNot, Value: value}, nil
	}
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() (ast.Expr, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch {
		case p.match(token.LParen):
			ident, ok := expr.(*ast.IdentExpr)
			if !ok {
				return nil, p.errorAtTokenPos(exprPos(expr), "expected function name before '('")
			}
			call := &ast.CallExpr{Pos: ident.Pos, Callee: ident.Name}
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
			expr = call
		case p.match(token.LBracket):
			ident, ok := expr.(*ast.IdentExpr)
			if !ok {
				return nil, p.errorAtTokenPos(exprPos(expr), "expected array name before '['")
			}
			index, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.consume(token.RBracket, "expected ']' after index"); err != nil {
				return nil, err
			}
			expr = &ast.IndexExpr{Pos: ident.Pos, Name: ident.Name, Index: index}
		default:
			return expr, nil
		}
	}
}

func (p *Parser) parsePrimary() (ast.Expr, error) {
	tok := p.peek()
	switch tok.Type {
	case token.IntLiteral:
		p.advance()
		return &ast.IntLiteral{Pos: tok.Pos, Lexeme: tok.Lexeme}, nil
	case token.FloatLiteral:
		p.advance()
		return &ast.FloatLiteral{Pos: tok.Pos, Lexeme: tok.Lexeme}, nil
	case token.CharLiteral:
		p.advance()
		return &ast.CharLiteral{Pos: tok.Pos, Lexeme: tok.Lexeme}, nil
	case token.StringLiteral:
		p.advance()
		return &ast.StringLiteral{Pos: tok.Pos, Lexeme: tok.Lexeme}, nil
	case token.Identifier:
		p.advance()
		return &ast.IdentExpr{Pos: tok.Pos, Name: tok.Lexeme}, nil
	case token.LParen:
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.consume(token.RParen, "expected ')' after expression"); err != nil {
			return nil, err
		}
		return expr, nil
	default:
		return nil, p.errorAt(tok, "expected expression")
	}
}

func (p *Parser) consume(want token.Type, message string) (token.Token, error) {
	tok := p.peek()
	if tok.Type != want {
		return token.Token{}, p.errorAt(tok, message)
	}
	p.advance()
	return tok, nil
}

func (p *Parser) check(want token.Type) bool { return p.peek().Type == want }

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

func (p *Parser) previous() token.Token {
	if p.index == 0 || p.index-1 >= len(p.tokens) {
		return token.Token{}
	}
	return p.tokens[p.index-1]
}

func (p *Parser) advance() {
	if p.index < len(p.tokens) {
		p.index++
	}
}

func (p *Parser) errorAt(tok token.Token, message string) error {
	return fmt.Errorf("%d:%d: %s, got %s %q", tok.Pos.Line, tok.Pos.Column, message, tok.Type, tok.Lexeme)
}

func (p *Parser) errorAtTokenPos(pos token.Position, message string) error {
	return fmt.Errorf("%d:%d: %s", pos.Line, pos.Column, message)
}

func exprPos(expr ast.Expr) token.Position {
	switch node := expr.(type) {
	case *ast.IntLiteral:
		return node.Pos
	case *ast.FloatLiteral:
		return node.Pos
	case *ast.CharLiteral:
		return node.Pos
	case *ast.StringLiteral:
		return node.Pos
	case *ast.IdentExpr:
		return node.Pos
	case *ast.IndexExpr:
		return node.Pos
	case *ast.CallExpr:
		return node.Pos
	case *ast.AssignExpr:
		return node.Pos
	case *ast.BinaryExpr:
		return node.Pos
	case *ast.UnaryExpr:
		return node.Pos
	default:
		return token.Position{}
	}
}

func isTypeToken(typ token.Type) bool {
	return typ == token.KeywordInt || typ == token.KeywordChar || typ == token.KeywordFloat || typ == token.KeywordVoid
}
