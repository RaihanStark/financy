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
- [ ] **First-class installers for every OS** — signed/notarized macOS `.dmg` and a
      Windows installer alongside the existing Linux `.deb` / `.rpm`.
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

Candidate features under consideration, roughly in priority order. Feedback and
👍 reactions on issues help decide what comes first.

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
