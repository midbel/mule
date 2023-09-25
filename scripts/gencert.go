package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

var now = time.Now()

func main() {
	var (
		client  = flag.Bool("c", false, "generate a client certificate")
		server  = flag.String("s", "localhost", "server name")
		subject = flag.String("subject", "", "certificate subject")
		issuer  = flag.String("issuer", "", "certificate issuer")
		dir     = flag.String("d", "", "certificate directory")
		root    = flag.Bool("r", false, "certificate root")
		bits    = flag.Int("b", 2048, "size of RSA key to generate")
		ttl     = flag.Duration("t", time.Hour*24, "time to life of generated certificate")
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
		NotBefore:    now,
		NotAfter:     now.Add(*ttl),

		KeyUsage:              getKeyUsage(*client, *root),
		ExtKeyUsage:           []x509.ExtKeyUsage{getExtKeyUsage(*client)},
		BasicConstraintsValid: true,
		IsCA:                  !*client && *root,
	}

	if *subject != "" {
		cert.Subject = pkix.Name{
			Organization: []string{*subject},
		}
	}
	if *issuer != "" {
		cert.Issuer = pkix.Name{
			Organization: []string{*issuer},
		}
	}

	if !*client {
		if ip := net.ParseIP(*server); ip == nil {
			cert.DNSNames = append(cert.DNSNames, *server)
		} else {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		}
	}
	var parent *x509.Certificate
	if *client {
		parent, err = loadParentCertificate(flag.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	if err := writeCertificate(&cert, parent, priv, *dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func loadParentCertificate(dir string) (*x509.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem"))
	if err != nil {
		return nil, err
	}
	return cert.Leaf, nil
}

func writeCertificate(cert, root *x509.Certificate, priv any, dir string) error {
	key, ok := priv.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("unexpected private key type")
	}
	if root == nil {
		root = cert
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, root, &key.PublicKey, priv)
	if err != nil {
		return err
	}
	if err := writePem(dir, der); err != nil {
		return err
	}
	return writeKey(dir, priv)
}

func writePem(dir string, der []byte) error {
	w, err := os.Create(filepath.Join(dir, "cert.pem"))
	if err != nil {
		return err
	}
	defer w.Close()

	block := pem.Block{
		Type:  "CERTIFICATE",
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
		Type:  "PRIVATE KEY",
		Bytes: raw,
	}
	return pem.Encode(w, &block)
}

func getSerialNumber() *big.Int {
	var limit big.Int
	serial, _ := rand.Int(rand.Reader, limit.Lsh(big.NewInt(1), 128))
	return serial
}

func getExtKeyUsage(client bool) x509.ExtKeyUsage {
	if client {
		return x509.ExtKeyUsageClientAuth
	}
	return x509.ExtKeyUsageServerAuth
}

func getKeyUsage(client, ca bool) x509.KeyUsage {
	usage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if ca && !client {
		usage |= x509.KeyUsageCertSign
	}
	return usage
}
