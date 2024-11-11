package json

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	errUndefined   = errors.New("undefined")
	errDiscard     = errors.New("discard")
	errType        = errors.New("type")
	errArgument    = errors.New("argument")
	errImplemented = errors.New("not implemented")
)

type Query interface {
	Get(any) (any, error)
}

type Expr interface {
	Eval(any) (any, error)
}

type query struct {
	expr Expr
}

func (q query) Get(doc any) (any, error) {
	a, err := q.expr.Eval(doc)
	return a, err
}

type compiler struct {
	scan *QueryScanner
	curr Token
	peek Token

	prefix map[rune]func() (Expr, error)
	infix  map[rune]func(Expr) (Expr, error)
}

func Compile(query string) (Query, error) {
	cp := compiler{
		scan: ScanQuery(strings.NewReader(query)),
	}
	cp.prefix = map[rune]func() (Expr, error){
		Ident:    cp.compileIdent,
		Func:     cp.compileIdent,
		Number:   cp.compileNumber,
		String:   cp.compileString,
		Boolean:  cp.compileBool,
		BegGrp:   cp.compileGroup,
		Wildcard: cp.compileWildcard,
		Descent:  cp.compileDescent,
		Sub:      cp.compileReverse,
		BegArr:   cp.compileArray,
		BegObj:   cp.compileObjectPrefix,
	}

	cp.infix = map[rune]func(Expr) (Expr, error){
		BegGrp:    cp.compileCall,
		BegArr:    cp.compileFilter,
		BegObj:    cp.compileObject,
		And:       cp.compileBinary,
		Or:        cp.compileBinary,
		Add:       cp.compileBinary,
		Sub:       cp.compileBinary,
		Mul:       cp.compileBinary,
		Wildcard:  cp.compileBinary,
		Div:       cp.compileBinary,
		Mod:       cp.compileBinary,
		Parent:    cp.compileBinary,
		Eq:        cp.compileBinary,
		Ne:        cp.compileBinary,
		Lt:        cp.compileBinary,
		Le:        cp.compileBinary,
		Gt:        cp.compileBinary,
		Ge:        cp.compileBinary,
		Concat:    cp.compileBinary,
		In:        cp.compileBinary,
		Map:       cp.compileMap,
		Ternary:   cp.compileTernary,
		Transform: cp.compileTransform,
	}

	cp.next()
	cp.next()
	return cp.Compile()
}

func (c *compiler) Compile() (Query, error) {
	return c.compile()
}

func (c *compiler) compile() (Query, error) {
	e, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	q := query{
		expr: e,
	}
	return q, nil
}

func (c *compiler) compileTransform(left Expr) (Expr, error) {
	expr := transform{
		expr: left,
	}
	c.next()
	next, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	expr.next = next
	return expr, nil
}

func (c *compiler) compileMap(left Expr) (Expr, error) {
	c.next()
	q := path{
		expr: left,
	}
	next, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	q.next = next
	return q, nil
}

func (c *compiler) compileFilter(left Expr) (Expr, error) {
	c.next()
	if c.is(EndArr) {
		c.next()
		a := arrayTransform{
			expr: left,
		}
		if c.is(BegArr) {
			left, err := c.compileFilter(left)
			if err != nil {
				return nil, err
			}
			a.expr = left
		}
		return a, nil
	}
	expr, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	if !c.is(EndArr) {
		return nil, fmt.Errorf("syntax error: missing ]")
	}
	c.next()

	f := filter{
		expr:  left,
		check: expr,
	}
	return f, nil
}

func (c *compiler) getString() string {
	defer c.next()
	return c.curr.Literal
}

func (c *compiler) getNumber() float64 {
	defer c.next()
	n, _ := strconv.ParseFloat(c.curr.Literal, 64)
	return n
}

func (c *compiler) getBool() bool {
	defer c.next()
	b, _ := strconv.ParseBool(c.curr.Literal)
	return b
}

