package eval

import (
	"os"
	"fmt"
)


type Object struct {}

type Date struct {}

type Math struct {}

type Console struct {}

func (c Console) Log(args ...Value) {
	fmt.Fprintln(os.Stdout, args...)
}

func (c Console) Error(args ...Value) {
	fmt.Fprintln(os.Stderr, args...)
}