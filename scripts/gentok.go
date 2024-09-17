package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/midbel/mule/jwt"
)

func main() {
	flag.Parse()

	r, err := os.Open(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer r.Close()

	var dat interface{}
	if err := json.NewDecoder(r).Decode(&dat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Println(dat)
	fmt.Println(jwt.Encode(dat))
}
