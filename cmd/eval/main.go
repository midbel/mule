package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/mule/eval"
)

func main() {
	scan := flag.Bool("s", false, "scan")
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	defer r.Close()
	switch {
	case *scan:
		err = scanFile(r)
	default:
		err = parseFile(r)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanFile(r io.Reader) error {
	scan := eval.Scan(r)
	for {
		tok := scan.Scan()
		if tok.Type == eval.EOF {
			break
		}
		fmt.Println(tok)
	}
	return nil
}

func parseFile(r io.Reader) error {
	expr, err := eval.Parse(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", expr)
	return nil
}
