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
