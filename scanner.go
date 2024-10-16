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
	"compress",
	"flow",
	"when",
	"exit",
	"goto",
	"set",
	"unset",
	"expect",
	// HTTP methods
	"do", // abstract request
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
	Lsub
	Rsub
	EndSubstitute
	String
	Number
	Lbrace
	Rbrace
	Substring
	TrimPrefix
	TrimLongPrefix
	TrimSuffix
	TrimLongSuffix
	Replace
	ReplaceAll
	ReplaceSuffix
	ReplacePrefix
	UpperFirst
	UpperAll
	LowerFirst
	LowerAll
	ValueUnset
	ValueSet
	ValueAssign
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
	case Lbrace:
		return "<lbrace>"
	case Rbrace:
		return "<rbrace>"
	case Lsub:
		return "<beg-substitute>"
	case Rsub:
		return "<end-substitute>"
	case Substring:
		return "<substring>"
	case TrimPrefix:
		return "<trim-prefix>"
	case TrimLongPrefix:
		return "<trim-long-prefix>"
	case TrimSuffix:
		return "<trim-suffix>"
	case TrimLongSuffix:
		return "<trim-long-suffix>"
	case Replace:
		return "<replace-first>"
	case ReplaceAll:
		return "<replace-all>"
	case ReplaceSuffix:
		return "<replace-suffix>"
	case ReplacePrefix:
		return "<replace-prefix>"
	case UpperAll:
		return "<upper-all>"
	case UpperFirst:
		return "<upper-first>"
	case LowerAll:
		return "<lower-all>"
	case LowerFirst:
		return "<lower-first>"
	case ValueUnset:
		return "<value-unset>"
	case ValueSet:
		return "<value-set>"
	case ValueAssign:
		return "<value-assign>"
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

type state int8

const (
	stateQuoted state = 1 << iota
	stateSubstitute
)

func (s state) quoted() bool {
	return s&stateQuoted == stateQuoted
}

func (s state) substitute() bool {
	return s&stateSubstitute == stateSubstitute
}

type Scanner struct {
	input []byte
	cursor
	old cursor

	state
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

	var tok Token
	tok.Position = s.cursor.Position
	if s.done() {
		tok.Type = EOF
		return tok
	}

	if s.quoted() {
		s.scanQuote(&tok)
		return tok
	} else if s.substitute() {
		s.scanSubstitute(&tok)
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

func (s *Scanner) scanSubstitute(tok *Token) {
	if s.char == rbrace {
		s.read()
		s.state = 0
		tok.Type = Rsub
		return
	}
	switch {
	case isTransform(s.char):
		s.scanModifier(tok)
	case isLetter(s.char):
		s.scanIdent(tok)
	case isQuote(s.char):
		s.scanString(tok)
	case isDigit(s.char):
		s.scanNumber(tok)
	default:
		tok.Type = Invalid
	}
}

func (s *Scanner) scanModifier(tok *Token) {
	switch s.char {
	case colon:
		tok.Type = Substring
		if k := s.peek(); k == plus {
			tok.Type = ValueUnset
		} else if k == minus {
			tok.Type = ValueSet
		} else if k == equal {
			tok.Type = ValueAssign
		}
		if tok.Type != Substring {
			s.read()
		}
	case comma:
		tok.Type = LowerFirst
		if s.peek() == s.char {
			s.read()
			tok.Type = LowerAll
		}
	case caret:
		tok.Type = UpperFirst
		if s.peek() == s.char {
			s.read()
			tok.Type = UpperAll
		}
	case pound:
		tok.Type = TrimPrefix
		if s.peek() == s.char {
			s.read()
			tok.Type = TrimLongPrefix
		}
	case percent:
		tok.Type = TrimSuffix
		if s.peek() == s.char {
			s.read()
			tok.Type = TrimLongSuffix
		}
	case slash:
		tok.Type = Replace
		if k := s.peek(); k == s.char {
			tok.Type = ReplaceAll
		} else if k == percent {
			tok.Type = ReplaceSuffix
		} else if k == pound {
			tok.Type = ReplacePrefix
		}
		if tok.Type != Replace {
			s.read()
		}
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
	}
}

func (s *Scanner) scanQuote(tok *Token) {
	switch {
	case isVariable(s.char):
		s.scanVariable(tok)
	case isTemplate(s.char):
		s.scanTemplate(tok)
	default:
		s.scanVerbatim(tok)
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
	if s.quoted() {
		s.state = 0
	} else {
		s.state = stateQuoted
	}
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
		body.WriteRune(nl)
	}
	tok.Type = String
	if !valid {
		tok.Type = Invalid
	}
	tok.Literal = body.String()
}

func (s *Scanner) scanVariable(tok *Token) {
	s.read()
	if s.char == lbrace {
		s.read()
		s.state = stateSubstitute
		tok.Type = Lsub
		return
	}
	s.scanIdent(tok)
	if tok.Type != Ident && tok.Type != Keyword {
		tok.Type = Invalid
		return
	}
	tok.Type = Variable
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
	colon      = ':'
	percent    = '%'
	slash      = '/'
	comma      = ','
	caret      = '^'
	plus       = '+'
	minus      = '-'
	equal      = '='
)

func isTransform(r rune) bool {
	return r == colon || r == percent || r == slash || r == comma || r == caret
}

func isMacro(r rune) bool {
	return r == arobase
}

func isHeredoc(r, k rune) bool {
	return r == langle && r == k
}

func isDelim(r rune) bool {
	return isBlank(r) || isPunct(r) || isTemplate(r) || isTransform(r)
}

func isPunct(r rune) bool {
	return r == lbrace || r == rbrace
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
