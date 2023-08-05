package mule

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/midbel/mule/env"
	"github.com/midbel/mule/eval"
)

type Parser struct {
	dispatch map[string]func(*Collection) error
	macros   map[string]func() (interface{}, error)

	file string

	scan *Scanner
	curr Token
	peek Token
}

func NewParser(r io.Reader) *Parser {
	p := Parser{
		scan: Scan(r),
	}
	if n, ok := r.(interface{ Name() string }); ok {
		p.file = filepath.Dir(n.Name())
	}
	p.macros = map[string]func() (interface{}, error){
		"include": p.parseIncludeMacro,
	}
	p.dispatch = map[string]func(*Collection) error{
		"variables":  p.parseVariables,
		"collection": p.parseCollection,
		"headers":    p.parseCollectionHeaders,
		"query":      p.parseCollectionQuery,
		"tls":        p.parseCollectionTLS,
		"get":        p.parseRequest,
		"post":       p.parseRequest,
		"put":        p.parseRequest,
		"delete":     p.parseRequest,
		"patch":      p.parseRequest,
		"head":       p.parseRequest,
		"option":     p.parseRequest,
	}
	p.next()
	p.next()

	return &p
}

func (p *Parser) Parse() (*Collection, error) {
	return p.parseMain()
}

func (p *Parser) parseMacro() (interface{}, error) {
	parse, ok := p.macros[p.curr.Literal]
	if !ok {
		return nil, p.unexpected()
	}
	p.next()
	return parse()
}

func (p *Parser) parseIncludeMacro() (interface{}, error) {
	uri, err := url.Parse(p.curr.Literal)
	if err != nil {
		return nil, err
	}
	var r io.ReadCloser
	switch uri.Scheme {
	case "http", "https":
		res, err := http.DefaultClient.Get(uri.String())
		if err != nil {
			return nil, err
		}
		r = res.Body
	case "file", "":
		f, err := os.Open(filepath.Join(p.file, uri.Path))
		if err != nil {
			return nil, err
		}
		r = f
	default:
		return nil, fmt.Errorf("%s can not be included - wrong scheme given %s", uri.Path, uri.Scheme)
	}
	p.next()
	p.skip(EOL)
	defer r.Close()
	return NewParser(r).Parse()
}

func (p *Parser) parseReadFileMacro() (interface{}, error) {
	uri, err := url.Parse(p.curr.Literal)
	if err != nil {
		return nil, err
	}
	var buf []byte
	switch uri.Scheme {
	case "http", "https":
		res, err := http.DefaultClient.Get(uri.String())
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		buf, err = io.ReadAll(res.Body)
	case "file", "":
		buf, err = os.ReadFile(filepath.Join(p.file, uri.Path))
	default:
		return nil, fmt.Errorf("%s can not be included - wrong scheme given %s", uri.Path, uri.Scheme)
	}
	p.next()
	p.skip(EOL)
	return string(buf), nil
}

func (p *Parser) parseMain() (*Collection, error) {
	collect := Empty("")
	for !p.done() {
		if err := p.startParse(collect); err != nil {
			return nil, err
		}
	}
	return collect, nil
}

func (p *Parser) startParse(collect *Collection) error {
	p.skip(Comment)
	p.skip(EOL)
	if p.is(Macro) {
		dat, err := p.parseMacro()
		if err != nil {
			return err
		}
		c, ok := dat.(*Collection)
		if !ok {
			return fmt.Errorf("no collection received from macro")
		}
		collect.collections = append(collect.collections, c)
		p.skip(EOL)
		return nil
	}
	if !p.is(Keyword) {
		return p.unexpected()
	}
	defer p.skip(EOL)

	parse, ok := p.dispatch[p.curr.Literal]
	if !ok {
		return p.unexpected()
	}
	return parse(collect)
}

func (p *Parser) parseCollection(parent *Collection) error {
	p.next()
	if !p.is(Ident) {
		return p.unexpected()
	}
	curr := Enclosed(p.curr.Literal, parent)
	p.next()
	if err := p.expect(Lbrace); err != nil {
		return err
	}
	defer p.skip(EOL)
	for !p.done() && !p.is(Rbrace) {
		p.skip(EOL)
		if err := p.startParse(curr); err != nil {
			return err
		}
		p.skip(EOL)
	}
	parent.AddCollection(curr)
	return p.expect(Rbrace)
}

func (p *Parser) parseRequest(collect *Collection) error {
	p.unregisterMacroFunc("include")
	p.registerMacroFunc("readfile", p.parseReadFileMacro)
	defer func() {
		p.registerMacroFunc("include", p.parseIncludeMacro)
		p.unregisterMacroFunc("readfile")
	}()

	method := p.curr.Literal
	p.next()
	if !p.is(Ident) {
		return p.unexpected()
	}
	var (
		req   = Prepare(p.curr.Literal, method)
		track = createTracker()
	)
	req.Order = len(collect.requests)
	p.next()

	if err := p.expect(Lbrace); err != nil {
		return err
	}
	defer p.skip(EOL)
	for !p.done() && !p.is(Rbrace) {
		p.skip(EOL)
		if !p.is(Ident) && !p.is(Keyword) {
			return p.unexpected()
		}
		var (
			kw  = p.curr.Literal
			err error
		)
		if err = track.Seen(kw); err != nil {
			return nil
		}
		p.next()
		switch kw {
		case "url":
			req.location, err = p.parseWord()
		case "retry":
			req.retry, err = p.parseWord()
		case "timeout":
			req.timeout, err = p.parseWord()
		case "headers":
			req.headers, err = p.parseBag()
		case "query":
			req.query, err = p.parseBag()
		case "body":
			req.body, err = p.parseWord()
		case "cookie":
		case "username":
			req.user, err = p.parseWord()
		case "password":
			req.pass, err = p.parseWord()
		case "pre":
			req.pre, err = p.parseScript(collect)
		case "post":
			req.post, err = p.parseScript(collect)
		case "expect":
			req.expect, err = p.parseExpect(collect)
		case "depends":
			req.depends, err = p.parseDepends()
		default:
			return p.unexpected()
		}
		if err != nil {
			return err
		}
		p.skip(EOL)
	}
	if req.location == nil {
		return fmt.Errorf("%s: missing url in request definition", req.Name)
	}
	collect.AddRequest(req)
	return p.expect(Rbrace)
}

