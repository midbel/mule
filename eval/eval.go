package eval

import (
	"io"

	"github.com/midbel/mule/env"
)

type Expression interface {
	Eval(env.Env) error
}

func Eval(r io.Reader, env env.Env) error {
	return nil
}
