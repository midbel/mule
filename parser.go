package mule

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	depth  int
	macros map[string]func() error
}

func Parse(r io.Reader) (*Parser, error) {
	p := Parser{
		scan: Scan(r),
	}
	p.macros = map[string]func() error{
		"include":  p.parseIncludeMacro,
		"readfile": p.parseReadFileMacro,
		"env":      p.parseEnvMacro,
	}
	p.next()
	p.next()
	return &p, nil
}

func (p *Parser) Parse() (*Collection, error) {
	root := Root()
	for !p.done() {
		if err := p.parse(root); err != nil {
			return nil, err
		}
	}
	return root, nil
}

func (p *Parser) parse(root *Collection) error {
	p.skip(Comment)

	var err error
	switch {
	case p.is(Macro):
		err = p.parseMacro()
	case p.is(Ident):
		child := Make(p.getCurrLiteral(), root)
		p.next()
		err = p.parseBraces("collection", func() error {
			return p.parseItem(child)
		})
		if err != nil {
			break
		}
		root.Collections = append(root.Collections, child)
	default:
		err = p.parseItem(root)
	}
	return err
}

func (p *Parser) parseItem(root *Collection) error {
	if !p.is(Keyword) {
		return p.unexpected("collection")
	}
	var (
		err error
		eol bool
	)
	switch p.getCurrLiteral() {
	case "collection":
		p.next()
		err = p.parse(root)
	case "before", "beforeAll":
		p.next()
		eol = true
		root.BeforeAll, err = p.parseScript()
	case "beforeEach":
		p.next()
		eol = true
		root.BeforeEach, err = p.parseScript()
	case "after", "afterAll":
		p.next()
		eol = true
		root.AfterAll, err = p.parseScript()
	case "afterEach":
		p.next()
		eol = true
		root.AfterEach, err = p.parseScript()
	case "auth":
		p.next()
		root.Auth, err = p.parseAuth()
	case "url":
		p.next()
		eol = true
		root.URL, err = p.parseValue()
	case "query":
		p.next()
		root.Query, err = p.parseSet("query")
	case "headers":
		p.next()
		root.Headers, err = p.parseSet("headers")
	case "variables":
		p.next()
		err = p.parseVariables(root)
	case "get", "post", "put", "patch", "delete":
		req, err1 := p.parseRequest()
		if err1 != nil {
			err = err1
			break
		}
		root.Requests = append(root.Requests, req)
	default:
		err = p.unexpected("collection")
	}
	if err == nil && eol && !p.is(EOL) {
		err = p.unexpected("collection")
	}
	p.skip(EOL)
	return err
}

func (p *Parser) parseScript() (string, error) {
	if p.is(Macro) {

	}
	if !p.is(String) {
		return "", p.unexpected("script")
	}
	script := p.getCurrLiteral()
	p.next()
	return script, nil
}

func (p *Parser) parseValue() (Value, error) {
	switch {
	case p.is(Macro):
		return nil, p.parseMacro()
	case p.is(Ident) || p.is(String) || p.is(Number) || p.is(Keyword):
		defer p.next()
		return createLiteral(p.getCurrLiteral()), nil
	case p.is(Variable):
		defer p.next()
		return createVariable(p.getCurrLiteral()), nil
	case p.is(Quote):
		p.next()
		var cs compound
		for !p.done() && !p.is(Quote) {
			v, err := p.parseValue()
			if err != nil {
				return nil, err
			}
			cs = append(cs, v)
		}
		if !p.is(Quote) {
			return nil, p.unexpected("value")
		}
		p.next()
		if len(cs) == 1 {
			return cs[0], nil
		}
		return cs, nil
	default:
		return nil, p.unexpected("value")
	}
}

func (p *Parser) parseBody() (Body, error) {
	if p.is(Lbrace) {
		set, err := p.parseSet("body")
		if err != nil {
			return nil, err
		}
		return jsonify(set), nil
	}
	if !p.is(Ident) {
		return nil, p.unexpected("body")
	}
	var create func(Set) Body
	switch p.getCurrLiteral() {
	case "urlencoded":
		create = urlEncoded
	case "json":
		create = jsonify
	case "xml":
		create = xmlify
	case "text":
		create = textify
	default:
		return nil, p.unexpected("body")
	}
	p.next()
	set, err := p.parseSet("body")
	if err != nil {
		return nil, err
	}
	return create(set), nil
}

func (p *Parser) parseAuth() (Authorization, error) {
	if !p.is(Ident) {
		return nil, p.unexpected("auth")
	}
	var (
		auth Authorization
		err  error
	)
	switch p.getCurrLiteral() {
	case "basic":
		auth, err = p.parseBasicAuth()
	case "bearer":
		auth, err = p.parseBearerAuth()
	case "digest":
		return nil, fmt.Errorf("digest: not yet implemented")
	case "jwt":
		return nil, fmt.Errorf("jwt: not yet implemented")
	default:
		return nil, p.unexpected("auth")
	}
	return auth, err
}

func (p *Parser) parseBearerAuth() (Authorization, error) {
	p.next()
	var (
		auth bearer
		err  error
	)
	if !p.is(Lbrace) {
		auth.Token, err = p.parseValue()
		return auth, err
	}
	err = p.parseBraces("bearer", func() error {
		if !p.is(Keyword) {
			return p.unexpected("bearer")
		}
		if p.getCurrLiteral() != "token" {
			return p.unexpected("bearer")
		}
		p.next()
		token, err := p.parseValue()
		if err == nil {
			auth.Token = token
		}
		return err
	})
	return auth, err
}

