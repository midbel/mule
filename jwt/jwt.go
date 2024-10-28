package jwt

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"reflect"
	"strings"
	"time"
)

var (
	ErrSign      = errors.New("invalid signature")
	ErrMalformed = errors.New("malformed")
)

const (
	JWT   = "JWT"
	HS256 = "HS256"
	HS384 = "HS384"
	HS512 = "HS512"
	RS256 = "RS256"
	RS384 = "RS384"
	RS512 = "RS512"
	NONE  = "none"
)

type StdClaims struct {
	Id        string
	Issuer    string
	Audience  []string
	Subject   string
	Expires   time.Time
	NotBefore time.Time
	IssueAt   time.Time
}

func (c StdClaims) MarshalJSON() ([]byte, error) {
	claims := make(map[string]any)

	addStrClaim := func(id, value string) {
		if value == "" {
			return
		}
		claims[id] = value
	}

	addTimeClaim := func(id string, value time.Time) {
		if value.IsZero() {
			return
		}
		claims[id] = value.Unix()
	}
	addStrClaim("id", c.Id)
	addStrClaim("iss", c.Issuer)
	if len(c.Audience) > 0 {
		claims["aud"] = c.Audience
	}
	addStrClaim("sub", c.Subject)
	addTimeClaim("exp", c.Expires)
	addTimeClaim("nbf", c.NotBefore)
	addTimeClaim("iat", c.IssueAt)

	return json.Marshal(claims)
}

type Config struct {
	Claims StdClaims
	Alg    string
	Secret string
	Ttl    time.Duration
}

func (c Config) getSigner() (Signer, error) {
	return getSigner(c.Alg, c.Secret)
}

func getSigner(alg, secret string) (Signer, error) {
	var sign hash.Hash
	switch alg {
	default:
		return nil, fmt.Errorf("%s: unsupported algorithm", alg)
	case HS256:
		sign = hmac.New(sha256.New, []byte(secret))
	case HS384:
		sign = hmac.New(sha512.New384, []byte(secret))
	case HS512:
		sign = hmac.New(sha512.New, []byte(secret))
	case NONE:
		return none{}, nil
	}
	mc := mac{
		Signer: sign,
	}
	return mc, nil
}

func Decode(token string, config *Config) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrMalformed
	}
	signer, err := getSignerFromHeader(parts[0], config.Secret)
	if err != nil {
		return nil, err
	}
	var (
		body  = parts[0] + "." + parts[1]
		check = signer.Sum([]byte(body))
	)
	if sign, err := std.DecodeString(parts[2]); err != nil || !bytes.Equal(sign, check) {
		return nil, ErrSign
	}
	payload := make(map[string]any)
	return payload, unmarshalPart(parts[1], &payload)
}

func Encode(payload any, config *Config) (string, error) {
	signer, err := config.getSigner()
	if err != nil {
		return "", err
	}
	if payload, err = prepare(config.Claims, payload); err != nil {
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

func getSignerFromHeader(hdr, secret string) (Signer, error) {
	jose, err := decodeHeader(hdr)
	if err != nil {
		return nil, err
	}
	return getSigner(jose.Alg, secret)
}

func decodeHeader(str string) (jwtHeader, error) {
	var hdr jwtHeader
	if err := unmarshalPart(str, &hdr); err != nil {
		return hdr, err
	}
	if hdr.Typ != JWT {
		return hdr, ErrMalformed
	}
	return hdr, nil
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
	Signer hash.Hash
}

func (m mac) Sum(msg []byte) []byte {
	defer m.Signer.Reset()
	m.Signer.Write(msg)
	return m.Signer.Sum(nil)
}

func prepare(claims StdClaims, payload any) (map[string]any, error) {
	body := make(map[string]any)

	addStrClaim := func(id, value string) {
		if value == "" {
			return
		}
		body[id] = value
	}

	addTimeClaim := func(id string, value time.Time) {
		if value.IsZero() {
			return
		}
		body[id] = value.Unix()
	}
	addStrClaim("id", claims.Id)
	addStrClaim("iss", claims.Issuer)
	if len(claims.Audience) > 0 {
		body["aud"] = claims.Audience
	}
	addStrClaim("sub", claims.Subject)
	addTimeClaim("exp", claims.Expires)
	addTimeClaim("nbf", claims.NotBefore)
	addTimeClaim("iat", claims.IssueAt)

	v := reflect.ValueOf(payload)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			var (
				f = v.Field(i)
				d = t.Field(i)
				n = d.Name
			)
			if tag, ok := d.Tag.Lookup("jwt"); ok {
				n = tag
			} else if tag, ok := d.Tag.Lookup("json"); ok {
				n = tag
			}
			body[n] = f.Interface()
		}
	case reflect.Map:
		it := v.MapRange()
		for it.Next() {
			body[it.Key().String()] = it.Value().Interface()
		}
	default:
		return nil, fmt.Errorf("invalid payload given")
	}
	return body, nil
}
