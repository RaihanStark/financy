package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// newEncDoc creates an encrypted document in a temp dir and returns it + path.
func newEncDoc(t *testing.T, pass string) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.financy")
	s, err := NewEncryptedDocument(path, pass)
	if err != nil {
		t.Fatalf("NewEncryptedDocument: %v", err)
	}
	return s, path
}

func TestEncryptedDocumentOnDisk(t *testing.T) {
	s, path := newEncDoc(t, "hunter2")
	if !s.Encrypted() {
		t.Fatal("new encrypted document reports Encrypted() == false")
	}
	if s.IsDirty() {
		t.Fatal("freshly created document is dirty")
	}
	_ = s.Close()

	// The file on disk must be encrypted, not a plain SQLite database.
	enc, err := IsEncrypted(path)
	if err != nil {
		t.Fatalf("IsEncrypted: %v", err)
	}
	if !enc {
		t.Fatal("encrypted document is not encrypted on disk")
	}
	data, _ := os.ReadFile(path)
	if len(data) >= 6 && string(data[:6]) == "SQLite" {
		t.Fatal("document on disk is plaintext SQLite")
	}
}

func TestEncryptedRoundTrip(t *testing.T) {
	s, path := newEncDoc(t, "s3cret")

	s.AddAccount(Account{ID: "bank", Name: "Bank", Type: Asset})
	if !s.AddTransaction(Transaction{Date: TodaySerial, Payee: "Open",
		Posts: []Posting{P("bank", 1_000), P("opening", -1_000)}}) {
		t.Fatal("AddTransaction failed")
	}
	if !s.IsDirty() {
		t.Fatal("document not marked dirty after mutations")
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if s.IsDirty() {
		t.Fatal("document still dirty after Save")
	}
	_ = s.Close()

	// Reopen with the correct passphrase: data is intact.
	s2, err := OpenStoreEncrypted(path, "s3cret")
	if err != nil {
		t.Fatalf("OpenStoreEncrypted: %v", err)
	}
	if a := s2.AccountByID("bank"); a == nil {
		t.Fatal("account did not survive save/reopen")
	}
	if s2.TotalAssets() != 1_000 {
		t.Fatalf("assets = %d, want 1000", s2.TotalAssets())
	}
	_ = s2.Close()

	// Wrong passphrase fails.
	if _, err := OpenStoreEncrypted(path, "nope"); !errors.Is(err, ErrBadPassphrase) {
		t.Fatalf("wrong passphrase: got %v, want ErrBadPassphrase", err)
	}
}

// Manual-save semantics: unsaved changes are NOT persisted.
func TestEncryptedManualSave(t *testing.T) {
	s, path := newEncDoc(t, "pw")
	s.AddAccount(Account{ID: "bank", Name: "Bank", Type: Asset})
	// Deliberately do NOT save.
	_ = s.Close()

	s2, err := OpenStoreEncrypted(path, "pw")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s2.Close() }()
	if s2.AccountByID("bank") != nil {
		t.Fatal("unsaved account was persisted — manual save not honored")
	}
}

func TestSetAndRemovePassword(t *testing.T) {
	// Start from a plain document.
	path := filepath.Join(t.TempDir(), "plain.financy")
	s, err := NewDocument(path)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	s.AddAccount(Account{ID: "bank", Name: "Bank", Type: Asset})
	if s.Encrypted() {
		t.Fatal("plain document reports Encrypted() == true")
	}

	// Encrypt it.
	if err := s.SetPassword("pw"); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	if !s.Encrypted() {
		t.Fatal("SetPassword did not switch to encrypted")
	}
	_ = s.Close()
	if enc, _ := IsEncrypted(path); !enc {
		t.Fatal("file not encrypted after SetPassword")
	}

	// Reopen, then remove the password.
	s2, err := OpenStoreEncrypted(path, "pw")
	if err != nil {
		t.Fatalf("reopen encrypted: %v", err)
	}
	if s2.AccountByID("bank") == nil {
		t.Fatal("account lost across SetPassword")
	}
	if err := s2.RemovePassword(); err != nil {
		t.Fatalf("RemovePassword: %v", err)
	}
	if s2.Encrypted() {
		t.Fatal("still encrypted after RemovePassword")
	}
	_ = s2.Close()
	if enc, _ := IsEncrypted(path); enc {
		t.Fatal("file still encrypted after RemovePassword")
	}

	// Now openable as a plain document again.
	s3, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore after RemovePassword: %v", err)
	}
	defer func() { _ = s3.Close() }()
	if s3.AccountByID("bank") == nil {
		t.Fatal("account lost across RemovePassword")
	}
}
