package main

import (
	"flag"
	"fmt"
	"io"
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
	err = runExecute(c, *print)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
		args := flag.Args()
		err = executeHelp(c, args[1:])
	case "all":
	default:
		args := flag.Args()
		err = c.Run(flag.Arg(0), args[1:], out)
	}
	return err
}

func executeHelp(c *mule.Collection, args []string) error {
	printRequests := func(c *mule.Collection) {
		for _, r := range c.GetRequests() {
			fmt.Printf("* [%s] %s", r.Method, r.Name)
			fmt.Println()
		}
	}

	printHelp := func(c *mule.Collection) {
		printRequests(c)
		for _, c := range c.GetCollections() {
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
	for _, n := range set.Args() {
		other, err := c.GetCollection(n)
		if err != nil {
			return err
		}
		printHelp(other)
	}
	return nil
}
