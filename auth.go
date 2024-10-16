package mule

import (
	"encoding/base64"
	"fmt"

	"github.com/midbel/mule/environ"
	"github.com/midbel/mule/jwt"
)

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
