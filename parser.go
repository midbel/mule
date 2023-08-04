package mule

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/midbel/mule/env"
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
		case "default":
			req.Default, err = p.parseBool(collect)
			continue
		case "url":
			req.location, err = p.parseURL(collect)
		case "retry":
			req.retry, err = p.parseNumber(collect)
		case "timeout":
			req.timeout, err = p.parseNumber(collect)
		case "headers":
			req.headers, err = p.parseHeaders(collect)
		case "query":
			req.query, err = p.parseQuery(collect)
		case "body":
			req.body, err = p.parseString(collect)
		case "cookie":
			cookie, err1 := p.parseCookie(collect)
			if err1 != nil {
				err = err1
				break
			}
			req.cookies = append(req.cookies, cookie)
		case "username":
			req.user, err = p.parseString(collect)
		case "password":
			req.pass, err = p.parseString(collect)
		case "tls":
			_, err = p.parseTLS(collect)
		case "before":
			err = p.parseScript(collect)
		case "after":
			err = p.parseScript(collect)
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

func (p *Parser) parseQuote(env env.Env) (string, error) {
	if err := p.expect(Quote); err != nil {
		return "", err
	}
	var parts []string
	for !p.done() && !p.is(Quote) {
		str, err := p.parseString(env)
		if err != nil {
			return "", err
		}
		parts = append(parts, str)
	}
	return strings.Join(parts, ""), p.expect(Quote)
}

func (p *Parser) parseURL(env env.Env) (*url.URL, error) {
	defer p.next()

	var str string
	switch {
	case p.is(Quote):
		v, err := p.parseQuote(env)
		if err != nil {
			return nil, err
		}
		str = v
	case p.is(Variable):
		v, err := env.Resolve(p.curr.Literal)
		if err != nil {
			return nil, err
		}
		str = v
	case p.is(Macro):
		dat, err := p.parseMacro()
		if err != nil {
			return nil, err
		}
		str, _ = dat.(string)
	case p.is(String) || p.is(Ident):
		str = p.curr.Literal
	default:
		return nil, p.unexpected()
	}
	return url.Parse(str)
}

func (p *Parser) parseString(env env.Env) (string, error) {
	defer p.next()

	switch {
	case p.is(Quote):
		return p.parseQuote(env)
	case p.is(Number) || p.is(String) || p.is(Ident):
		return p.curr.Literal, nil
	case p.is(Variable):
		return env.Resolve(p.curr.Literal)
	case p.is(Macro):
		dat, err := p.parseMacro()
		if err != nil {
			return "", err
		}
		str, _ := dat.(string)
		return str, nil
	default:
		return "", p.unexpected()
	}
}

func (p *Parser) parseBool(env env.Env) (bool, error) {
	if p.is(EOL) {
		return true, nil
	}
	defer p.next()
	switch {
	case p.is(Number):
		v, err := strconv.Atoi(p.curr.Literal)
		if err != nil {
			return false, err
		}
		return v != 0, nil
	case p.is(String) || p.is(Ident):
		return strconv.ParseBool(p.curr.Literal)
	case p.is(Variable):
		v, err := env.Resolve(p.curr.Literal)
		if err != nil {
			return false, err
		}
		return strconv.ParseBool(v)
	default:
		return false, p.unexpected()
	}
}

func (p *Parser) parseNumber(env env.Env) (int, error) {
	defer p.next()

	var str string
	switch {
	case p.is(Number) || p.is(String) || p.is(Ident):
		str = p.curr.Literal
	case p.is(Variable):
		v, err := env.Resolve(p.curr.Literal)
		if err != nil {
			return 0, err
		}
		str = v
	case p.is(Macro):
		dat, err := p.parseMacro()
		if err != nil {
			return 0, err
		}
		str, _ = dat.(string)
	default:
		return 0, p.unexpected()
	}
	return strconv.Atoi(str)
}

func (p *Parser) parseKeyValues(env env.Env, set func(string, string)) error {
	p.skip(EOL)
	if !p.is(Ident) {
		return p.unexpected()
	}
	ident := p.curr.Literal
	p.next()
	for !p.done() && !p.is(EOL) {
		switch {
		case p.is(Ident) || p.is(String) || p.is(Number):
			set(ident, p.curr.Literal)
		case p.is(Quote):
			value, err := p.parseQuote(env)
			if err != nil {
				return err
			}
			set(ident, value)
			continue
		case p.is(Variable):
			value, err := env.Resolve(p.curr.Literal)
			if err != nil {
				return err
			}
			set(ident, value)
		case p.is(Macro):
			dat, err := p.parseMacro()
			if err != nil {
				return err
			}
			value, _ := dat.(string)
			set(ident, value)
		default:
			return p.unexpected()
		}
		p.next()
	}
	return nil
}

