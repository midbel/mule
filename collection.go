package mule

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
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
	environ.Environment[Value]

	BeforeAll  string
	BeforeEach string
	AfterAll   string
	AfterEach  string

	// collection of requests
	Steps []*Step
}

func (f *Flow) Execute(args []string, out io.Writer) error {
	return nil
}

type Step struct {
	Request string
	Next    []StepBody
}

type StepBody struct {
	Predicate Value
	Commands  []any
}

type gotog struct {
	Ident Value
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
	// scripts to be run
	BeforeAll  string
	BeforeEach string
	AfterAll   string
	AfterEach  string

	// collection of requests
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

func (c Collection) GetCollections() []*Collection {
	return c.Collections
}

func (c Collection) GetRequests() []*Request {
	return c.Requests
}

func (c Collection) Resolve(ident string) (Value, error) {
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

func (c *Collection) Execute() error {
	return nil
}

func (c *Collection) Get(name string) (*http.Request, error) {
	req, err := c.FindRequest(name)
	if err != nil {
		return nil, err
	}
	return req.createRequest(c)
}

func (c *Collection) Run(name string, args []string, w io.Writer) error {
	var (
		rest  string
		found bool
	)
	name, rest, found = strings.Cut(name, ".")
	if !found {
		req, err := c.GetRequest(name)
		if err != nil {
			other, err := c.GetCollection(name)
			if err != nil {
				return err
			}
			return other.Execute()
		}
		return req.Execute(c, args, w)
	}
	other, err := c.GetCollection(name)
	if err != nil {
		return err
	}
	return other.Run(rest, args, w)
}

func (c *Collection) FindRequest(name string) (*Request, error) {
	var (
		rest  string
		found bool
	)
	name, rest, found = strings.Cut(name, ".")
	if !found {
		return c.GetRequest(name)
	}
	other, err := c.GetCollection(name)
	if err != nil {
		return nil, err
	}
	return other.FindRequest(rest)
}

func (c *Collection) FindCollection(name string) (*Collection, error) {
	name, rest, ok := strings.Cut(name, ".")

	sub, err := c.GetCollection(name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return sub, nil
	}
	return sub.FindCollection(rest)
}

func (c *Collection) GetCollection(name string) (*Collection, error) {
	ix := slices.IndexFunc(c.Collections, func(other *Collection) bool {
		return other.Name == name
	})

	if ix < 0 {
		return nil, fmt.Errorf("%s: collection not found", name)
	}

	sub := *c.Collections[ix]

	sub.URL = getUrl(c.URL, sub.URL, sub)
	sub.Headers = sub.Headers.Merge(c.Headers)
	sub.Query = sub.Query.Merge(c.Query)

	return &sub, nil
}

func (c *Collection) GetRequest(name string) (*Request, error) {
	ix := slices.IndexFunc(c.Requests, func(other *Request) bool {
		return other.Name == name
	})

	if ix < 0 {
		return nil, fmt.Errorf("%s: request not found", name)
	}

	req := *c.Requests[ix]
	if req.Abstract {
		return nil, fmt.Errorf("%s: request can not be executed", name)
	}

	req.URL = getUrl(c.URL, req.URL, c)
	req.Headers = req.Headers.Merge(c.Headers)
	req.Query = req.Query.Merge(c.Query)

	return &req, nil
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
}

func (r *Request) Merge(other *Request) error {
	if other.URL == nil && r.URL != nil {
		other.URL = r.URL.clone()
	}
	if other.Auth == nil && r.Auth != nil {
		other.URL = r.Auth.clone()
	}
	if other.Retry == nil && r.Retry != nil {
		other.URL = r.Retry.clone()
	}
	if other.Timeout == nil && r.Timeout != nil {
		other.URL = r.Timeout.clone()
	}
	if other.Redirect == nil && r.Redirect != nil {
		other.URL = r.Redirect.clone()
	}
	if other.Body == nil && r.Body != nil {
		other.URL = r.Body.clone()
	}
	if other.Compressed == nil && r.Compressed != nil {
		other.URL = r.Compressed.clone()
	}
	if other.Before == "" && r.Before != "" {
		other.Before = r.Before
	}
	if other.After == "" && r.After != "" {
		other.After = r.After
	}
	if len(other.Depends) == 0 {
		for i := range r.Depends {
			other.Depends = append(other.Depends, r.Depends[i].clone())
		}
	}
	return nil
}

func (r *Request) Execute(ctx *Collection, args []string, out io.Writer) error {
	if err := r.parseArgs(ctx, args); err != nil {
		return err
	}
	req, err := r.createRequest(ctx)
	if err != nil {
		return err
	}
	var (
		root = play.Enclosed(play.Default())
		obj  = muleObject{
			when: time.Now(),
			req:  getMuleRequest(req, nil),
			ctx:  getMuleCollection(ctx),
			vars: getMuleVars(),
		}
		env = play.Enclosed(root)
	)
	root.Define(muleVarName, &obj)

	if _, err := play.EvalWithEnv(strings.NewReader(r.Before), env); err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	res.Body = io.NopCloser(bytes.NewReader(buf))

	obj.res = getMuleResponse(res, buf)
	root.Define(muleVarName, &obj)

	if _, err := play.EvalWithEnv(strings.NewReader(r.After), env); err != nil {
		return err
	}
	io.Copy(out, res.Body)
	if res.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf(http.StatusText(res.StatusCode))
	}
	return nil
}

func (r *Request) parseArgs(ctx *Collection, args []string) error {
	set := flag.NewFlagSet(r.Name, flag.ExitOnError)
	if is, ok := ctx.Environment.(interface{ Identifiers() []string }); ok {
		for _, i := range is.Identifiers() {
			set.Func(i, "", func(str string) error {
				return ctx.Define(i, createLiteral(str))
			})
		}
	}
	set.Func("method", "", func(str string) error {
		r.Method = str
		return nil
	})
	set.Func("header", "", func(str string) error {
		name, value, ok := strings.Cut(str, ":")
		if !ok {
			return fmt.Errorf("%s: invalid header value", str)
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		r.Headers[name] = append(r.Headers[name], createLiteral(value))
		return nil
	})
	set.Func("body", "", func(str string) error {
		r, err := os.Open(str)
		if err == nil {
			defer r.Close()
		} else {

		}
		return nil
	})
	return set.Parse(args)
}

func (r *Request) createRequest(env environ.Environment[Value]) (*http.Request, error) {
	target, err := r.target(env)
	if err != nil {
		return nil, err
	}

	headers, err := r.Headers.Headers(env)
	if err != nil {
		return nil, err
	}
	if r.Auth != nil {
		val, err := r.Auth.Expand(env)
		if err != nil {
			return nil, err
		}
		val = fmt.Sprintf("%s %s", r.Auth.Method(), val)
		headers.Set("Authorization", val)
	}

	var body io.Reader
	if r.Body != nil {
		b, err := r.Body.Expand(env)
		if err != nil {
			return nil, err
		}
		body = strings.NewReader(b)
		headers.Set("content-type", r.Body.ContentType())
		headers.Set("content-length", strconv.Itoa(len(b)))
	}
	req, err := http.NewRequest(r.Method, target, body)
	if err == nil {
		req.Header = headers
	}
	return req, err
}

func (r *Request) target(env environ.Environment[Value]) (string, error) {
	target, err := r.URL.Expand(env)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", err
	}

	vs, err := r.Query.Query(env)
	if err != nil {
		return "", err
	}
	if u.RawQuery == "" {
		u.RawQuery = vs.Encode()
	}
	return u.String(), nil
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

func getUrl(left, right Value, env environ.Environment[Value]) Value {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}

	if str, err := right.Expand(env); err == nil {
		u, err := url.Parse(str)
		if err == nil && u.Host != "" {
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

func getValue(left, right Value) Value {
	if left != nil {
		return left
	}
	return right
}
