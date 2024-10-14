package mule

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/midbel/mule/environ"
	"github.com/midbel/mule/play"
)

var (
	ErrAbort     = errors.New("abort")
	ErrCancel    = errors.New("cancel")
	ErrImmutable = errors.New("immutable")
)

const muleVarName = "mule"

type muleObject struct {
	when time.Time
	req  *muleRequest
	res  *muleResponse
	ctx  *muleCollection
	vars *muleVars

	play.EventHandler
}

func getMuleObject(ctx *Collection) *muleObject {
	return &muleObject{
		when: time.Now(),
		ctx:  getMuleCollection(ctx),
		vars: getMuleVars(),
	}
}

func (m *muleObject) reset() {
	m.req = nil
	m.res = nil
}

func (_ *muleObject) String() string {
	return muleVarName
}

func (_ *muleObject) True() play.Value {
	return play.NewBool(true)
}

func (m *muleObject) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "cancel":
		return play.Void{}, ErrCancel
	case "abort":
		return play.Void{}, ErrAbort
	case "elapsed":
		millis := time.Since(m.when).Milliseconds()
		return play.NewFloat(float64(millis)), nil
	default:
		return nil, fmt.Errorf("%s: undefined fonction", ident)
	}
}

func (m *muleObject) Get(ident play.Value) (play.Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, play.ErrEval
	}
	switch prop := str.String(); prop {
	case "collection":
		return m.ctx, nil
	case "request":
		return m.req, nil
	case "response":
		return m.res, nil
	case "variables":
		return m.vars, nil
	case "environ":
		return &muleEnviron{}, nil
	default:
		return nil, fmt.Errorf("%s: property not known", prop)
	}
}

type muleCollection struct {
	collection *Collection
}

func getMuleCollection(ctx *Collection) *muleCollection {
	return &muleCollection{
		collection: ctx,
	}
}

func (m *muleCollection) String() string {
	return m.collection.Name
}

func (_ *muleCollection) True() play.Value {
	return play.NewBool(true)
}

func (m *muleCollection) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "get":
		if len(args) != 1 {
			return play.Void{}, play.ErrArgument
		}
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		v, err := m.collection.Resolve(str.String())
		if err != nil {
			return play.Void{}, nil
		}
		res, err := v.Expand(m.collection)
		if err != nil {
			return play.Void{}, err
		}
		return play.NewString(res), nil
	case "set":
		if len(args) != 2 {
			return play.Void{}, play.ErrArgument
		}
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		val, ok := args[1].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		res := createLiteral(val.String())
		return play.Void{}, m.collection.Define(str.String(), res)
	case "has":
		if len(args) != 1 {
			return play.Void{}, play.ErrArgument
		}
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		_, err := m.collection.Resolve(str.String())
		if err != nil {
			return play.NewBool(false), nil
		}
		return play.NewBool(true), nil
	default:
		return play.Void{}, fmt.Errorf("%s: unknown function", ident)
	}
}

type muleRequest struct {
	request *http.Request
	name    string
	auth    Authorization
	body    []byte
}

func getMuleRequest(req *http.Request, name string, body []byte) *muleRequest {
	return &muleRequest{
		request: req,
		name:    name,
		body:    body,
	}
}

func (_ *muleRequest) String() string {
	return "request"
}

func (m *muleRequest) True() play.Value {
	ok := m.request != nil
	return play.NewBool(ok)
}

func (m *muleRequest) Get(ident play.Value) (play.Value, error) {
	prop, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, play.ErrEval
	}
	switch ident := prop.String(); ident {
	case "body":
		return play.NewString(""), nil
	case "name":
		return play.NewString(m.name), nil
	case "url":
		return play.NewURL(m.request.URL), nil
	case "method":
		return play.NewString(m.request.Method), nil
	case "token":
		token := m.request.Header.Get("authorization")
		token, _ = strings.CutPrefix(token, "Bearer ")
		return play.NewString(token), nil
	case "username":
		user, _, _ := m.request.BasicAuth()
		return play.NewString(user), nil
	case "password":
		_, pass, _ := m.request.BasicAuth()
		return play.NewString(pass), nil
	case "header":
		return &muleHeader{
			headers:   m.request.Header,
			immutable: false,
		}, nil
	case "auth":
		return play.Void{}, nil
	default:
		return play.Void{}, nil
	}
}

type muleResponse struct {
	response *http.Response
	body     []byte
}

