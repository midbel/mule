package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	addr := flag.String("a", ":9090", "listening address")
	flag.Parse()

	dump := Dump()
	http.Handle("/", dump)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type dumper struct {
	counter atomic.Uint64
	when    time.Time
}

func Dump() *dumper {
	return &dumper{
		when: time.Now(),
	}
}

func (d *dumper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := struct {
		Count   uint64        `json:"id"`
		When    time.Time     `json:"timestamp"`
		Uptime  time.Duration `json:"uptime"`
		Headers http.Header   `json:"headers"`
		Params  url.Values    `json:"params"`
	}{
		Count:   d.counter.Add(1),
		When:    time.Now(),
		Headers: r.Header,
		Params:  r.URL.Query(),
		Uptime:  time.Since(d.when),
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c)
}

func dump(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, false)
	if err != nil {
		http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	os.Stdout.Write(dump)
	fmt.Fprintln(os.Stdout, "==========")
}
