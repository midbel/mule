package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/midbel/mule/codecs/json"
)

func main() {
	flag.Parse()
	doc, err := loadDocument(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	doc, err = queryDocument(flag.Arg(1), doc)
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
	} else {
		ws := json.NewWriter(os.Stdout)
		ws.Write(doc)
	}
}

func queryDocument(query string, doc any) (any, error) {
	q, err := json.Compile(query)
	if err != nil {
		return nil, err
	}
	return q.Get(doc)
}

func loadDocument(file string) (any, error) {
	u, err := url.Parse(file)
	if err != nil {
		return nil, err
	}
	var r io.Reader
	switch u.Scheme {
	case "http", "https":
		return nil, nil
	case "", "file":
		r, err = os.Open(file)
	default:
		return nil, fmt.Errorf("can not get document from file")
	}
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return json.Parse(r)
}
