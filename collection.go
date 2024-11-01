package mule

import (
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/midbel/mule/codecs/json"
	"github.com/midbel/mule/codecs/xml"
	"github.com/midbel/mule/environ"
	"github.com/midbel/mule/play"
)

const MaxFlowDepth = 100

var ErrNotFound = errors.New("not found")

type ErrorExit struct {
	Code int
}

func exit(code int) error {
	return ErrorExit{
		Code: code,
	}
}

func (e ErrorExit) Error() string {
	return "exit"
}

type Common struct {
	Name string
	Desc string

	URL      Value
	Auth     Authorization
	Retry    Value
	Timeout  Value
	Redirect Value
	Body     Body

	Headers Set
	Query   Set
	Tls     *tls.Config
}

type Flow struct {
	Common
	Usage string
	environ.Environment[Value]

	Before     string
	BeforeEach string
	After      string
	AfterEach  string

	Steps []*Step
	depth int
}

func (f *Flow) Execute(ctx *Collection, args []string, stdout, stderr io.Writer) error {
	f.reset()
	if err := f.parseArgs(args); err != nil {
		return err
	}
	var (
		obj  = getMuleObject(ctx)
		root = play.Enclosed(play.Default())
		env  = play.Enclosed(root)
	)
	root.Define(muleVarName, obj)

	if err := runScript(env, f.Before); err != nil {
		return err
	}
	step := f.Steps[0]
	step.ctx = ctx
	step.env = env
	if err := f.execute(obj, step, stdout, stderr); err != nil {
		return err
	}
	if err := runScript(env, f.After); err != nil {
		return err
	}
	return nil
}

func (f *Flow) execute(obj *muleObject, step *Step, stdout, stderr io.Writer) error {
	f.enter()
	defer f.leave()

	if f.depth >= MaxFlowDepth {
		return fmt.Errorf("max flow depth reached")
	}
	obj.reset()

	if err := runScript(step.env, f.BeforeEach); err != nil {
		return err
	}

	res, err := step.Execute(obj)
	if err != nil {
		return err
	}

	next, err := step.guessNext(res.StatusCode, f.Steps)
	if err != nil || next == nil {
		return err
	}
	if err := runScript(step.env, f.AfterEach); err != nil {
		return err
	}
	return f.execute(obj, next, stdout, stderr)
}

