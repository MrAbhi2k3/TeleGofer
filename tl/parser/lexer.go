package parser

import (
	"strings"
	"unicode"
)

// TokenType identifies the lexical category of a TL token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenSectionTypes
	TokenSectionFunctions
	TokenColon
	TokenEquals
	TokenSemicolon
	TokenQuestion
	TokenHash
	TokenHexID
	TokenAngleOpen
	TokenAngleClose
	TokenBraceOpen
	TokenBraceClose
)

// Token represents a single lexical token.
type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// Lexer breaks raw TL text into a sequence of tokens.
type Lexer struct {
	input []rune
	pos   int
	line  int
}

// NewLexer creates a Lexer from a raw string.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: []rune(input),
		pos:   0,
		line:  1,
	}
}

// NextToken returns the next non-whitespace, non-comment token.
func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Line: l.line}
	}

	startLine := l.line
	ch := l.input[l.pos]

	// Section markers: ---types--- and ---functions---
	if ch == '-' && l.peek(1) == '-' && l.peek(2) == '-' {
		l.pos += 3
		var word strings.Builder
		for l.pos < len(l.input) && (unicode.IsLetter(l.input[l.pos]) || l.input[l.pos] == '-') {
			word.WriteRune(l.input[l.pos])
			l.pos++
		}
		str := strings.Trim(word.String(), "-")
		if str == "types" {
			return Token{Type: TokenSectionTypes, Value: "types", Line: startLine}
		}
		if str == "functions" {
			return Token{Type: TokenSectionFunctions, Value: "functions", Line: startLine}
		}
	}

	// Single-character symbols
	switch ch {
	case ':':
		l.pos++
		return Token{Type: TokenColon, Value: ":", Line: startLine}
	case '=':
		l.pos++
		return Token{Type: TokenEquals, Value: "=", Line: startLine}
	case ';':
		l.pos++
		return Token{Type: TokenSemicolon, Value: ";", Line: startLine}
	case '?':
		l.pos++
		return Token{Type: TokenQuestion, Value: "?", Line: startLine}
	case '<':
		l.pos++
		return Token{Type: TokenAngleOpen, Value: "<", Line: startLine}
	case '>':
		l.pos++
		return Token{Type: TokenAngleClose, Value: ">", Line: startLine}
	case '{':
		l.pos++
		return Token{Type: TokenBraceOpen, Value: "{", Line: startLine}
	case '}':
		l.pos++
		return Token{Type: TokenBraceClose, Value: "}", Line: startLine}
	case '#':
		l.pos++
		// Check if immediately followed by hex digits (e.g. #b304a621)
		if l.pos < len(l.input) && isHexDigit(l.input[l.pos]) {
			var hexStr strings.Builder
			for l.pos < len(l.input) && isHexDigit(l.input[l.pos]) {
				hexStr.WriteRune(l.input[l.pos])
				l.pos++
			}
			return Token{Type: TokenHexID, Value: hexStr.String(), Line: startLine}
		}
		// Standalone # (used in TL flags definition, e.g. flags:#)
		return Token{Type: TokenHash, Value: "#", Line: startLine}
	}

	// Identifiers (including dotted names like auth.sendCode, flags.0, Vector)
	if isIdentStart(ch) {
		var ident strings.Builder
		for l.pos < len(l.input) && isIdentPart(l.input[l.pos]) {
			ident.WriteRune(l.input[l.pos])
			l.pos++
		}
		return Token{Type: TokenIdent, Value: ident.String(), Line: startLine}
	}

	l.pos++
	return Token{Type: TokenIdent, Value: string(ch), Line: startLine}
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\n' {
			l.line++
			l.pos++
			continue
		}
		if unicode.IsSpace(ch) {
			l.pos++
			continue
		}
		// Skip line comments: // ...
		if ch == '/' && l.peek(1) == '/' {
			l.pos += 2
			for l.pos < len(l.input) && l.input[l.pos] != '\n' {
				l.pos++
			}
			continue
		}
		break
	}
}

func (l *Lexer) peek(offset int) rune {
	idx := l.pos + offset
	if idx < len(l.input) {
		return l.input[idx]
	}
	return 0
}

func isIdentStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_' || ch == '%'
}

func isIdentPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.' || ch == '%'
}

func isHexDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}
