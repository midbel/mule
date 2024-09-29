package play

import (
	"fmt"
	"net/url"
	"time"
)

type Env struct{}

func (e *Env) True() Value {
	return getBool(true)
}

func (e *env) String() string {
	return "env"
}

func (a *Array) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	str = os.Getenv(str)
	return getString(str, nil)
}

type Date struct {
	value time.Time
}

func (d *Date) True() Value {
	return getBool(true)
}

func (d *Date) String() string {
	return "date"
}

func (d *Date) Call(ident string, args []Value) (Value, error) {
	return Void{}, ErrImpl
}

type Url struct {
	value *url.URL
}

func NewURL(str *url.URL) Value {
	return &Url{
		value: str,
	}
}

func (u *Url) True() Value {
	return getBool(true)
}

func (u *Url) String() string {
	return u.value.String()
}

func (u *Url) Get(ident Value) (Value, error) {
	str, ok := ident.(fmt.Stringer)
	if !ok {
		return nil, ErrEval
	}
	switch name := str.String(); name {
	case "host", "hostname":
		return getString(u.value.Hostname()), nil
	case "port":
		return getString(u.value.Port()), nil
	case "path":
		return getString(u.value.Path), nil
	case "query":
		return getString(u.value.RawQuery), nil
	case "scheme":
		return getString(u.value.Scheme), nil
	default:
		return nil, fmt.Errorf("%s: undefined property", name)
	}
}
