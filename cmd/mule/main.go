package main

import (
	"flag"
	"fmt"
	"net/http/httputil"
	"os"
	"strings"

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
	err = runCommand(c, *print)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCommand(c *mule.Collection, print bool) error {
	var err error
	switch args := flag.Args(); flag.Arg(0) {
	case "help":
		err = executeHelp(c, args[1:])
	case "all":
	case "debug":
		err = executeDebug(c, args[1:])
	default:
		args := flag.Args()
		err = c.Run(flag.Arg(0), args[1:], os.Stdout, os.Stderr)
	}
	return err
}

func executeDebug(c *mule.Collection, args []string) error {
	var (
		set   = flag.NewFlagSet("debug", flag.ExitOnError)
		color = set.Bool("c", false, "colorize")
	)
	if err := set.Parse(args); err != nil {
		return err
	}
	_ = color

	req, err := c.Get(set.Arg(0))
	if err != nil {
		return err
	}
	if req.Body != nil {
		defer req.Body.Close()
	}
	buf, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(buf))
	return nil
}

func executeHelp(c *mule.Collection, args []string) error {
	printRequests := func(c *mule.Collection) {
		for _, r := range c.Requests {
			fmt.Printf("* [%s] %s", r.Method, r.Name)
			fmt.Println()
		}
	}

	printHelp := func(c *mule.Collection) {
		printRequests(c)
		for _, c := range c.Collections {
			fmt.Println(c.Name)
			fmt.Println(strings.Repeat("-", len(c.Name)))
			printRequests(c)
		}
	}

	set := flag.NewFlagSet("help", flag.ExitOnError)
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() == 0 {
		printHelp(c)
		return nil
	}
	return nil
}
