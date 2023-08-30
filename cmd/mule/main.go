package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/midbel/mule"
)

func main() {
	var (
		file  = flag.String("f", "sample.mu", "read request from file")
		print = flag.Bool("p", false, "print response to stdout")
	)

	flag.Parse()

	c, err := mule.Open(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var out io.Writer = io.Discard
	if *print {
		out = os.Stdout
	}
	switch flag.Arg(0) {
	case "help":
	default:
		err = c.Run(flag.Arg(0), out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
