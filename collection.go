package mule

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
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

	"github.com/midbel/mule/environ"
	"github.com/midbel/mule/jwt"
	"github.com/midbel/mule/play"
)

var ErrNotFound = errors.New("not found")

type ErrorExit struct {
	Code int
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

	Steps    []*Step
	Requests []*Request
}

func (f *Flow) Execute(ctx *Collection, args []string, stdout, stderr io.Writer) error {
	if err := f.parseArgs(args); err != nil {
		return err
	}
	obj := muleObject{
		when: time.Now(),
		ctx:  getMuleCollection(ctx),
		vars: getMuleVars(),
	}

	root := play.Enclosed(play.Default())
	root.Define(muleVarName, &obj)

	tmp := play.Enclosed(root)
	if err := runScript(tmp, f.Before); err != nil {
		return err
	}
	if err := f.execute(ctx, f.Steps[0], f.Requests[0], stdout, stderr); err != nil {
		return err
	}
	if err := runScript(tmp, f.After); err != nil {
		return err
	}
	return nil
}

func (f *Flow) execute(ctx *Collection, step *Step, req *Request, stdout, stderr io.Writer) error {
	if step.Before != "" {
		req.Before = step.Before
	}
	if step.After != "" {
		req.After = step.After
	}
	tmp := play.Enclosed(play.Default())
	if err := runScript(tmp, f.BeforeEach); err != nil {
		return err
	}
	res, err := req.Execute(ctx, nil, stdout, stderr)
	if err != nil {
		return err
	}
	if err := runScript(tmp, f.AfterEach); err != nil {
		return err
	}

	var ix int
	for _, body := range step.Next {
		if ok := slices.Contains(body.Codes, res.StatusCode); !ok {
			continue
		}
		ix = slices.IndexFunc(f.Steps, func(s *Step) bool {
			return s.Request == body.Target
		})
		for _, c := range body.Commands {
			switch c.(type) {
			case set:
			case unset:
			case exit:
			default:
				return nil
			}
		}
		if ix >= 0 {
			break
		}
	}
	if ix < 0 {
		return nil
	}
	return f.execute(ctx, f.Steps[ix], f.Requests[ix], stdout, stderr)
}

func (f *Flow) parseArgs(args []string) error {
	set := flag.NewFlagSet(f.Name, flag.ExitOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	return nil
}

type Step struct {
	Request string
	Before  string
	After   string
	Next    []StepBody
}

type StepBody struct {
	Codes    []int
	Target   string
	Commands []any
}

type exit struct {
	Code Value
}

type set struct {
	Source Value
	Target Value
}

type unset struct {
	Ident Value
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
		flow.Requests = append(flow.Requests, req)
	}
	return flow.Execute(c, args, stdout, stderr)
}

