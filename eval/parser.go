package eval

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

func ParseString(str string) (Expression, error) {
	return Parse(strings.NewReader(str))
}

func Parse(r io.Reader) (Expression, error) {
	return NewParser(r).Parse()
}

type Parser struct {
	file string

	scan *Scanner
	curr Token
	peek Token

	keywords map[string]func() (Expression, error)
	prefix   map[rune]func() (Expression, error)
	infix    map[rune]func(Expression) (Expression, error)
}

func NewParser(r io.Reader) *Parser {
	p := Parser{
		scan:     Scan(r),
		infix:    make(map[rune]func(Expression) (Expression, error)),
		prefix:   make(map[rune]func() (Expression, error)),
		keywords: make(map[string]func() (Expression, error)),
	}
	if n, ok := r.(interface{ Name() string }); ok {
		p.file = n.Name()
	}
	p.registerInfix(Assign, p.parseAssignment)
	p.registerInfix(AddAssign, p.parseAssignment)
	p.registerInfix(SubAssign, p.parseAssignment)
	p.registerInfix(MulAssign, p.parseAssignment)
	p.registerInfix(DivAssign, p.parseAssignment)
	p.registerInfix(ModAssign, p.parseAssignment)
	p.registerInfix(PowAssign, p.parseAssignment)
	p.registerInfix(LshiftAssign, p.parseAssignment)
	p.registerInfix(RshiftAssign, p.parseAssignment)
	p.registerInfix(BandAssign, p.parseAssignment)
	p.registerInfix(BorAssign, p.parseAssignment)
	p.registerInfix(Add, p.parseBinary)
	p.registerInfix(Sub, p.parseBinary)
	p.registerInfix(Mul, p.parseBinary)
	p.registerInfix(Div, p.parseBinary)
	p.registerInfix(Mod, p.parseBinary)
	p.registerInfix(Pow, p.parseBinary)
	p.registerInfix(Lshift, p.parseBinary)
	p.registerInfix(Rshift, p.parseBinary)
	p.registerInfix(Eq, p.parseBinary)
	p.registerInfix(Ne, p.parseBinary)
	p.registerInfix(Lt, p.parseBinary)
	p.registerInfix(Le, p.parseBinary)
	p.registerInfix(Gt, p.parseBinary)
	p.registerInfix(Ge, p.parseBinary)
	p.registerInfix(And, p.parseBinary)
	p.registerInfix(Or, p.parseBinary)
	p.registerInfix(Band, p.parseBinary)
	p.registerInfix(Bor, p.parseBinary)
	p.registerInfix(Lsquare, p.parseIndex)
	p.registerInfix(Lparen, p.parseCall)
	p.registerInfix(Dot, p.parseDot)
	p.registerInfix(Question, p.parseTernary)
	// p.registerInfix(Nullish, p.parseInfix)
	// p.registerInfix(Optional, p.parseInfix)

	p.registerPrefix(Ident, p.parseIdentifier)
	p.registerPrefix(String, p.parseString)
	p.registerPrefix(Number, p.parseNumber)
	p.registerPrefix(Boolean, p.parseBool)
	p.registerPrefix(Lsquare, p.parseArray)
	p.registerPrefix(Lbrace, p.parseHash)
	p.registerPrefix(Lparen, p.parseGroup)
	p.registerPrefix(Not, p.parseUnary)
	p.registerPrefix(Sub, p.parseUnary)
	p.registerPrefix(Keyword, p.parseKeyword)

	p.registerKeyword("let", p.parseLet)
	p.registerKeyword("const", p.parseConst)
	p.registerKeyword("if", p.parseIf)
	p.registerKeyword("else", p.parseElse)
	p.registerKeyword("switch", p.parseSwitch)
	p.registerKeyword("while", p.parseWhile)
	p.registerKeyword("for", p.parseFor)
	p.registerKeyword("function", p.parseFunction)
	p.registerKeyword("try", p.parseTry)
	p.registerKeyword("catch", p.parseCatch)
	p.registerKeyword("throw", p.parseThrow)
	p.registerKeyword("return", p.parseReturn)
	p.registerKeyword("break", p.parseBreak)
	p.registerKeyword("continue", p.parseContinue)

	p.next()
	p.next()
	return &p
}

