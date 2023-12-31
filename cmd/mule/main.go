package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/midbel/mule"
)

func main() {
	var (
		file   = flag.String("f", "sample.mu", "read request from file")
		print  = flag.Bool("p", false, "print response to stdout")
		listen = flag.Bool("l", false, "listen")
		addr   = flag.String("a", ":9000", "listening address")
	)

	flag.Parse()

	c, err := mule.Open(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *listen {
		err = runListen(c, *addr)
	} else {
		err = runExecute(c, *print)
	}
}

func runListen(c *mule.Collection, addr string) error {
	return http.ListenAndServe(addr, nil)
}

func runExecute(c *mule.Collection, print bool) error {
	var (
		out io.Writer = io.Discard
		err error
	)
	if print {
		out = os.Stdout
	}
	switch flag.Arg(0) {
	case "help":
	default:
		err = c.Run(flag.Arg(0), out)
	}
	return err
}
