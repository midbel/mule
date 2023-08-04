package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/midbel/mule"
)

func main() {
	file := flag.String("f", "sample.mu", "")
	flag.Parse()

	c, err := mule.Open(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	switch flag.Arg(0) {
	case "help":
	default:
		err = c.Execute(flag.Arg(0), os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}