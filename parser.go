package mule

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/eval"
	"github.com/midbel/enjoy/parser"
	"github.com/midbel/enjoy/value"
)

type tlsConfig struct {
	certFile string
	certKey  string

	Config tls.Config
}

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
		"url":        p.parseCollectionURL,
		"username":   p.parseCollectionUser,
		"password":   p.parseCollectionPass,
		"variables":  p.parseVariables,
		"collection": p.parseCollection,
		"headers":    p.parseCollectionHeaders,
		"query":      p.parseCollectionQuery,
		"tls":        p.parseCollectionTLS,
		"beforeEach": p.parseCollectionScript,
		"afterEach":  p.parseCollectionScript,
		"before":     p.parseCollectionScript,
		"after":      p.parseCollectionScript,
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
	p.skip(EOL)
	for !p.done() && !p.is(Rbrace) {
		if err := p.startParse(curr); err != nil {
			return err
		}
		p.skip(EOL)
	}
	parent.AddCollection(curr)
	return p.expect(Rbrace)
}

func (p *Parser) parseCollectionUser(collect *Collection) error {
	p.next()

	var err error
	collect.user, err = p.parseWord()
	return err
}

func (p *Parser) parseCollectionPass(collect *Collection) error {
	p.next()

	var err error
	collect.pass, err = p.parseWord()
	return err
}

func (p *Parser) parseCollectionURL(collect *Collection) error {
	p.next()
	var err error
	collect.base, err = p.parseWord()
	return err
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
			req.body, err = p.parseBody()
		case "cookie":
		case "username":
			req.user, err = p.parseWord()
		case "password":
			req.pass, err = p.parseWord()
		case "before":
			req.before, err = p.parseScript(collect)
		case "after":
			req.after, err = p.parseScript(collect)
		case "expect":
			req.expect, err = p.parseExpect(collect)
		case "depends":
			req.depends, err = p.parseDepends()
		case "tls":
			req.config, err = p.parseTLS(collect)
		default:
			return p.unexpected()
		}
		if err != nil {
			return err
		}
		p.skip(EOL)
	}
	collect.AddRequest(req)
	return p.expect(Rbrace)
}

func (p *Parser) parseBody() (Body, error) {
	defer p.next()
	return PrepareBody(p.curr.Literal)
}

func (p *Parser) parseScript(ev env.Environ[string]) (value.Evaluable, error) {
	w, err := p.parseWord()
	if err != nil {
		return nil, err
	}
	str, err := w.Expand(ev)
	if err == nil {
		n, err := parser.ParseString(str)
		if err != nil {
			return nil, fmt.Errorf("enjoy: %s", err)
		}
		return eval.EvaluableNode(n), nil
	}
	return nil, err
}

func (p *Parser) parseExpect(ev env.Environ[string]) (ExpectFunc, error) {
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

func (p *Parser) parseString(ev env.Environ[string]) (string, error) {
	w, err := p.parseWord()
	if err != nil {
		return "", err
	}
	return w.Expand(ev)
}

func (p *Parser) parseBool(ev env.Environ[string]) (bool, error) {
	w, err := p.parseWord()
	if err != nil {
		return false, err
	}
	return w.ExpandBool(ev)
}

func (p *Parser) parseCertPool(ev env.Environ[string]) (*x509.CertPool, error) {
	if p.is(EOL) {
		return x509.SystemCertPool()
	}
	w, err := p.parseWord()
	if err != nil {
		return nil, err
	}
	str, err := w.Expand(ev)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if i, err := os.Stat(str); err == nil {
		if i.Mode().IsRegular() {
			cert, err := os.ReadFile(str)
			if err != nil {
				return nil, err
			}
			pool.AppendCertsFromPEM(cert)
			return pool, nil
		} else if i.IsDir() {
			files, err := os.ReadDir(str)
			if err != nil {
				return nil, err
			}
			for _, f := range files {
				cert, err := os.ReadFile(filepath.Join(str, f.Name()))
				if err != nil {
					return nil, err
				}
				pool.AppendCertsFromPEM(cert)
			}
		} else {
			return nil, fmt.Errorf("certificates can not be loaded from %s", i.Name())
		}
	}
	return pool, nil
}

func (p *Parser) parseVersionTLS(ev env.Environ[string]) (uint16, error) {
	w, err := p.parseWord()
	if err != nil {
		return 0, err
	}
	str, err := w.Expand(ev)
	if err != nil {
		return 0, err
	}
	var version uint16
	switch strings.ToLower(str) {
	default:
		return 0, fmt.Errorf("unsupported TLS version")
	case "tls-1.0":
		version = tls.VersionTLS10
	case "tls-1.1":
		version = tls.VersionTLS11
	case "tls-1.2":
		version = tls.VersionTLS12
	case "tls-1.3":
		version = tls.VersionTLS13
	}
	return version, nil
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
	var frozen bool
	if p.is(Frozen) {
		p.next()
		frozen = true
	}
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)
	var (
		bag = Standard()
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
	if frozen {
		bag = Freeze(bag)
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
		collect.Define(ident, value, false)
		p.next()
		p.skip(EOL)
	}
	return p.expect(Rbrace)
}

func (p *Parser) parseTLS(env env.Environ[string]) (*tls.Config, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)
	var (
		cfg   tlsConfig
		track = createTracker()
	)
	for !p.done() && !p.is(Rbrace) {
		p.skip(EOL)
		if !p.is(Ident) && !p.is(Keyword) {
			return nil, p.unexpected()
		}
		var (
			kw  = p.curr.Literal
			err error
		)
		if err = track.Seen(kw); err != nil {
			return nil, err
		}
		p.next()
		switch kw {
		case "certFile":
			cfg.certFile, err = p.parseString(env)
		case "certKey":
			cfg.certKey, err = p.parseString(env)
		case "certCA":
			cfg.Config.RootCAs, err = p.parseCertPool(env)
		case "serverName":
			cfg.Config.ServerName, err = p.parseString(env)
		case "insecure":
			cfg.Config.InsecureSkipVerify, err = p.parseBool(env)
		case "minVersion":
			cfg.Config.MinVersion, err = p.parseVersionTLS(env)
		case "maxVersion":
			cfg.Config.MaxVersion, err = p.parseVersionTLS(env)
		default:
			return nil, p.unexpected()
		}
		if err != nil {
			return nil, err
		}
		p.skip(EOL)
	}
	if cfg.certFile != "" && cfg.certKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.certFile, cfg.certKey)
		if err != nil {
			return nil, err
		}
		cfg.Config.Certificates = append(cfg.Config.Certificates, cert)
	}
	return &cfg.Config, p.expect(Rbrace)
}

func (p *Parser) parseCollectionTLS(collect *Collection) error {
	p.next()
	cfg, err := p.parseTLS(collect)
	if err == nil {
		collect.config = cfg
	}
	return err
}

func (p *Parser) parseCollectionScript(collect *Collection) error {
	ident := p.curr.Literal
	p.next()
	ev, err := p.parseScript(collect)
	if err != nil {
		return err
	}
	switch ident {
	case "beforeEach":
		collect.beforeEach = append(collect.beforeEach, ev)
	case "afterEach":
		collect.afterEach = append(collect.afterEach, ev)
	default:
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
	pos := p.curr.Position
	return fmt.Errorf("%d,%d: unexpected token %s", pos.Line, pos.Column, p.curr)
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
