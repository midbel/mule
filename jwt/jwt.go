package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrSign   = errors.New("invalid signature")
	ErrFormed = errors.New("malformed")
)

const (
	JWT   = "JWT"
	HS256 = "HS256"
	HS384 = "HS384"
	HS512 = "HS512"
	NONE  = "none"
)

type Claims struct {
	Id        string    `json:"jti,omitempty"`
	Issuer    string    `json:"iss,omitempty"`
	Audience  string    `json:"aud,omitempty"`
	Subject   string    `json:"sub,omitempty"`
	Expires   time.Time `json:"exp,omitempty"`
	NotBefore time.Time `json:"nbf,omitempty"`
	IssueAt   time.Time `json:"iat,omitempty"`
}

type Config struct {
	Claims
	Alg    string
	Secret string
	Ttl    time.Duration
}

func (c Config) getSigner() (Signer, error) {
	var (
		sign   Signer
		secret = []byte(c.Secret)
	)
	switch c.Alg {
	default:
		return nil, fmt.Errorf("%s: unsupported algorithm", c.Alg)
	case HS256:
		sign = hmac.New(sha256.New, secret)
	case HS384:
		sign = hmac.New(sha512.New384, secret)
	case HS512:
		sign = hmac.New(sha512.New, secret)
	case NONE:
		sign = none{}
	}
	return sign, nil
}

func Decode(token string, config *Config) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ErrFormed
	}
	signer, err := config.getSigner()
	if err != nil {
		return err
	}
	check := signer.Sum(parts[0] + "." + parts[1])
	if sign, err := std.DecodeString(parts[2]); err != nil || !bytes.Equal(sign, check) {
		return ErrSign
	}
	return nil
}

func Encode(payload any, config *Config) (string, error) {
	signer, err := config.getSigner()
	if err != nil {
		return "", err
	}
	var (
		hdr, _ = encodeHeader(config.Alg)
		body   = marshalPart(payload)
		token  = hdr + "." + body
		sign   = signer.Sum([]byte(token))
	)
	return token + "." + std.EncodeToString(sign), nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func decodeHeader(str string) (jwtHeader, error) {
	var hdr jwtHeader
	return hdr, unmarshalPart(str, &hdr)
}

func encodeHeader(alg string) (string, error) {
	hdr := jwtHeader{
		Alg: alg,
		Typ: JWT,
	}
	return marshalPart(hdr), nil
}

var std = base64.URLEncoding.WithPadding(base64.NoPadding)

func marshalPart(v any) string {
	buf, _ := json.Marshal(v)
	return std.EncodeToString(buf)
}

func unmarshalPart(s string, v interface{}) error {
	bs, err := std.DecodeString(s)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, v)
}

type Signer interface {
	Sum([]byte) []byte
}

type none struct{}

func (n none) Sum(_ []byte) []byte {
	return nil
}

type mac struct {
	Signer
}
