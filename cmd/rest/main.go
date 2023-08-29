package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Data struct {
	Id      int       `json:"id"`
	Name    string    `json:"label"`
	Created time.Time `json:"date_created"`
	Updated time.Time `json:"date_updated"`
}

func New(id int, name string) Data {
	return Data{
		Id:      id,
		Name:    name,
		Created: time.Now(),
		Updated: time.Now(),
	}
}

func main() {
	addr := flag.String("a", ":9001", "listening address")
	flag.Parse()

	set := []struct {
		Route string
		Data  string
	}{
		{Route: "/pets/", Data: "animals.txt"},
		{Route: "/cars/", Data: "cars.txt"},
		{Route: "/colors/", Data: "colors.txt"},
		{Route: "/companies/", Data: "companies.txt"},
		{Route: "/months/", Data: "months.txt"},
		{Route: "/males/", Data: "males.txt"},
		{Route: "/females/", Data: "females.txt"},
	}
	for _, s := range set {
		h, err := Prepare(s.Data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		http.Handle(s.Route, h)
	}
	if err := http.ListenAndServe(*addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Prepare(file string) (http.Handler, error) {
	data, err := Load(file)
	if err != nil {
		return nil, err
	}
	h := handler{
		Data: data,
	}
	return h, nil
}

type Limit struct {
	Offset int
	Count  int
}

type handler struct {
	Data []Data
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.setHeadersCORS(w)
	if r.Method == http.MethodOptions {
		return
	}
	limit, err := h.getLimitFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(h.Data[limit.Offset : limit.Offset+limit.Count])
}

func (h handler) setHeadersCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Methods", "GET")
	w.Header().Add("Access-Control-Allow-Methods", "OPTIONS")
	w.Header().Add("Access-Control-Allow-Headers", "Accept")
	w.Header().Add("Access-Control-Allow-Headers", "Accept-Language")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Language")
	w.Header().Add("Access-Control-Allow-Headers", "Origin")
}

func (h handler) getLimitFromRequest(r *http.Request) (Limit, error) {
	var (
		lim    Limit
		err    error
		size   = len(h.Data)
		q      = r.URL.Query()
		offset = q.Get("offset")
		count  = q.Get("count")
	)
	if lim.Offset, err = strconv.Atoi(offset); err != nil && offset != "" {
		return lim, err
	}
	if lim.Count, err = strconv.Atoi(count); err != nil && offset != "" {
		return lim, err
	}

	if lim.Offset < 0 {
		lim.Offset = size + lim.Offset
	}
	if lim.Offset < 0 || lim.Offset >= size {
		return lim, fmt.Errorf("invalid offset")
	}
	if lim.Offset+lim.Count >= size {
		lim.Count = size - lim.Offset
	} else if lim.Count == 0 {
		lim.Count = size
	}
	return lim, nil
}

//go:embed resources/*txt
var datadir embed.FS

func Load(file string) ([]Data, error) {
	f, err := datadir.Open(filepath.Join("resources", file))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		scan = bufio.NewScanner(f)
		list []Data
	)
	for i := 1; scan.Scan(); i++ {
		line := scan.Text()
		if line == "" {
			continue
		}
		list = append(list, New(i, line))
	}
	slices.SortFunc(list, func(i, j Data) int {
		return strings.Compare(i.Name, j.Name)
	})
	return list, scan.Err()
}