func (c *Collection) runRequest(req *Request, args []string, stdout, stderr io.Writer) error {
	req.URL = mergeURL(c.URL, req.URL, c)
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
	curr.Query = curr.Query.Merge(c.Query)
	curr.Headers = curr.Headers.Merge(c.Headers)
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
	req.Headers = req.Headers.Merge(c.Headers)
	req.Query = req.Query.Merge(c.Query)
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

func (r *Request) Execute(ctx *Collection, args []string, stdout, stderr io.Writer) (*http.Response, error) {
	if err := r.parseArgs(args); err != nil {
		return nil, err
	}
	req, err := r.build(ctx)
	if err != nil {
		return nil, err
	}

	var (
		root = play.Enclosed(play.Default())
		obj  = muleObject{
			when: time.Now(),
			req:  getMuleRequest(req, nil),
			ctx:  getMuleCollection(ctx),
			vars: getMuleVars(),
		}
		tmp = play.Enclosed(root)
	)
	root.Define(muleVarName, &obj)

	if err := runScript(tmp, r.Before); err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if err := r.Expect(res); err != nil {
		return nil, err
	}
	buf, _ := io.ReadAll(res.Body)

	obj.res = getMuleResponse(res, buf)
	root.Define(muleVarName, &obj)

	if err := runScript(tmp, r.After); err != nil {
		return nil, err
	}
	res.Body = io.NopCloser(bytes.NewReader(buf))
	return res, nil
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

func mergeURL(left, right Value, env environ.Environment[Value]) Value {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}

	if str, err := right.Expand(env); err == nil {
		u, err := url.Parse(str)
		if err == nil && u.IsAbs() {
			return right
		}
	}

	var cs compound
	if c, ok := left.(compound); ok {
		cs = append(cs, c...)
	} else {
		cs = append(cs, left)
	}
	if c, ok := right.(compound); ok {
		cs = append(cs, c...)
	} else {
		cs = append(cs, right)
	}
	return cs
}

type Body interface {
	Value
	Compressed() bool
	ContentType() string
}

type xmlBody struct {
	Set
}

func xmlify(set Set) Body {
	return xmlBody{
		Set: set,
	}
}

func (b xmlBody) Expand(env environ.Environment[Value]) (string, error) {
	vs, err := b.Set.Map(env)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := xml.NewEncoder(&buf).Encode(vs); err != nil {
		return "", err
	}
	return buf.String(), nil
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
	vs, err := b.Set.Map(env)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(vs); err != nil {
		return "", err
	}
	return buf.String(), nil
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

type Value interface {
	Expand(environ.Environment[Value]) (string, error)
	clone() Value
}

type Authorization interface {
	Value
	Method() string
}

type basic struct {
	User Value
	Pass Value
}

func (b basic) Method() string {
	return "Basic"
}

func (b basic) clone() Value {
	return basic{
		User: b.User.clone(),
		Pass: b.Pass.clone(),
	}
}

func (b basic) Expand(env environ.Environment[Value]) (string, error) {
	user, err := b.User.Expand(env)
	if err != nil {
		return "", err
	}
	pass, err := b.Pass.Expand(env)
	if err != nil {
		return "", err
	}
	str := fmt.Sprintf("%s:%s", user, pass)
	return base64.URLEncoding.EncodeToString([]byte(str)), nil
}

type token struct {
	Claims Set
	Alg    string
	Secret string
}

func (_ token) Method() string {
	return "Bearer"
}

func (_ token) clone() Value {
	return nil
}

func (t token) Expand(env environ.Environment[Value]) (string, error) {
	var (
		res = make(map[string]any)
		cfg = jwt.Config{
			Alg:    t.Alg,
			Secret: t.Secret,
		}
	)
	for k, vs := range t.Claims {
		var rs []string
		for i := range vs {
			str, err := vs[i].Expand(env)
			if err != nil {
				return "", err
			}
			rs = append(rs, str)
		}
		if len(rs) == 1 {
			res[k] = rs[0]
		} else {
			res[k] = rs
		}
	}
	return jwt.Encode(res, &cfg)
}

type bearer struct {
	Token Value
}

func (b bearer) Method() string {
	return "Bearer"
}

func (b bearer) clone() Value {
	return bearer{
		Token: b.Token.clone(),
	}
}

func (b bearer) Expand(env environ.Environment[Value]) (string, error) {
	return b.Token.Expand(env)
}

type literal string

func createLiteral(str string) Value {
	return literal(str)
}

func (i literal) clone() Value {
	return i
}

func (i literal) Expand(_ environ.Environment[Value]) (string, error) {
	return string(i), nil
}

type variable string

func createVariable(str string) Value {
	return variable(str)
}

func (v variable) clone() Value {
	return v
}

func (v variable) Expand(e environ.Environment[Value]) (string, error) {
	val, err := e.Resolve(string(v))
	if err != nil {
		return "", err
	}
	return val.Expand(e)
}

type compound []Value

func (c compound) clone() Value {
	var cs compound
	for i := range c {
		cs = append(cs, c[i].clone())
	}
	return cs
}

func (c compound) Expand(e environ.Environment[Value]) (string, error) {
	var parts []string
	for i := range c {
		v, err := c[i].Expand(e)
		if err != nil {
			return "", err
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, ""), nil
}

type Set map[string][]Value

func (s Set) Headers(env environ.Environment[Value]) (http.Header, error) {
	hs := make(http.Header)
	for k := range s {
		for _, v := range s[k] {
			str, err := v.Expand(env)
			if err != nil {
				return nil, err
			}
			hs.Add(k, str)
		}
	}
	return hs, nil
}

func (s Set) Map(env environ.Environment[Value]) (map[string]interface{}, error) {
	vs := make(map[string]interface{})
	for k := range s {
		var arr []string
		for _, v := range s[k] {
			str, err := v.Expand(env)
			if err != nil {
				return nil, err
			}
			arr = append(arr, str)
		}
		var dat interface{} = arr
		if len(arr) == 1 {
			dat = arr[0]
		}
		vs[k] = dat
	}
	return vs, nil
}

func (s Set) Query(env environ.Environment[Value]) (url.Values, error) {
	vs := make(url.Values)
	for k := range s {
		for _, v := range s[k] {
			str, err := v.Expand(env)
			if err != nil {
				return nil, err
			}
			vs.Add(k, str)
		}
	}
	return vs, nil
}

func (s Set) Merge(other Set) Set {
	var ns = make(Set)
	for k := range s {
		ns[k] = slices.Clone(s[k])
	}
	for k := range other {
		ns[k] = slices.Concat(ns[k], slices.Clone(other[k]))
	}
	return ns
}
