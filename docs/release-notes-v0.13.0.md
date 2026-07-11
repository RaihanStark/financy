## Financy v0.13.0 — Debts, reworked 💳

The biggest module release since the budget: Debts grows beyond BNPL plans
into **full debt tracking** — loans with real interest math, credit cards,
and IOUs — with payoff projections and a strategy comparison to plan your way
out. Everything ships on **desktop and mobile alike**, and your existing
`.financy` files upgrade automatically (a `.bak` is written first, as always).

### 🏦 Every debt type

- **Amortizing loans** (mortgage, auto, student, personal) — enter the
  principal, APR, and either the term or the payment; Financy computes the
  other and generates the full amortization schedule. Every payment posts a
  real **principal-vs-interest split**: principal draws the loan down,
  interest lands in an expense category, and the loan's balance always equals
  the remaining principal. **Extra payments** either shorten the term (the
  default — saves the most interest) or lower the payment, with a live
  preview of the payoff date and interest saved.
- **Revolving credit** (credit cards, lines of credit) — attach the card
  account you already track (its history stays yours) or create a new one,
  with a credit limit, utilization bar, and a minimum-payment rule. Interest
  is never auto-posted: once a month a **statement review** proposes an
  editable charge for you to confirm, correct, or skip — your real statement
  is ground truth. Card payments are budget-neutral transfers, so your
  envelopes are never double-counted.
- **Informal debts** (an IOU, money from a friend) — just a balance with an
  optional due date; pay any amount anytime, and optionally record which
  account the borrowed cash landed in so your books balance to the cent.
- **BNPL / installment plans** work exactly as before.

### 📈 Payoff projections & strategies

- Every debt shows its **payoff date and interest remaining** under the
  current plan — including the classic minimum-payment trap on a card, which
  is reported honestly ("this never pays off") instead of pretending.
- **Snowball vs. avalanche**: tell Financy what extra you can put toward
  debts each month and compare both strategies side by side — payoff order,
  debt-free date, and total interest, with the cheaper plan highlighted.
  Every cleared debt's payment rolls into the next target automatically.
- A **debt overview dashboard** sums it up: total owed, balance-weighted
  average APR, what's due now, and your projected debt-free date.

### ✨ Easy to set up

- The add-debt form is **type-driven and minimal**: pick a kind and only its
  essentials appear (a loan is just name + principal + APR + term), with a
  live plan preview before you save. Everything optional folds into an
  **Advanced setup** section with defaults that just work.
- The desktop Debts screen is a **compact table** — kind, APR, balance,
  progress, status — and clicking a row opens the debt's full detail in a
  popup: schedule with pay/undo, extra payments, statement review, activity.

### 🔧 Fixed

- Deleting a debt no longer removes transactions of yours that were linked to
  installments — they're restored to their original categories first.

### 📥 Install

**Android:** download `Financy-v0.13.0-android-arm64.apk` below and sideload it
(you'll need to allow installs from your browser/file manager).

**Desktop:** grab the package for your OS below (`.deb` / `.rpm` / `.tar.xz` /
`.exe` / macOS `.zip`).

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.12.0...v0.13.0