func (p *Parser) Parse() (Expression, error) {
	var b Block
	for !p.done() {
		p.skip(Comment)
		e, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		b.List = append(b.List, e)
		p.skip(EOL)
	}
	return b, nil
}

func (p *Parser) parseBinary(left Expression) (Expression, error) {
	b := Binary{
		Op:   p.curr.Type,
		Left: left,
	}
	p.next()
	right, err := p.parseExpression(bindings[b.Op])
	if err != nil {
		return nil, err
	}
	b.Right = right
	return b, nil
}

func (p *Parser) parseAssignment(left Expression) (Expression, error) {
	if _, ok := left.(Variable); !ok {
		return nil, fmt.Errorf("expected variable")
	}
	op := p.curr.Type
	p.next()
	right, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	var ag Assignment
	switch op {
	default:
	case AddAssign:
		op = Add
	case SubAssign:
		op = Sub
	case MulAssign:
		op = Mul
	case DivAssign:
		op = Div
	case PowAssign:
		op = Pow
	case ModAssign:
		op = Mod
	case BandAssign:
		op = Band
	case BorAssign:
		op = Bor
	case LshiftAssign:
		op = Lshift
	case RshiftAssign:
		op = Rshift
	}
	ag.Ident = left
	if op != Assign {
		right = Binary{
			Op:    op,
			Left:  left,
			Right: right,
		}
	}
	ag.Expr = right
	return ag, nil
}

func (p *Parser) parseCall(left Expression) (Expression, error) {
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	call := Call{
		Ident: left,
	}
	for !p.done() && !p.is(Rparen) {
		e, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, e)
		switch p.curr.Type {
		case Comma:
			p.next()
			if p.is(Rparen) {
				return nil, p.unexpected()
			}
		case Rparen:
		default:
			return nil, p.unexpected()
		}
	}
	return call, p.expect(Rparen)
}

func (p *Parser) parseIndex(left Expression) (Expression, error) {
	if err := p.expect(Lsquare); err != nil {
		return nil, err
	}
	ix := Index{
		Expr: left,
	}
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	ix.Index = expr
	return ix, p.expect(Rsquare)
}

func (p *Parser) parseDot(left Expression) (Expression, error) {
	p.next()
	ch := Chain{
		Left: left,
	}
	next, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	ch.Next = next
	return ch, nil
}

func (p *Parser) parseTernary(left Expression) (Expression, error) {
	var (
		expr If
		err  error
	)
	expr.Cdt = left
	p.next()
	expr.Csq, err = p.parseExpression(bindings[Question])
	if err != nil {
		return nil, err
	}
	if err := p.expect(Colon); err != nil {
		return nil, err
	}
	expr.Alt, err = p.parseExpression(bindings[Question])
	return expr, err
}

func (p *Parser) parseIdentifier() (Expression, error) {
	defer p.next()
	return createVariable(p.curr.Literal), nil
}

func (p *Parser) parseString() (Expression, error) {
	defer p.next()
	return createString(p.curr.Literal), nil
}

func (p *Parser) parseNumber() (Expression, error) {
	defer p.next()
	n, err := strconv.ParseFloat(p.curr.Literal, 64)
	if err == nil {
		return createNumber(n), nil
	}
	x, err := strconv.ParseInt(p.curr.Literal, 0, 64)
	if err != nil {
		return nil, err
	}
	return createNumber(float64(x)), nil
}

func (p *Parser) parseBool() (Expression, error) {
	defer p.next()
	b, err := strconv.ParseBool(p.curr.Literal)
	if err != nil {
		return nil, err
	}
	return createBool(b), nil
}

func (p *Parser) parseArray() (Expression, error) {
	if err := p.expect(Lsquare); err != nil {
		return nil, err
	}
	var arr Array
	for !p.done() && !p.is(Rsquare) {
		e, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		arr.List = append(arr.List, e)
		switch p.curr.Type {
		case Comma:
			p.next()
		case Rsquare:
		default:
			return nil, p.unexpected()
		}
	}
	return arr, p.expect(Rsquare)
}

func (p *Parser) parseHash() (Expression, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	obj := Hash{
		List: make(map[Expression]Expression),
	}
	for !p.done() && !p.is(Rbrace) {
		key, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		if err := p.expect(Colon); err != nil {
			return nil, err
		}
		val, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		obj.List[key] = val
		switch p.curr.Type {
		case Comma:
			p.next()
		case Rbrace:
		default:
			return nil, p.unexpected()
		}
	}
	return obj, p.expect(Rbrace)
}

