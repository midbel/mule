package mule

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/midbel/mule/env"
)

type Word interface {
	Expand(env.Env[string]) (string, error)
	ExpandBool(env.Env[string]) (bool, error)
	ExpandInt(env.Env[string]) (int, error)
	ExpandURL(env.Env[string]) (*url.URL, error)
}

type compound []Word

func (cs compound) Expand(e env.Env[string]) (string, error) {
	var list []string
	for _, w := range cs {
		str, err := w.Expand(e)
		if err != nil {
			return "", err
		}
		list = append(list, str)
	}
	return strings.Join(list, ""), nil
}

func (cs compound) ExpandBool(e env.Env[string]) (bool, error) {
	str, err := cs.Expand(e)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(str)
}

func (cs compound) ExpandInt(e env.Env[string]) (int, error) {
	str, err := cs.Expand(e)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}

func (cs compound) ExpandURL(e env.Env[string]) (*url.URL, error) {
	str, err := cs.Expand(e)
	if err != nil {
		return nil, err
	}
	return url.Parse(str)
}

type literal string

func createLiteral(str string) Word {
	return literal(str)
}

func (i literal) Expand(_ env.Env[string]) (string, error) {
	return string(i), nil
}

func (i literal) ExpandBool(_ env.Env[string]) (bool, error) {
	return strconv.ParseBool(string(i))
}

func (i literal) ExpandInt(_ env.Env[string]) (int, error) {
	return strconv.Atoi(string(i))
}

func (i literal) ExpandURL(_ env.Env[string]) (*url.URL, error) {
	return url.Parse(string(i))
}

type variable string

func createVariable(str string) Word {
	return variable(str)
}

func (v variable) Expand(e env.Env[string]) (string, error) {
	return e.Resolve(string(v))
}

func (v variable) ExpandBool(e env.Env[string]) (bool, error) {
	str, err := v.Expand(e)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(str)
}

func (v variable) ExpandInt(e env.Env[string]) (int, error) {
	str, err := v.Expand(e)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}

func (v variable) ExpandURL(e env.Env[string]) (*url.URL, error) {
	str, err := v.Expand(e)
	if err != nil {
		return nil, err
	}
	return url.Parse(str)
}
