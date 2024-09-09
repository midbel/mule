package play

import (
	"fmt"
	"io"
	"strconv"
)

const (
	powLowest int = iota
	powComma
	powAssign
	powKeyword
	powOr
	powAnd
	powEq
	powCmp
	powAdd
	powMul
	powPow
	powObject
	powPostfix
	powPrefix
	powAccess
	powGroup
)

var bindings = map[rune]int{
	Comma:    powComma,
	Question: powAssign,
	Assign:   powAssign,
	Colon:    powAssign,
	Arrow:    powAssign,
	Keyword:  powAssign,
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
	Lparen:   powGroup,
	Dot:      powAccess,
	Lsquare:  powAccess,
	Lcurly:   powObject,
	Incr:     powPostfix,
	Decr:     powPrefix,
}

type (
	prefixFunc func() (Node, error)
	infixFunc  func(Node) (Node, error)
)

type Parser struct {
	prefix map[rune]prefixFunc
	infix  map[rune]infixFunc

	scan *Scanner
	curr Token
	peek Token
}

func ParseReader(r io.Reader) (Node, error) {
	p := Parse(r)
	return p.Parse()
}

func Parse(r io.Reader) *Parser {
	p := Parser{
		scan:   Scan(r),
		prefix: make(map[rune]prefixFunc),
		infix:  make(map[rune]infixFunc),
	}

	p.registerPrefix(Not, p.parseNot)
	p.registerPrefix(Sub, p.parseRev)
	p.registerPrefix(Add, p.parseFloat)
	p.registerPrefix(Incr, p.parseIncrPrefix)
	p.registerPrefix(Decr, p.parseDecrPrefix)
	p.registerPrefix(Ident, p.parseIdent)
	p.registerPrefix(Text, p.parseString)
	p.registerPrefix(Number, p.parseNumber)
	p.registerPrefix(Boolean, p.parseBoolean)
	p.registerPrefix(Lparen, p.parseGroup)
	p.registerPrefix(Lsquare, p.parseList)
	p.registerPrefix(Lcurly, p.parseMap)
	p.registerPrefix(Keyword, p.parseKeyword)
	p.registerPrefix(TypeOf, p.parseTypeOf)
	p.registerPrefix(Del, p.parseDelete)
	p.registerPrefix(Spread, p.parseSpread)
	p.registerPrefix(Decorate, p.parseDecorator)

	p.registerInfix(Dot, p.parseDot)
	p.registerInfix(Optional, p.parseDot)
	p.registerInfix(Assign, p.parseAssign)
	p.registerInfix(Nullish, p.parseBinary)
	p.registerInfix(Add, p.parseBinary)
	p.registerInfix(Sub, p.parseBinary)
	p.registerInfix(Mul, p.parseBinary)
	p.registerInfix(Div, p.parseBinary)
	p.registerInfix(Mod, p.parseBinary)
	p.registerInfix(Pow, p.parseBinary)
	p.registerInfix(And, p.parseBinary)
	p.registerInfix(Or, p.parseBinary)
	p.registerInfix(Eq, p.parseBinary)
	p.registerInfix(Ne, p.parseBinary)
	p.registerInfix(Lt, p.parseBinary)
	p.registerInfix(Le, p.parseBinary)
	p.registerInfix(Gt, p.parseBinary)
	p.registerInfix(Ge, p.parseBinary)
	p.registerInfix(Nullish, p.parseBinary)
	p.registerInfix(InstanceOf, p.parseBinary)
	p.registerInfix(Incr, p.parseIncrPostfix)
	p.registerInfix(Decr, p.parseDecrPostfix)
	p.registerInfix(Arrow, p.parseArrow)
	p.registerInfix(Lparen, p.parseCall)
	p.registerInfix(Lsquare, p.parseIndex)
	p.registerInfix(Question, p.parseTernary)
	p.registerInfix(Keyword, p.parseKeywordCtrl)

	p.next()
	p.next()
	return &p
}

