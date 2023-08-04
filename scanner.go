package mule

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"
)

var keywords = []string{
	// configuration block
	"collection",
	"variables",
	"headers",
	"tls",
	"default",
	"query",
	"cookie",
	// HTTP methods
	"get",
	"post",
	"put",
	"delete",
	"patch",
	"head",
	"option",
}

func isSpecial(str string) bool {
	sort.Strings(keywords)
	i := sort.SearchStrings(keywords, str)
	return i < len(keywords) && keywords[i] == str
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
	Type    rune
	Literal string
	Offset  int
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

	if !s.quoted {
		s.skip(isSpace)
	}

	var tok Token
	tok.Offset = s.curr
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}

	if s.quoted && !isVariable(s.char) && !isDouble(s.char) {
		s.scanVerbatim(&tok)
		return tok
	}

	switch {
	case isNL(s.char):
		s.scanNL(&tok)
	case isComment(s.char):
		s.scanComment(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isVariable(s.char):
		s.scanVariable(&tok)
	case isHeredoc(s.char, s.peek()):
		s.scanHeredoc(&tok)
	case isDouble(s.char):
		s.scanQuote(&tok)
	case isSingle(s.char):
		s.scanString(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isMacro(s.char):
		s.scanMacro(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	default:
		tok.Type = Invalid
	}
	return tok
}

func (s *Scanner) scanQuote(tok *Token) {
	s.read()
	s.quoted = !s.quoted
	tok.Type = Quote
}

func (s *Scanner) scanVerbatim(tok *Token) {
	for !s.done() && !isDouble(s.char) && !isVariable(s.char) {
		s.write()
		s.read()
	}
	tok.Type = String
	tok.Literal = s.literal()
	if s.done() {
		tok.Type = Invalid
	}
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
	for !s.done() {
		s.skip(isBlank)
		for !s.done() && !isNL(s.char) {
			s.write()
			s.read()
		}
		line := s.literal()
		if line == delim {
			break
		}
		if len(line) == 0 {
			continue
		}
		body.WriteString(line)
	}
	tok.Type = String
	tok.Literal = body.String()
}

func (s *Scanner) scanNL(tok *Token) {
	s.skip(isBlank)
	tok.Type = EOL
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	s.skip(isBlank)
	for !isNL(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Comment
}

func (s *Scanner) scanVariable(tok *Token) {
	s.read()
	var brace bool
	if brace = s.char == lbrace; brace {
		s.read()
	}
	s.scanIdent(tok)
	if tok.Type != Ident {
		tok.Type = Invalid
		return
	}
	tok.Type = Variable
	if brace && s.char != rbrace {
		tok.Type = Invalid
	} else {
		s.read()
	}
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

func (s *Scanner) scanIdent(tok *Token) {
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Ident

	if isSpecial(tok.Literal) {
		tok.Type = Keyword
	}
}

func (s *Scanner) scanString(tok *Token) {
	s.read()
	for !s.done() && !isSingle(s.char) {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = String
	if !isSingle(s.char) {
		tok.Type = Invalid
	}
	if tok.Type == String {
		s.read()
	}
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

func (s *Scanner) unread() {
	c, z := utf8.DecodeRune(s.input[s.curr:])
	s.char, s.curr, s.next = c, s.curr-z, s.curr
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
)

func isMacro(r rune) bool {
	return r == arobase
}

func isHeredoc(r, k rune) bool {
	return r == langle && r == k
}

func isDelim(r rune) bool {
	return isBlank(r) || isPunct(r)
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

func isSpace(r rune) bool {
	return r == space || r == tab
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

func isNL(r rune) bool {
	return r == nl || r == cr
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}
