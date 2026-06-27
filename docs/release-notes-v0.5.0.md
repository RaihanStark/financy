# 💰 Financy v0.5.0 — Zero-based budgeting

Financy is now a **zero-based budgeting (ZBB)** app: give every dollar a job until
*Ready to Assign* hits zero, then track your envelopes through the month. It's all
built on the same local-first, double-entry foundation — your budget, balances and
net worth are derived from one journal in a single file you own, so they can never
drift out of balance. Plus a new dark mode.

## ✨ Highlights

**🎯 Zero-based budget**
- A new **Budget** screen: pick a month and give each category a job. Every category
  shows what you **Assigned**, its **Activity**, and a rolling **Available** balance.
- **Available rolls forward** month to month. Cash overspending turns a category red
  and is absorbed rather than dragging the envelope negative forever.
- Click a category to assign, with one-tap **quick-budget** suggestions (last month's
  assigned/spent, 3-month average), **Auto-Assign** to copy last month's plan, and a
  one-field **+ Add Category**.

**💰 An honest Ready to Assign**
- Ready to Assign is a **single global pool** drawn from your real on-budget account
  balances. It reads the same whatever month you view.
- Money assigned to **any** month — even a future one — draws it down today, so you
  can never assign money you don't have. The goal is to budget until it reaches **0**.

**💳 Credit cards & tracking accounts done right**
- Spending on a credit card depletes its envelope **and** nets against your funds, so
  it's budget-neutral the way a good ZBB tool should be.
- A per-account **Include in budget** toggle lets you mark investments or a mortgage
  as *tracking* — counted toward net worth, but not as assignable cash.

**🔒 Locked history**
- Past months are **view-only** — you budget the current month and beyond, and a
  budget you've already lived through can't be rewritten.

**🎨 Dark mode**
- A toolbar toggle switches the whole app between light and a new dark theme. Your
  choice is remembered and restored on the next launch.

**🧪 A complete demo**
- Load the sample document to see a fully-worked example: ~6 months of transactions,
  an off-budget Investments account, sinking funds, and a current month budgeted all
  the way to *Ready to Assign = 0* — tying out across Budget, Accounts, Transactions,
  Recurring, Analytics and Reports.

## 🗃️ Data & compatibility

- Two append-only schema migrations: budget assignments (**v3**) and a per-account
  off-budget flag (**v4**). Existing `.financy` files upgrade automatically on open
  (a `.bak` is written first).
- Amounts remain integer minor units; money is never stored as a float.

## 📦 Install

Grab the build for your platform from the assets below.

- **Debian / Ubuntu:** `sudo apt install ./Financy-v0.5.0-linux-amd64.deb`
- **Fedora / RHEL / openSUSE:** `sudo dnf install ./Financy-v0.5.0-linux-x86_64.rpm`
- **Other Linux:** extract `Financy-v0.5.0-linux-amd64.tar.xz` and run the app.
- **Windows:** run the `.exe` (SmartScreen → *More info → Run anyway*; unsigned).
- **macOS:** unzip, move to Applications, first launch **right-click → Open** (unsigned).

The full list of changes is in [CHANGELOG.md](../CHANGELOG.md).