func (p *Parser) Parse() (Node, error) {
	var body Body
	p.skip(p.eol)
	for !p.done() {
		n, err := p.parseNode()
		if err != nil {
			return nil, err
		}
		body.Nodes = append(body.Nodes, n)
		p.skip(p.eol)
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
	switch p.curr.Literal {
	case "let":
		return p.parseLet()
	case "const":
		return p.parseConst()
	case "using":
		return p.parseUsing()
	case "if":
		return p.parseIf()
	case "else":
		return p.parseElse()
	case "switch":
		return p.parseSwitch()
	case "case":
		return p.parseCase()
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
	case "catch":
		return p.parseCatch()
	case "finally":
		return p.parseFinally()
	case "throw":
		return p.parseThrow()
	case "null":
		return p.parseNull()
	case "undefined":
		return p.parseUndefined()
	default:
		return nil, fmt.Errorf("%s: keyword not supported/known", p.curr.Literal)
	}
}

func (p *Parser) parseKeywordCtrl(left Node) (Node, error) {
	switch p.curr.Literal {
	case "of":
		expr := OfCtrl{
			Ident: left,
		}
		p.next()
		right, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		expr.Iter = right
		return expr, nil
	case "in":
		expr := InCtrl{
			Ident: left,
		}
		p.next()
		right, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		expr.Iter = right
		return expr, nil
	default:
		return nil, fmt.Errorf("%s: keyword not supported/known", p.curr.Literal)
	}
}

func (p *Parser) parseLet() (Node, error) {
	expr := Let{
		Position: p.curr.Position,
	}
	p.next()
	ident, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	expr.Node = ident
	if p.is(Keyword) {
		return p.parseKeywordCtrl(expr)
	}
	if !p.is(Assign) {
		expr.Node = Assignment{
			Ident: ident,
			Node:  Undefined{},
		}
		return expr, nil
	}
	p.next()
	value, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = Assignment{
		Ident: ident,
		Node:  value,
	}
	return expr, nil
}

func (p *Parser) parseConst() (Node, error) {
	expr := Const{
		Position: p.curr.Position,
	}
	p.next()
	ident, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	if !p.is(Assign) {
		return nil, p.unexpected()
	}
	p.next()
	value, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = Assignment{
		Ident: ident,
		Node:  value,
	}
	return expr, nil
}

func (p *Parser) parseUsing() (Node, error) {
	return nil, nil
}

func (p *Parser) parseIf() (Node, error) {
	expr := If{
		Position: p.curr.Position,
	}
	p.next()
	cdt, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	expr.Cdt = cdt
	if expr.Csq, err = p.parseBody(); err != nil {
		return nil, err
	}
	p.skip(p.eol)
	if p.is(Keyword) {
		expr.Alt, err = p.parseKeyword()
	}
	return expr, err
}

func (p *Parser) parseElse() (Node, error) {
	p.next()
	if p.is(Keyword) {
		return p.parseKeyword()
	}
	node, err := p.parseBody()
	if err == nil {
		p.skip(p.eol)
	}
	return node, err
}

func (p *Parser) parseSwitch() (Node, error) {
	expr := Switch{
		Position: p.curr.Position,
	}
	p.next()
	cdt, err := p.parseSwitchIdent()
	if err != nil {
		return nil, err
	}
	expr.Cdt = cdt
	expr.Cases, expr.Default, err = p.parseSwitchCases()
	if err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *Parser) parseCase() (Node, error) {
	expr := Case{
		Position: p.curr.Position,
	}
	p.next()
	ident, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	expr.Value = ident
	if !p.is(Colon) {
		return nil, p.unexpected()
	}
	p.next()
	p.skip(p.eol)
	var body Body
	for !p.done() && !p.is(Rcurly) {
		if p.is(Keyword) && (p.curr.Literal == "case" || p.curr.Literal == "default") {
			break
		}
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		p.skip(p.eol)
		body.Nodes = append(body.Nodes, expr)
	}
	expr.Body = body
	return expr, nil
}

func (p *Parser) parseSwitchCases() ([]Node, Node, error) {
	if !p.is(Lcurly) {
		return nil, nil, p.unexpected()
	}
	p.next()
	p.skip(p.eol)

	var nodes []Node
	for !p.done() && !p.is(Rcurly) {
		if !p.is(Keyword) {
			return nil, nil, p.unexpected()
		}
		if p.curr.Literal == "default" {
			break
		}
		expr, err := p.parseExpression(powKeyword)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, expr)
	}

	var alt Node
	if p.is(Keyword) && p.curr.Literal == "default" {
		p.next()
		if !p.is(Colon) {
			return nil, nil, p.unexpected()
		}
		p.next()
		p.skip(p.eol)
		var body Body
		for !p.done() && !p.is(Rcurly) {
			node, err := p.parseExpression(powKeyword)
			if err != nil {
				return nil, nil, err
			}
			p.skip(p.eol)
			body.Nodes = append(body.Nodes, node)
		}
		alt = body
	}
	if !p.is(Rcurly) {
		return nil, nil, p.unexpected()
	}
	p.next()
	return nodes, alt, nil
}