func (p *Parser) parseBasicAuth() (Authorization, error) {
	p.next()
	var (
		auth basic
		err  error
	)
	err = p.parseBraces("basic", func() error {
		if !p.is(Keyword) {
			return p.unexpected("basic")
		}
		var err error
		switch p.getCurrLiteral() {
		case "username":
			p.next()
			auth.User, err = p.parseValue()
		case "password":
			p.next()
			auth.Pass, err = p.parseValue()
		default:
			return p.unexpected("basic")
		}
		if err == nil {
			if !p.is(EOL) {
				return p.unexpected("basic")
			}
			p.next()
		}
		return err
	})
	return auth, err
}

func (p *Parser) parseRequest() (*Request, error) {
	req := Request{
		Method: strings.ToUpper(p.getCurrLiteral()),
	}
	p.next()
	if !p.is(Lbrace) {
		if !p.is(Ident) && !p.is(String) {
			return nil, p.unexpected("request")
		}
		req.Name = p.getCurrLiteral()
		p.next()
	}
	err := p.parseBraces("request", func() error {
		if !p.is(Keyword) {
			return p.unexpected("request")
		}
		var (
			err error
			eol bool
		)
		switch p.getCurrLiteral() {
		case "depends":
			p.next()
			eol = true
			for !p.is(EOL) && p.done() {
				d, err := p.parseValue()
				if err != nil {
					return err
				}
				req.Depends = append(req.Depends, d)
			}
		case "body":
			p.next()
			req.Body, err = p.parseBody()
		case "compress":
			p.next()
			req.Compressed, err = p.parseValue()
		case "before":
			p.next()
			eol = true
			req.Before, err = p.parseScript()
		case "after":
			p.next()
			eol = true
			req.After, err = p.parseScript()
		case "url":
			p.next()
			eol = true
			req.URL, err = p.parseValue()
		case "retry":
			p.next()
			eol = true
			req.Retry, err = p.parseValue()
		case "timeout":
			p.next()
			eol = true
			req.Timeout, err = p.parseValue()
		case "redirect":
		case "auth":
			p.next()
			req.Auth, err = p.parseAuth()
		case "query":
			p.next()
			req.Query, err = p.parseSet("query")
		case "headers":
			p.next()
			req.Headers, err = p.parseSet("headers")
		default:
			err = p.unexpected("request")
		}
		if err == nil && eol && !p.is(EOL) {
			err = p.unexpected("request")
		}
		p.skip(EOL)
		return err
	})
	return &req, err
}

func (p *Parser) parseSet(ctx string) (Set, error) {
	set := make(Set)
	return set, p.parseBraces(ctx, func() error {
		p.skip(EOL)
		if !p.is(Ident) && !p.is(String) && !p.is(Keyword) {
			return p.unexpected("set")
		}
		ident := p.getCurrLiteral()
		p.next()
		for !p.done() && !p.is(EOL) {
			v, err := p.parseValue()
			if err != nil {
				return err
			}
			set[ident] = append(set[ident], v)
		}
		if !p.is(EOL) {
			return p.unexpected("set")
		}
		p.next()
		return nil
	})
}

func (p *Parser) parseVariables(root *Collection) error {
	return p.parseBraces("variables", func() error {
		p.skip(EOL)
		if !p.is(Ident) && !p.is(Keyword) && !p.is(String) {
			return p.unexpected("variables")
		}
		ident := p.getCurrLiteral()
		p.next()
		value, err := p.parseValue()
		if err != nil {
			return err
		}
		if !p.is(EOL) {
			return p.unexpected("variables")
		}
		p.next()
		root.Define(ident, value)
		return nil
	})
}

func (p *Parser) parseBraces(ctx string, fn func() error) error {
	if !p.is(Lbrace) {
		return p.unexpected(ctx)
	}
	p.next()
	for !p.done() && !p.is(Rbrace) {
		if err := fn(); err != nil {
			return err
		}
	}
	if !p.is(Rbrace) {
		return p.unexpected(ctx)
	}
	p.next()
	return nil
}

func (p *Parser) parseMacro() error {
	fn, ok := p.macros[p.getCurrLiteral()]
	if !ok {
		return fmt.Errorf("%s: undefined macro")
	}
	return fn()
}

func (p *Parser) parseIncludeMacro() error {
	p.next()
	r, err := os.Open(p.getCurrLiteral())
	if err != nil {
		return err
	}
	defer r.Close()
	p.next()

	px, err := Parse(r)
	if err != nil {
		return err
	}
	_, err = px.Parse()
	return err
}

func (p *Parser) parseReadFileMacro() error {
	p.next()
	buf, err := os.ReadFile(p.getCurrLiteral())
	defer p.next()
	if err != nil {
		return err
	}
	fmt.Println(string(buf))
	return nil
}

func (p *Parser) parseEnvMacro() error {
	p.next()
	os.Getenv(p.getCurrLiteral())
	p.next()
	return nil
}

func (p *Parser) done() bool {
	return p.is(EOF)
}

func (p *Parser) getCurrLiteral() string {
	return p.curr.Literal
}

func (p *Parser) is(kind rune) bool {
	return p.curr.Type == kind
}

func (p *Parser) skip(kind rune) {
	for p.is(kind) {
		p.next()
	}
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) enter() {
	p.depth++
}

func (p *Parser) leave() {
	p.depth--
}

func (p *Parser) nested() bool {
	return p.depth > 0
}

func (p *Parser) unexpected(ctx string) error {
	return unexpected(ctx, p.curr)
}

func unexpected(ctx string, tok Token) error {
	pos := tok.Position
	return fmt.Errorf("[%d:%d] unexpected token in %s: %s", pos.Line, pos.Column, ctx, tok)
}
