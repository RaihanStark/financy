## Financy v0.12.0 — Debt payments & Back-button polish 📱

A focused mobile release: paying a debt now works exactly like the desktop, and
the Android Back button behaves the way you'd expect around popups.

### 💳 Link a transaction when paying a debt

Paying a debt installment on mobile now mirrors the desktop. Instead of always
posting a fresh payment, a **"record this payment as"** dropdown lets you:

- **➕ Create a new transaction (today)**, or
- **🔗 Link an existing transaction** already in your books (paid manually,
  imported, or a slightly different amount) — a confident match is pre-selected
  so an already-recorded installment links in one tap, and
- **🔍 Browse all transactions…** opens a searchable picker of every transaction
  on the pay-from account.

Linking re-points that transaction onto the debt's liability, so the outstanding
balance stays correct instead of double-posting.

### 🔙 Back-button behaviour

- With a **dropdown, dialog, or the calendar picker open**, the Android Back
  button now dismisses just that popup instead of navigating the page away.

### 📥 Install

**Android:** download `Financy-v0.12.0-android-arm64.apk` below and sideload it
(you'll need to allow installs from your browser/file manager).

**Desktop:** grab the package for your OS below (`.deb` / `.rpm` / `.tar.xz` /
`.exe` / macOS `.zip`).

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.11.0...v0.12.0
