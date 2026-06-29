# 💰 Financy v0.7.0 — Faster bookkeeping & update checks

This release speeds up data entry and helps you stay current. Enter a backlog of
transactions in a single pass with the new **bulk add** form, and let Financy tell
you when a new version is out — on your terms, never installing anything on its own.

## ✨ Highlights

**⚡ Bulk add for faster bookkeeping**
- A new bulk-entry form with two appendable tables — income/expense rows and
  transfers — so you can knock out a backlog in one sitting.
- Every row carries its own **date**, money account, amount, category and payee, so
  entries spanning several days go in together; a date picker keeps dates tidy.
- Saving commits every valid row in a **single batch** instead of one dialog per
  transaction.

**🔔 Update checks**
- Financy now checks GitHub Releases for a newer version once a day on launch, or on
  demand via **File → Check for Updates…**, and shows a popup with a **Download**
  link.
- It **never** downloads or replaces the binary itself — you decide when and how to
  install.
- The check is an anonymous request to GitHub and sends **none of your data**. Use
  **Skip This Version** to silence the prompt for a release you're not ready for.

## 🔧 Other changes

- **Toolbar +** now consistently opens the **Add Transaction** form on every screen
  (it previously changed meaning per page). Per-screen adds remain as dedicated
  buttons within those screens.
- **Pre-upgrade backups** now embed the document's schema version in their name
  (e.g. `My Budget.financy.v6.bak`), so you can tell which earlier Financy version
  still opens a backup if you ever need to downgrade.
- A journal row with no payee now shows the transaction type (e.g. **Expense**,
  **Transfer**) instead of the category name, so the row title reads consistently.

## 🗑️ Removed

- The **Reports** screen (Income Statement, Balance Sheet, Cash Flow) has been
  removed — its figures were purely derived from the journal and overlapped with
  Analytics.

## 📦 Install

Grab the build for your platform from the assets below.

- **Debian / Ubuntu:** `sudo apt install ./Financy-v0.7.0-linux-amd64.deb`
- **Fedora / RHEL / openSUSE:** `sudo dnf install ./Financy-v0.7.0-linux-x86_64.rpm`
- **Other Linux:** extract `Financy-v0.7.0-linux-amd64.tar.xz` and run the app.
- **Windows:** run the `.exe` (SmartScreen → *More info → Run anyway*; unsigned).
- **macOS:** unzip, move to Applications, first launch **right-click → Open** (unsigned).

The full list of changes is in [CHANGELOG.md](../CHANGELOG.md).
