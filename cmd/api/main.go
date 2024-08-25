package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
)

//go:embed resources/*
var resources embed.FS

func main() {
	flag.Parse()

	attach := []func() error {
		geo,
		animals,
	}
	for _, fn := range attach {
		if err := fn(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	if err := http.ListenAndServe(flag.Arg(0), nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func geo() error {
	data, err := load("resources/countries.json")
	if err != nil {
		return err
	}
	if ds, ok := data.(map[string]interface{}); ok {
		http.Handle("/continents/", handle(ds["continents"]))
		http.Handle("/countries/", handle(ds["countries"]))
	} else {
		return fmt.Errorf("geo: unexpected data")
	}
	return nil
}

func animals() error {
	data, err := load("resources/animals.json")
	if err != nil {
		return err
	}
	http.Handle("/animals/", handle(data))
	return nil
}

func handle(data interface{}) http.Handler {
	fn := func(res http.ResponseWriter, req *http.Request) {
		json.NewEncoder(res).Encode(data)
	}
	return http.HandlerFunc(fn)
}

func load(file string) (interface{}, error) {
	blob, err := fs.ReadFile(resources, file)
	if err != nil {
		return nil, err
	}
	var data interface{}
	return data, json.Unmarshal(blob, &data)
}