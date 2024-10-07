package mule

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/midbel/mule/environ"
	"github.com/midbel/mule/jwt"
	// "github.com/midbel/mule/play"
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
	Usage string
	environ.Environment[Value]

	BeforeAll  string
	BeforeEach string
	AfterAll   string
	AfterEach  string

	Steps []*Step
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

	BeforeAll  string
	BeforeEach string
	AfterAll   string
	AfterEach  string

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

func (c Collection) Find(name string) (*Collection, error) {
	return nil, nil
}

func (c Collection) Run(name string, args []string, out io.Writer) error {
	return nil
}

func (c Collection) Get(name string) (*http.Request, error) {
	return nil, nil
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
	return nil
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
