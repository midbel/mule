package mule

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

var keywords = []string{
	// configuration block
	"username",
	"password",
	"token",
	"auth",
	"collection",
	"variables",
	"headers",
	"tls",
	"default",
	"query",
	"cookie",
	"before",
	"beforeAll",
	"beforeEach",
	"after",
	"afterAll",
	"afterEach",
	"url",
	"usage",
	"description",
	"body",
	// HTTP methods
	"get",
	"post",
	"put",
	"delete",
	"patch",
}

const (
	EOF rune = -(iota + 1)
	EOL
	Quote
	Comment
	Ident
	Keyword
	Macro
	Variable
	String
	Number
	Dot
	Lbrace
	Rbrace
	Invalid
)

type Token struct {
	Literal string
	Type    rune
	Position
}

func (t Token) String() string {
	var prefix string
	switch t.Type {
	case EOF:
		return "<eof>"
	case EOL:
		return "<eol>"
	case Quote:
		return "<quote>"
	case Dot:
		return "<dot>"
	case Lbrace:
		return "<lbrace>"
	case Rbrace:
		return "<rbrace>"
	case Keyword:
		prefix = "keyword"
	case Macro:
		prefix = "macro"
	case Ident:
		prefix = "identifier"
	case String:
		prefix = "string"
	case Number:
		prefix = "number"
	case Comment:
		prefix = "comment"
	case Variable:
		prefix = "variable"
	case Invalid:
		prefix = "invalid"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
}

type Position struct {
	Line   int
	Column int
}

type cursor struct {
	char rune
	curr int
	next int
	Position
}

type Scanner struct {
	input []byte
	cursor
	old cursor

	quoted bool
	str    bytes.Buffer
}

func Scan(r io.Reader) *Scanner {
	buf, _ := io.ReadAll(r)
	buf, _ = bytes.CutPrefix(buf, []byte{0xef, 0xbb, 0xbf})
	s := Scanner{
		input: buf,
	}
	s.cursor.Line = 1
	s.read()
	s.skip(isBlank)
	return &s
}

func (s *Scanner) Scan() Token {
	defer s.reset()

	var tok Token
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}

	if s.quoted {
		if isVariable(s.char) {
			s.scanVariable(&tok)
		} else if isTemplate(s.char) {
			s.scanTemplate(&tok)
		} else {
			s.scanVerbatim(&tok)
		}
		return tok
	}

	s.skip(isSpace)
	switch {
	case isMacro(s.char):
		s.scanMacro(&tok)
	case isComment(s.char):
		s.scanComment(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	case isNL(s.char):
		s.scanNL(&tok)
	case isTemplate(s.char):
		s.scanTemplate(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isVariable(s.char):
		s.scanVariable(&tok)
	case isHeredoc(s.char, s.peek()):
		s.scanHeredoc(&tok)
	default:
		s.scanLiteral(&tok)
	}

	return tok
}

func (s *Scanner) scanMacro(tok *Token) {
	s.read()
	s.scanIdent(tok)
	if tok.Type != Ident {
		tok.Type = Invalid
	} else {
		tok.Type = Macro
	}
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	s.skip(isBlank)
	for !isNL(s.char) && !s.done() {
		s.write()
		s.read()
	}
	s.skip(isBlank)
	tok.Literal = s.literal()
	tok.Type = Comment
}

func (s *Scanner) scanIdent(tok *Token) {
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Ident

	if slices.Contains(keywords, tok.Literal) {
		tok.Type = Keyword
	}
}

func (s *Scanner) scanVerbatim(tok *Token) {
	for !s.done() && !isTemplate(s.char) && !isVariable(s.char) {
		s.write()
		s.read()
	}
	tok.Type = String
	tok.Literal = s.literal()
}

func (s *Scanner) scanLiteral(tok *Token) {
	for !s.done() && !isBlank(s.char) && !isTemplate(s.char) {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = String
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for !s.done() && !isQuote(s.char) {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = String
	if !isQuote(s.char) && s.char != quote {
		tok.Type = Invalid
		return
	}
	s.read()
}

func (s *Scanner) scanNumber(tok *Token) {
	for isDigit(s.char) && !s.done() {
		s.write()
		s.read()
	}
	if s.char == dot {
		s.write()
		s.read()
		for isDigit(s.char) && !s.done() {
			s.write()
			s.read()
		}
	}
	tok.Literal = s.literal()
	tok.Type = Number
}

func (s *Scanner) scanTemplate(tok *Token) {
	s.read()
	s.quoted = !s.quoted
	tok.Type = Quote
}

func (s *Scanner) scanHeredoc(tok *Token) {
	s.read()
	s.read()
	var (
		delim string
		body  bytes.Buffer
	)
	for !isNL(s.char) {
		s.write()
		s.read()
	}
	delim = s.literal()
	if s.done() {
		tok.Type = Invalid
		tok.Literal = s.literal()
		return
	}
	s.reset()

	var valid bool
	for !s.done() {
		s.skip(isBlank)
		s.reset()
		for !s.done() && !isNL(s.char) {
			s.write()
			s.read()
		}
		line := s.literal()
		if delim == strings.TrimSpace(line) {
			valid = true
			break
		}
		if len(line) == 0 {
			continue
		}
		body.WriteString(line)
	}
	tok.Type = String
	if !valid {
		tok.Type = Invalid
	}
	tok.Literal = body.String()
}

func (s *Scanner) scanVariable(tok *Token) {
	s.read()
	var brace bool
	if brace = s.char == lbrace; brace {
		s.read()
	}
	s.scanIdent(tok)
	if tok.Type != Ident && tok.Type != Keyword {
		tok.Type = Invalid
		return
	}
	tok.Type = Variable
	if brace && s.char != rbrace {
		tok.Type = Invalid
	} else if brace {
		s.read()
	}
}

func (s *Scanner) scanNL(tok *Token) {
	s.skip(isBlank)
	tok.Type = EOL
}

func (s *Scanner) scanPunct(tok *Token) {
	switch s.char {
	case lbrace:
		tok.Type = Lbrace
	case rbrace:
		tok.Type = Rbrace
	case dot:
		tok.Type = Dot
	default:
		tok.Type = Invalid
	}
	s.read()
	if tok.Type == Lbrace || tok.Type == Rbrace {
		s.skip(isBlank)
	}
}

func (s *Scanner) done() bool {
	return s.char == utf8.RuneError || s.char == 0
}

func (s *Scanner) read() {
	if s.curr >= len(s.input) {
		s.char = utf8.RuneError
		return
	}
	r, n := utf8.DecodeRune(s.input[s.next:])
	if r == utf8.RuneError {
		s.char = r
		s.next = len(s.input)
		return
	}
	s.old.Position = s.cursor.Position
	if r == nl {
		s.cursor.Line++
		s.cursor.Column = 0
	}
	s.cursor.Column++
	s.char, s.curr, s.next = r, s.next, s.next+n
}

func (s *Scanner) peek() rune {
	r, _ := utf8.DecodeRune(s.input[s.next:])
	return r
}

func (s *Scanner) reset() {
	s.str.Reset()
}

func (s *Scanner) write() {
	s.str.WriteRune(s.char)
}

func (s *Scanner) literal() string {
	return s.str.String()
}

func (s *Scanner) skip(accept func(rune) bool) {
	if s.done() {
		return
	}
	for accept(s.char) && !s.done() {
		s.read()
	}
}

func (s *Scanner) save() {
	s.old = s.cursor
}

func (s *Scanner) restore() {
	s.cursor = s.old
}

const (
	lbrace     = '{'
	rbrace     = '}'
	space      = ' '
	tab        = '\t'
	nl         = '\n'
	cr         = '\r'
	squote     = '\''
	dquote     = '"'
	underscore = '_'
	pound      = '#'
	dot        = '.'
	dollar     = '$'
	langle     = '<'
	arobase    = '@'
	star       = '*'
	backquote  = '`'
)

func isMacro(r rune) bool {
	return r == arobase
}

func isHeredoc(r, k rune) bool {
	return r == langle && r == k
}

func isDelim(r rune) bool {
	return isBlank(r) || isPunct(r) || isTemplate(r)
}

func isPunct(r rune) bool {
	return r == dot || r == lbrace || r == rbrace
}

func isComment(r rune) bool {
	return r == pound
}

func isVariable(r rune) bool {
	return r == dollar
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r) || r == underscore
}

func isTemplate(r rune) bool {
	return r == backquote
}

func isQuote(r rune) bool {
	return isSingle(r) || isDouble(r)
}

func isDouble(r rune) bool {
	return r == dquote
}

func isSingle(r rune) bool {
	return r == squote
}

func isSpace(r rune) bool {
	return r == space || r == tab
}

func isNL(r rune) bool {
	return r == nl || r == cr
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}
