package mule

import (
	"fmt"
	"io"
	"net/url"
)

type Parser struct {
	scan *Scanner
	curr Token
	peek Token

	depth    int
	macros   map[string]func() error
	dispatch map[string]func(*Collection) error
}

func Parse(r io.Reader) (*Parser, error) {
	p := Parser{
		scan: Scan(r),
	}
	p.macros = map[string]func() error{
		"readfile": p.parseReadFileMacro,
		"include":  p.parseIncludeMacro,
	}
	p.dispatch = map[string]func(*Collection) error{
		"collection": p.parseCollection,
		"variables":  p.parseVariables,
		"query":      nil,
		"headers":    nil,
		"tls":        nil,
		"username":   nil,
		"password":   nil,
		"url":        p.parseURL,
		"before":     nil,
		"beforeAll":  nil,
		"beforeEach": nil,
		"after":      nil,
		"afterAll":   nil,
		"afterEach":  nil,
		"get":        p.parseRequest,
		"post":       p.parseRequest,
		"put":        p.parseRequest,
		"patch":      p.parseRequest,
		"delete":     p.parseRequest,
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
	switch {
	case p.is(Macro):
		return p.parseMacro()
	case p.is(Ident):
		child := Make(p.getCurrLiteral(), root)
		p.next()
		if err := p.parseMain(child); err != nil {
			return err
		}
		root.Collections = append(root.Collections, child)
	case p.is(Keyword):
		fn, ok := p.dispatch[p.getCurrLiteral()]
		if !ok {
			return p.unexpected("collection")
		}
		return fn(root)
	default:
		return p.unexpected("collection")
	}
	return nil
}

func (p *Parser) parseMain(root *Collection) error {
	return p.parseBraces("collection", func() error {
		for !p.done() && !p.is(Rbrace) {
			if err := p.parse(root); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *Parser) parseCollection(root *Collection) error {
	p.next()
	return p.parse(root)
}

func (p *Parser) parseURL(root *Collection) error {
	p.next()
	if !p.is(Ident) && !p.is(String) {
		return p.unexpected("url")
	}
	u, err := url.Parse(p.getCurrLiteral())
	if err != nil {
		return err
	}
	root.URL = *u
	p.next()
	if !p.is(EOL) {
		return p.unexpected("url")
	}
	p.next()
	return nil
}

func (p *Parser) parseRequest(root *Collection) error {
	req := Request{
		Method: p.getCurrLiteral(),
	}
	p.next()
	if !p.is(Ident) {
		return p.unexpected("request")
	}
	req.Name = p.getCurrLiteral()
	p.next()
	err := p.parseBraces("request", func() error {
		for !p.done() && !p.is(Rbrace) {
			p.next()
		}
		return nil
	})
	if err == nil {
		root.Requests = append(root.Requests, &req)
	}
	return err
}

func (p *Parser) parseVariables(root *Collection) error {
	p.next()
	return p.parseBraces("variables", func() error {
		return nil
	})
}

func (p *Parser) parseBraces(ctx string, fn func() error) error {
	if !p.is(Lbrace) {
		return p.unexpected(ctx)
	}
	p.next()
	if err := fn(); err != nil {
		return err
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
	return nil
}

func (p *Parser) parseReadFileMacro() error {
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
