package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/mule/play"
)

func main() {
	flag.Parse()
	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer r.Close()

	if err := scanFile(r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scanFile(r io.Reader) error {
	scan := play.Scan(r)
	for {
		tok := scan.Scan()
		fmt.Println(tok)
		if tok.Type == play.EOF || tok.Type == play.Invalid {
			break
		}
	}
	return nil
}
