package tokenencrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Encrypter cifra tokens en reposo (AES-GCM).
type Encrypter struct {
	gcm cipher.AEAD
}

// New construye un encrypter. keyMaterial vacío usa fallback (p. ej. JWT secret) vía SHA-256 → 32 bytes.
// Con Gmail OAuth en producción, pasar siempre TOKEN_ENCRYPTION_KEY y usar fallback "" (ver configs.Validate).
func New(keyMaterial, fallback string) (*Encrypter, error) {
	raw := keyMaterial
	if raw == "" {
		raw = fallback
	}
	if raw == "" {
		return nil, errors.New("token encryption: TOKEN_ENCRYPTION_KEY or JWT secret required")
	}
	sum := sha256.Sum256([]byte(raw))
	key := sum[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encrypter{gcm: gcm}, nil
}

// Encrypt devuelve base64(nonce||ciphertext).
func (e *Encrypter) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt invierte Encrypt.
func (e *Encrypter) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	ns := e.gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := data[:ns], data[ns:]
	pt, err := e.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
