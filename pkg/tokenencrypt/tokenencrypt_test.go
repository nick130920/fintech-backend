package tokenencrypt

import (
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	e, err := New("my-32-byte-ish-secret-key-material", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	const plain = "refresh_token_value_xyz"
	enc, err := e.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if enc == "" || enc == plain {
		t.Fatalf("expected ciphertext, got %q", enc)
	}
	out, err := e.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if out != plain {
		t.Fatalf("want %q got %q", plain, out)
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	e, err := New("", "only-fallback-secret")
	if err != nil {
		t.Fatal(err)
	}
	enc, err := e.Encrypt("")
	if err != nil || enc != "" {
		t.Fatalf("empty encrypt: enc=%q err=%v", enc, err)
	}
	out, err := e.Decrypt("")
	if err != nil || out != "" {
		t.Fatalf("empty decrypt: out=%q err=%v", out, err)
	}
}

func TestNew_ErrWhenNoKey(t *testing.T) {
	_, err := New("", "")
	if err == nil {
		t.Fatal("expected error when no key material and no fallback")
	}
}
