package mule

import (
	"net/http"

	"github.com/midbel/mule/play"
)

type muleObject struct {
	req *muleRequest
	res *muleResponse
	ctx *Collection
}

func (_ *muleObject) True() play.Value {
	return play.NewBool(true)
}

func (m *muleObject) Get(ident play.Value) (play.Value, error) {
	return nil, nil
}

type muleCollection struct {
	collection *Collection
}

func (_ *muleCollection) True() play.Value {
	return play.NewBool(true)
}

func (m *muleCollection) Call(ident string, args []play.Value) (play.Value, error) {
	switch ident {
	case "getVariable":
	case "setVariable":
	case "hasVariable":
	default:
		return nil, fmt.Errorf("%s: unknown function", ident)
	}
	return nil, play.ErrImpl
}

type muleRequest struct {
	request *http.Request
}

func (m *muleRequest) True() play.Value {
	ok := m.request != nil
	return play.NewBool(ok)
}

func (m *muleRequest) Get(ident play.Value) (play.Value, error) {
	return nil, nil
}

type muleResponse struct {
	response *http.Response
}

func (m *muleResponse) True() play.Value {
	ok := m.response != nil
	return play.NewBool(ok)
}

func (m *muleResponse) Get(ident play.Value) (play.Value, error) {
	return nil, nil
}

type muleHeader struct {
	headers http.Header
}

func (_ *muleHeader) True() play.Value {
	return nil
}
