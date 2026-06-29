# 🗺️ Financy Roadmap

This document tracks where Financy is heading. It's a living plan — priorities can
shift, and dates are intentionally omitted. For the detailed history of what has
already shipped, see [CHANGELOG.md](CHANGELOG.md).

**Guiding principles** — Financy stays **local-first, private, and yours**: a single
`.financy` file you own, no cloud account, no tracking, and double-entry accounting
as the trustworthy engine underneath. Anything on this roadmap is held to those
principles.

---

## ✅ Where we are today (0.x)

The core product is feature-complete and in active use:

- 🎯 Zero-based budgeting (Ready to Assign, envelopes, quick-budget, Auto-Assign)
- 📒 Real double-entry accounting core with derived balances & net worth
- 🏦 Accounts & net worth, per-account register with running balance
- 🧾 Transactions (Income / Expense / Transfer), filters, search, bulk add & recategorize
- 🗓️ Debts (BNPL / installments) integrated into the budget
- 🔁 Recurring transactions with review-before-post
- 📊 Analytics (KPIs, income-vs-expenses, net worth, spending by category)
- 🔁 Reconciliation, CSV import & export
- 🔒 Optional password encryption (XChaCha20-Poly1305 + Argon2id)
- 🌗 Dark mode, update check, Linux `.deb` / `.rpm` packages

---

## 🚀 v1.0 — "Stable & shippable"

The goal of **v1.0** is not a pile of new features — it's declaring the current
feature set **stable, polished, and trustworthy** enough to commit to a `1.x` line
with backward-compatible file formats.

**Must-have for 1.0:**

- [ ] **Stabilize the `.financy` file format** — commit to forward/backward
      compatibility guarantees within the `1.x` series; document the schema.
- [ ] **Linux as the supported install target** — polished, first-class Linux
      `.deb` / `.rpm` packages are the officially supported way to install v1.0.
- [ ] **Unsigned Windows & macOS builds** — ship Windows and macOS bundles for 1.0
      too, but **unsigned** (no code-signing / notarization yet); users get the usual
      OS "unidentified developer" prompt. Signing/notarization is a post-1.0 goal
      (see below).
- [ ] **Onboarding polish** — refine the first-run setup wizard and demo data so a
      new user reaches a useful budget in minutes.
- [ ] **Documentation pass** — complete the user guide for every screen and a
      "concepts" explainer for zero-based budgeting + double-entry.
- [ ] **Accessibility & keyboard navigation** — full keyboard flow for the common
      paths (add transaction, assign budget, navigate months).
- [ ] **Hardening** — broaden test coverage on the `core` package, fuzz the CSV
      importer, and shake out edge cases in encryption save/close flows.
- [ ] **Performance** — confirm large documents (years of history, thousands of
      transactions) stay responsive.

---

## 🔭 Upcoming — after 1.0

The north star after 1.0 is to make Financy an **all-in-one personal finance
app** — budgeting, accounting, debts, *and* investments in one local-first,
private file — so you can see your whole financial picture in one place.

Candidate features under consideration, roughly in priority order. Feedback and
👍 reactions on issues help decide what comes first.

- **📈 Investment tracker** — turn the existing off-budget *Investments* accounts
  into a true portfolio tracker. Record **holdings** (stocks, ETFs, funds, crypto)
  as quantity + cost basis, log **buy / sell / dividend** transactions through the
  double-entry journal (so net worth always ties out), and see **per-holding and
  total market value, unrealized/realized gains, and allocation**. Prices can be
  updated **manually** first (privacy-first, no network), with an *optional* quote
  fetch later — and a dedicated **Investments** screen plus net-worth and
  allocation charts in Analytics. This is the centerpiece of the all-in-one vision.
- **💳 In-depth debt management** — grow the Debts module beyond fixed BNPL/
  installment schedules into full loan tracking: **interest / APR** with proper
  principal-vs-interest split per payment and an **amortization schedule**, support
  for **revolving credit** (credit cards, lines of credit) and amortizing loans
  (mortgage, auto, student), **extra / early payments** and minimum-payment
  handling, **payoff strategies** (snowball vs. avalanche) with a comparison view,
  **payoff-date and total-interest projections**, and a debt overview dashboard
  showing total owed, weighted average rate, and progress over time.
- **Budget goals / targets** — per-category monthly or sinking-fund targets with
  progress indicators, so Ready to Assign can suggest how much to fund.
- **Multi-currency** — hold accounts in different currencies within one document,
  with exchange-rate handling for net worth.
- **Reports, reborn** — bring back focused, printable statements (P&L, balance
  sheet, cash flow) as an opt-in export rather than a permanent screen.
- **Scheduled reports & spending alerts** — optional nudges when a category is
  overspent or a bill is due.
- **Attachments** — attach a receipt or note to a transaction.
- **Saved filters / views** on the Transactions screen.
- **Importer improvements** — more bank formats, OFX/QFX support, smarter
  payee/category learning.
- **Signed & notarized builds** — code-sign the Windows installer and
  sign/notarize the macOS bundle so they install without security warnings, and
  add proper platform installers (`.dmg` / Windows setup).
- **Optional self-hosted sync** — an *opt-in*, end-to-end-encrypted way to sync a
  document across your own devices, without a Financy cloud account.
- **Localization** — translate the UI beyond English.

---

## 💡 Have an idea?

Open an [issue](https://github.com/raihanstark/financy/issues) or start a
discussion. If a feature here matters to you, a 👍 helps us prioritize. And if
Financy helps you give every dollar a job, consider
[sponsoring the project](https://github.com/sponsors/RaihanStark) 💚.
</content>
</invoke>