func getMuleResponse(res *http.Response, body []byte) *muleResponse {
	return &muleResponse{
		response: res,
		body:     body,
	}
}

func (_ *muleResponse) String() string {
	return "response"
}

func (m *muleResponse) True() play.Value {
	ok := m.response != nil
	return play.NewBool(ok)
}

func (m *muleResponse) Get(ident play.Value) (play.Value, error) {
	prop, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, play.ErrEval
	}
	switch ident := prop.String(); ident {
	case "body":
		return play.NewString(string(m.body)), nil
	case "code":
		return play.NewFloat(float64(m.response.StatusCode)), nil
	case "headers":
		return &muleHeader{
			headers:   m.response.Header,
			immutable: true,
		}, nil
	default:
		return play.Void{}, nil
	}
}

func (m *muleResponse) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "json":
		var (
			obj interface{}
			buf = bytes.NewReader(m.body)
		)
		if err := json.NewDecoder(buf).Decode(&obj); err != nil {
			return play.Void{}, err
		}
		return play.NativeToValues(obj)
	case "success":
		ok := m.response.StatusCode < http.StatusBadRequest
		return play.NewBool(ok), nil
	case "fail":
		ok := m.response.StatusCode >= http.StatusBadRequest
		return play.NewBool(ok), nil
	case "badRequest":
		ok := m.response.StatusCode >= http.StatusBadRequest && m.response.StatusCode < http.StatusInternalServerError
		return play.NewBool(ok), nil
	case "serverError":
		ok := m.response.StatusCode >= http.StatusInternalServerError
		return play.NewBool(ok), nil
	default:
		return nil, fmt.Errorf("%s: unknown function", ident)
	}
}

type muleHeader struct {
	headers   http.Header
	immutable bool
}

func (_ *muleHeader) True() play.Value {
	return play.NewBool(true)
}

func (m *muleHeader) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "get":
		if len(args) != 1 {
			return nil, play.ErrArgument
		}
		id, ok := args[0].(fmt.Stringer)
		if !ok {
			return nil, play.ErrEval
		}
		arr := play.NewArray()
		for _, h := range m.headers[id.String()] {
			arr.Append(play.NewString(h))
		}
		return arr, nil
	case "set":
		if m.immutable {
			return nil, ErrImmutable
		}
	case "has":
		if len(args) != 1 {
			return nil, play.ErrArgument
		}
		id, ok := args[0].(fmt.Stringer)
		if !ok {
			return nil, play.ErrEval
		}
		_, ok = m.headers[id.String()]
		return play.NewBool(ok), nil
	case "entries":
		arr := play.NewArray()
		for n, hs := range m.headers {
			var (
				sub = play.NewArray()
				all = play.NewArray()
			)
			for _, h := range hs {
				all.Append(play.NewString(h))
			}
			sub.Append(play.NewString(n))
			sub.Append(all)
			arr.Append(sub)
		}
		return arr, nil
	default:
		return nil, fmt.Errorf("%s: unknown function", ident)
	}
	return nil, play.ErrImpl
}

type muleEnviron struct{}

func (_ *muleEnviron) String() string {
	return "environ"
}

func (_ *muleEnviron) True() play.Value {
	return play.NewBool(true)
}

func (_ *muleEnviron) Get(ident play.Value) (play.Value, error) {
	prop, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, play.ErrEval
	}
	return play.NewString(os.Getenv(prop.String())), nil
}

type muleVars struct {
	env environ.Environment[play.Value]
}

func getMuleVars() *muleVars {
	return &muleVars{
		env: play.Empty(),
	}
}

func (_ *muleVars) String() string {
	return "variables"
}

func (_ *muleVars) True() play.Value {
	return play.NewBool(true)
}

func (v *muleVars) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "get":
		if len(args) != 1 {
			return play.Void{}, play.ErrArgument
		}
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		return v.env.Resolve(str.String())
	case "set":
		if len(args) != 2 {
			return play.Void{}, play.ErrArgument
		}
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		return play.Void{}, v.env.Define(str.String(), args[1])
	case "has":
		if len(args) != 1 {
			return play.Void{}, play.ErrArgument
		}
		str, ok := args[0].(fmt.Stringer)
		if !ok {
			return play.Void{}, play.ErrEval
		}
		_, err := v.env.Resolve(str.String())
		if err == nil {
			return play.NewBool(true), nil
		}
		return play.NewBool(false), nil
	default:
		return play.Void{}, fmt.Errorf("%s: unknown function", ident)
	}
}
