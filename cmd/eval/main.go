package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/mule/eval"
)

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	scan := eval.Scan(r)
	for {
		tok := scan.Scan()
		if tok.Type == eval.EOF {
			break
		}
		fmt.Println(tok)
	}
}
