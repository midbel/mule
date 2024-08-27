package play

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

func Eval(r io.Reader) error {
	return nil
}

type Node interface{}

type Body struct {
	Nodes []Node
}

type Null struct{}

type Undefined struct{}

type Literal[T string | float64 | bool] struct {
	Value T
}

type Identifier struct {
	Value string
}

type Index struct {
	Left  Node
	Value Node
}

type Unary struct {
	Op rune
	Node
}

type Binary struct {
	Op    rune
	Left  Node
	Right Node
}

type Let struct {
	Node
}

type Const struct {
	Node
}

type Increment struct {
	Node
}

type Decrement struct {
	Node
}

type If struct {
	Cdt Node
	Csq Node
	Alt Node
}

type Switch struct {
	Cdt     Node
	Cases   []Node
	Default Node
}

type Case struct {
	Value Node
	Body  Node
}

type While struct {
	Cdt  Node
	Body Node
}

type For struct {
	Init  Node
	Cdt   Node
	After Node
	Body  Node
}

type ForOf struct {
	Var  Node
	Iter Node
	Body Node
}

type ForIn struct {
	Var  Node
	Iter Node
	Body Node
}

type Break struct{}

type Continue struct{}

type Try struct {
	Node
	Catch   Node
	Finally Node
}

type Catch struct {
	Err  Node
	Body Node
}

type Throw struct {
	Node
}

type Return struct {
	Node
}

type Call struct {
	Ident string
	Args  []Node
}

type Func struct {
	Ident string
	Args  []Node
	Body  Node
}

const (
	powLowest int = iota
	powComma
	powAssign
	powOr
	powAnd
	powEq
	powCmp
	powAdd
	powMul
	powPow
	powUnary
	powObject
	powGroup
)

var bindings = map[rune]int{
	Comma:    powComma,
	Question: powAssign,
	Assign:   powAssign,
	Colon:    powAssign,
	Or:       powOr,
	And:      powAnd,
	Eq:       powEq,
	Ne:       powEq,
	Lt:       powCmp,
	Le:       powCmp,
	Gt:       powCmp,
	Ge:       powCmp,
	Add:      powAdd,
	Sub:      powAdd,
	Mul:      powMul,
	Div:      powMul,
	Mod:      powMul,
	Pow:      powPow,
	Dot:      powObject,
	Lparen:   powObject,
	Lsquare:  powObject,
	Lcurly:   powObject,
}

type prefixFunc func() (Node, error)
type infixFunc func(Node) (Node, error)

type Parser struct {
	prefix map[rune]prefixFunc
	infix  map[rune]infixFunc

	scan *Scanner
	curr Token
	peek Token
}

func Parse(r io.Reader) *Parser {
	p := Parser{
		scan:   Scan(r),
		prefix: make(map[rune]prefixFunc),
		infix:  make(map[rune]infixFunc),
	}

	p.registerPrefix(Not, p.parseNot)
	p.registerPrefix(Sub, p.parseRev)
	p.registerPrefix(Incr, p.parseIncr)
	p.registerPrefix(Decr, p.parseDecr)
	p.registerPrefix(Ident, p.parseIdent)
	p.registerPrefix(String, p.parseString)
	p.registerPrefix(Number, p.parseNumber)
	p.registerPrefix(Boolean, p.parseBoolean)
	p.registerPrefix(Lsquare, p.parseArray)
	p.registerPrefix(Lcurly, p.parseObject)

	p.registerInfix(Dot, p.parseDot)
	p.registerInfix(Add, p.parseBinary)
	p.registerInfix(Sub, p.parseBinary)
	p.registerInfix(Mul, p.parseBinary)
	p.registerInfix(Div, p.parseBinary)
	p.registerInfix(Pow, p.parseBinary)
	p.registerInfix(Assign, p.parseBinary)
	p.registerInfix(And, p.parseBinary)
	p.registerInfix(Or, p.parseBinary)
	p.registerInfix(Eq, p.parseBinary)
	p.registerInfix(Ne, p.parseBinary)
	p.registerInfix(Lt, p.parseBinary)
	p.registerInfix(Le, p.parseBinary)
	p.registerInfix(Gt, p.parseBinary)
	p.registerInfix(Ge, p.parseBinary)
	p.registerInfix(Lparen, p.parseCall)
	p.registerInfix(Lsquare, p.parseIndex)
	p.registerInfix(Question, p.parseTernary)

	p.next()
	p.next()
	return &p
}

func (p *Parser) Parse() (Node, error) {
	return p.parse()
}

func (p *Parser) parse() (Node, error) {
	var body Body
	for !p.done() {
		n, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		body.Nodes = append(body.Nodes, n)
	}
	return body, nil
}

func (p *Parser) parseNode() (Node, error) {
	if p.is(Keyword) {
		return p.parseKeyword()
	}
	return p.parseExpression(powLowest)
}

func (p *Parser) parseKeyword() (Node, error) {
	keyword := p.curr.Literal
	p.next()
	switch keyword {
	case "let":
		return p.parseLet()
	case "const":
		return p.parseConst()
	case "if":
		return p.parseIf()
	case "switch":
		return p.parseSwitch()
	case "for":
		return p.parseFor()
	case "do":
		return p.parseDo()
	case "while":
		return p.parseWhile()
	case "break":
		return p.parseBreak()
	case "continue":
		return p.parseContinue()
	case "return":
		return p.parseReturn()
	case "function":
		return p.parseFunction()
	case "import":
		return p.parseImport()
	case "export":
		return p.parseExport()
	case "try":
		return p.parseTry()
	case "throw":
		return p.parseThrow()
	case "null":
		return p.parseNull()
	case "undefined":
		return p.parseUndefined()
	default:
		return nil, fmt.Errorf("%s: keyword not supported/known")
	}
}

