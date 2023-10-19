package mule

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/midbel/enjoy/env"
	"github.com/midbel/enjoy/eval"
	"github.com/midbel/enjoy/value"
)

var errReusable = errors.New("not reusable")

const (
	reqUri      = "requestUri"
	reqName     = "requestName"
	reqDuration = "requestDuration"
	resStatus   = "responseStatus"
	resBody     = "responseBody"
)

type Context struct {
	value.Global
	root *Collection
	Cache
}

func MuleEnv(ctx *Context) env.Environ[value.Value] {
	top := eval.Default()
	sub := env.EnclosedEnv[value.Value](top)
	sub.Define("mule", ctx, true)

	return env.EnclosedEnv[value.Value](env.Immutable(sub))
}

func MuleContext(root *Collection) (*Context, error) {
	obj := Context{
		Global: value.CreateGlobal("mule"),
		root:   root,
	}
	obj.RegisterProp("variables", createMuleVars(root))
	obj.RegisterProp("environ", createEnvVars())

	cache, err := Bolt()
	if err != nil {
		return nil, err
	}
	obj.Cache = cache

	return &obj, nil
}

func (c *Context) Store(res *http.Response) error {
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err := c.Cache.Put("", data); err != nil {
		return err
	}
	res.Body = io.NopCloser(bytes.NewReader(data))
	return nil
}

func (c *Context) Reusable(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return nil, errReusable
	}
	body, err := c.Cache.Get(req.URL.String(), time.Second*5)
	if err != nil {
		return nil, err
	}
	var res http.Response
	res.StatusCode = http.StatusNotModified
	res.Status = http.StatusText(res.StatusCode)
	res.Proto = "HTTP/1.1"
	res.ProtoMajor = 1
	res.ProtoMinor = 1
	res.Header = make(http.Header)
	res.Body = io.NopCloser(bytes.NewReader(body))
	res.ContentLength = 0
	res.Uncompressed = true
	res.Request = req
	return &res, nil
}

func (c *Context) Get(prop string) (value.Value, error) {
	switch prop {
	case "collections":
		var list []value.Value
		for _, c := range c.root.Collections() {
			list = append(list, value.CreateString(c))
		}
		return value.CreateArray(list), nil
	default:
		return c.Global.Get(prop)
	}
}

type responseValue struct {
	res *http.Response
}

func createResponseValue(res *http.Response) value.Value {
	return responseValue{
		res: res,
	}
}

func (_ responseValue) True() bool {
	return true
}

func (_ responseValue) Type() string {
	return "object"
}

func (_ responseValue) String() string {
	return "<response>"
}

func (r responseValue) Get(prop string) (value.Value, error) {
	switch prop {
	case "headers":
		list := make(map[string]value.Value)
		for k, v := range r.res.Header {
			list[k] = value.CreateString(v[0])
		}
		return value.CreateObject(list), nil
	case "status":
		return value.CreateString(r.res.Status), nil
	case "code":
		return value.CreateFloat(float64(r.res.StatusCode)), nil
	case "contentLength":
		return value.CreateFloat(float64(r.res.ContentLength)), nil
	default:
		return value.Undefined(), nil
	}
}

type headersValue struct {
	req *http.Request
}

func createHeadersValue(req *http.Request) value.Value {
	return headersValue{
		req: req,
	}
}

func (_ headersValue) True() bool {
	return true
}

func (_ headersValue) Type() string {
	return "object"
}

func (_ headersValue) String() string {
	return "<headers>"
}

func (h headersValue) Get(prop string) (value.Value, error) {
	values := h.req.Header.Values(prop)
	if len(values) == 1 {
		return value.CreateString(values[0]), nil
	}
	var arr []value.Value
	for i := range values {
		arr = append(arr, value.CreateString(values[i]))
	}
	return value.CreateArray(arr), nil
}

func (h headersValue) Set(prop string, val value.Value) error {
	switch v := val.(type) {
	case *value.Array:
		// for i := range v {
		// 	h.req.Header.Add(prop, v[i].String())
		// }
	default:
		h.req.Header.Add(prop, v.String())
	}
	return nil
}

type requestValue struct {
	req *http.Request
}

func createRequestValue(req *http.Request) value.Value {
	return requestValue{
		req: req,
	}
}

func (_ requestValue) True() bool {
	return true
}

func (_ requestValue) Type() string {
	return "object"
}

func (_ requestValue) String() string {
	return "<request>"
}

func (r requestValue) Get(prop string) (value.Value, error) {
	switch prop {
	case "method":
		return value.CreateString(r.req.Method), nil
	case "url":
		s := r.req.URL.String()
		return value.CreateString(s), nil
	case "headers":
		return createHeadersValue(r.req), nil
	default:
		return value.Undefined(), nil
	}
}

func (r requestValue) Set(prop string, val value.Value) error {
	switch prop {
	case "url":
		u, err := url.Parse(val.String())
		if err != nil {
			return err
		}
		r.req.URL = u
	case "body":
		if r.req.Body != nil {
			r.req.Body.Close()
		}
		tmp := strings.NewReader(val.String())
		r.req.Body = io.NopCloser(tmp)
	default:
	}
	return nil
}

type envVars struct{}

func createEnvVars() value.Value {
	return envVars{}
}

func (_ envVars) True() bool {
	return true
}

func (_ envVars) Type() string {
	return "object"
}

func (_ envVars) String() string {
	return "<environ>"
}

func (v envVars) Get(prop string) (value.Value, error) {
	n := strings.ToUpper(prop)
	s := os.Getenv(n)
	return value.CreateString(s), nil
}

func (v envVars) Call(fn string, args []value.Value) (value.Value, error) {
	switch fn {
	case "get":
		return v.Get(args[0].String())
	default:
		return nil, value.ErrOperation
	}
}

type muleVars struct {
	context env.Environ[string]
}

func createMuleVars(ev env.Environ[string]) value.Value {
	return muleVars{
		context: ev,
	}
}

func (_ muleVars) True() bool {
	return true
}

func (_ muleVars) Type() string {
	return "object"
}

func (_ muleVars) String() string {
	return "<variables>"
}

func (v muleVars) Call(fn string, args []value.Value) (value.Value, error) {
	switch fn {
	case "set":
		err := v.context.Assign(args[0].String(), args[1].String())
		return value.Undefined(), err
	case "get":
		s, err := v.context.Resolve(args[0].String())
		return value.CreateString(s), err
	default:
		return nil, value.ErrOperation
	}
}
