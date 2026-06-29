# Changelog

All notable changes to Financy are recorded here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

How to use this file: add one line under `Unreleased` for every change that lands
on `main`. An entry under **Added** means the next release is a minor bump
(`0.x.0`); only **Changed** / **Fixed** entries mean a patch (`0.0.x`). At release
time, rename `Unreleased` to the new version + date and start a fresh empty block.
See `RELEASING.md`.

## [Unreleased]

## [0.7.0] - 2026-06-29

### Added
- **Update check** — Financy now checks GitHub Releases for a newer version once a
  day on launch, or on demand via **File → Check for Updates…**, and shows a popup
  with a **Download** link. It never downloads or replaces the binary itself — you
  decide when and how to install. The check is an anonymous request to GitHub and
  sends none of your data; **Skip This Version** silences the prompt for a release
  you're not ready for.
- **Bulk add** — a new bulk-entry form for faster bookkeeping: two appendable
  tables (income/expense rows and transfers), each row carrying its own date, money
  account, amount, category and payee, so a backlog spanning several days is entered
  in one pass. Saving commits every valid row in a single batch instead of one
  dialog per transaction.

### Changed
- **Toolbar +** — now consistently opens the **Add Transaction** form on every
  screen. It previously changed meaning by page (an account form on Accounts, a
  budget-category form on Budget); per-screen adds remain as dedicated buttons
  within those screens.
- **Pre-upgrade backups** — the automatic `.bak` written before a file is migrated
  now embeds the document's schema version in its name (e.g. `My Budget.financy.v6.bak`),
  so you can tell which earlier Financy version still opens it if you need to downgrade.

### Removed
- **Reports** — the *Reports* screen (Income Statement, Balance Sheet, Cash Flow)
  has been removed. It added complexity without meaningful returns; the figures
  were purely derived from the journal and overlapped with Analytics.

### Fixed
- **Transactions** — a journal row with no payee now shows the transaction type
  (e.g. **Expense**, **Transfer**) instead of the category name, so the row title
  reads consistently regardless of category.

## [0.6.0] - 2026-06-27

### Added
- **Debts** — a new module for tracking Buy-Now-Pay-Later (and other installment)
  debts, with a **Debts** screen in the toolbar between Recurring and Analytics. Add
  a debt (name, lender, pay-from account, total, number of installments, purchase
  date, first due date, frequency) and a near-equal schedule is generated
  automatically. Each debt is a **tab** with an inline installment table — **Pay** or
  **Undo** each installment, or tap an unpaid one to edit its due date/amount. The
  header summarises what's **Outstanding**, **Due now**, due **This month**, and
  **Next month**. Installment payments are dated when you actually pay (today), not
  the scheduled due date.