func (p *Parser) parseLet() (Node, error) {
	return nil, nil
}

func (p *Parser) parseConst() (Node, error) {
	return nil, nil
}

func (p *Parser) parseIf() (Node, error) {
	return nil, nil
}

func (p *Parser) parseSwitch() (Node, error) {
	return nil, nil
}

func (p *Parser) parseDo() (Node, error) {
	return nil, nil
}

func (p *Parser) parseWhile() (Node, error) {
	return nil, nil
}

func (p *Parser) parseFor() (Node, error) {
	return nil, nil
}

func (p *Parser) parseBreak() (Node, error) {
	return nil, nil
}

func (p *Parser) parseContinue() (Node, error) {
	return nil, nil
}

func (p *Parser) parseReturn() (Node, error) {
	return nil, nil
}

func (p *Parser) parseFunction() (Node, error) {
	return nil, nil
}

func (p *Parser) parseImport() (Node, error) {
	return nil, nil
}

func (p *Parser) parseExport() (Node, error) {
	return nil, nil
}

func (p *Parser) parseTry() (Node, error) {
	return nil, nil
}

func (p *Parser) parseThrow() (Node, error) {
	return nil, nil
}

func (p *Parser) parseNull() (Node, error) {
	return nil, nil
}

func (p *Parser) parseUndefined() (Node, error) {
	return nil, nil
}

func (p *Parser) parseExpression(pow int) (Node, error) {
	return nil, nil
}

func (p *Parser) parseNot() (Node, error) {
	return nil, nil
}

func (p *Parser) parseRev() (Node, error) {
	return nil, nil
}

func (p *Parser) parseIncr() (Node, error) {
	return nil, nil
}

func (p *Parser) parseDecr() (Node, error) {
	return nil, nil
}

func (p *Parser) parseIdent() (Node, error) {
	return nil, nil
}

func (p *Parser) parseString() (Node, error) {
	return nil, nil
}

func (p *Parser) parseNumber() (Node, error) {
	return nil, nil
}

func (p *Parser) parseBoolean() (Node, error) {
	return nil, nil
}

func (p *Parser) parseArray() (Node, error) {
	return nil, nil
}

func (p *Parser) parseObject() (Node, error) {
	return nil, nil
}

func (p *Parser) parseDot(left Node) (Node, error) {
	return nil, nil
}

func (p *Parser) parseBinary(left Node) (Node, error) {
	return nil, nil
}

func (p *Parser) parseTernary(left Node) (Node, error) {
	return nil, nil
}

func (p *Parser) parseIndex(left Node) (Node, error) {
	return nil, nil
}

func (p *Parser) parseCall(left Node) (Node, error) {
	return nil, nil
}

func (p *Parser) registerPrefix(kind rune, fn prefixFunc) {
	p.prefix[kind] = fn
}

func (p *Parser) registerInfix(kind rune, fn infixFunc) {
	p.infix[kind] = fn
}

func (p *Parser) done() bool {
	return p.is(EOF)
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

const (
	EOF = -(iota + 1)
	EOL
	Keyword
	Ident
	String
	Number
	Boolean
	Invalid
	Incr
	Decr
	Add
	Sub
	Mul
	Div
	Mod
	Pow
	Assign
	Not
	Eq
	Ne
	Lt
	Le
	Gt
	Ge
	And
	Or
	Dot
	Comma
	Question
	Colon
	Lparen
	Rparen
	Lsquare
	Rsquare
	Lcurly
	Rcurly
)

var keywords = []string{
	"let",
	"const",
	"break",
	"continue",
	"for",
	"in",
	"of",
	"if",
	"else",
	"switch",
	"case",
	"default",
	"while",
	"function",
	"return",
	"import",
	"export",
	"from",
	"as",
	"true",
	"false",
	"try",
	"catch",
	"finally",
	"throw",
	"null",
	"undefined",
	"instanceof",
	"typeof",
}

type Position struct {
	Line   int
	Column int
}

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
	case Dot:
		return "<dot>"
	case Comma:
		return "<comma>"
	case Lparen:
		return "<lparen>"
	case Rparen:
		return "<rparen>"
	case Lsquare:
		return "<lsquare>"
	case Rsquare:
		return "<rsquare>"
	case Lcurly:
		return "<lcurly>"
	case Rcurly:
		return "<rcurly>"
	case Incr:
		return "<incr>"
	case Decr:
		return "<decr>"
	case Add:
		return "<add>"
	case Sub:
		return "<sub>"
	case Mul:
		return "<mul>"
	case Div:
		return "<div>"
	case Mod:
		return "<mod>"
	case Pow:
		return "<pow>"
	case Assign:
		return "<assign>"
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
	case And:
		return "<and>"
	case Or:
		return "<or>"
	case Question:
		return "<question>"
	case Colon:
		return "<colon>"
	case Keyword:
		prefix = "keyword"
	case Boolean:
		prefix = "boolean"
	case Ident:
		prefix = "identifier"
	case String:
		prefix = "string"
	case Number:
		prefix = "number"
	case Invalid:
		prefix = "invalid"
	default:
		prefix = "unknown"
	}
	return fmt.Sprintf("%s(%s)", prefix, t.Literal)
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
	tok.Type = Ident

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
	}
}

func (s *Scanner) scanOperator(tok *Token) {
	switch s.char {
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
	case colon:
		tok.Type = Colon
	case dot:
		tok.Type = Dot
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
		r == question || r == colon || r == dot || r == comma
}
