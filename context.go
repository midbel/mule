package mule

import (
	"fmt"
	"net/http"

	"github.com/midbel/mule/play"
)

type muleObject struct {
	req *muleRequest
	res *muleResponse
	ctx *muleCollection
}

func (_ *muleObject) String() string {
	return "mule"
}

func (_ *muleObject) True() play.Value {
	return play.NewBool(true)
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
	default:
		return nil, fmt.Errorf("%s: property not known", prop)
	}
}

type muleCollection struct {
	collection *Collection
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
	case "set":
	case "has":
	default:
		return nil, fmt.Errorf("%s: unknown function", ident)
	}
	return nil, play.ErrImpl
}

type muleRequest struct {
	request *http.Request
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
	case "url":
		return play.NewURL(m.request.URL), nil
	case "method":
		return play.NewString(m.request.Method), nil
	case "token":
		return play.NewString(""), nil
	case "username":
		user, _, _ := m.request.BasicAuth()
		return play.NewString(user), nil
	case "password":
		_, pass, _ := m.request.BasicAuth()
		return play.NewString(pass), nil
	case "header":
		return &muleHeader{
			headers: m.request.Header,
		}, nil
	default:
		return play.Void{}, fmt.Errorf("%s: unknown property", ident)
	}
}

type muleResponse struct {
	response *http.Response
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
		return play.NewString(""), nil
	case "code":
		return play.NewFloat(0), nil
	case "header":
		return &muleHeader{
			headers: m.response.Header,
		}, nil
	default:
		return play.Void{}, fmt.Errorf("%s: unknown property", ident)
	}
}

type muleHeader struct {
	headers http.Header
}

func (_ *muleHeader) True() play.Value {
	return nil
}

func (_ *muleHeader) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "get":
	case "set":
	case "has":
	default:
		return nil, fmt.Errorf("%s: unknown function", ident)
	}
	return nil, play.ErrImpl
}
