# 💰 Financy v0.1.0 — Initial Release

The first release of **Financy**, a fast, local-first personal finance manager built on
**real double-entry accounting**. Your data lives in a single file you own, and the books
can never drift out of balance.

## ✨ What's included

**Accounting core**
- 📒 True double-entry — every transaction is balanced postings that sum to zero; balances and net worth are *derived*, never stored, so they can't go inconsistent.
- 💵 Proper money handling — amounts stored as integer minor units (cents), with formatting for **Rp · $ · € · £** (no floating-point money).

**Accounts & transactions**
- 🏦 Account cards for assets & liabilities, a net-worth overview, and a per-account **register with running balance**.
- 🧾 A date-grouped **transaction journal** with a familiar Income / Expense / Transfer entry form, plus filters and search.
- 🏷️ Income & expense **categories**, managed from Preferences.

**Desktop experience**
- 🖱️ Right-click **context menus**, hover **tooltips**, an icon toolbar, and a minimal File menu.
- ⚡ First-run **setup**: load demo data (USD) or start from scratch with your chosen currency.

**Your data**
- 💾 Each document is a single **`.financy` SQLite file**, always auto-saved (ACID writes — no save button).
- 📂 New / Open / Open Recent / Save a Copy; reopens your last file on launch.
- 🔒 Safe by design — atomic writes, append-only schema migrations, and an automatic `.bak` before any file upgrade.

## ⬇️ Install

Download the build for your platform below, then:

- 🐧 **Linux** — `Financy-v0.1.0-linux-amd64.tar.xz`
  `tar xf Financy-v0.1.0-linux-amd64.tar.xz` and run the extracted app.
- 🪟 **Windows** — `Financy-v0.1.0-windows-amd64.exe`
  Run it. SmartScreen may warn (unsigned): *More info → Run anyway*.
- 🍎 **macOS** — `Financy-v0.1.0-macos.zip`
  Unzip and move `Financy.app` to Applications. First launch: right-click → *Open* (unsigned app).

> Builds are currently **unsigned**, so Windows/macOS may ask you to confirm before running.

## 🚀 Getting started

On first launch, choose **Load demo data (USD)** to explore, or **Start from scratch** and
pick your currency — then choose where to save your `.financy` file. That's it.

## ⚠️ Known limitations

- One currency per document (no multi-currency / FX within a file yet).
- No reporting, budgeting, or charts yet — this release covers accounts, transactions, and categories.
- Don't open the same `.financy` file from two machines at the same time (SQLite isn't built for that).

---

Built with [Go](https://go.dev) + [Fyne](https://fyne.io) + [SQLite](https://pkg.go.dev/modernc.org/sqlite).
Licensed under the MIT License.
