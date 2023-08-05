package eval

import (
	"fmt"
	"io"

	"github.com/midbel/mule/env"
)

type Expression interface{}

func Eval(r io.Reader) (Object, error) {
	expr, err := Parse(r)
	if err != nil {
		return nil, err
	}
	return EvalExpr(expr, env.EmptyEnv[Object]())
}

func EvalExpr(node Node, ev env.Env[Object]) (Object, error) {
	return eval(node, ev)
}

func eval(node Node, ev env.Env[Object]) (Object, error) {
	switch n := node.(type) {
	case Primitive[float64]:
		return evalNumber(n, ev)
	case Primitive[string]:
		return evalString(n, ev)
	case Primitive[bool]:
		return evalBool(n, ev)
	case Variable:
		return evalVariable(n, ev)
	case Chain:
		return evalChain(n, ev)
	case Index:
		return evalIndex(n, ev)
	case Array:
		return evalArray(n, ev)
	case Object:
		return evalObject(n, ev)
	case Call:
		return evalCall(n, ev)
	case Block:
		return evalBlock(n, ev)
	case Binary:
		return evalBinary(n, ev)
	case Unary:
		return evalUnary(n, ev)
	case Let:
		return evalLet(n, ev)
	case If:
	case Switch:
	case For:
	case While:
	case Try:
	case Catch:
	case Function:
	default:
		return nil, fmt.Errorf("%T unsupported node type", node)
	}
	return nil, nil
}

func evalString(p Primitive[string], _ env.Env[Object]) (Object, error) {
	return CreateObject(p.Literal)
}

func evalNumber(p Primitive[float64], _ env.Env[Object]) (Object, error) {
	return CreateObject(p.Literal)
}

func evalBool(p Primitive[bool], _ env.Env[Object]) (Object, error) {
	return CreateObject(p.Literal)
}

func evalVariable(v Variable, ev env.Env[Object]) (Object, error) {
	return ev.Resolve(v.Ident)
}

func evalChain(c Chain, ev env.Env[Object]) (Object, error) {
	return nil, nil
}

func evalIndex(i Index, ev env.Env[Object]) (Object, error) {
	return nil, nil
}

func evalArray(a Array, ev env.Env[Object]) (Object, error) {
	var arr []Object
	for i := range a.List {
		v, err := eval(a.List[i], ev)
		if err != nil {
			return nil, err
		}
		arr = append(arr, v)
	}
	return CreateArray(arr), nil
}

func evalObject(o Object, ev env.Env[Object]) (Object, error) {
	return nil, nil
}

func evalCall(c Call, ev env.Env[Object]) (Object, error) {
	return nil, nil
}

func evalBlock(b Block, ev env.Env[Object]) (Object, error) {
	var (
		res Object
		err error
		tmp = env.EnclosedEnv[Object](ev)
	)
	for i := range b.List {
		res, err = eval(b.List[i], tmp)
		if err != nil {
			return nil, err
		}
	}
	return res, err
}

func evalBinary(b Binary, ev env.Env[Object]) (Object, error) {
	left, err := eval(b.Left, ev)
	if err != nil {
		return nil, err
	}
	right, err := eval(b.Right, ev)
	if err != nil {
		return nil, err
	}
	switch b.Op {
	case Add:
		if a, ok := left.(adder); ok {
			return a.Add(right)
		}
	case Sub:
		if s, ok := left.(suber); ok {
			return s.Sub(right)
		}
	case Mul:
		if m, ok := left.(muler); ok {
			return m.Mul(right)
		}
	case Div:
		if d, ok := left.(diver); ok {
			return d.Div(right)
		}
	case Pow:
		if p, ok := left.(power); ok {
			return p.Pow(right)
		}
	case Mod:
		if m, ok := left.(moder); ok {
			return m.Mod(right)
		}
	case Lshift:
	case Rshift:
	case Band:
	case Bor:
	case And:
		return leftAndRight(left, right)
	case Or:
		return leftOrRight(left, right)
	case Eq:
	case Ne:
	case Lt:
	case Le:
	case Gt:
	case Ge:
	default:
		return nil, fmt.Errorf("unsupported operator")
	}
	return nil, ErrOperation
}

func evalAssignment(a Assignment, ev env.Env[Object]) (Object, error) {
	ident, ok := a.Ident.(Variable)
	if !ok {
		return nil, fmt.Errorf("variable expected")
	}
	value, err := eval(a.Expr, ev)
	if err != nil {
		return nil, err
	}
	ev.Define(ident.Ident, value)
	return value, nil
}

func evalUnary(u Unary, ev env.Env[Object]) (Object, error) {
	return nil, nil
}

func evalLet(e Let, ev env.Env[Object]) (Object, error) {
	val, err := eval(e.Expr, ev)
	if err == nil {
		ev.Define(e.Ident, val)
	}
	return val, err
}
