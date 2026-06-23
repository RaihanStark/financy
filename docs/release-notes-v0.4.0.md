# 💰 Financy v0.4.0 — Recurring, Reports & faster cleanup

This release is about **doing more with the data you already have**: automate the
entries that repeat, read the three classic financial statements, export back out
to CSV, and tidy categories in bulk. Still the same local-first, double-entry
Financy in a single file you own.

## ✨ Highlights

**🔁 Recurring transactions**
- A new **Recurring** screen for templates — rent, salary, subscriptions — each with
  a frequency (Weekly / Biweekly / Monthly / Quarterly / Yearly) and a next-due date.
- When entries come due, Financy **prompts you** (on open, or via **Review & Post**,
  where due templates are highlighted) — it never posts silently.
- For each occurrence you pick, from a dropdown, to **post a new transaction** or
  **link it to the transaction that already paid it**. Financy pre-selects a close
  match so you can't double-post; the amount needn't match exactly, and a
  *Browse all transactions…* picker lets you link anything in the account's history.
- Missed periods are caught up, and a row's ⋮ menu offers **Post now (early)**.

**📄 Reports — the three financial statements**
- A new **Reports** screen with three tabs: an **Income Statement** (P&L), a
  **Balance Sheet** (assets, liabilities & equity that always satisfies
  *Assets = Liabilities + Equity*), and a **Cash Flow** (per-account opening/closing/
  change).
- Shares the Analytics period selector (This month / Last 3·6·12 months / YTD).
- Everything is **read-only and derived** from the journal, so the statements can
  never drift from your transactions.

**🏷️ Bulk recategorize**
- The Transactions screen has a **Select** mode: tick any number of rows (or
  **Select all shown** to grab everything matching the current filters), then
  **Recategorize…** to move them all to one category at once.
- Transfers, opening balances, and mismatched entries are left untouched, and it
  reports how many actually changed — a fast way to clean up after a CSV import.

**📤 CSV export** (`File ▸ Export CSV…`)
- Write every transaction to a flat, spreadsheet-friendly CSV
  (`Date, Payee, Category, Account, Amount, Memo`). It round-trips with the importer,
  so you can export, edit in a spreadsheet, and re-import.

## 🐞 Fixes
- Button hover no longer washes out — primary buttons darken slightly instead of
  turning flat gray, and secondary buttons show a visible hover.
- Hover tooltips (toolbar icons, analytics charts) now appear at the cursor instead
  of offset too far below it.
- The **Setup Wizard** and **File ▸ New** dialogs now have a Close/Cancel button.

## ⬇️ Install

- 🐧 **Linux** — `Financy-v0.4.0-linux-amd64.tar.xz` → `tar xf …` and run.
- 🪟 **Windows** — `Financy-v0.4.0-windows-amd64.exe` (SmartScreen → *More info → Run anyway*).
- 🍎 **macOS** — `Financy-v0.4.0-macos.zip` → move to Applications, first launch *right-click → Open*.

> Builds are unsigned, so Windows/macOS may ask you to confirm before the first run.

## ⬆️ Upgrading from v0.3.0

Just open your existing `.financy` file. This release adds a schema migration for
recurring templates, so Financy upgrades the file automatically on open — writing a
`.bak` alongside it first, as always.

## ⚠️ Known limitations

- One currency per document (no multi-currency / FX within a file yet).
- Cash Flow is a straight cash-movement view, not a GAAP operating/investing/
  financing breakdown (Financy doesn't classify transactions).
- CSV import/export only (no OFX/QIF yet); no automatic transfer detection.
- Don't open the same `.financy` file from two machines at once.

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.3.0...v0.4.0

Built with [Go](https://go.dev) + [Fyne](https://fyne.io) + [SQLite](https://pkg.go.dev/modernc.org/sqlite).
Licensed under the MIT License.