func (p *Parser) parseSwitchIdent() (Node, error) {
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	ident, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return ident, nil
}

func (p *Parser) parseDo() (Node, error) {
	do := Do{
		Position: p.curr.Position,
	}
	p.next()
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	do.Body = body
	p.skip(p.eol)
	if !p.is(Keyword) && p.curr.Literal != "while" {
		return nil, p.unexpected()
	}
	p.next()
	if do.Cdt, err = p.parseCondition(); err != nil {
		return nil, err
	}
	return do, nil
}

func (p *Parser) parseWhile() (Node, error) {
	expr := While{
		Position: p.curr.Position,
	}
	p.next()
	cdt, err := p.parseCondition()
	if err != nil {
		return nil, err
	}
	expr.Cdt = cdt
	if expr.Body, err = p.parseBody(); err != nil {
		return nil, err
	}
	p.skip(p.eol)
	return expr, nil
}

func (p *Parser) parseCondition() (Node, error) {
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return expr, nil
}

func (p *Parser) parseBody() (Node, error) {
	if !p.is(Lcurly) {
		return p.parseExpression(powLowest)
	}
	p.next()
	var b Body
	for !p.done() && !p.is(Rcurly) {
		p.skip(p.eol)
		n, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		b.Nodes = append(b.Nodes, n)
		p.skip(p.eol)
	}
	if !p.is(Rcurly) {
		return nil, p.unexpected()
	}
	p.next()
	return b, nil
}

func (p *Parser) parseFor() (Node, error) {
	loop := For{
		Position: p.curr.Position,
	}
	p.next()
	ctrl, err := p.parseForControl()
	if err != nil {
		return nil, err
	}
	loop.Ctrl = ctrl

	if loop.Body, err = p.parseBody(); err != nil {
		return nil, err
	}
	p.skip(p.eol)
	return loop, nil
}

func (p *Parser) parseForControl() (Node, error) {
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()

	var ctrl ForCtrl
	if !p.is(EOL) {
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		switch expr.(type) {
		case OfCtrl, InCtrl:
			if !p.is(Rparen) {
				return nil, p.unexpected()
			}
			p.next()
			return expr, nil
		default:
			ctrl.Init = expr
		}
	}
	if !p.is(EOL) {
		return nil, p.unexpected()
	}
	p.next()
	if !p.is(EOL) {
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		ctrl.Cdt = expr
	}
	if !p.is(EOL) {
		return nil, p.unexpected()
	}
	p.next()
	if !p.is(Rparen) {
		expr, err := p.parseExpression(powLowest)
		if err != nil {
			return nil, err
		}
		ctrl.After = expr
	}

	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return ctrl, nil
}

func (p *Parser) parseBreak() (Node, error) {
	expr := Break{
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Ident) {
		expr.Label = p.curr.Literal
		p.next()
	}
	return expr, nil
}

func (p *Parser) parseContinue() (Node, error) {
	expr := Continue{
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Ident) {
		expr.Label = p.curr.Literal
		p.next()
	}
	return expr, nil
}

