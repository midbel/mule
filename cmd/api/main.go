package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"

	"github.com/midbel/mule/jwt"
)

//go:embed resources/*
var resources embed.FS

func main() {
	flag.Parse()

	attach := []func() error{
		geo,
		animals,
	}
	for _, fn := range attach {
		if err := fn(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	http.Handle("/token/new", createToken())
	http.Handle("/token", readToken())
	http.Handle("/codes/400", handleCode(http.StatusBadRequest))
	http.Handle("/codes/401", handleCode(http.StatusUnauthorized))
	http.Handle("/codes/403", handleCode(http.StatusForbidden))
	http.Handle("/codes/404", handleCode(http.StatusNotFound))
	http.Handle("/codes/500", handleCode(http.StatusInternalServerError))
	http.Handle("/codes/200", handleCode(http.StatusOK))
	http.Handle("/codes/201", handleCode(http.StatusCreated))
	http.Handle("/codes/204", handleCode(http.StatusNoContent))
	if err := http.ListenAndServe(flag.Arg(0), nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

const (
	secret   = "supersecretapikey11!"
	issuer   = "http://fakeapi.org"
	audience = "client"
)

func readToken() http.Handler {
	cfg := jwt.Config{
		Alg:    jwt.HS256,
		Secret: secret,
	}
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		tok := struct {
			Token string `json:"token"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&tok); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := jwt.Decode(tok.Token, &cfg); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	return http.HandlerFunc(fn)
}

func createToken() http.Handler {
	cfg := jwt.Config{
		Alg:    jwt.HS256,
		Secret: secret,
	}
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var (
			dat = make(map[string]any)
			err error
		)
		if err = json.NewDecoder(r.Body).Decode(&dat); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		dat["iss"] = issuer
		dat["aud"] = audience

		token, err := jwt.Encode(dat, &cfg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res := struct {
			Token string `json:"token"`
		}{
			Token: token,
		}
		json.NewEncoder(w).Encode(res)
	}
	return http.HandlerFunc(fn)
}

func handleCode(code int) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
	}
	return http.HandlerFunc(fn)
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
	if ds, ok := data.(map[string]interface{}); ok {
		http.Handle("/reigns/", handle(ds["reigns"]))
		http.Handle("/animals/", handle(ds["animals"]))
		http.Handle("/diets/", handle(ds["diets"]))
		http.Handle("/types/", handle(ds["types"]))
	} else {
		return fmt.Errorf("animals: unexpected data")
	}
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