- **Debts in the budget (zero-based)** — each debt is its own envelope under a
  **Debt Payments** group: you fund it monthly and paying an installment is budgeted
  spending that draws the liability down, leaving **Ready to Assign** neutral. Every
  debt also opens an off-budget **Liability** account that mirrors the outstanding
  balance, so it counts in **Accounts** and **Net Worth**. The purchase is booked as
  a balance-sheet-only financing event on its **purchase date** (distinct from the
  first payment's due date) against an equity contra — Net Worth drops by what you
  owe, with no expense category charged. Persisted via schema migrations v5–v7.
- **Pay debt from Add Transaction** — the Add Transaction form gains a **Pay debt**
  type: choose a debt and one of its unpaid installments to post the payment without
  leaving the transaction flow.

## [0.5.1] - 2026-06-27

### Changed
- **`make run-dev`** — run against an isolated dev profile so development never touches
  your real data. It repoints `$XDG_CONFIG_HOME` at `./.devdata/config` (gitignored),
  giving dev its own `prefs.json`, recent-files list, and `.financy` database; your
  `~/.config/financy` is left untouched. (Linux/BSD only — macOS ignores `XDG_CONFIG_HOME`.)
- **Funding & sponsorship** — added a GitHub Sponsors funding configuration and a
  "Support / sponsorship" section to the README. No application changes.

## [0.5.0] - 2026-06-26

### Added
- **Richer demo data** — the sample document now exercises every module coherently:
  an off-budget **Investments** account that grows via monthly auto-invest, plus a full
  **budget** — monthly assignments for every category, sinking funds (Emergency Fund,
  Insurance, Vacation) that roll an Available balance forward, a mid-year vacation that
  draws its envelope down, and a few categories left intentionally tight so overspending
  shows. Accounts, Transactions, Budget, Recurring, Analytics and Reports all tie out.
- **Budget** — a new zero-based ("envelope") budgeting screen modelled on YNAB.
  Pick a month and give every unit of currency a job: each Expense category shows
  what you **Assigned**, its **Activity**, and a rolling **Available** balance, with
  a **Ready to Assign** banner up top. Ready to Assign is a single global pool —
  your on-budget funds minus everything you've committed to a category in *any*
  month — so it reads the same whatever month you view, money assigned to a future
  month draws it down today, and the goal is to budget until it reaches 0.
  **Past months are locked** (view-only) — you can only change the current and
  future months, so a budget you've already lived through can't be rewritten.
  Available carries forward month to month;
  cash overspending turns a category red and is absorbed by the next month's Ready
  to Assign (it doesn't drag the envelope negative forever). Click a category to
  assign an amount with one-tap quick-budget suggestions (last month's assigned,
  last month's spent, 3-month average), or **Auto-Assign** to copy the previous
  month's plan. Assignments are saved in the document (schema v3).
- **On-budget / tracking accounts** — accounts have an **Include in budget** toggle.
  Ready to Assign is computed from your on-budget **Asset and Liability** balances
  (so a credit card's debt nets against your cash and card spending is budget-neutral,
  the way YNAB handles it), while tracking accounts you exclude — investments, a
  mortgage — no longer inflate the money you can assign. Stored per account (schema v4).
- **Dark mode** — a 🎨 toggle in the toolbar switches the whole app between the
  light and a new dark theme (deep-slate surfaces with accents tuned to read well
  on a dark ground). Your choice is remembered and restored on the next launch.
- **Linux `.deb` and `.rpm` packages** — releases now ship native Debian/Ubuntu and
  Fedora/RHEL packages (built with nfpm). Installing puts `financy` on your `PATH`,
  adds a desktop launcher and app icon, and registers the `.financy` file type so
  documents open with a double-click. See the README install section.

### Changed
- **Repositioned as a zero-based budgeting (ZBB) app** — the README, website and docs
  now lead with budgeting ("give every dollar a job"), with double-entry accounting as
  the trustworthy engine underneath. Added a Budget user-guide page and updated the
  desktop entry/package metadata. The app name, file format and data are unchanged.

## [0.4.0] - 2026-06-23

### Added
- **Bulk recategorize** — the Transactions screen has a **Select** mode: tick any
  number of rows (with **Select all shown** to grab everything matching the current
  filters), then **Recategorize…** to move them all to one category at once.
  Transfers, opening balances, and entries whose kind doesn't match the chosen
  category are left untouched, and it reports how many actually changed. Pairs well
  with CSV import for cleaning up uncategorized rows in bulk.
- **Reports** — a new *Reports* screen presenting the three core financial
  statements as tabs: an **Income Statement** (P&L: income and expense categories
  for the period, ending in net income), a **Balance Sheet** (assets, liabilities,
  and equity as of the period end, where equity is opening balances plus retained
  earnings — it always satisfies *Assets = Liabilities + Equity*), and a **Cash
  Flow** (per-asset-account opening/closing/change for the period). It shares the
  Analytics period selector (This month / Last 3·6·12 months / YTD). Everything is
  read-only and derived from the journal, so the statements can't drift from your
  transactions — no schema change.
- **CSV export** (`File ▸ Export CSV…`) — write all transactions to a flat,
  spreadsheet-friendly CSV (`Date, Payee, Category, Account, Amount, Memo`),
  chronological, with plain decimal amounts. It round-trips with the importer, so
  you can export, edit in a spreadsheet, and re-import.
- **Recurring transactions** — a new *Recurring* screen for templates (rent,
  salary, subscriptions) with a frequency (Weekly/Biweekly/Monthly/Quarterly/
  Yearly) and next-due date. When entries are due, Financy prompts you (on open, or
  via **Review & Post** on the screen, where due templates are highlighted) — for
  each occurrence you choose, from a dropdown, to **post a new transaction** or
  **link it to the existing transaction that already paid it** (Financy pre-selects
  a close match if it finds one, so you can't double-post; the amount needn't match
  exactly, and a *Browse all transactions…* picker lets you search the account's
  full history to link any transaction). It never posts silently. Missed periods are
  caught up. A row's ⋮ menu also offers **Post now (early)** — a review to create a
  new transaction today or link the occurrence to an existing one. Posted entries
  are normal, editable transactions. Adds a schema migration (existing files upgrade
  automatically, with the usual `.bak`).

### Fixed
- Button hover no longer washes out — primary (blue) buttons now darken slightly on
  hover instead of turning flat gray, and secondary buttons show a visible hover
  (the theme's hover color was opaque, replacing the button colour instead of
  tinting it).
- Hover tooltips (toolbar icons, analytics charts) now appear at the cursor instead
  of being offset too far below it — the in-canvas File menu was pushing the
  floating tooltip layer down.
- The **Setup Wizard** and **File ▸ New** dialogs now have a Close/Cancel button, so
  they can be dismissed without creating a document.

## [0.3.0] - 2026-06-23

### Added
- **Account reconciliation** — right-click an account → *Reconcile…* (or the
  **Reconcile** button in its register). Enter the real balance from your bank and
  Financy posts a single adjustment transaction for the difference so the account
  matches. The contra goes to a *Reconciliation* equity account, so net worth
  updates but income/expense analytics aren't distorted; the adjustment is a normal,
  auditable journal entry.
- **Live amount formatting** — money fields (transaction amount, opening balance)
  now group thousands as you type (e.g. `1,000,000`), and a new **Number format**
  setting in Preferences lets you choose the separator style (`1,234.56` /
  `1.234,56` / `1 234,56` / …) independently of the currency symbol.
- **Guided CSV import** (`File ▸ Import CSV…`, or the Transactions screen) — bring in
  bank/card exports. A mapping step auto-detects columns and date format from the
  header (any layout; single signed-amount **or** separate debit/credit columns),
  then a preview shows each row's parsed date, payee, amount and category before you
  commit. Single-entry rows become balanced double-entry transactions (the money
  account plus a contra category). Rows **auto-categorize by payee** — matching
  against transactions you've already categorized (exact, then token/substring, so
  `AMAZON.COM*A1B2C` still matches a learned `Amazon`), so recurring payees come in
  tagged; the rest default to Uncategorized. Every row has a **category dropdown**
  in the preview so you can set/adjust before committing. Likely duplicates (same
  date+payee+amount already in the account) are flagged and excluded by default;
  unparseable rows are skipped. Imports commit atomically in one batch.

## [0.2.0] - 2026-06-23

### Added
- **Analytics** screen — a read-only insights dashboard with KPI cards (net worth
  + change, income, expenses, net saved, savings rate), an Income-vs-Expenses bar
  chart, a Net-worth-over-time line chart, and ranked Spending-by-category, all
  over a selectable period (This month / Last 3 / 6 / 12 / Year to date). Charts
  are drawn natively with Fyne canvas primitives (no new dependency) and are
  interactive: hovering a month highlights it and shows a tooltip (per-month
  income/expense/net on the bar chart, net worth on the line chart). Both charts
  have a Y axis with value gridlines and compact tick labels ($2.5k, $26k, …).
  Reachable from the toolbar.
- **Setup Wizard…** in the File menu — re-open the first-run welcome wizard at any
  time to spin up a fresh document with demo data or start from scratch.
- Calendar date picker on the transaction date field — click the calendar button
  to pick a day instead of typing. The field still accepts/validates `YYYY-MM-DD`.
- Refuse to open a `.financy` file whose schema is newer than the running app,
  with a clear "update Financy" dialog, instead of risking a misread. The file is
  left untouched.

### Changed
- Demo data now spans ~6 months (~90 transactions) of recurring income, bills,
  groceries, dining, transfers and credit-card activity, so first-run users see
  balances, registers and period flows behave like a real file. Screenshots
  regenerated.

### Fixed
- **File ▸ New** now asks for the document's currency before saving (like the
  Setup Wizard), instead of silently defaulting every new file to Rp.

## [0.1.0] - 2026-06-22

Initial release — a fast, local-first personal finance manager built on real
double-entry accounting, with data in a single `.financy` file you own.

### Added
- Double-entry accounting core: every transaction is balanced postings summing to
  zero; balances and net worth are derived, never stored.
- Integer-minor-unit (cents) money handling with formatting for Rp · $ · € · £ (no
  floating-point money).
- Account cards for assets & liabilities, a net-worth overview, and a per-account
  register with running balance.
- Date-grouped transaction journal with an Income / Expense / Transfer entry form,
  plus filters and search.
- Income & expense categories, managed from Preferences.
- Desktop experience: right-click context menus, hover tooltips, an icon toolbar,
  and a minimal File menu.
- First-run setup: load demo data (USD) or start from scratch with a chosen currency.
- Single-file `.financy` SQLite documents with always-on auto-save (ACID writes).
- New / Open / Open Recent / Save a Copy; reopens the last file on launch.
- Safe-by-design persistence: atomic writes, append-only schema migrations, and an
  automatic `.bak` before any file upgrade.
- Cross-platform release bundles (Linux / Windows / macOS) published from CI on tag.

[Unreleased]: https://github.com/raihanstark/financy/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/raihanstark/financy/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/raihanstark/financy/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/raihanstark/financy/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/raihanstark/financy/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/raihanstark/financy/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/raihanstark/financy/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/raihanstark/financy/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/raihanstark/financy/releases/tag/v0.1.0
