# 💰 Financy v0.2.0 — Analytics & polish

This release adds an **Analytics dashboard**, makes the demo data feel like a real file,
and smooths a handful of everyday rough edges. Still the same local-first, double-entry
Financy — your data lives in a single `.financy` file you own.

## ✨ Highlights

**📊 Analytics (new)**
- A read-only insights screen with **KPI cards** — net worth (and its change over the
  period), income, expenses, net saved, and **savings rate**.
- **Income vs Expenses** — grouped monthly bars.
- **Net worth over time** — a trend line.
- **Spending by category** — ranked, with each category's share.
- Pick the window: **This month / Last 3 / 6 / 12 months / Year to date**.
- Charts are drawn **natively** (no new dependencies), with **Y-axis gridlines + value
  labels** and **hover tooltips** showing the exact figures for each month.

**🧪 Richer demo data**
- "Load demo data" now seeds **~6 months (~90 transactions)** of recurring income, bills,
  groceries, dining, transfers and credit-card activity — so balances, registers and the
  new analytics all look like a real, lived-in file.

## 🛠️ Improvements & fixes
- 🗓️ **Calendar date picker** on the transaction date field — click to pick a day instead
  of typing (it still accepts/validates `YYYY-MM-DD`).
- 🧭 **Setup Wizard…** added to the **File** menu — re-open the first-run wizard any time to
  start a fresh document (demo data or from scratch).
- 💱 **File ▸ New** now asks for the document's **currency** before saving, instead of
  silently defaulting new files to Rp.
- 🛡️ Financy now **refuses to open a file written by a newer version** (with a clear
  "update Financy" message) rather than risk misreading it — the file is left untouched.

## ⬇️ Install

Download the build for your platform below, then:

- 🐧 **Linux** — `Financy-v0.2.0-linux-amd64.tar.xz`
  `tar xf Financy-v0.2.0-linux-amd64.tar.xz` and run the extracted app.
- 🪟 **Windows** — `Financy-v0.2.0-windows-amd64.exe`
  Run it. SmartScreen may warn (unsigned): *More info → Run anyway*.
- 🍎 **macOS** — `Financy-v0.2.0-macos.zip`
  Unzip and move `Financy.app` to Applications. First launch: right-click → *Open*.

> Builds are currently **unsigned**, so Windows/macOS may ask you to confirm before running.

## ⬆️ Upgrading from v0.1.0

Just open your existing `.financy` file in v0.2.0 — there are no schema changes in this
release, so it opens as-is. (When a future release does change the schema, Financy writes a
`.bak` before upgrading the file automatically.)

## ⚠️ Known limitations

- One currency per document (no multi-currency / FX within a file yet).
- No budgeting or importing yet — this release covers accounts, transactions, categories and
  analytics.
- Don't open the same `.financy` file from two machines at the same time.

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.1.0...v0.2.0

Built with [Go](https://go.dev) + [Fyne](https://fyne.io) + [SQLite](https://pkg.go.dev/modernc.org/sqlite).
Licensed under the MIT License.