func (c *compiler) compileExpr(pow int) (Expr, error) {
	fn, ok := c.prefix[c.curr.Type]
	if !ok {
		return nil, fmt.Errorf("syntax error: invalid prefix expression")
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for !c.is(EndArr) && pow < bindings[c.curr.Type] {
		fn, ok := c.infix[c.curr.Type]
		if !ok {
			return nil, fmt.Errorf("syntax error: invalid infix expression")
		}
		left, err = fn(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (c *compiler) compileArray() (Expr, error) {
	c.next()
	var b arrayBuilder
	for !c.done() && !c.is(EndArr) {
		expr, err := c.compileExpr(powComma)
		if err != nil {
			return nil, err
		}
		b.expr = append(b.expr, expr)
		switch {
		case c.is(Comma):
			c.next()
		case c.is(EndArr):
		default:
			return nil, fmt.Errorf("syntax error: expected ',' or ']")
		}
	}
	if !c.is(EndArr) {
		return nil, fmt.Errorf("syntax error: missing ']")
	}
	c.next()
	return b, nil
}

func (c *compiler) compileObjectPrefix() (Expr, error) {
	return c.compileObject(nil)
}

func (c *compiler) compileObject(left Expr) (Expr, error) {
	c.next()
	b := objectBuilder{
		expr: left,
		list: make(map[Expr]Expr),
	}
	for !c.done() && !c.is(EndObj) {
		key, err := c.compileExpr(powLowest)
		if err != nil {
			return nil, err
		}
		if !c.is(Colon) {
			return nil, fmt.Errorf("syntax error: expected ':'")
		}
		c.next()
		val, err := c.compileExpr(powLowest)
		if err != nil {
			return nil, err
		}
		b.list[key] = val
		switch {
		case c.is(Comma):
			c.next()
		case c.is(EndObj):
		default:
			return nil, fmt.Errorf("syntax error: expected ',' or '}")
		}
	}
	if !c.is(EndObj) {
		return nil, fmt.Errorf("syntax error: expected '}")
	}
	c.next()
	return b, nil
}

func (c *compiler) compileWildcard() (Expr, error) {
	defer c.next()
	return wildcard{}, nil
}

func (c *compiler) compileDescent() (Expr, error) {
	defer c.next()
	return descent{}, nil
}

func (c *compiler) compileIdent() (Expr, error) {
	i := identifier{
		ident: c.getString(),
	}
	return i, nil
}

func (c *compiler) compileNumber() (Expr, error) {
	i := literal[float64]{
		value: c.getNumber(),
	}
	return i, nil
}

func (c *compiler) compileString() (Expr, error) {
	i := literal[string]{
		value: c.getString(),
	}
	return i, nil
}

func (c *compiler) compileBool() (Expr, error) {
	i := literal[bool]{
		value: c.getBool(),
	}
	return i, nil
}

func (c *compiler) compileGroup() (Expr, error) {
	c.next()
	expr, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	if !c.is(EndGrp) {
		return nil, fmt.Errorf("syntax error: missing ')'")
	}
	c.next()
	return expr, nil
}

func (c *compiler) compileReverse() (Expr, error) {
	c.next()
	expr, err := c.compileExpr(powPrefix)
	if err != nil {
		return nil, err
	}
	r := reverse{
		expr: expr,
	}
	return r, nil
}

func (c *compiler) compileTernary(left Expr) (Expr, error) {
	c.next()
	t := ternary{
		cdt: left,
	}
	fmt.Println(left)
	csq, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	if !c.is(Colon) {
		return nil, fmt.Errorf("syntax error: missing ':'")
	}
	c.next()
	alt, err := c.compileExpr(powLowest)
	if err != nil {
		return nil, err
	}
	t.csq = csq
	t.alt = alt
	return t, nil
}

func (c *compiler) compileBinary(left Expr) (Expr, error) {
	if c.is(Wildcard) {
		c.curr.Type = Mul
	} else if c.is(Parent) {
		c.curr.Type = Mod
	}
	var (
		pow = bindings[c.curr.Type]
		err error
	)
	bin := binary{
		left: left,
		op:   c.curr.Type,
	}
	c.next()
	bin.right, err = c.compileExpr(pow)
	return bin, err
}

func (c *compiler) compileCall(left Expr) (Expr, error) {
	ident, ok := left.(identifier)
	if !ok {
		return nil, fmt.Errorf("syntax error: identifier expected")
	}
	expr := call{
		ident: ident.ident,
	}
	c.next()
	for !c.done() && !c.is(EndGrp) {
		a, err := c.compileExpr(powLowest)
		if err != nil {
			return nil, err
		}
		expr.args = append(expr.args, a)
		switch {
		case c.is(Comma):
			c.next()
			if c.is(EndGrp) {
				return nil, fmt.Errorf("syntax error: trailing comma")
			}
		case c.is(EndGrp):
		default:
			return nil, fmt.Errorf("syntax error: unexpected token")
		}
	}
	if !c.is(EndGrp) {
		return nil, fmt.Errorf("syntax error: missing ')'")
	}
	c.next()
	return expr, nil
}

func (c *compiler) done() bool {
	return c.is(EOF)
}

func (c *compiler) is(kind rune) bool {
	return c.curr.Type == kind
}

func (c *compiler) next() {
	c.curr = c.peek
	c.peek = c.scan.Scan()
}

type queryMode int8

const (
	pathMode queryMode = 1 << iota
	filterMode
)

type QueryScanner struct {
	input *bufio.Reader
	char  rune

	mode queryMode

	str bytes.Buffer
}

func ScanQuery(r io.Reader) *QueryScanner {
	scan := QueryScanner{
		input: bufio.NewReader(r),
		mode:  pathMode,
	}
	scan.read()
	return &scan
}

func (s *QueryScanner) Scan() Token {
	defer s.str.Reset()
	s.skipBlank()

	var tok Token
	if s.done() {
		tok.Type = EOF
		return tok
	}
	switch {
	case isLetter(s.char):
		s.scanIdent(&tok)
	case isBackQuote(s.char):
		s.scanQuotedIdent(&tok)
	case isNumber(s.char):
		s.scanNumber(&tok)
	case isQuote(s.char):
		s.scanString(&tok)
	case isDelim(s.char) || s.char == '(' || s.char == ')':
		s.scanDelimiter(&tok)
	case isOperator(s.char):
		s.scanOperator(&tok)
	case isDollar(s.char):
		s.scanDollar(&tok)
	case isTransform(s.char):
		s.scanTransform(&tok)
	default:
		tok.Type = Invalid
	}
	s.setMode(tok)
	return tok
}

func (s *QueryScanner) setMode(tok Token) {
	if tok.Type == BegArr {
		s.mode = filterMode
	} else if tok.Type == EndArr {
		s.mode = pathMode
	}
}

func (s *QueryScanner) scanQuotedIdent(tok *Token) {
	for !s.done() && !isBackQuote(s.char) {
		s.write()
		s.read()
	}
	tok.Type = Ident
	tok.Literal = s.str.String()
	if !isBackQuote(s.char) {
		tok.Type = Invalid
	} else {
		s.read()
	}
}

func (s *QueryScanner) scanTransform(tok *Token) {
	s.read()
	tok.Type = Transform
}

func (s *QueryScanner) scanDollar(tok *Token) {
	s.read()
	if !isLetter(s.char) {
		tok.Type = Invalid
		return
	}
	s.scanIdent(tok)
	if tok.Type == Ident {
		tok.Type = Func
	}
}

func (s *QueryScanner) scanIdent(tok *Token) {
	for !s.done() && isAlpha(s.char) {
		s.write()
		s.read()
	}
	tok.Literal = s.str.String()
	switch tok.Literal {
	case "true", "false":
		tok.Type = Boolean
	case "null":
		tok.Type = Null
	case "and":
		tok.Type = And
	case "or":
		tok.Type = Or
	case "in":
		tok.Type = In
	default:
		tok.Type = Ident
	}
}

func (s *QueryScanner) scanString(tok *Token) {
	s.read()
	for !s.done() && s.char != '"' {
		s.write()
		s.read()
	}
	tok.Literal = s.str.String()
	tok.Type = String
	if s.char != '"' {
		tok.Type = Invalid
	} else {
		s.read()
	}
}

func (s *QueryScanner) scanNumber(tok *Token) {
	tok.Type = Number
	for !s.done() && isNumber(s.char) {
		s.write()
		s.read()
	}
	tok.Literal = s.str.String()
	if s.char == '.' {
		s.write()
		s.read()
		if !isNumber(s.char) {
			tok.Type = Invalid
			return
		}
		for !s.done() && isNumber(s.char) {
			s.write()
			s.read()
		}
		tok.Literal = s.str.String()
	}
	if s.char == 'e' || s.char == 'E' {
		s.write()
		s.read()
		if s.char == '-' || s.char == '+' {
			s.write()
			s.read()
		}
		if !isNumber(s.char) {
			tok.Type = Invalid
			return
		}
		for !s.done() && isNumber(s.char) {
			s.write()
			s.read()
		}
		tok.Literal = s.str.String()
	}
}

func (s *QueryScanner) scanOperator(tok *Token) {
	switch s.char {
	case '+':
		tok.Type = Add
	case '-':
		tok.Type = Sub
	case '*':
		if s.mode == pathMode {
			tok.Type = Wildcard
			if k := s.peek(); k == s.char {
				s.read()
				tok.Type = Descent
			}
		} else {
			tok.Type = Mul
		}
	case '/':
		tok.Type = Div
	case '%':
		if s.mode == pathMode {
			tok.Type = Parent
		} else {
			tok.Type = Mod
		}
	case '?':
		tok.Type = Ternary
	case ':':
	case '!':
		tok.Type = Invalid
		if k := s.peek(); k == '=' {
			s.read()
			tok.Type = Ne
		}
	case '=':
		tok.Type = Eq
	case '<':
		tok.Type = Lt
		if k := s.peek(); k == '=' {
			s.read()
			tok.Type = Le
		}
	case '>':
		tok.Type = Gt
		if k := s.peek(); k == '=' {
			s.read()
			tok.Type = Ge
		}
	case '.':
		tok.Type = Map
		if k := s.peek(); k == s.char {
			s.read()
			tok.Type = Range
		}
	case '&':
		tok.Type = Concat
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
	}
}

func (s *QueryScanner) scanDelimiter(tok *Token) {
	switch s.char {
	case '(':
		tok.Type = BegGrp
	case ')':
		tok.Type = EndGrp
	case '[':
		tok.Type = BegArr
	case ']':
		tok.Type = EndArr
	case '{':
		tok.Type = BegObj
	case '}':
		tok.Type = EndObj
	case ',':
		tok.Type = Comma
	case ':':
		tok.Type = Colon
	default:
		tok.Type = Invalid
	}
	if tok.Type != Invalid {
		s.read()
	}
}

func (s *QueryScanner) write() {
	s.str.WriteRune(s.char)
}

func (s *QueryScanner) read() {
	char, _, err := s.input.ReadRune()
	if errors.Is(err, io.EOF) {
		char = utf8.RuneError
	}
	s.char = char
}

func (s *QueryScanner) peek() rune {
	defer s.input.UnreadRune()
	r, _, _ := s.input.ReadRune()
	return r
}

func (s *QueryScanner) done() bool {
	return s.char == utf8.RuneError
}

func (s *QueryScanner) skipBlank() {
	for !s.done() && unicode.IsSpace(s.char) {
		s.read()
	}
}
