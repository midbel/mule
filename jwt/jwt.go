package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"
)

const (
	JWT   = "JWT"
	HS256 = "HS256"
)

type Option struct {
	Id        string    `json:"jti"`
	Issuer    string    `json:"iss"`
	Audience  string    `json:"aud"`
	Subject   string    `json:"sub"`
	Expires   time.Time `json:"exp"`
	NotBefore time.Time `json:"nbf"`
	IssueAt   time.Time `json:"iat"`

	Secret string
}

func Encode(payload interface{}) (string, error) {
	var (
		hdr, _ = encodeHeader()
		body   = marshalPart(payload)
		token  = hdr + "." + body
		mac    = hmac.New(sha256.New, []byte("supersecret11"))
	)
	mac.Write([]byte(token))
	return token + "." + std.EncodeToString(mac.Sum(nil)), nil
}

func encodeHeader() (string, error) {
	header := struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{
		Alg:  HS256,
		Typ: JWT,
	}
	return marshalPart(header), nil
}

var std = base64.URLEncoding.WithPadding(base64.NoPadding)

func marshalPart(v interface{}) string {
	bs, _ := json.Marshal(v)
	return std.EncodeToString(bs)
}
