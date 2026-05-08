package lexer

import (
	"testing"

	"github.com/xwhiz/compiler/internal/token"
)

func TestTokenizeSkipsCommentsAndTracksLines(t *testing.T) {
	source := "int main() {\n    // comment\n    return 42;\n}\n"

	tokens, err := Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	got := []token.Token{
		tokens[0], tokens[1], tokens[2], tokens[3], tokens[4],
		tokens[5], tokens[6], tokens[7], tokens[8], tokens[9],
	}
	want := []token.Token{
		{Type: token.KeywordInt, Lexeme: "int", Pos: token.Position{Line: 1, Column: 1}},
		{Type: token.Identifier, Lexeme: "main", Pos: token.Position{Line: 1, Column: 5}},
		{Type: token.LParen, Lexeme: "(", Pos: token.Position{Line: 1, Column: 9}},
		{Type: token.RParen, Lexeme: ")", Pos: token.Position{Line: 1, Column: 10}},
		{Type: token.LBrace, Lexeme: "{", Pos: token.Position{Line: 1, Column: 12}},
		{Type: token.KeywordReturn, Lexeme: "return", Pos: token.Position{Line: 3, Column: 5}},
		{Type: token.IntLiteral, Lexeme: "42", Pos: token.Position{Line: 3, Column: 12}},
		{Type: token.Semicolon, Lexeme: ";", Pos: token.Position{Line: 3, Column: 14}},
		{Type: token.RBrace, Lexeme: "}", Pos: token.Position{Line: 4, Column: 1}},
		{Type: token.EOF, Lexeme: "", Pos: token.Position{Line: 5, Column: 1}},
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestTokenizeRejectsUnexpectedCharacter(t *testing.T) {
	_, err := Tokenize("@")
	if err == nil {
		t.Fatal("Tokenize() error = nil, want non-nil")
	}

	if err.Error() != "1:1: unexpected character '@'" {
		t.Fatalf("Tokenize() error = %q, want %q", err.Error(), "1:1: unexpected character '@'")
	}
}

func TestTokenizeFloatCharAndStringLiterals(t *testing.T) {
	source := "float x = 2.5; char c = 'A'; print_str(\"hi\");"
	tokens, err := Tokenize(source)
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}

	wantTypes := []token.Type{
		token.KeywordFloat, token.Identifier, token.Assign, token.FloatLiteral, token.Semicolon,
		token.KeywordChar, token.Identifier, token.Assign, token.CharLiteral, token.Semicolon,
		token.Identifier, token.LParen, token.StringLiteral, token.RParen, token.Semicolon, token.EOF,
	}
	if len(tokens) != len(wantTypes) {
		t.Fatalf("len(tokens) = %d, want %d", len(tokens), len(wantTypes))
	}
	for i, want := range wantTypes {
		if tokens[i].Type != want {
			t.Fatalf("token %d type = %s, want %s", i, tokens[i].Type, want)
		}
	}
	if tokens[3].Lexeme != "2.5" || tokens[8].Lexeme != "'A'" || tokens[12].Lexeme != "\"hi\"" {
		t.Fatalf("literal lexemes = %q %q %q", tokens[3].Lexeme, tokens[8].Lexeme, tokens[12].Lexeme)
	}
}

func TestTokenizeRejectsUnterminatedString(t *testing.T) {
	_, err := Tokenize("\"oops")
	if err == nil {
		t.Fatal("Tokenize() error = nil, want non-nil")
	}
	if err.Error() != "1:1: unterminated string literal" {
		t.Fatalf("Tokenize() error = %q, want %q", err.Error(), "1:1: unterminated string literal")
	}
}

func TestTokenizeRejectsInvalidNumericTokenStartingIdentifier(t *testing.T) {
	_, err := Tokenize("int 1res = 10;")
	if err == nil {
		t.Fatal("Tokenize() error = nil, want non-nil")
	}
	if err.Error() != "1:5: invalid numeric token \"1res\"" {
		t.Fatalf("Tokenize() error = %q, want %q", err.Error(), "1:5: invalid numeric token \"1res\"")
	}
}

func TestTokenizeRejectsInvalidFloatTokenStartingIdentifier(t *testing.T) {
	_, err := Tokenize("float 2.5value = 1;")
	if err == nil {
		t.Fatal("Tokenize() error = nil, want non-nil")
	}
	if err.Error() != "1:7: invalid numeric token \"2.5value\"" {
		t.Fatalf("Tokenize() error = %q, want %q", err.Error(), "1:7: invalid numeric token \"2.5value\"")
	}
}

func TestTokenizeAllowsIdentifierWithTrailingDigits(t *testing.T) {
	tokens, err := Tokenize("int x1 = 10;")
	if err != nil {
		t.Fatalf("Tokenize() error = %v", err)
	}
	if tokens[1].Type != token.Identifier || tokens[1].Lexeme != "x1" {
		t.Fatalf("tokens[1] = %#v, want identifier x1", tokens[1])
	}
}
