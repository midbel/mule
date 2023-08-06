package eval

import (
	"fmt"
	"os"
)

type Object struct{}

func (_ Object) Not() (Value, error) {
	return nil, ErrOperation
}

func (_ Object) True() bool {
	return false
}

func (_ Object) Raw() any {
	return nil
}


type Date struct{
	Object
}

func (d Date) Apply(ident string, args ...Value) (Value, error) {
	return nil, nil
}

type Math struct{
	Object
}

func (m Math) Apply(ident string, args ...Value) (Value, error) {
	return nil, nil
}

type Console struct{
	Object
}

func (c Console) Apply(ident string, args ...Value) (Value, error) {
	return nil, nil
}

func (c Console) Log(args ...Value) {
	fmt.Fprintln(os.Stdout)
}

func (c Console) Error(args ...Value) {
	fmt.Fprintln(os.Stderr)
}
