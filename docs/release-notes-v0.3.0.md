# 💰 Financy v0.3.0 — Import, reconcile & docs

This release is about getting **real data in** and **keeping it honest**: a guided
CSV importer, account reconciliation, smarter amount entry, and a full
documentation site. Still the same local-first, double-entry Financy.

## ✨ Highlights

**📥 Guided CSV import** (`File ▸ Import CSV…`, or the Transactions screen)
- Import bank/card exports of almost any shape. A mapping step **auto-detects** the
  date, payee and amount columns and the date format from the header.
- Handles commas/semicolons/tabs/pipes, single signed-amount **or** separate
  debit/credit columns, parentheses negatives, and a sign-flip toggle for cards.
- A **preview** shows every row's date, payee, amount and category before you
  commit, with **per-row category dropdowns**.
- Rows **auto-categorize by payee** — exact, then token/substring matching, so
  `AMAZON.COM*A1B2C` still matches a learned `Amazon`. New payees default to
  Uncategorized.
- Likely **duplicates** are flagged and excluded by default; imports commit
  atomically. Ten example files live in [`sample_data/csv/`](https://github.com/raihanstark/financy/tree/main/sample_data/csv).

**🔄 Account reconciliation**
- Right-click an account → **Reconcile…** (or the **Reconcile** button in its
  register). Enter the real balance from your bank and Financy posts a single
  adjustment for the difference so the account matches.
- The adjustment is an auditable journal entry; net worth updates, but it doesn't
  distort income/expense analytics.

**🔢 Live amount formatting & number formats**
- Money fields group thousands **as you type** (`1,000,000`).
- A new **Number format** setting (Preferences) lets you pick the separator style
  (`1,234.56` / `1.234,56` / `1 234,56` / …) independently of the currency.

**📖 Documentation site**
- A full user guide is now published at
  **[raihanstark.github.io/financy/docs](https://raihanstark.github.io/financy/docs/)**
  — getting started, concepts, the guide, importing, reconciliation, settings, and
  data/backup.

## ⬇️ Install

- 🐧 **Linux** — `Financy-v0.3.0-linux-amd64.tar.xz` → `tar xf …` and run.
- 🪟 **Windows** — `Financy-v0.3.0-windows-amd64.exe` (SmartScreen → *More info → Run anyway*).
- 🍎 **macOS** — `Financy-v0.3.0-macos.zip` → move to Applications, first launch *right-click → Open*.

> Builds are unsigned, so Windows/macOS may ask you to confirm before the first run.

## ⬆️ Upgrading from v0.2.0

Just open your existing `.financy` file — there are no schema changes in this
release, so it opens as-is.

## ⚠️ Known limitations

- One currency per document (no multi-currency / FX within a file yet).
- CSV import only (no OFX/QIF yet); no automatic transfer detection.
- Don't open the same `.financy` file from two machines at once.

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.2.0...v0.3.0

Built with [Go](https://go.dev) + [Fyne](https://fyne.io) + [SQLite](https://pkg.go.dev/modernc.org/sqlite).
Licensed under the MIT License.