func (p *Parser) parseGroup() (Expression, error) {
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	return expr, p.expect(Rparen)
}

func (p *Parser) parseUnary() (Expression, error) {
	u := Unary{
		Op: p.curr.Type,
	}
	p.next()
	right, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	u.Right = right
	return u, nil
}

func (p *Parser) parseKeyword() (Expression, error) {
	parse, ok := p.keywords[p.curr.Literal]
	if !ok {
		return nil, p.unexpected()
	}
	return parse()
}

func (p *Parser) parseLet() (Expression, error) {
	p.next()
	var let Let
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	let.Ident = p.curr.Literal
	p.next()
	if !p.is(Assign) {
		return nil, p.unexpected()
	}
	p.next()
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	let.Expr = expr
	return let, nil
}

func (p *Parser) parseConst() (Expression, error) {
	p.next()
	return nil, nil
}

func (p *Parser) parseFor() (Expression, error) {
	p.next()
	var (
		loop For
		err  error
	)
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	if !p.is(EOL) {
		loop.Init, err = p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(EOL); err != nil {
		return nil, err
	}
	if !p.is(EOL) {
		loop.Cdt, err = p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(EOL); err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		loop.Incr, err = p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
	}
	if err := p.expect(Rparen); err != nil {
		return nil, err
	}
	loop.Body, err = p.parseBlock()
	return loop, err
}

func (p *Parser) parseWhile() (Expression, error) {
	p.next()
	var (
		loop While
		err  error
	)
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	loop.Cdt, err = p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if err := p.expect(Rparen); err != nil {
		return nil, err
	}
	loop.Body, err = p.parseBlock()
	return loop, err
}

func (p *Parser) parseBreak() (Expression, error) {
	p.next()
	var br Break
	if p.is(Ident) {
		br.Label = p.curr.Literal
		p.next()
	}
	return br, nil
}

func (p *Parser) parseContinue() (Expression, error) {
	p.next()
	var ct Continue
	if p.is(Ident) {
		ct.Label = p.curr.Literal
		p.next()
	}
	return ct, nil
}

func (p *Parser) parseIf() (Expression, error) {
	p.next()
	var (
		expr If
		err  error
	)
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	expr.Cdt, err = p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if err := p.expect(Rparen); err != nil {
		return nil, err
	}
	expr.Csq, err = p.parseBlock()
	if err != nil {
		return nil, err
	}
	if p.is(Keyword) {
		expr.Alt, err = p.parseKeyword()
	}
	return expr, err
}

func (p *Parser) parseElse() (Expression, error) {
	p.next()
	if p.is(Keyword) {
		return p.parseKeyword()
	}
	return p.parseBlock()
}

func (p *Parser) parseThrow() (Expression, error) {
	p.next()
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	t := Throw{
		Expr: expr,
	}
	return t, nil
}

func (p *Parser) parseTry() (Expression, error) {
	p.next()
	var (
		try Try
		err error
	)
	try.Body, err = p.parseBlock()
	if err != nil {
		return nil, err
	}
	try.Catch, err = p.parseKeyword()
	return try, err
}

func (p *Parser) parseCatch() (Expression, error) {
	p.next()
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	if err := p.expect(Ident); err != nil {
		return nil, err
	}
	var (
		err   error
		catch Catch
	)
	catch = Catch{
		Err: p.curr.Literal,
	}
	if err = p.expect(Rparen); err != nil {
		return nil, err
	}
	catch.Body, err = p.parseBlock()
	return catch, err
}

func (p *Parser) parseSwitch() (Expression, error) {
	p.next()
	var (
		sw  Switch
		err error
	)
	if err := p.expect(Lparen); err != nil {
		return nil, err
	}
	sw.Cdt, err = p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if err := p.expect(Rparen); err != nil {
		return nil, err
	}
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	for !p.done() && !p.is(Rbrace) {

	}
	return sw, p.expect(Rbrace)
}

func (p *Parser) parseCase() (Expression, error) {
	p.next()
	var (
		ca  Case
		err error
	)
	ca.Value, err = p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if err := p.expect(Colon); err != nil {
		return nil, err
	}
	return ca, nil
}

func (p *Parser) parseFunction() (Expression, error) {
	p.next()
	var (
		fn  Function
		err error
	)
	if !p.is(Ident) {
		return nil, err
	}
	fn.Name = p.curr.Literal
	p.next()
	if err = p.expect(Lparen); err != nil {
		return nil, err
	}
	for !p.done() && !p.is(Rparen) {
		a, err := p.parseArgument()
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, a)
		switch p.curr.Type {
		case Comma:
			p.next()
			if p.is(Rparen) {
				return nil, p.unexpected()
			}
		case Rparen:
		default:
			return nil, p.unexpected()
		}
	}
	if err = p.expect(Rparen); err != nil {
		return nil, err
	}
	fn.Body, err = p.parseBlock()
	return fn, err
}

