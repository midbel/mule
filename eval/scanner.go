package eval

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"
)

var keywords = []string{
	"let",
	"const",
	"for",
	"if",
	"switch",
	"case",
	"function",
	"return",
	"break",
	"continue",
	"try",
	"catch",
	"while",
	"do",
	"null",
	"undefined",
	"true",
	"false",
	"delete",
	"typeof",
	"new",
}

func isKeyword(str string) bool {
	sort.Strings(keywords)
	i := sort.SearchStrings(keywords, str)
	return i < len(keywords) && keywords[i] == str
}

const (
	EOF rune = -(iota + 1)
	EOL
	Comment
	Ident
	Keyword
	String
	Number
	Boolean
	Dot
	Lparen
	Rparen
	Lsquare
	Rsquare
	Lbrace
	Rbrace
	Assign
	Not
	And
	Or
	Eq
	Ne
	Lt
	Le
	Gt
	Ge
	Add
	Sub
	Mul
	Div
	Pow
	Mod
	Lshift
	Rshift
	Band
	Bor
	Bnot
	Comma
	Colon
	Question
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
	case Dot:
		return "<dot>"
	case Lbrace:
		return "<lbrace>"
	case Rbrace:
		return "<rbrace>"
	case Lparen:
		return "<lparen>"
	case Rparen:
		return "<rparen>"
	case Lsquare:
		return "<lsquare>"
	case Rsquare:
		return "<rsquare>"
	case Colon:
		return "<colon>"
	case Comma:
		return "<comma>"
	case Question:
		return "<question>"
	case Assign:
		return "<assign>"
	case Add:
		return "<add>"
	case Sub:
		return "<sub>"
	case Div:
		return "<div>"
	case Mul:
		return "<mul>"
	case Pow:
		return "<pow>"
	case Mod:
		return "<mod>"
	case Not:
		return "<not>"
	case Eq:
		return "<eq>"
	case Ne:
		return "<ne>"
	case Lt:
		return "<lt>"
	case Le:
		return "<le>"
	case Gt:
		return "<gt>"
	case Ge:
		return "<ge>"
	case Keyword:
		prefix = "keyword"
	case String:
		prefix = "string"
	case Number:
		prefix = "number"
	case Boolean:
		prefix = "boolean"
	case Comment:
		prefix = "comment"
	case Ident:
		prefix = "identifier"
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

	str bytes.Buffer
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

	s.skip(isBlank)

	var tok Token
	tok.Offset = s.curr
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}

	switch {
	case isComment(s.char, s.peek()):
		s.scanComment(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isEOL(s.char):
		s.scanEOL(&tok)
	default:
		s.scanPunct(&tok)
	}
	return tok
}

func (s *Scanner) scanEOL(tok *Token) {
	s.skip(isEOL)
	tok.Type = EOL
}

func (s *Scanner) scanComment(tok *Token) {
	s.read()
	s.read()
	s.skip(isSpace)
	for !s.done() && !isNL(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Comment
	tok.Literal = s.literal()
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	for !s.done() && s.char != quote {
		s.write()
		s.read()
	}
	tok.Type = String
	if s.char != quote {
		tok.Type = Invalid
	} else {
		s.read()
	}
	tok.Literal = s.literal()
}

func (s *Scanner) scanNumber(tok *Token) {
	for !s.done() && isDigit(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Number
	tok.Literal = s.literal()
	if s.char != dot {
		return
	}
	s.write()
	s.read()
	for !s.done() && isDigit(s.char) {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
}

func (s *Scanner) scanIdent(tok *Token) {
	for !s.done() && isAlpha(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Ident
	tok.Literal = s.literal()
	if isKeyword(tok.Literal) {
		tok.Type = Keyword
	}
	if tok.Literal == "true" || tok.Literal == "false" {
		tok.Type = Boolean
	}
}

func (s *Scanner) scanPunct(tok *Token) {
	switch s.char {
	case dot:
		tok.Type = Dot
	case comma:
		tok.Type = Comma
	case colon:
		tok.Type = Colon
	case lbrace:
		tok.Type = Lbrace
	case rbrace:
		tok.Type = Rbrace
	case lparen:
		tok.Type = Lparen
	case rparen:
		tok.Type = Rparen
	case lsquare:
		tok.Type = Lsquare
	case rsquare:
		tok.Type = Rsquare
	case plus:
		tok.Type = Add
	case minus:
		tok.Type = Sub
	case star:
		tok.Type = Mul
		if s.peek() == star {
			s.read()
			tok.Type = Pow
		}
	case slash:
		tok.Type = Div
	case percent:
		tok.Type = Mod
	case ampersand:
		tok.Type = Band
		if s.peek() == ampersand {
			s.read()
			tok.Type = And
		}
	case pipe:
		tok.Type = Bor
		if s.peek() == pipe {
			s.read()
			tok.Type = Or
		}
	case equal:
		tok.Type = Assign
		if s.peek() == equal {
			s.read()
			tok.Type = Eq
		}
	case bang:
		tok.Type = Not
		if s.peek() == equal {
			s.read()
			tok.Type = Ne
		}
	case langle:
		tok.Type = Lt
		if s.peek() == equal {
			s.read()
			tok.Type = Le
		}
	case rangle:
		tok.Type = Gt
		if s.peek() == equal {
			s.read()
			tok.Type = Ge
		}
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
	lparen     = '('
	rparen     = ')'
	lsquare    = '['
	rsquare    = ']'
	langle     = '<'
	rangle     = '>'
	space      = ' '
	tab        = '\t'
	nl         = '\n'
	cr         = '\r'
	squote     = '\''
	dquote     = '"'
	underscore = '_'
	pound      = '#'
	dot        = '.'
	plus       = '+'
	minus      = '-'
	star       = '*'
	slash      = '/'
	percent    = '%'
	ampersand  = '&'
	pipe       = '|'
	question   = '?'
	bang       = '!'
	equal      = '='
	comma      = ','
	colon      = ':'
	semicolon  = ';'
)

func isComment(r, k rune) bool {
	return r == slash && r == k
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == underscore
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return isLetter(r) || isDigit(r)
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

func isEOL(r rune) bool {
	return isNL(r) || r == semicolon
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}
