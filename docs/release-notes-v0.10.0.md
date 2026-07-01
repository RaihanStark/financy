## Financy v0.10.0 — Faster entry & a friendlier budget ✍️

This release sharpens everyday use on **mobile** — bulk transaction entry, a
calendar picker for every date, and accounts grouped by type — and makes the
**budget** safer and the **demo** far more realistic on both platforms.

### 📱 Mobile

- **Bulk transaction entry** — a new **Bulk add** screen lets you enter many
  transactions in one pass: a scrollable list of editable rows, each its own
  type, amount, account, category (or transfer), date, payee and memo. Add as
  many rows as you like and commit them all in a single batch. It's reached from
  a **Material-style speed dial** that expands from the Transactions ＋ button
  (Add / Bulk add).
- **Calendar date picker everywhere** — every date field on mobile (add/edit
  transaction, reconcile as-of, and a debt's purchase / first-due dates) now
  opens a tappable month calendar instead of making you type `YYYY-MM-DD`.
- **Accounts split by type** — the Home screen now lists **Assets** and
  **Liabilities** as separate groups, each with its own count and color-coded
  subtotal, instead of one flat mixed list (matching the desktop Accounts
  screen).

### 💰 Budget

- **Auto-Assign previews before overwriting** — on desktop and mobile, running
  Auto-Assign now shows a confirmation popup listing every category that will
  change (current amount → new amount) so existing assignments are never
  silently overwritten. Nothing is applied until you confirm.

### ✨ Demo data

- **Every month is now funded** — the seeded demo budgets all of its months, not
  just the current one, so past months show a real funded budget with sinking
  funds visibly accumulating month over month, instead of reading "Assigned 0"
  against real spending.

### 📥 Install

**Android:** download `Financy-v0.10.0-android-arm64.apk` below and sideload it
(you'll need to allow installs from your browser/file manager).

**Desktop:** grab the package for your OS below (`.deb` / `.rpm` / `.tar.xz` /
`.exe` / macOS `.zip`).

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.9.0...v0.10.0
