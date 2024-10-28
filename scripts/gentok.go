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

	cfg.Claims = jwt.StdClaims{
		Issuer:   "token.midbel.org",
		Subject:  "demo",
		Audience: []string{"account"},
	}

	dat := struct {
		Issuer string `json:"iss"`
		Name   string `jwt:"name"`
		Level  int    `jwt:"level"`
	}{
		Issuer: "https://localhost/iam/",
		Name:   "foobar",
		Level:  100,
	}
	token, err := jwt.Encode(dat, &cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(token)

	body, err := jwt.Decode(token, &cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", body)
}
