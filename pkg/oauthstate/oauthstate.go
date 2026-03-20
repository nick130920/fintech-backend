package oauthstate

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type payload struct {
	UID uint  `json:"uid"`
	Exp int64 `json:"exp"`
}

// Sign genera state firmado (base64url(payload).base64url(hmac)).
func Sign(userID uint, ttl time.Duration, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("oauth state secret empty")
	}
	p := payload{UID: userID, Exp: time.Now().Add(ttl).Unix()}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	pb := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(pb))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return pb + "." + sig, nil
}

// Verify valida y devuelve userID.
func Verify(state, secret string) (uint, error) {
	if secret == "" || state == "" {
		return 0, errors.New("invalid state")
	}
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return 0, errors.New("invalid state format")
	}
	pb, sig := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(pb))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return 0, errors.New("invalid state signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(pb)
	if err != nil {
		return 0, err
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, err
	}
	if time.Now().Unix() > p.Exp {
		return 0, errors.New("state expired")
	}
	if p.UID == 0 {
		return 0, errors.New("invalid uid in state")
	}
	return p.UID, nil
}
