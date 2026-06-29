# 💰 Financy v0.8.0 — Password-protected files & smarter debt payments

This release puts your data behind a passphrase and makes paying off debts as
flexible as posting recurring transactions. Payee fields everywhere now
autocomplete from your history, too.

## ✨ Highlights

**🔒 Password encryption**
- A `.financy` file can now be protected with a passphrase. The document is
  encrypted at rest with **XChaCha20-Poly1305** and an **Argon2id**-derived key, so a
  lost or leaked file is unreadable and tamper-evident without your password.
- Set a password when creating a file, or add/change/remove one later via
  **File → Set Password…**.
- Encrypted documents use **manual save** (Cmd/Ctrl-S) and prompt you to save unsaved
  changes before closing; your data is only decrypted in memory while the file is
  open, never persisted to disk in plaintext.
- A forgotten passphrase is **unrecoverable** — there is no backdoor.

**🤝 Match debt payments**
- Paying an installment now opens the same review dialog as recurring transactions:
  post a new payment, or **link an installment to an existing transaction** that
  already paid it (re-pointed onto the debt's liability so the balance stays correct)
  instead of always posting a fresh one.
- Handles installments already paid manually or imported, and amounts that aren't
  exact.

## 🔧 Other changes

- **Payee autocomplete** — every payee field (Add Transaction, Bulk Add, Recurring,
  Debts) is now a searchable dropdown that suggests matching payees from your history
  as you type. Keep typing to filter and pick an existing payee, or enter a brand-new
  one inline.

## 📦 Install

Grab the build for your platform from the assets below.

- **Debian / Ubuntu:** `sudo apt install ./Financy-v0.8.0-linux-amd64.deb`
- **Fedora / RHEL / openSUSE:** `sudo dnf install ./Financy-v0.8.0-linux-x86_64.rpm`
- **Other Linux:** extract `Financy-v0.8.0-linux-amd64.tar.xz` and run the app.
- **Windows:** run the `.exe` (SmartScreen → *More info → Run anyway*; unsigned).
- **macOS:** unzip, move to Applications, first launch **right-click → Open** (unsigned).

The full list of changes is in [CHANGELOG.md](../CHANGELOG.md).
