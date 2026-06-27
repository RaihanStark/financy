# 💰 Financy v0.6.0 — Debt tracking

Financy now tracks **debt** as a first-class part of your finances. Add a
Buy-Now-Pay-Later or installment plan and Financy generates the schedule, opens a
matching liability account, and wires it into your zero-based budget — so the money
you owe shows up in your net worth and your monthly plan, never drifting out of
balance with the rest of your journal.

## ✨ Highlights

**💳 A Debts module**
- A new **Debts** screen in the toolbar (between Recurring and Analytics). Add a debt
  with its name, lender, pay-from account, total, number of installments, purchase
  date, first due date, and frequency — Financy generates a near-equal schedule
  automatically.
- Each debt is a **tab** with an inline installment table: **Pay** or **Undo** each
  installment, or tap an unpaid one to edit its due date or amount.
- The header summarises what's **Outstanding**, **Due now**, due **This month**, and
  **Next month**. Payments are dated when you actually pay (today), not the scheduled
  due date.

**🎯 Debts in your zero-based budget**
- Each debt is its own envelope under a **Debt Payments** group — fund it monthly, and
  paying an installment is budgeted spending that draws the liability down, leaving
  **Ready to Assign** neutral.
- Every debt also opens an off-budget **Liability** account that mirrors the
  outstanding balance, so it counts in **Accounts** and **Net Worth**.
- The original purchase is booked as a balance-sheet-only financing event on its
  **purchase date** (distinct from the first payment's due date) against an equity
  contra — Net Worth drops by what you owe, with no expense category charged.

**⚡ Pay a debt from Add Transaction**
- The Add Transaction form gains a **Pay debt** type: choose a debt and one of its
  unpaid installments to post the payment without leaving the transaction flow.

## 🗃️ Data & compatibility

- Append-only schema migrations **v5–v7** add debt persistence. Existing `.financy`
  files upgrade automatically on open (a `.bak` is written first).
- Amounts remain integer minor units; money is never stored as a float.

## 📦 Install

Grab the build for your platform from the assets below.

- **Debian / Ubuntu:** `sudo apt install ./Financy-v0.6.0-linux-amd64.deb`
- **Fedora / RHEL / openSUSE:** `sudo dnf install ./Financy-v0.6.0-linux-x86_64.rpm`
- **Other Linux:** extract `Financy-v0.6.0-linux-amd64.tar.xz` and run the app.
- **Windows:** run the `.exe` (SmartScreen → *More info → Run anyway*; unsigned).
- **macOS:** unzip, move to Applications, first launch **right-click → Open** (unsigned).

The full list of changes is in [CHANGELOG.md](../CHANGELOG.md).