func (f *Flow) parseArgs(args []string) error {
	set := flag.NewFlagSet(f.Name, flag.ExitOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	return nil
}

func (f *Flow) enter() {
	f.depth++
}

func (f *Flow) leave() {
	f.depth--
}

func (f *Flow) reset() {
	f.depth = 0
}

type Step struct {
	Request string
	Before  string
	After   string
	Next    []StepBody

	req *Request
	ctx *Collection
	env environ.Environment[play.Value]
}

func (s *Step) Execute(obj *muleObject) (*http.Response, error) {
	req, err := s.req.build(s.ctx)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	obj.req = getMuleRequest(req, s.req.Name, body)
	if err := s.runBefore(); err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if err := s.req.Expect(res); err != nil {
		return nil, err
	}
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	obj.res = getMuleResponse(res, buf)
	if err := s.runAfter(); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Step) runBefore() error {
	script := s.req.Before
	if s.Before != "" {
		script = s.Before
	}
	return runScript(s.env, script)
}

func (s *Step) runAfter() error {
	script := s.req.After
	if s.After != "" {
		script = s.After
	}
	return runScript(s.env, script)
}

func (s *Step) guessNext(code int, others []*Step) (*Step, error) {
	var next *Step
	for _, body := range s.Next {
		if ok := slices.Contains(body.Codes, code); !ok {
			continue
		}
		ix := slices.IndexFunc(others, func(s *Step) bool {
			return s.Request == body.Target
		})
		if ix < 0 {
			continue
		}
		for _, c := range body.Commands {
			switch c := c.(type) {
			case cmdSet:
			case cmdUnset:
			case cmdExit:
				v, err := c.Code.Expand(s.ctx)
				if err != nil {
					return nil, err
				}
				x, _ := strconv.Atoi(v)
				return nil, exit(x)
			default:
				return nil, nil
			}
		}
		next = others[ix]
		break
	}
	if next != nil {
		next.ctx = s.ctx
		next.env = s.env
	}
	return next, nil
}

type StepBody struct {
	Target   string
	Codes    []int
	Commands []any
}

type cmdGoto struct {
	Target string
}

type cmdExit struct {
	Code Value
}

type cmdSet struct {
	Source Value
	Target Value
}

type cmdUnset struct {
	Ident Value
}

type Exchange struct {
	Total time.Duration
	*http.Request
	*http.Response
}

type Collection struct {
	Common
	environ.Environment[Value]

	Before string
	After  string

	Requests    []*Request
	Collections []*Collection
	Flows       []*Flow
}

func Open(file string) (*Collection, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	p, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}

func Root() *Collection {
	return Make("", nil)
}

func Make(name string, parent environ.Environment[Value]) *Collection {
	info := Common{
		Name: name,
	}
	return &Collection{
		Common:      info,
		Environment: environ.Enclosed[Value](nil),
	}
}

func (c *Collection) Resolve(ident string) (Value, error) {
	switch {
	case c.URL != nil && ident == "url":
		return c.URL, nil
	case (ident == "username" || ident == "password") && c.Auth != nil:
		b, ok := c.Auth.(basic)
		if !ok {
			break
		}
		if ident == "username" {
			return b.User, nil
		}
		if ident == "password" {
			return b.Pass, nil
		}
	case ident == "token" && c.Auth != nil:
		b, ok := c.Auth.(bearer)
		if !ok {
			break
		}
		return b.Token, nil
	default:
	}
	return c.Environment.Resolve(ident)
}

func (c *Collection) Find(name string) (*Collection, error) {
	return nil, nil
}

func (c *Collection) Get(name string) (*http.Request, error) {
	name, rest, ok := strings.Cut(name, ".")
	if !ok {
		req, err := c.findRequestByName(name)
		if err != nil {
			return nil, err
		}
		return req.build(c)
	}
	other, err := c.findCollectionByName(name)
	if err != nil {
		return nil, err
	}
	return other.Get(rest)
}

func (c *Collection) Run(name string, args []string, stdout, stderr io.Writer) error {
	name, rest, ok := strings.Cut(name, ".")
	if !ok {
		if ex, err := c.findFlowByName(name); err == nil {
			return c.runFlow(ex, args, stdout, stderr)
		}
		if ex, err := c.findRequestByName(name); err == nil {
			return c.runRequest(ex, args, stdout, stderr)
		}
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	other, err := c.findCollectionByName(name)
	if err != nil {
		return err
	}
	return other.Run(rest, args, stdout, stderr)
}

func (c *Collection) runFlow(flow *Flow, args []string, stdout, stderr io.Writer) error {
	for _, s := range flow.Steps {
		req, err := c.findRequestByName(s.Request)
		if err != nil {
			return err
		}
		s.req = req
	}
	return flow.Execute(c, args, stdout, stderr)
}

func (c *Collection) runRequest(req *Request, args []string, stdout, stderr io.Writer) error {
	_, err := req.Execute(c, args, stdout, stderr)
	return err
}

func (c *Collection) findCollectionByName(name string) (*Collection, error) {
	name, rest, ok := strings.Cut(name, ".")
	if ok {
		other, err := c.findCollectionByName(name)
		if err != nil {
			return nil, err
		}
		return other.findCollectionByName(rest)
	}
	x := slices.IndexFunc(c.Collections, func(curr *Collection) bool {
		return curr.Name == name
	})
	if x < 0 {
		return nil, fmt.Errorf("%w: collection %s", ErrNotFound, name)
	}
	curr := c.Collections[x]
	curr.URL = mergeURL(c.URL, curr.URL, c)
	curr.Query = curr.Query.Merge(c.Query)
	curr.Headers = curr.Headers.Merge(c.Headers)

	if curr.Auth == nil && c.Auth != nil {
		curr.Auth = c.Auth
	}
	return curr, nil
}

func (c *Collection) findFlowByName(name string) (*Flow, error) {
	x := slices.IndexFunc(c.Flows, func(curr *Flow) bool {
		return curr.Name == name
	})
	if x < 0 {
		return nil, fmt.Errorf("%w: flow %s", ErrNotFound, name)
	}
	return c.Flows[x], nil
}

func (c *Collection) findRequestByName(name string) (*Request, error) {
	name, rest, ok := strings.Cut(name, ".")
	if ok {
		other, err := c.findCollectionByName(name)
		if err != nil {
			return nil, err
		}
		return other.findRequestByName(rest)
	}
	x := slices.IndexFunc(c.Requests, func(curr *Request) bool {
		return curr.Name == name
	})
	if x < 0 {
		return nil, fmt.Errorf("%w: request %s", ErrNotFound, name)
	}
	req := c.Requests[x]
	req.URL = mergeURL(c.URL, req.URL, c)
	req.Headers = req.Headers.Merge(c.Headers)
	req.Query = req.Query.Merge(c.Query)
	if req.Auth == nil && c.Auth != nil {
		req.Auth = c.Auth
	}
	return req, nil
}

type ExpectFunc func(*http.Response) error

func checkResponseCode(codes []int) ExpectFunc {
	return func(res *http.Response) error {
		ok := slices.Contains(codes, res.StatusCode)
		if ok {
			return nil
		}
		return fmt.Errorf("request ends with unexpected code %d", res.StatusCode)
	}
}

func expectRequestNoop(_ *http.Response) error {
	return nil
}

func expectRequestSucceed(res *http.Response) error {
	if err := expectRequestFail(res); err == nil {
		return fmt.Errorf("request fail")
	}
	return nil
}

func expectRequestFail(res *http.Response) error {
	if res.StatusCode >= http.StatusBadRequest {
		return nil
	}
	return fmt.Errorf("request succeed")
}

type Request struct {
	Common
	Abstract   bool
	Usage      string
	Compressed Value
	Method     string
	Depends    []Value
	Before     string
	After      string

	Expect ExpectFunc
}

func (r *Request) Merge(other *Request) error {
	return nil
}

func (r *Request) Execute(ctx *Collection, args []string, stdout, stderr io.Writer) (*Exchange, error) {
	if err := r.parseArgs(args); err != nil {
		return nil, err
	}
	var exchange Exchange

	req, err := r.build(ctx)
	if err != nil {
		return nil, err
	}
	exchange.Request = req

	var (
		root = play.Enclosed(play.Default())
		tmp  = play.Enclosed(root)
		obj  = getMuleObject(ctx)
	)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	obj.req = getMuleRequest(req, r.Name, body)
	root.Define(muleVarName, obj)

	if err := runScript(tmp, r.Before); err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	exchange.Response = res
	defer res.Body.Close()
	if err := r.Expect(res); err != nil {
		return nil, err
	}
	buf, _ := io.ReadAll(res.Body)

	obj.res = getMuleResponse(res, buf)

	if err := runScript(tmp, r.After); err != nil {
		return nil, err
	}
	res.Body = io.NopCloser(bytes.NewReader(buf))
	return &exchange, nil
}

func (r *Request) parseArgs(args []string) error {
	set := flag.NewFlagSet(r.Name, flag.ExitOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	return nil
}

func (r *Request) build(env environ.Environment[Value]) (*http.Request, error) {
	target, err := r.target(env)
	if err != nil {
		return nil, err
	}

	headers, err := r.Headers.Headers(env)
	if err != nil {
		return nil, err
	}
	if r.Auth != nil {
		auth, err := r.Auth.Expand(env)
		if err != nil {
			return nil, err
		}
		auth = fmt.Sprintf("%s %s", r.Auth.Method(), auth)
		headers.Set("Authorization", auth)
	}

	var body io.Reader
	if r.Body != nil {
		bs, err := r.Body.Expand(env)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(bs)
		headers.Set("content-type", r.Body.ContentType())
		headers.Set("content-length", strconv.Itoa(len(bs)))
	}

	req, err := http.NewRequest(r.Method, target, body)
	if err == nil {
		req.Header = headers
	}
	return req, err
}

func (r *Request) target(env environ.Environment[Value]) (string, error) {
	str, err := r.URL.Expand(env)
	if err != nil {
		return "", err
	}
	target, err := url.Parse(str)
	if err != nil {
		return "", err
	}

	qs, err := r.Query.Query(env)
	if err != nil {
		return "", err
	}
	if target.RawQuery != "" {
		qs2 := target.Query()
		for k := range qs {
			qs2[k] = append(qs2[k], qs[k]...)
		}
		qs = qs2
	}
	target.RawQuery = qs.Encode()
	return target.String(), nil
}

func runScript(env environ.Environment[play.Value], script string) error {
	_, err := play.EvalWithEnv(strings.NewReader(script), env)
	return err
}

type Body interface {
	Value
	Compressed() bool
	ContentType() string
}

type xmlBody struct {
	root string
	elem Value
}

func xmlify(set Set) (Body, error) {
	if len(set) != 1 {
		return nil, fmt.Errorf("only one root element allowed")
	}
	var root string
	for k := range set {
		root = k
		break
	}
	if len(set[root]) != 1 {
		return nil, fmt.Errorf("invalid root element")
	}
	return xmlBody{
		root: root,
		elem: set[root][0],
	}, nil
}

func (b xmlBody) Expand(env environ.Environment[Value]) (string, error) {
	var (
		root = xml.NewElement(b.root, "")
		doc  = xml.NewDocument(root)
	)

	el, ok := b.elem.(Set)
	if !ok {
		return "", fmt.Errorf("invalid element type")
	}
	for k, vs := range el {
		for _, v := range vs {
			el, err := b.tree(env, k, v)
			if err != nil {
				return "", err
			}
			doc.Append(el)
		}
	}
	return doc.WriteString()
}

func (b xmlBody) tree(env environ.Environment[Value], name string, val Value) (xml.Node, error) {
	set, ok := val.(Set)
	if !ok {
		str, err := val.Expand(env)
		if err != nil {
			return nil, err
		}
		return xml.NewText(str), nil
	}
	el := xml.NewElement(name, "")
	for k, vs := range set {
		if strings.HasPrefix(k, "_") {
			if len(vs) != 1 {
				return nil, fmt.Errorf("multi value attribute")
			}
			str, err := vs[0].Expand(env)
			if err != nil {
				return nil, err
			}
			a := xml.NewAttribute(str, strings.TrimPrefix(k, "_"), "")
			el.Attrs = append(el.Attrs, a)
			continue
		}
		for i := range vs {
			sub := xml.NewElement(k, "")
			child, err := b.tree(env, k, vs[i])
			if err != nil {
				return nil, err
			}
			sub.Append(child)
			el.Append(sub)
		}
	}
	return el, nil
}

func (b xmlBody) clone() Value {
	return b
}

func (b xmlBody) Compressed() bool {
	return false
}

func (b xmlBody) ContentType() string {
	return "text/xml"
}

type jsonBody struct {
	Set
}

func jsonify(set Set) Body {
	return jsonBody{
		Set: set,
	}
}

func (b jsonBody) Expand(env environ.Environment[Value]) (string, error) {
	doc, err := b.expand(env)
	if err != nil {
		return "", err
	}
	var (
		buf bytes.Buffer
		ws  = json.NewWriter(&buf)
	)
	if err = ws.Write(doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (b jsonBody) expand(env environ.Environment[Value]) (map[string]any, error) {
	var expand func(Set) (map[string]any, error)
	expand = func(set Set) (map[string]any, error) {
		vs := make(map[string]any)
		for k := range set {
			var (
				arr []any
				sub any
				err error
			)
			for _, v := range set[k] {
				if x, ok := v.(Set); ok {
					sub, err = expand(x)
				} else {
					sub, err = v.Expand(env)
				}
				if err != nil {
					return nil, err
				}
				arr = append(arr, sub)
			}
			var dat interface{} = arr
			if len(arr) == 1 {
				dat = arr[0]
			}
			vs[k] = dat
		}
		return vs, nil
	}
	return expand(b.Set)
}

func (b jsonBody) clone() Value {
	return b
}

func (b jsonBody) Compressed() bool {
	return false
}

func (b jsonBody) ContentType() string {
	return "application/json"
}

type octetstreamBody struct {
	stream string
}

func octetstream() Body {
	return octetstreamBody{}
}

func (b octetstreamBody) Expand(env environ.Environment[Value]) (string, error) {
	return "", nil
}

func (b octetstreamBody) clone() Value {
	return b
}

func (b octetstreamBody) Compressed() bool {
	return false
}

func (b octetstreamBody) ContentType() string {
	return "application/octet-stream"
}

type textBody struct {
	stream string
}

func textify() Body {
	return textBody{}
}

func (b textBody) Expand(env environ.Environment[Value]) (string, error) {
	return "", nil
}

func (b textBody) clone() Value {
	return b
}

func (b textBody) Compressed() bool {
	return false
}

func (b textBody) ContentType() string {
	return "text/plain"
}

type urlencodedBody struct {
	Set
}

func urlEncoded(set Set) Body {
	return urlencodedBody{
		Set: set,
	}
}

func (b urlencodedBody) Expand(env environ.Environment[Value]) (string, error) {
	qs, err := b.Set.Query(env)
	if err != nil {
		return "", err
	}
	return qs.Encode(), nil
}

func (b urlencodedBody) clone() Value {
	return b
}

func (b urlencodedBody) Compressed() bool {
	return false
}

func (b urlencodedBody) ContentType() string {
	return "application/x-www-form-urlencoded"
}
