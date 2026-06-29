package core

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Encrypted documents.
//
// A plain .financy file is a live SQLite database written through on every
// mutation. An encrypted document can't work that way without leaving plaintext
// on disk, so it uses a different model:
//
//   - The working database lives in memory (":memory:"); reads and write-through
//     mutations behave exactly as for a plain file.
//   - Saving is explicit. Save() serializes the in-memory database (via SQLite's
//     VACUUM INTO to a short-lived 0600 temp file), encrypts the bytes with the
//     user's passphrase and atomically replaces the document on disk.
//   - The on-disk file is therefore always encrypted; plaintext exists only in
//     RAM and, transiently, in a private temp file during open/save.
//
// The UI surfaces this as a manual Save command plus an "unsaved changes"
// prompt when closing a dirty encrypted document.

// IsEncrypted reports whether the file at path is a password-protected Financy
// document (as opposed to a plain SQLite file).
func IsEncrypted(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, len(cryptoMagic))
	n, _ := io.ReadFull(f, buf)
	return isEncrypted(buf[:n]), nil
}

// OpenStoreEncrypted decrypts and opens a password-protected document into an
// in-memory store. It returns ErrBadPassphrase when the passphrase is wrong (or
// the file is corrupt/tampered) and ErrFileTooNew when the file's schema is
// newer than this build understands.
func OpenStoreEncrypted(path, passphrase string) (*Store, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := decryptBytes(blob, passphrase)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "financy-open-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	tmpPath := filepath.Join(tmpDir, "doc.db")
	if err := os.WriteFile(tmpPath, plain, 0o600); err != nil {
		return nil, err
	}

	// Pre-migration backup: if this file predates our schema it will be migrated
	// on open, so keep a copy of the original (still encrypted) bytes first —
	// mirroring the plain-file behavior in the UI.
	oldVersion, _ := dbVersion(tmpPath)
	if oldVersion > 0 && oldVersion < schemaVersion() {
		_ = os.WriteFile(path+BackupSuffix(oldVersion), blob, 0o600)
	}

	fileDB, err := openDB(tmpPath) // migrates the temp copy in place
	if err != nil {
		return nil, err
	}
	s, err := loadStore(fileDB)
	_ = fileDB.Close()
	if err != nil {
		return nil, err
	}

	// Move the working database into memory so no plaintext database persists on
	// disk for the duration of the session.
	if err := s.adoptInMemory(); err != nil {
		return nil, err
	}
	s.path = path
	s.encrypted = true
	s.passphrase = passphrase
	s.dirty = false

	// If the file was migrated to a newer schema, persist the upgrade so the
	// encrypted file on disk matches what we just loaded.
	if oldVersion < schemaVersion() {
		if err := s.Save(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// NewEncryptedDocument creates a fresh password-protected document seeded with
// the default chart of accounts and writes the initial encrypted file.
func NewEncryptedDocument(path, passphrase string) (*Store, error) {
	db, err := openDB(":memory:")
	if err != nil {
		return nil, err
	}
	seed := &Store{db: db, currency: "Rp", year: settingsYear, assignments: map[string]int{}}
	for _, a := range seedAccounts() {
		if err := seed.dbUpsertAccount(a); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	if err := seed.dbSetSettings(); err != nil {
		_ = db.Close()
		return nil, err
	}
	s, err := loadStore(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	s.path = path
	s.encrypted = true
	s.passphrase = passphrase
	if err := s.Save(); err != nil {
		_ = db.Close()
		return nil, err
	}
	s.dirty = false
	return s, nil
}

// adoptInMemory replaces the store's current database handle with a fresh
// in-memory one populated from the in-memory model, then closes the old handle.
// Used to detach an encrypted document from any on-disk SQLite file.
func (s *Store) adoptInMemory() error {
	mem, err := openDB(":memory:")
	if err != nil {
		return err
	}
	old := s.db
	s.db = mem
	if err := s.writeAll(); err != nil {
		s.db = old
		_ = mem.Close()
		return err
	}
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// writeAll persists the entire in-memory model into s.db. It assumes s.db points
// at a freshly migrated, empty database (e.g. a new ":memory:" handle).
func (s *Store) writeAll() error {
	for _, a := range s.accounts {
		if err := s.dbUpsertAccount(a); err != nil {
			return err
		}
	}
	if len(s.txns) > 0 {
		if err := s.dbInsertTxnBatch(s.txns); err != nil {
			return err
		}
	}
	for _, r := range s.recurring {
		if err := s.dbUpsertRecurring(r); err != nil {
			return err
		}
	}
	for key, amt := range s.assignments {
		month, catID, ok := strings.Cut(key, "|")
		if !ok {
			continue
		}
		if err := s.dbUpsertAssignment(month, catID, amt); err != nil {
			return err
		}
	}
	for _, d := range s.debts {
		if err := s.dbUpsertDebt(d); err != nil {
			return err
		}
	}
	for _, in := range s.installments {
		if err := s.dbUpsertInstallment(in); err != nil {
			return err
		}
	}
	return s.dbSetSettings()
}

// Save serializes an encrypted document and atomically replaces its file on
// disk. For plain (auto-saved) documents it is a no-op so callers can call it
// unconditionally.
func (s *Store) Save() error {
	if s.db == nil || !s.encrypted {
		return nil
	}
	plain, err := s.dumpDB()
	if err != nil {
		return err
	}
	blob, err := encryptBytes(plain, s.passphrase)
	if err != nil {
		return err
	}
	if err := atomicWrite(s.path, blob); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

// dumpDB returns the working database's bytes as a standalone SQLite image. It
// uses VACUUM INTO a private 0600 temp file, reads it back, and removes it.
func (s *Store) dumpDB() ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "financy-save-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	tmpPath := filepath.Join(tmpDir, "doc.db")
	if _, err := s.db.Exec(`VACUUM INTO ?`, tmpPath); err != nil {
		return nil, err
	}
	return os.ReadFile(tmpPath)
}

// SetPassword turns an unencrypted document into an encrypted one (or changes
// the passphrase of an already-encrypted one). The plaintext file on disk is
// replaced with the encrypted document.
func (s *Store) SetPassword(passphrase string) error {
	if passphrase == "" {
		return ErrBadPassphrase
	}
	if s.encrypted {
		return s.ChangePassword(passphrase)
	}
	// Detach from the plaintext file before we overwrite it, so future writes go
	// to memory rather than to disk in the clear.
	if err := s.adoptInMemory(); err != nil {
		return err
	}
	s.encrypted = true
	s.passphrase = passphrase
	if err := s.Save(); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

// ChangePassword re-keys an encrypted document with a new passphrase and saves.
func (s *Store) ChangePassword(passphrase string) error {
	if passphrase == "" {
		return ErrBadPassphrase
	}
	if !s.encrypted {
		return s.SetPassword(passphrase)
	}
	s.passphrase = passphrase
	return s.Save()
}

// RemovePassword turns an encrypted document back into a plain auto-saved
// SQLite file at the same path.
func (s *Store) RemovePassword() error {
	if !s.encrypted {
		return nil
	}
	plain, err := s.dumpDB()
	if err != nil {
		return err
	}
	if err := atomicWrite(s.path, plain); err != nil {
		return err
	}
	old := s.db
	fileDB, err := openDB(s.path)
	if err != nil {
		return err
	}
	s.db = fileDB
	if old != nil {
		_ = old.Close()
	}
	s.encrypted = false
	s.passphrase = ""
	s.dirty = false
	return nil
}

// atomicWrite writes data to a sibling temp file then renames it over path, so a
// failure mid-write can't leave a half-written document.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
