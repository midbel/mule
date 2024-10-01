package main

import (
	"flag"
	"fmt"

	"github.com/midbel/mule/jwt"
)

func main() {
	var cfg jwt.Config
	flag.StringVar(&cfg.Secret, "s", "supersecret11", "secret")
	flag.StringVar(&cfg.Alg, "a", jwt.HS256, "")
	flag.Parse()

	dat := struct {
		Name  string `json:"name"`
		Level int    `json:"level"`
		jwt.Claims
	}{
		Name:  "foobar",
		Level: 100,
		Claims: jwt.Claims{
			Issuer:   "token.midbel.org",
			Subject:  "demo",
			Audience: "account",
		},
	}
	fmt.Println(jwt.Encode(dat, &cfg))
}
