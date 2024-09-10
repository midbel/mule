package play

import (
	"bytes"
	"io"
	"slices"
	"unicode/utf8"
)

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

	s.skip(isSpace)
	switch {
	case isDigit(s.char):
		s.scanNumber(&tok)
	case isPunct(s.char):
		s.scanPunct(&tok)
	case isOperator(s.char):
		s.scanOperator(&tok)
	case isNL(s.char):
		s.scanNL(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isLetter(s.char):
		s.scanIdent(&tok)
	default:
		tok.Type = Invalid
	}

	return tok
}

func (s *Scanner) scanIdent(tok *Token) {
	for !isDelim(s.char) && !s.done() {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	switch tok.Literal {
	case "new":
		tok.Type = New
	case "instanceof":
		tok.Type = InstanceOf
	case "typeof":
		tok.Type = TypeOf
	case "delete":
		tok.Type = Del
	default:
		tok.Type = Ident
	}
	if slices.Contains(keywords, tok.Literal) {
		tok.Type = Keyword
		if tok.Literal == "true" || tok.Literal == "false" {
			tok.Type = Boolean
		}
	}
}

func (s *Scanner) scanString(tok *Token) {
	quote := s.char
	s.read()
	for !s.done() && s.char != quote {
		s.write()
		s.read()
	}
	tok.Literal = s.literal()
	tok.Type = Text
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

func (s *Scanner) scanPunct(tok *Token) {
	switch s.char {
	case lparen:
		tok.Type = Lparen
	case rparen:
		tok.Type = Rparen
	case lsquare:
		tok.Type = Lsquare
	case rsquare:
		tok.Type = Rsquare
	case lcurly:
		tok.Type = Lcurly
	case rcurly:
		tok.Type = Rcurly
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
		s.skip(isBlank)
	}
}

func (s *Scanner) scanOperator(tok *Token) {
	switch s.char {
	case arobase:
		tok.Type = Decorate
	case plus:
		tok.Type = Add
		if s.peek() == plus {
			s.read()
			tok.Type = Incr
		}
	case minus:
		tok.Type = Sub
		if s.peek() == minus {
			s.read()
			tok.Type = Decr
		}
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
	case equal:
		tok.Type = Assign
		if s.peek() == rangle {
			s.read()
			tok.Type = Arrow
		}
		if s.peek() == equal {
			s.read()
			tok.Type = Eq
		}
		if s.peek() == equal && tok.Type == Eq {
			s.read()
			tok.Type = Seq
		}
	case bang:
		tok.Type = Not
		if s.peek() == equal {
			s.read()
			tok.Type = Ne
		}
		if s.peek() == equal && tok.Type == Ne {
			s.read()
			tok.Type = Sne
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
	case ampersand:
		tok.Type = Invalid
		if s.peek() == ampersand {
			s.read()
			tok.Type = And
		}
	case pipe:
		tok.Type = Invalid
		if s.peek() == pipe {
			s.read()
			tok.Type = Or
		}
	case question:
		tok.Type = Question
		if s.peek() == question {
			s.read()
			tok.Type = Nullish
		}
		if s.peek() == dot {
			s.read()
			tok.Type = Optional
		}
	case colon:
		tok.Type = Colon
	case dot:
		tok.Type = Dot
		if s.peek() == dot {
			s.save()
			s.read()
			s.read()
			if s.char != dot {
				s.restore()
				break
			}
			tok.Type = Spread
		}
	case comma:
		tok.Type = Comma
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
	}
}

func (s *Scanner) scanNL(tok *Token) {
	s.skip(isBlank)
	tok.Type = EOL
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
	space      = ' '
	tab        = '\t'
	nl         = '\n'
	cr         = '\r'
	dot        = '.'
	squote     = '\''
	dquote     = '"'
	underscore = '_'
	pipe       = '|'
	ampersand  = '&'
	equal      = '='
	plus       = '+'
	minus      = '-'
	star       = '*'
	slash      = '/'
	percent    = '%'
	bang       = '!'
	comma      = ','
	langle     = '<'
	rangle     = '>'
	lparen     = '('
	rparen     = ')'
	lsquare    = '['
	rsquare    = ']'
	lcurly     = '{'
	rcurly     = '}'
	semi       = ';'
	question   = '?'
	colon      = ':'
	arobase    = '@'
)

func isDelim(r rune) bool {
	return isBlank(r) || isOperator(r) || isPunct(r)
}

func isPunct(r rune) bool {
	return r == lparen || r == rparen ||
		r == lcurly || r == rcurly ||
		r == lsquare || r == rsquare
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
	return r == nl || r == cr || r == semi
}

func isBlank(r rune) bool {
	return isSpace(r) || isNL(r)
}

func isOperator(r rune) bool {
	return r == equal || r == ampersand || r == pipe ||
		r == plus || r == minus || r == star || r == slash ||
		r == bang || r == langle || r == rangle || r == percent ||
		r == question || r == colon || r == dot || r == comma ||
		r == arobase
}
