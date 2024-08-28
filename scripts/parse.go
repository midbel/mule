package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

	root, err := play.ParseReader(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "    ")
	e.Encode(root)
}