func (p *Parser) parseScript(ev env.Env[string]) (eval.Expression, error) {
	w, err := p.parseWord()
	if err != nil {
		return "", err
	}
	str, err := w.Expand(ev)
	if err == nil {
		return eval.ParseString(str)
	}
	return "", err
}

func (p *Parser) parseExpect(ev env.Env[string]) (ExpectFunc, error) {
	w, err := p.parseWord()
	if err != nil {
		return nil, err
	}
	n, err := w.ExpandInt(ev)
	if err == nil {
		return expectCode(n)
	}
	str, err := w.Expand(ev)
	if err != nil {
		return nil, err
	}
	return expectCodeRange(str)
}

func (p *Parser) parseWord() (Word, error) {
	switch {
	case p.is(Macro):
		dat, err := p.parseMacro()
		if err != nil {
			return nil, err
		}
		str, _ := dat.(string)
		return createLiteral(str), nil
	case p.is(Quote):
		return p.parseQuote()
	case p.is(Variable):
		defer p.next()
		return createVariable(p.curr.Literal), nil
	default:
		defer p.next()
		return createLiteral(p.curr.Literal), nil
	}
}

func (p *Parser) parseQuote() (Word, error) {
	if err := p.expect(Quote); err != nil {
		return nil, err
	}
	p.next()
	var ws compound
	for !p.done() && !p.is(Quote) {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}
	var w Word = ws
	if len(ws) == 1 {
		w = ws[0]
	}
	return w, p.expect(Quote)
}

func (p *Parser) parseKeyValues(set func(string, Word)) error {
	p.skip(EOL)
	if !p.is(Ident) {
		return p.unexpected()
	}
	ident := p.curr.Literal
	p.next()
	for !p.done() && !p.is(EOL) {
		word, err := p.parseWord()
		if err != nil {
			return err
		}
		set(ident, word)
	}
	return nil
}

func (p *Parser) parseDepends() ([]Word, error) {
	var list []Word
	for !p.done() && !p.is(EOL) {
		w, err := p.parseWord()
		if err != nil {
			return nil, err
		}
		list = append(list, w)
	}
	return list, nil
}

func (p *Parser) parseBag() (Bag, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)
	var (
		bag = make(Bag)
		err error
	)
	for !p.done() && !p.is(Rbrace) {
		err = p.parseKeyValues(func(key string, word Word) {
			bag.Add(key, word)
		})
		if err != nil {
			return nil, err
		}
		p.skip(EOL)
	}
	return bag, p.expect(Rbrace)
}

func (p *Parser) parseVariables(collect *Collection) error {
	p.next()
	if err := p.expect(Lbrace); err != nil {
		return err
	}
	defer p.skip(EOL)
	for !p.done() && !p.is(Rbrace) {
		p.skip(EOL)
		if !p.is(Ident) {
			return p.unexpected()
		}
		var (
			ident = p.curr.Literal
			value string
		)
		p.next()
		switch {
		case p.is(Ident) || p.is(String) || p.is(Number):
			value = p.curr.Literal
		case p.is(Variable):
			v, err := collect.Resolve(p.curr.Literal)
			if err != nil {
				return err
			}
			value = v
		default:
			return p.unexpected()
		}
		collect.Define(ident, value)
		p.next()
		p.skip(EOL)
	}
	return p.expect(Rbrace)
}

func (p *Parser) parseTLS(env env.Env[string]) (interface{}, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)

	return nil, p.expect(Rbrace)
}

func (p *Parser) parseCollectionTLS(collect *Collection) error {
	p.next()
	if _, err := p.parseTLS(collect); err != nil {
		return err
	}
	return nil
}

func (p *Parser) parseCollectionQuery(collect *Collection) error {
	p.next()
	bg, err := p.parseBag()
	if err == nil {
		collect.query = bg
	}
	return err
}

func (p *Parser) parseCollectionHeaders(collect *Collection) error {
	p.next()
	bg, err := p.parseBag()
	if err == nil {
		collect.headers = bg
	}
	return err
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

func (p *Parser) done() bool {
	return p.is(EOF)
}

func (p *Parser) next() {
	p.curr = p.peek
	p.peek = p.scan.Scan()
}

func (p *Parser) registerMacroFunc(name string, fn func() (interface{}, error)) {
	p.macros[name] = fn
}

func (p *Parser) unregisterMacroFunc(name string) {
	delete(p.macros, name)
}

type tracker[T comparable] struct {
	seen  map[T]struct{}
	empty struct{}
}

func createTracker() tracker[string] {
	return tracker[string]{
		seen:  make(map[string]struct{}),
		empty: struct{}{},
	}
}

func (t tracker[T]) Seen(v T) error {
	_, ok := t.seen[v]
	if !ok {
		return nil
	}
	t.seen[v] = t.empty
	return fmt.Errorf("already seen")
}
