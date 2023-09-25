package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"encoding/xml"
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
	XMLName xml.Name  `json:"-" xml:"data"`
	Id      int       `json:"id" xml:"id,attr"`
	Label   string    `json:"label" xml:"label"`
	Created time.Time `json:"date_created" xml:"date-created"`
	Updated time.Time `json:"date_updated" xml:"date-updated"`
}

func New(id int, name string) Data {
	return Data{
		Id:      id,
		Label:   name,
		Created: time.Now(),
		Updated: time.Now(),
	}
}

func LoadCertPool(ca string) (*x509.CertPool, error) {
	if ca == "" {
		return x509.SystemCertPool()
	}
	pool := x509.NewCertPool()
	i, err := os.Stat(ca)
	if err != nil {
		return nil, err
	}
	if i.Mode().IsRegular() {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, err
		}
		pool.AppendCertsFromPEM(pem)
		return pool, nil
	}
	if !i.IsDir() {
		return nil, fmt.Errorf("can not read certificates from %s", ca)
	}
	files, err := os.ReadDir(ca)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		pem, err := os.ReadFile(filepath.Join(ca, f.Name()))
		if err != nil {
			return nil, err
		}
		pool.AppendCertsFromPEM(pem)
	}
	return pool, nil
}

func getTLS(server, ca, opt string) (*tls.Config, error) {
	pool, err := LoadCertPool(ca)
	if err != nil {
		return nil, err
	}
	cfg := &tls.Config{
		ServerName: server,
		ClientCAs:  pool,
	}
	switch opt {
	default:
		cfg.ClientAuth = tls.NoClientCert
	case "client-request":
		cfg.ClientAuth = tls.RequestClientCert
	case "client-any":
		cfg.ClientAuth = tls.RequireAnyClientCert
	case "verify-cert":
		cfg.ClientAuth = tls.VerifyClientCertIfGiven
	case "require-verify":
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}

func main() {
	var (
		addr      = flag.String("a", ":9001", "listening address")
		forceAuth = flag.Bool("x", false, "enable authentication")
		certFile  = flag.String("cert-file", "", "certificate file")
		certKey   = flag.String("cert-key", "", "certificate key")
		certCA    = flag.String("cert-ca", "", "certificate ca")
		certOpt   = flag.String("cert-opt", "", "certificate option")
		server    = flag.String("server-name", "localhost", "server name")
	)
	flag.Parse()

	config, err := getTLS(*server, *certCA, *certOpt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	serv := http.Server{
		Addr:      *addr,
		TLSConfig: config,
	}

	set := []struct {
		Route string
		Data  string
	}{
		{Route: "/animals/", Data: "animals.txt"},
		{Route: "/cars/", Data: "cars.txt"},
		{Route: "/colors/", Data: "colors.txt"},
		{Route: "/companies/", Data: "companies.txt"},
		{Route: "/months/", Data: "months.txt"},
		{Route: "/males/", Data: "males.txt"},
		{Route: "/females/", Data: "females.txt"},
	}
	for _, s := range set {
		h, err := Prepare(s.Data)
		if *forceAuth {
			h = WithAuth(h)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		http.Handle(s.Route, h)
	}
	if *certFile != "" && *certKey != "" {
		err = serv.ListenAndServeTLS(*certFile, *certKey)
	} else {
		err = serv.ListenAndServe()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
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

type Encoder interface {
	Encode(interface{}) error
}

type authHandler struct {
	http.Handler
}

func WithAuth(h http.Handler) http.Handler {
	return authHandler{
		Handler: h,
	}
}

func (h authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("authorization")
	if auth == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	h.Handler.ServeHTTP(w, r)
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
	data, err := h.getData(r)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusBadRequest)
	case len(data) == 0:
		w.WriteHeader(http.StatusNoContent)
	default:
	}
	enc, err := h.getEncoder(w, r)
	if err != nil {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	enc.Encode(data)
}

func (h handler) getEncoder(w http.ResponseWriter, r *http.Request) (Encoder, error) {
	accept := r.Header.Values("accept")
	if len(accept) == 0 {
		return json.NewEncoder(w), nil
	}
	var list []WeightString
	for _, a := range accept {
		list = append(list, Weighted(a))
	}
	slices.SortFunc(list, func(i, j WeightString) int {
		return j.Weight - i.Weight
	})
	for _, str := range list {
		switch str.Value {
		case "application/json":
			w.Header().Set("content-type", str.Value)
			return json.NewEncoder(w), nil
		case "text/xml":
			w.Header().Set("content-type", str.Value)
			return xml.NewEncoder(w), nil
		default:
		}
	}
	return nil, fmt.Errorf("not acceptable")
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

func (h handler) getData(r *http.Request) ([]Data, error) {
	var (
		lim    Limit
		err    error
		size   = len(h.Data)
		q      = r.URL.Query()
		offset = q.Get("offset")
		count  = q.Get("count")
	)
	if lim.Offset, err = strconv.Atoi(offset); err != nil && offset != "" {
		return nil, err
	}
	if lim.Count, err = strconv.Atoi(count); err != nil && offset != "" {
		return nil, err
	}

	if lim.Offset < 0 {
		lim.Offset = size + lim.Offset
	}
	if lim.Offset < 0 || lim.Offset >= size {
		return nil, fmt.Errorf("invalid offset")
	}
	if lim.Offset+lim.Count >= size {
		lim.Count = size - lim.Offset
	} else if lim.Count == 0 {
		lim.Count = size
	}
	return h.Data[lim.Offset : lim.Offset+lim.Count], nil
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
		return strings.Compare(i.Label, j.Label)
	})
	return list, scan.Err()
}

type WeightString struct {
	Value  string
	Weight int
}

const prefix = ";q="

func Weighted(str string) WeightString {
	var (
		q = 100
		x = strings.Index(str, prefix)
	)
	if x > 0 {
		tmp, _ := strconv.ParseFloat(str[x+len(prefix):], 64)
		q = int(tmp * 100)
		str = str[:x]
	}
	return WeightString{
		Value:  str,
		Weight: q,
	}
}
