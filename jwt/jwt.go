package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrSign = errors.New("invalid signature")

const (
	JWT   = "JWT"
	HS256 = "HS256"
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
	Secret string
}

func Decode(token string, config *Config) (any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrSign
	}
	if parts[2] == "" {

	}
	return nil, nil
}

func Encode(payload any, config *Config) (string, error) {
	var alg string
	if config == nil || config.Secret == "" {
		config = new(Config)
		alg = NONE
	}
	var (
		hdr, _ = encodeHeader(alg)
		body   = marshalPart(payload)
		token  = hdr + "." + body
		sign   = signPart(token, config.Secret)
	)
	return token + "." + sign, nil
}

func encodeHeader(alg string) (string, error) {
	header := struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{
		Alg: alg,
		Typ: JWT,
	}
	return marshalPart(header), nil
}

var std = base64.URLEncoding.WithPadding(base64.NoPadding)

func marshalPart(v any) string {
	buf, _ := json.Marshal(v)
	return std.EncodeToString(buf)
}

func signPart(token, secret string) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return std.EncodeToString(mac.Sum(nil))
}
