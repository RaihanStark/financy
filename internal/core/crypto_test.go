package core

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plain := []byte("SQLite format 3\x00 ...pretend database bytes...")
	blob, err := encryptBytes(plain, "correct horse battery staple")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !isEncrypted(blob) {
		t.Fatal("encrypted blob is not recognized as encrypted")
	}
	if bytes.Contains(blob, plain) {
		t.Fatal("plaintext leaked into the encrypted blob")
	}
	got, err := decryptBytes(blob, "correct horse battery staple")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plain)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	blob, err := encryptBytes([]byte("secret"), "right")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := decryptBytes(blob, "wrong"); !errors.Is(err, ErrBadPassphrase) {
		t.Fatalf("wrong passphrase: got %v, want ErrBadPassphrase", err)
	}
}

func TestDecryptTampered(t *testing.T) {
	blob, err := encryptBytes([]byte("secret data here"), "pw")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	// Flip a bit in the last byte (ciphertext) — authentication must fail.
	blob[len(blob)-1] ^= 0x01
	if _, err := decryptBytes(blob, "pw"); !errors.Is(err, ErrBadPassphrase) {
		t.Fatalf("tampered blob: got %v, want ErrBadPassphrase", err)
	}
}

func TestIsEncryptedDetection(t *testing.T) {
	blob, _ := encryptBytes([]byte("x"), "pw")
	cases := []struct {
		name string
		in   []byte
		want bool
	}{
		{"encrypted", blob, true},
		{"plain sqlite", []byte("SQLite format 3\x00rest"), false},
		{"garbage", []byte("not a known file"), false},
		{"empty", nil, false},
	}
	for _, c := range cases {
		if got := isEncrypted(c.in); got != c.want {
			t.Errorf("%s: isEncrypted = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDecryptNotEncrypted(t *testing.T) {
	if _, err := decryptBytes([]byte("SQLite format 3\x00"), "pw"); !errors.Is(err, ErrNotEncrypted) {
		t.Fatalf("got %v, want ErrNotEncrypted", err)
	}
}