func (p *Parser) parseArgument() (Expression, error) {
	var (
		arg Argument
		err error
	)
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	arg.Ident = p.curr.Literal
	p.next()
	if p.is(Assign) {
		p.next()
		arg.Default, err = p.parseExpression(powLowest)
	}
	return arg, err
}

func (p *Parser) parseReturn() (Expression, error) {
	p.next()
	var (
		ret Return
		err error
	)
	ret.Expr, err = p.parseExpression(powLowest)
	return ret, err
}

func (p *Parser) parseBlock() (Expression, error) {
	var b Block
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	for !p.done() && !p.is(Rbrace) {
		e, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		b.List = append(b.List, e)
	}
	if err := p.expect(Rbrace); err != nil {
		return nil, err
	}
	if len(b.List) == 1 {
		return b.List[0], nil
	}
	return b, nil
}

func (p *Parser) parseExpression(pow int) (Expression, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}
	for !p.done() && !p.eol() && pow < bindings[p.curr.Type] {
		left, err = p.parseInfix(left)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseInfix(left Expression) (Expression, error) {
	fn, ok := p.infix[p.curr.Type]
	if !ok {
		return nil, p.unexpected()
	}
	return fn(left)
}

func (p *Parser) parsePrefix() (Expression, error) {
	fn, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, p.unexpected()
	}
	return fn()
}

func (p *Parser) registerInfix(kind rune, fn func(Expression) (Expression, error)) {
	p.infix[kind] = fn
}

func (p *Parser) registerPrefix(kind rune, fn func() (Expression, error)) {
	p.prefix[kind] = fn
}

func (p *Parser) registerKeyword(kw string, fn func() (Expression, error)) {
	p.keywords[kw] = fn
}

func (p *Parser) skip(kind rune) {
	for p.is(kind) {
		p.next()
	}
}

func (p *Parser) expect(kind rune) error {
	if !p.is(kind) {
		return p.unexpected()
	}
	p.next()
	return nil
}

func (p *Parser) unexpected() error {
	return fmt.Errorf("unexpected token %s", p.curr)
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) eol() bool {
	return p.is(EOL)
}

func (p *Parser) done() bool {
	return p.is(EOF)
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

const (
	powLowest int = iota
	powAssign
	powTernary
	powLogical
	powBitwise
	powEqual
	powCompare
	powShift
	powAdd
	powMul
	powCall
	powIndex
	powDot
	powUnary
)

var bindings = map[rune]int{
	Assign:       powAssign,
	AddAssign:    powAssign,
	SubAssign:    powAssign,
	DivAssign:    powAssign,
	MulAssign:    powAssign,
	PowAssign:    powAssign,
	ModAssign:    powAssign,
	BandAssign:   powAssign,
	BorAssign:    powAssign,
	LshiftAssign: powAssign,
	RshiftAssign: powAssign,
	Question:     powTernary,
	Colon:        powAssign,
	And:          powLogical,
	Or:           powLogical,
	Band:         powBitwise,
	Bor:          powBitwise,
	Eq:           powEqual,
	Ne:           powEqual,
	Lt:           powCompare,
	Le:           powCompare,
	Gt:           powCompare,
	Ge:           powCompare,
	Lshift:       powShift,
	Rshift:       powShift,
	Add:          powAdd,
	Sub:          powAdd,
	Mul:          powMul,
	Div:          powMul,
	Mod:          powMul,
	Pow:          powMul,
	Lparen:       powCall,
	Lsquare:      powIndex,
	Dot:          powDot,
}
