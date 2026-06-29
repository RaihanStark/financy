---
title: Your Data
weight: 2
---

Financy is **local-first**: there's no account, no cloud, no telemetry. Your data
is a file on your disk that you fully control.

## The `.financy` file

Each document is a single **SQLite database** with a `.financy` extension. Move it,
copy it, back it up, or keep it in a synced folder (while the app is closed). It's
a normal file.

## Saving

How a document saves depends on whether it's encrypted:

- **Unencrypted files** have **no Save button**. Every change is written through to
  the file immediately with ACID guarantees, so a crash or power loss can't leave
  you with a half-written document.
- **Encrypted files** save **manually** — press **Save** (`Cmd/Ctrl-S`) or **File ▸
  Save**. A document with unsaved changes shows a `•` in the title bar and prompts
  you to *Save / Discard / Cancel* when you close it or open another file. This is
  the deliberate trade-off that keeps your decrypted data in memory and never writes
  it to disk in plaintext.

{{< callout type="warning" >}}
SQLite isn't designed for two machines writing the same file at once. Don't open
the same `.financy` from multiple devices simultaneously.
{{< /callout >}}

## Password encryption

You can protect a document with a passphrase. An encrypted `.financy` file is sealed
with **XChaCha20-Poly1305** (authenticated, so tampering is detected) under a key
derived from your passphrase with **Argon2id** (a memory-hard function that resists
brute-force attacks). The file on disk is therefore unreadable — and unmodifiable —
without your password.

- **Create encrypted** — set a password in the first-run **Setup Wizard** or **File ▸
  New…** (leave it blank for an unencrypted file).
- **Set / Change Password…** — encrypt an existing document, or re-key one that's
  already encrypted.
- **Remove Password…** — turn an encrypted document back into a plain file.

While an encrypted document is open, its data lives only in memory; it is written to
disk only when you **Save**, and only ever in encrypted form.

{{< callout type="warning" >}}
**There is no backdoor.** If you forget the passphrase, the file **cannot** be
recovered — not by you, and not by the Financy authors. Store it somewhere safe.
{{< /callout >}}

## Files menu

- **New…** — create a fresh document (asks for currency and an optional password).
- **Open…** / **Open Recent** — open existing files; the last file reopens on launch.
  Encrypted files prompt for the passphrase.
- **Save** — write an encrypted document to disk (`Cmd/Ctrl-S`; not needed for
  unencrypted files, which auto-save).
- **Set / Change Password…** · **Remove Password…** — manage encryption (see above).
- **Save a Copy…** — write a clean snapshot elsewhere (e.g. a dated backup).
- **Setup Wizard…** — re-run the first-run wizard any time.

## Safety by design

- **Atomic writes** — the file is never left in a partial state.
- **Automatic backup on upgrade** — opening a file made by an older version writes
  a `.bak` copy *before* migrating it to the current schema.
- **Forward-compatibility guard** — Financy refuses to open a file written by a
  *newer* version than you're running (with a clear message), rather than risk
  misreading it. Update the app, then open the file.

## Backups

Because it's just a file: copy the `.financy` somewhere safe, or use **Save a
Copy…** periodically. That's a complete, portable backup of everything.

## Export to CSV

**File ▸ Export CSV…** writes all your transactions to a plain CSV
(`Date, Payee, Category, Account, Amount, Memo`), in date order, with plain decimal
amounts — handy for spreadsheets, taxes, or sharing. The format round-trips with
[CSV import](../../importing/csv/): export, edit in a spreadsheet, and bring it back
in.