func (p *Parser) parseReturn() (Node, error) {
	expr := Return{
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseArrow(left Node) (Node, error) {
	var (
		args []Node
		err  error
	)
	switch a := left.(type) {
	case List:
		args = a.Nodes
	case Group:
		args = a.Nodes
	default:
		args = append(args, left)
	}
	fn := Func{
		Args:     args,
		Arrow:    true,
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Lcurly) {
		fn.Body, err = p.parseBody()
	} else {
		fn.Body, err = p.parseExpression(powLowest)
	}
	return fn, err
}

func (p *Parser) parseFunction() (Node, error) {
	fn := Func{
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Ident) {
		fn.Ident = p.curr.Literal
		p.next()
	}
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		arg, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, arg)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	fn.Body = body
	return fn, nil
}

func (p *Parser) parseImport() (Node, error) {
	expr := Import{
		Position: p.curr.Position,
	}
	p.next()
	switch {
	case p.is(Mul):
		p.next()
		if !p.is(Keyword) && p.curr.Literal != "as" {
			return nil, p.unexpected()
		}
		p.next()
		if !p.is(Ident) {
			return nil, p.unexpected()
		}
		p.next()
	case p.is(Lcurly):
		p.next()
		for !p.done() && !p.is(Rcurly) {

		}
		if !p.is(Rcurly) {
			return nil, p.unexpected()
		}
		p.next()
	case p.is(Ident):
		p.next()
	case p.is(Keyword) && p.curr.Literal == "from":
	default:
		return nil, p.unexpected()
	}
	if !p.is(Keyword) && p.curr.Literal != "from" {
		return nil, p.unexpected()
	}
	p.next()
	if !p.is(String) {
		return nil, p.unexpected()
	}
	expr.From = p.curr.Literal
	p.next()
	return expr, nil
}

func (p *Parser) parseExport() (Node, error) {
	expr := Export{
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseTry() (Node, error) {
	try := Try{
		Position: p.curr.Position,
	}
	p.next()
	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	try.Node = body
	if p.is(Keyword) && p.curr.Literal == "catch" {
		try.Catch, err = p.parseKeyword()
		if err != nil {
			return nil, err
		}
	}
	if p.is(Keyword) && p.curr.Literal == "finally" {
		try.Finally, err = p.parseKeyword()
		if err != nil {
			return nil, err
		}
	}
	return try, nil
}

func (p *Parser) parseCatch() (Node, error) {
	catch := Catch{
		Position: p.curr.Position,
	}
	p.next()
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()
	ident, err := p.parseIdent()
	if err != nil {
		return nil, err
	}
	catch.Err = ident
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	catch.Body, err = p.parseBody()
	return catch, err
}

func (p *Parser) parseFinally() (Node, error) {
	p.next()
	return p.parseBody()
}

func (p *Parser) parseThrow() (Node, error) {
	t := Throw{
		Position: p.curr.Position,
	}
	p.next()
	expr, err := p.parseExpression(powLowest)
	if err != nil {
		return nil, err
	}
	t.Node = expr
	return t, nil
}

func (p *Parser) parseNull() (Node, error) {
	defer p.next()
	expr := Null{
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseUndefined() (Node, error) {
	defer p.next()
	expr := Undefined{
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseExpression(pow int) (Node, error) {
	fn, ok := p.prefix[p.curr.Type]
	if !ok {
		return nil, fmt.Errorf("unknown prefix expression %s", p.curr)
	}
	left, err := fn()
	if err != nil {
		return nil, err
	}
	for !p.done() && !p.eol() && pow < p.power() {
		fn, ok := p.infix[p.curr.Type]
		if !ok {
			return nil, fmt.Errorf("unknown infix expression %s", p.curr)
		}
		if left, err = fn(left); err != nil {
			return nil, err
		}
	}
	return left, nil
}

func (p *Parser) parseDelete() (Node, error) {
	expr := Delete{
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseTypeOf() (Node, error) {
	expr := Unary{
		Op:       TypeOf,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseDecorator() (Node, error) {
	return nil, nil
}

func (p *Parser) parseSpread() (Node, error) {
	expr := Extend{
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseNot() (Node, error) {
	expr := Unary{
		Op:       Not,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseFloat() (Node, error) {
	expr := Unary{
		Op:       Add,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseRev() (Node, error) {
	expr := Unary{
		Op:       Sub,
		Position: p.curr.Position,
	}
	p.next()
	n, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	expr.Node = n
	return expr, nil
}

func (p *Parser) parseIncrPrefix() (Node, error) {
	incr := Increment{
		Position: p.curr.Position,
	}
	p.next()
	right, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	incr.Node = right
	return incr, nil
}

func (p *Parser) parseDecrPrefix() (Node, error) {
	decr := Decrement{
		Position: p.curr.Position,
	}
	p.next()
	right, err := p.parseExpression(powPrefix)
	if err != nil {
		return nil, err
	}
	decr.Node = right
	return decr, nil
}

func (p *Parser) parseIncrPostfix(left Node) (Node, error) {
	incr := Increment{
		Node:     left,
		Post:     true,
		Position: p.curr.Position,
	}
	p.next()
	return incr, nil
}

func (p *Parser) parseDecrPostfix(left Node) (Node, error) {
	decr := Decrement{
		Node:     left,
		Post:     true,
		Position: p.curr.Position,
	}
	p.next()
	return decr, nil
}

func (p *Parser) parseIdent() (Node, error) {
	defer p.next()
	if !p.is(Ident) {
		return nil, p.unexpected()
	}
	expr := Identifier{
		Name:     p.curr.Literal,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseString() (Node, error) {
	defer p.next()
	if !p.is(Text) {
		return nil, p.unexpected()
	}
	expr := Literal[string]{
		Value:    p.curr.Literal,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseNumber() (Node, error) {
	if !p.is(Number) {
		return nil, p.unexpected()
	}
	n, err := strconv.ParseFloat(p.curr.Literal, 64)
	if err != nil {
		return nil, err
	}
	defer p.next()
	expr := Literal[float64]{
		Value:    n,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseBoolean() (Node, error) {
	if !p.is(Boolean) {
		return nil, p.unexpected()
	}
	n, err := strconv.ParseBool(p.curr.Literal)
	if err != nil {
		return nil, err
	}
	defer p.next()
	expr := Literal[bool]{
		Value:    n,
		Position: p.curr.Position,
	}
	return expr, nil
}

func (p *Parser) parseList() (Node, error) {
	if !p.is(Lsquare) {
		return nil, p.unexpected()
	}
	list := List{
		Position: p.curr.Position,
	}
	p.next()
	for !p.done() && !p.is(Rsquare) {
		p.skip(p.eol)
		n, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		list.Nodes = append(list.Nodes, n)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rsquare):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rsquare) {
		return nil, fmt.Errorf("missing ] at end of array")
	}
	p.next()
	return list, nil
}

func (p *Parser) parseKey() (Node, error) {
	if p.is(Text) {
		return p.parseString()
	}
	if p.is(Number) {
		return p.parseNumber()
	}
	if p.is(Boolean) {
		return p.parseBoolean()
	}
	if p.is(Spread) {
		return p.parseSpread()
	}

	if !p.is(Ident) {
		return nil, p.unexpected()
	}

	ident := Identifier{
		Name:     p.curr.Literal,
		Position: p.curr.Position,
	}
	p.next()
	if p.is(Comma) || p.is(Colon) {
		return ident, nil
	}
	if !p.is(Lparen) {
		return nil, p.unexpected()
	}
	p.next()

	fn := Func{
		Ident:    ident.Name,
		Position: ident.Position,
	}

	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		arg, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		fn.Args = append(fn.Args, arg)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()

	body, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	p.skip(p.eol)
	fn.Body = body
	return fn, nil
}

func (p *Parser) parseMap() (Node, error) {
	if !p.is(Lcurly) {
		return nil, p.unexpected()
	}
	obj := Map{
		Position: p.curr.Position,
		Nodes:    make(map[Node]Node),
	}
	p.next()
	for !p.done() && !p.is(Rcurly) {
		p.skip(p.eol)
		key, err := p.parseKey()
		if err != nil {
			return nil, err
		}
		if p.is(Comma) || p.is(Rcurly) {
			val := key
			if fn, ok := key.(Func); ok {
				key = Identifier{
					Name: fn.Ident,
				}
				val = fn
			}
			obj.Nodes[key] = val
			if p.is(Comma) {
				p.next()
			}
			p.skip(p.eol)
			continue
		}
		if !p.is(Colon) {
			return nil, p.unexpected()
		}
		p.next()
		val, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		obj.Nodes[key] = val
		switch {
		case p.is(EOL):
			p.skip(p.eol)
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rcurly):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rcurly) {
		return nil, p.unexpected()
	}
	p.next()
	return obj, nil
}

func (p *Parser) parseGroup() (Node, error) {
	grp := Group{
		Position: p.curr.Position,
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		node, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		grp.Nodes = append(grp.Nodes, node)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	p.skip(p.eol)
	return grp, nil
}

func (p *Parser) parseDot(left Node) (Node, error) {
	access := Access{
		Optional: p.is(Optional),
		Node:     left,
		Position: p.curr.Position,
	}
	p.next()
	expr, err := p.parseExpression(powAccess)
	if err != nil {
		return nil, err
	}
	access.Ident = expr
	return access, nil
}

func (p *Parser) parseAssign(left Node) (Node, error) {
	expr := Assignment{
		Ident: left,
	}
	p.next()
	right, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	expr.Node = right
	return expr, nil
}

func (p *Parser) parseBinary(left Node) (Node, error) {
	expr := Binary{
		Left:     left,
		Op:       p.curr.Type,
		Position: p.curr.Position,
	}
	p.next()

	right, err := p.parseExpression(bindings[expr.Op])
	if err != nil {
		return nil, err
	}
	expr.Right = right
	return expr, nil
}

func (p *Parser) parseTernary(left Node) (Node, error) {
	expr := If{
		Cdt:      left,
		Position: p.curr.Position,
	}
	p.next()
	csq, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	if !p.is(Colon) {
		return nil, p.unexpected()
	}
	p.next()

	alt, err := p.parseExpression(powAssign)
	if err != nil {
		return nil, err
	}
	expr.Csq = csq
	expr.Alt = alt
	return expr, nil
}

func (p *Parser) parseIndex(left Node) (Node, error) {
	ix := Index{
		Position: p.curr.Position,
		Ident:    left,
	}
	p.next()
	x, err := p.parseExpression(powAccess)
	if err != nil {
		return nil, err
	}
	ix.Expr = x
	if !p.is(Rsquare) {
		return nil, p.unexpected()
	}
	p.next()
	return ix, nil
}

func (p *Parser) parseCall(left Node) (Node, error) {
	call := Call{
		Ident:    left,
		Position: p.curr.Position,
	}
	p.next()
	for !p.done() && !p.is(Rparen) {
		p.skip(p.eol)
		a, err := p.parseExpression(powComma)
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, a)
		switch {
		case p.is(Comma):
			p.next()
			p.skip(p.eol)
		case p.is(Rparen):
		default:
			return nil, p.unexpected()
		}
	}
	if !p.is(Rparen) {
		return nil, p.unexpected()
	}
	p.next()
	return call, nil
}

func (p *Parser) registerPrefix(kind rune, fn prefixFunc) {
	p.prefix[kind] = fn
}

func (p *Parser) registerInfix(kind rune, fn infixFunc) {
	p.infix[kind] = fn
}

func (p *Parser) power() int {
	pow, ok := bindings[p.curr.Type]
	if !ok {
		return powLowest
	}
	return pow
}

func (p *Parser) skip(ok func() bool) {
	for ok() {
		p.next()
	}
}

func (p *Parser) eol() bool {
	return p.is(EOL)
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

func (p *Parser) unexpected() error {
	pos := p.curr.Position
	return fmt.Errorf("%s unexpected token at %d:%d", p.curr, pos.Line, pos.Column)
}
