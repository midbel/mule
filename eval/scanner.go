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
	"else",
	"switch",
	"case",
	"function",
	"return",
	"break",
	"continue",
	"try",
	"catch",
	"finally",
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
	AddAssign
	Sub
	SubAssign
	Mul
	MulAssign
	Div
	DivAssign
	Pow
	PowAssign
	Mod
	ModAssign
	Lshift
	LshiftAssign
	Rshift
	RshiftAssign
	Band
	BandAssign
	Bor
	BorAssign
	Bnot
	BnotAssign
	Comma
	Colon
	Question
	Nullish
	Optional
	Arrow
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
	case Arrow:
		return "<arrow>"
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
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case Band:
		return "<bin-and>"
	case BandAssign:
		return "<bin-and-assign>"
	case Bor:
		return "<bin-or>"
	case BorAssign:
		return "<bin-or-assign>"
	case Bnot:
		return "<bin-not>"
	case Assign:
		return "<assign>"
	case Add:
		return "<add>"
	case AddAssign:
		return "<add-assign>"
	case Sub:
		return "<sub>"
	case SubAssign:
		return "<sub-assign>"
	case Div:
		return "<div>"
	case DivAssign:
		return "<div-assign>"
	case Mul:
		return "<mul>"
	case MulAssign:
		return "<mul-assign>"
	case Pow:
		return "<pow>"
	case PowAssign:
		return "<pow-assign>"
	case Mod:
		return "<mod>"
	case ModAssign:
		return "<mod-assign>"
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
	s.read()
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

func (s *Scanner) scanBinary(tok *Token) {
	s.write()
	s.read()
	for !s.done() && isBin(s.char) {
		s.write()
		s.read()
	}
}

func (s *Scanner) scanHexa(tok *Token) {
	s.write()
	s.read()
	for !s.done() && isHex(s.char) {
		s.write()
		s.read()
	}
}

func (s *Scanner) scanOctal(tok *Token) {
	s.write()
	s.read()
	for !s.done() && isOctal(s.char) {
		s.write()
		s.read()
	}
}

func (s *Scanner) scanNumber(tok *Token) {
	if k := s.peek(); s.char == '0' && k == 'b' || k == 'x' || k == 'o' {
		s.write()
		s.read()
		switch s.char {
		case 'o':
			s.scanOctal(tok)
		case 'b':
			s.scanBinary(tok)
		case 'x':
			s.scanHexa(tok)
		}
		return
	}
	var zeros int
	if s.char == '0' {
		for !s.done() && s.char == '0' {
			s.write()
			s.read()
			zeros++
		}
		if zeros > 1 {
			tok.Type = Invalid
		}
	}
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
		if s.peek() == equal {
			s.read()
			tok.Type = AddAssign
		}
	case minus:
		tok.Type = Sub
		if s.peek() == equal {
			s.read()
			tok.Type = SubAssign
		}
	case star:
		tok.Type = Mul
		if s.peek() == star {
			s.read()
			tok.Type = Pow
			if s.peek() == equal {
				s.read()
				tok.Type = PowAssign
			}
		} else if s.peek() == equal {
			s.read()
			tok.Type = MulAssign
		}
	case slash:
		tok.Type = Div
		if s.peek() == equal {
			s.read()
			tok.Type = DivAssign
		}
	case percent:
		tok.Type = Mod
		if s.peek() == equal {
			s.read()
			tok.Type = ModAssign
		}
	case ampersand:
		tok.Type = Band
		if s.peek() == ampersand {
			s.read()
			tok.Type = And
		} else if s.peek() == equal {
			s.read()
			tok.Type = BandAssign
		}
	case pipe:
		tok.Type = Bor
		if s.peek() == pipe {
			s.read()
			tok.Type = Or
		} else if s.peek() == equal {
			s.read()
			tok.Type = BorAssign
		}
	case equal:
		tok.Type = Assign
		if s.peek() == equal {
			s.read()
			tok.Type = Eq
		} else if s.peek() == rangle {
			s.read()
			tok.Type = Arrow
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
		} else if s.peek() == langle {
			s.read()
			tok.Type = Lshift
			if s.peek() == equal {
				s.read()
				tok.Type = LshiftAssign
			}
		}
	case rangle:
		tok.Type = Gt
		if s.peek() == equal {
			s.read()
			tok.Type = Ge
		} else if s.peek() == rangle {
			s.read()
			tok.Type = Rshift
			if s.peek() == equal {
				s.read()
				tok.Type = RshiftAssign
			}
		}
	case question:
		tok.Type = Question
		if s.peek() == question {
			s.read()
			tok.Type = Nullish
		} else if s.peek() == dot {
			s.read()
			tok.Type = Optional
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

func isBin(r rune) bool {
	return r == '0' || r == '1'
}

func isOctal(r rune) bool {
	return r >= '0' && r <= '7'
}

func isHex(r rune) bool {
	return isDigit(r) || (r >= 'a' && r <= 'f') && (r >= 'A' && r <= 'F')
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
