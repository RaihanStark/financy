<div align="center">

# 💰 Financy

**Open-source zero-based budgeting (ZBB) for your desktop — give every dollar a job.**

A fast, local-first **zero-based budget** in a native desktop app: assign every unit
of money to a category until *Ready to Assign* hits zero, then watch your envelopes
through the month. Underneath it's **real double-entry accounting**, so your budget,
balances and net worth are all derived from one journal and can never drift out of
balance. Your data lives in a single file you own — no cloud, no account, no tracking —
and can be **protected with strong password encryption**, so even the file on disk is
unreadable without your passphrase.

![CI](https://github.com/raihanstark/financy/actions/workflows/ci.yml/badge.svg)
&nbsp;![govulncheck](https://github.com/raihanstark/financy/actions/workflows/govulncheck.yml/badge.svg)
&nbsp;![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
&nbsp;![Fyne](https://img.shields.io/badge/UI-Fyne-1A1A2E)
&nbsp;![SQLite](https://img.shields.io/badge/Storage-SQLite-003B57?logo=sqlite&logoColor=white)
&nbsp;![Encryption](https://img.shields.io/badge/Encryption-XChaCha20--Poly1305-4B0082)
&nbsp;![License](https://img.shields.io/badge/License-MIT-green)

📖 **[Read the documentation](https://raihanstark.github.io/financy/docs/)** · 🌐 **[Website](https://raihanstark.github.io/financy/)**

<img src="docs/screenshots/budget.png" alt="Zero-based budget" width="820">

</div>

---

## ✨ Features

- 🎯 **Zero-based budgeting** — the heart of the app. Every month, give each category a job until **Ready to Assign** reaches zero. Track **Assigned · Activity · Available** per category, with envelope balances that roll forward and overspending shown in red. One-tap quick-budget suggestions (last month's assigned/spent, 3-month average) and **Auto-Assign** to copy a month's plan.
- 💰 **A Ready-to-Assign that's honest** — it's a single pool drawn from your real **on-budget account balances** (assets net of credit-card debt), so money assigned to any month — even a future one — draws it down, and you can never assign money you don't have.
- 💳 **Credit cards done right** — spending on a card depletes its envelope *and* nets against the budget, so it's budget-neutral the way a good ZBB tool should be. Mark investments or a mortgage as **off-budget** so they count toward net worth but not assignable cash.
- 🔒 **Locked history** — past months are view-only; you budget the current month and beyond, and a budget you've already lived through can't be rewritten.
- 📒 **Real double-entry accounting under the hood** — every transaction is balanced postings that sum to zero, so your budget, balances and net worth are *derived* from one journal and can never go inconsistent.
- 💵 **Proper money handling** — amounts are stored as integer minor units (cents), with per-currency formatting for **Rp · $ · € · £** (no floating-point money, ever).
- 🏦 **Accounts & net worth** — asset & liability cards, a net-worth overview, and a per-account **register with running balance**.
- 🧾 **Transactions** — a clean, date-grouped journal with a familiar **Income / Expense / Transfer** entry form, live filters and search, plus a **bulk-select mode** to recategorize many entries at once.
- 🗓️ **Debts (BNPL & installments)** — track Buy-Now-Pay-Later and other installment debts on a fixed schedule. Each debt opens an off-budget **Liability** account (so it shows in Accounts & Net Worth) and becomes its own **budget envelope** you fund monthly — paying an installment is budgeted spending that draws the balance down, leaving Ready to Assign neutral. The purchase is booked on its **purchase date** against an equity contra (no expense double-count). A tabbed screen shows each debt's installment table with **Pay / Undo**, plus an **Outstanding · Due now · This month · Next month** summary.
- 📊 **Analytics** — KPIs, income-vs-expenses, net-worth-over-time and spending-by-category charts — all derived from the journal, so they never drift from your transactions.
- 🔐 **Password encryption (optional)** — protect a document with a passphrase and the `.financy` file is encrypted at rest with **XChaCha20-Poly1305** and an **Argon2id**-derived key, and is tamper-evident. Your data is only ever decrypted in memory, never persisted to disk in plaintext. A forgotten passphrase is unrecoverable — there's no backdoor.
- 💾 **Local-first persistence** — each document is a single **`.financy` SQLite file** you own. **Unencrypted** files are *always auto-saved* (ACID writes — no save button, no lost work); **encrypted** files use **manual save** (Cmd/Ctrl-S) with an *unsaved-changes prompt* when you close — the trade-off that keeps plaintext off your disk.
- ⚡ **Quick start** — first-run setup lets you **load demo data** (a complete worked budget) or **start from scratch** with your chosen currency and an optional password.
- 🛡️ **Safe by design** — atomic writes, append-only schema migrations, and an automatic `.bak` before any file upgrade.

---

## 📸 Screenshots

<div align="center"><img src="docs/screenshots/budget.png" alt="Zero-based budget — Ready to Assign, category envelopes" width="820"></div>

| Accounts & net worth | Analytics |
| :-: | :-: |
| <img src="docs/screenshots/accounts.png" width="420"> | <img src="docs/screenshots/analytics.png" width="420"> |

| Transactions | First-run setup |
| :-: | :-: |
| <img src="docs/screenshots/transactions.png" width="420"> | <img src="docs/screenshots/setup.png" width="420"> |

---

## 🚀 Getting Started

### Install (Linux packages)

Grab the `.deb` or `.rpm` for the latest release from the
[Releases page](https://github.com/raihanstark/financy/releases/latest):

```sh
# Debian / Ubuntu
sudo apt install ./Financy-vX.Y.Z-linux-amd64.deb

# Fedora / RHEL / openSUSE
sudo dnf install ./Financy-vX.Y.Z-linux-x86_64.rpm
```

This installs the `financy` command, a desktop launcher, and registers the
`.financy` file type so you can double-click documents to open them.

### Prerequisites (building from source)
- **Go 1.25+**
- A C toolchain + OpenGL/X11 dev headers (Fyne requirement). On Debian/Ubuntu:
  ```sh
  sudo apt-get install gcc pkg-config libgl1-mesa-dev xorg-dev
  ```

### Run it
```sh
go run .        # or: make run
```

### Build a binary
```sh
make build      # produces ./financy with the version stamped in
```

On first launch you'll be greeted with a setup dialog — pick **Load demo data (USD)**
to explore, or **Start from scratch** and choose your currency. Then choose where to
save your `.financy` file.

---

## 🧭 Usage

- **Budget the month** — open **💰 Budget**, then click a category to assign money
  (with quick-budget suggestions) until **Ready to Assign** is zero. Navigate months
  with ◀ ▶, **Auto-Assign** to copy last month, and **+ Add Category** to grow your plan.
- **Add a transaction** — the toolbar **`+`**, or the *Add Transaction* button. Choose
  Income / Expense / Transfer; the app writes the correct double-entry postings for you,
  and the spending flows straight into your budget's *Activity*.
- **Track a debt** — open **🗓️ Debts** to add a BNPL/installment plan (set a purchase
  date and first due date). Pay installments from the Debts screen or via *Add
  Transaction ▸ Pay debt*, and fund each debt's envelope on the Budget screen.
- **Open an account's register** — click an account card. Right-click a card (or use the
  **⋮** button) for *New Transaction · View Register · Edit · Delete*.
- **Filter the journal** — by month, type, account, or free-text search.
- **See your trends** — the **📊 Analytics** screen shows KPIs and charts over This month / Last 3 / 6 / 12 / Year to date; hover any month for a tooltip.
- **Manage categories & currency** — the **⚙ Preferences** dialog (Configuration · Categories · Data Summary).
- **Files** — `File ▸ New / Open / Open Recent / Save a Copy / Close`.

---

## 💾 Your Data

- Each document is a self-contained **SQLite** database with a `.financy` extension —
  move it, back it up, or drop it in a cloud folder (while the app is closed).
- It's **always saved** as you work (ACID writes). There is no "Save" button.
- **Save a Copy** snapshots the document elsewhere; opening an older file makes a `.bak`
  before migrating it to the current schema.

> ⚠️ SQLite isn't built for two machines editing the same file at once — don't open the
> same `.financy` from multiple devices simultaneously.

---

## 🏗️ Architecture

```
main.go                  entry point (embeds the icon, calls ui.Run)
internal/
  core/                  domain model · double-entry Store · SQLite · formatting   (no UI deps)
  ui/
    style/               palette + theme
    component/           reusable widgets (cards, table, rows, app bar, tooltips, charts)
    view/                screens (Budget, Accounts, Transactions, Debts, Recurring, Analytics) + Preferences + forms
    (root)               app shell, controller, toolbar, File menu, document manager
```

The **`core`** package has zero UI dependencies and holds all the accounting logic, so it's
independently testable — that's where new logic and tests belong.

---

## 🛠️ Development

```sh
make check      # build + vet + test   (run before committing)
make test       # tests only
make run        # run the app
make run-dev    # run with an isolated dev profile + database (won't touch your real data)
make shot       # regenerate screenshots (into /tmp/financy-shots)
```

`make run-dev` repoints `$XDG_CONFIG_HOME` at `./.devdata/config` (gitignored) so dev
gets its own `prefs.json`, recent-files list, and `.financy` database — your real
`~/.config/financy` is never read or written. First run shows the setup wizard; save the
dev document inside `./.devdata/` and it reopens next time. (Linux/BSD only — macOS
doesn't honour `XDG_CONFIG_HOME`.)

**Two rules that protect user data:**
1. **Migrations are append-only** — add a new entry to `migrations` in `internal/core/db.go`; never edit an existing one.
2. **Money stays integer minor units** — never introduce floats for money.

---

## 🚢 Releasing

```sh
make release VERSION=0.2.0     # stamps version, verifies, builds
git commit -am "Release v0.2.0"
git tag v0.2.0 && git push --tags
```

Pushing a `v*` tag triggers CI to package **Linux / Windows / macOS** bundles and attach
them to the GitHub Release. See [RELEASING.md](RELEASING.md) for the full guide.

---

## 🧰 Tech Stack

- **[Go](https://go.dev)** — application language
- **[Fyne](https://fyne.io)** — cross-platform native GUI
- **[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)** — pure-Go SQLite (no cgo for the DB)

---

## 🗺️ Roadmap

Financy's core is feature-complete and the project is working toward a **stable
v1.0**. The supported install target for 1.0 is **Linux** (`.deb` / `.rpm`);
**Windows and macOS ship too, but unsigned first** (you'll see the usual
"unidentified developer" prompt), with code-signing/notarization coming after 1.0.
The rest of the v1.0 work is locking down the `.financy` file format plus a
docs/accessibility/hardening pass. After 1.0 the goal is to grow Financy into an
**all-in-one finance app**. The **first priority** is **💳 in-depth debt
management** (interest/APR, amortization schedules, snowball vs. avalanche payoff
strategies and payoff projections), followed by a full **📈 investment tracker**
(holdings, cost basis, gains and allocation, built on the same double-entry
journal). Other candidates include **budget goals/targets**, **multi-currency**,
an **OFX/QFX-aware importer**, signed installers, and an *opt-in, self-hosted*
end-to-end-encrypted **sync** (never a Financy cloud account).

See **[ROADMAP.md](ROADMAP.md)** for the full plan, and
[CHANGELOG.md](CHANGELOG.md) for what's already shipped.

---

## ❤️ Support the Project

Financy is free and open-source, built and maintained in my spare time. If it helps you
give every dollar a job, please consider [**sponsoring on GitHub**](https://github.com/sponsors/RaihanStark) —
it directly funds continued development and is hugely appreciated. A ⭐ on the repo helps too!

---

## 📄 License

Released under the [MIT License](LICENSE).
