package core

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// Password encryption for .financy documents.
//
// A document is either a plain SQLite file (begins with the SQLite magic
// "SQLite format 3\000") or an encrypted blob written by this package (begins
// with cryptoMagic). The two are mutually distinguishable from their first
// bytes, so opening code can detect encryption without a password.
//
// The encrypted payload is the document's raw SQLite bytes sealed with
// XChaCha20-Poly1305 (authenticated, so tampering is detected) under a key
// derived from the user's passphrase with Argon2id (memory-hard). The KDF
// parameters and the random salt/nonce are stored in the header so the file
// stays self-describing and the parameters can be tuned in future versions.

// cryptoMagic prefixes every encrypted document. Plain SQLite files start with
// "SQLite format 3\000", which shares no prefix with this, so the two are
// unambiguous.
var cryptoMagic = []byte("FINCRYPT")

// cryptoVersion is the on-disk format version of the encrypted container.
const cryptoVersion byte = 1

// Argon2id parameters. These are the defaults used when sealing; the values
// actually used to open a file are read from its header, so raising these later
// does not break existing files.
const (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // 64 MiB
	argonThreads uint8  = 4
	argonKeyLen  uint32 = chacha20poly1305.KeySize
	saltLen             = 16
)

// ErrBadPassphrase is returned when a document can't be decrypted: a wrong
// passphrase, or a corrupted/tampered file (the two are indistinguishable by
// design, since the cipher is authenticated).
var ErrBadPassphrase = errors.New("incorrect passphrase or corrupted file")

// ErrNotEncrypted is returned when decryption is asked of bytes that aren't an
// encrypted Financy document.
var ErrNotEncrypted = errors.New("not an encrypted Financy document")

// isEncrypted reports whether b begins with the encrypted-document magic.
func isEncrypted(b []byte) bool {
	return len(b) >= len(cryptoMagic) && string(b[:len(cryptoMagic)]) == string(cryptoMagic)
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

// deriveKey stretches a passphrase into a cipher key with Argon2id.
func deriveKey(passphrase string, salt []byte, time, memory uint32, threads uint8) []byte {
	return argon2.IDKey([]byte(passphrase), salt, time, memory, uint8(threads), argonKeyLen)
}

// encryptBytes seals plaintext under a passphrase and returns a self-describing
// encrypted document (magic + KDF params + salt + nonce + ciphertext).
func encryptBytes(plaintext []byte, passphrase string) ([]byte, error) {
	salt, err := randomBytes(saltLen)
	if err != nil {
		return nil, err
	}
	key := deriveKey(passphrase, salt, argonTime, argonMemory, argonThreads)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return nil, err
	}

	// Header: magic | version | time(4) | memory(4) | threads(1) | saltLen(1) |
	//         salt | nonceLen(1) | nonce, then the ciphertext.
	var hdr []byte
	hdr = append(hdr, cryptoMagic...)
	hdr = append(hdr, cryptoVersion)
	hdr = binary.LittleEndian.AppendUint32(hdr, argonTime)
	hdr = binary.LittleEndian.AppendUint32(hdr, argonMemory)
	hdr = append(hdr, argonThreads)
	hdr = append(hdr, byte(len(salt)))
	hdr = append(hdr, salt...)
	hdr = append(hdr, byte(len(nonce)))
	hdr = append(hdr, nonce...)

	// Authenticate the header alongside the plaintext so its parameters can't be
	// altered without detection.
	return aead.Seal(hdr, nonce, plaintext, hdr), nil
}

// decryptBytes opens an encrypted document produced by encryptBytes. It returns
// ErrBadPassphrase for a wrong passphrase or any tampering.
func decryptBytes(blob []byte, passphrase string) ([]byte, error) {
	if !isEncrypted(blob) {
		return nil, ErrNotEncrypted
	}
	r := blob[len(cryptoMagic):]
	if len(r) < 1 {
		return nil, ErrBadPassphrase
	}
	version := r[0]
	r = r[1:]
	if version != cryptoVersion {
		return nil, ErrFileTooNew
	}
	if len(r) < 4+4+1+1 {
		return nil, ErrBadPassphrase
	}
	time := binary.LittleEndian.Uint32(r[0:4])
	memory := binary.LittleEndian.Uint32(r[4:8])
	threads := r[8]
	saltL := int(r[9])
	r = r[10:]
	if len(r) < saltL+1 {
		return nil, ErrBadPassphrase
	}
	salt := r[:saltL]
	r = r[saltL:]
	nonceL := int(r[0])
	r = r[1:]
	if len(r) < nonceL {
		return nil, ErrBadPassphrase
	}
	nonce := r[:nonceL]
	ciphertext := r[nonceL:]

	// Reconstruct the authenticated header (everything up to the ciphertext).
	hdr := blob[:len(blob)-len(ciphertext)]

	key := deriveKey(passphrase, salt, time, memory, threads)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, ErrBadPassphrase
	}
	if len(nonce) != aead.NonceSize() {
		return nil, ErrBadPassphrase
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, hdr)
	if err != nil {
		return nil, ErrBadPassphrase
	}
	return plaintext, nil
}
