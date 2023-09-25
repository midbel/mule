package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
  "encoding/pem"
	"flag"
	"fmt"
	"math/big"
  "path/filepath"
  "net"
	"os"
	"time"
)

var now = time.Now()

func main() {
	var (
		server = flag.String("s", "localhost", "server name")
		org    = flag.String("o", "midbel", "organization name")
		dir    = flag.String("d", "", "certificate directory")
		root   = flag.Bool("r", false, "certificate root")
		bits   = flag.Int("b", 2048, "size of RSA key to generate")
		ttl    = flag.Duration("t", time.Hour*24, "time to life of generated certificate")
	)
	flag.Parse()

	if err := os.MkdirAll(*dir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	priv, err := rsa.GenerateKey(rand.Reader, *bits)
  if err != nil {
    fmt.Fprintln(os.Stderr, "fail to generate key: %s", err)
    os.Exit(2)
  }

	cert := x509.Certificate{
		SerialNumber: getSerialNumber(),
		Subject: pkix.Name{
			Organization: []string{*org},
		},
		NotBefore: now,
		NotAfter:  now.Add(*ttl),

		KeyUsage:              getKeyUsage(*root),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
    IsCA: *root,
	}

  if ip := net.ParseIP(*server); ip == nil {
    cert.DNSNames = append(cert.DNSNames, *server)
  } else {
    cert.IPAddresses = append(cert.IPAddresses, ip)
  }

  if err := writeCertificate(&cert, priv, *dir); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(2)
  }
}

func writeCertificate(cert *x509.Certificate, priv any, dir string) error {
  key, ok := priv.(*rsa.PrivateKey)
  if !ok {
    return fmt.Errorf("unexpected private key type")
  }
  der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, priv)
  if err != nil {
    return err
  }
  if err := writePem(dir, der); err != nil {
    return err
  }
  return nil
}

func writePem(dir string, der []byte) error {
  w, err := os.Create(filepath.Join(dir, "cert.pem"))
  if err != nil {
    return err
  }
  defer w.Close()

  block := pem.Block{
    Type: "CERTIFICATE",
    Bytes: der,
  }
  return pem.Encode(w, &block)
}

func writeKey(dir string, priv any) error {
  w, err := os.Create(filepath.Join(dir, "key.pem"))
  if err != nil {
    return err
  }
  defer w.Close()

  raw, err := x509.MarshalPKCS8PrivateKey(priv)
  if err != nil {
    return err
  }
  block := pem.Block{
    Type: "PRIVATE KEY",
    Bytes: raw,
  }
  return pem.Encode(w, &block)
}

func getSerialNumber() *big.Int {
	var limit big.Int
	serial, _ := rand.Int(rand.Reader, limit.Lsh(big.NewInt(1), 128))
	return serial
}

func getKeyUsage(ca bool) x509.KeyUsage {
	usage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
  if ca {
    usage |= x509.KeyUsageCertSign
  }
  return usage
}