func (p *Parser) parseCookie(env env.Env) (http.Cookie, error) {
	var (
		cookie http.Cookie
		err    error
		track  = createTracker()
	)
	if !p.is(Ident) {
		return cookie, p.unexpected()
	}
	cookie.Name = p.curr.Literal
	p.next()
	if err := p.expect(Lbrace); err != nil {
		return cookie, err
	}
	for !p.done() && !p.is(Rbrace) {
		p.skip(EOL)
		if !p.is(Ident) && !p.is(Keyword) {
			return cookie, p.unexpected()
		}
		kw := p.curr.Literal
		if err := track.Seen(kw); err != nil {
			return cookie, err
		}
		p.next()
		switch kw {
		case "value":
			cookie.Value, err = p.parseString(env)
		case "domain":
			cookie.Domain, err = p.parseString(env)
		case "secure":
			cookie.Secure, err = p.parseBool(env)
		case "http-only":
			cookie.HttpOnly, err = p.parseBool(env)
		case "max-age":
			cookie.MaxAge, err = p.parseNumber(env)
		case "same-site":
		default:
			return cookie, p.unexpected()
		}
		if err != nil {
			return cookie, err
		}
		p.skip(EOL)
	}
	return cookie, p.expect(Rbrace)
}

func (p *Parser) parseExpect(env env.Env) (ExpectFunc, error) {
	var (
		fn  ExpectFunc
		err error
	)
	switch {
	case p.is(Number):
		code, _ := strconv.Atoi(p.curr.Literal)
		fn, err = expectCode(code)
	case p.is(Ident):
		fn, err = expectCodeRange(p.curr.Literal)
	default:
		return nil, p.unexpected()
	}
	p.next()
	return fn, err
}

func (p *Parser) parseDepends(env env.Env) ([]string, error) {
	var list []string
	for !p.done() && !p.is(EOL) {
		switch {
		case p.is(Ident):
			list = append(list, p.curr.Literal)
		case p.is(Variable):
			v, err := env.Resolve(p.curr.Literal)
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		default:
			return nil, p.unexpected()
		}
		p.next()
	}
	return list, nil
}

func (p *Parser) parseScript(env env.Env) error {
	var script string
	switch {
	case p.is(Macro):
		dat, err := p.parseMacro()
		if err != nil {
			return err
		}
		s, ok := dat.(string)
		if !ok {
			return fmt.Errorf("no string received from macro")
		}
		script = s
	case p.is(String):
		script = p.curr.Literal
	default:
		return fmt.Errorf("script should be given in string/heredoc or via a macro")
	}
	_ = script
	return nil
}

func (p *Parser) parseQuery(env env.Env) (url.Values, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)
	var (
		query = make(url.Values)
		err   error
	)
	for !p.done() && !p.is(Rbrace) {
		err = p.parseKeyValues(env, func(key, value string) {
			query.Add(key, value)
		})
		if err != nil {
			return nil, err
		}
		p.skip(EOL)
	}
	return query, p.expect(Rbrace)
}

func (p *Parser) parseHeaders(env env.Env) (http.Header, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)
	var (
		hdr = make(http.Header)
		err error
	)
	for !p.done() && !p.is(Rbrace) {
		err = p.parseKeyValues(env, func(key, value string) {
			hdr.Add(key, value)
		})
		if err != nil {
			return nil, err
		}
		p.skip(EOL)
	}
	return hdr, p.expect(Rbrace)
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

func (p *Parser) parseTLS(env env.Env) (interface{}, error) {
	if err := p.expect(Lbrace); err != nil {
		return nil, err
	}
	defer p.skip(EOL)
	var err error
	for !p.done() && !p.is(Rbrace) {
		err = p.parseKeyValues(env, func(key, value string) {

		})
		if err != nil {
			return nil, err
		}
		p.skip(EOL)
	}
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
	query, err := p.parseQuery(collect)
	if err == nil {
		collect.query = query
	}
	return err
}

func (p *Parser) parseCollectionHeaders(collect *Collection) error {
	p.next()
	hdr, err := p.parseHeaders(collect)
	if err == nil {
		collect.headers = hdr
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
