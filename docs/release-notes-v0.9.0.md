## Financy v0.9.0 — Financy on your phone 📱

This release brings **Financy to Android** — a touch-first mobile app built from
the same codebase as the desktop version, sharing the exact same double-entry
core and the same cross-platform `.financy` file format.

> **Beta.** The core bookkeeping works end-to-end; some desktop features aren't
> ported yet. See [MOBILE.md](https://github.com/raihanstark/financy/blob/main/MOBILE.md).

### 📱 Mobile app (Android)

- **Home, Transactions, Budget, Debts** as bottom tabs, each with full-screen
  **drill-down detail pages** and **add / edit / delete** throughout.
- **Account detail** — running-balance register, reconcile, edit, delete, and add
  a transaction pre-filled to that account.
- **Transaction detail** — a kind-tinted amount, a labelled breakdown, edit/delete.
- **Budget** — navigate months, tap a category to assign money (with last-month /
  spent / average quick-fills), and auto-assign; add categories.
- **Debts** — add a debt (with a generated installment schedule), pay/undo each
  installment, edit, and delete.
- A **context-aware ＋ button** (adds an account, transaction, category, or debt
  depending on the tab) and proper **Android Back** navigation.
- **Documents** — one auto-saved file in the app sandbox; open/export any
  `.financy` file (including password-protected ones), with per-session
  write-back sync to the file you opened — so one file in a cloud folder works on
  both desktop and phone.

### 🧱 Under the hood

- The mobile UI is a separate package (`internal/mobileui`) that shares **only**
  the domain/data core — the desktop UI is never linked into the APK.
- APKs are **16 KB-aligned** (Android 15 / Play requirement).

### 📥 Install

**Android:** download `Financy-v0.9.0-android-arm64.apk` below and sideload it
(you'll need to allow installs from your browser/file manager).

**Desktop:** grab the package for your OS below (`.deb` / `.rpm` /
`.tar.xz` / `.exe` / macOS `.zip`).

---

Not on mobile yet (available on desktop): Analytics, Recurring, transaction
filters/search, CSV import/export, password management, and preferences — see
[MOBILE.md](https://github.com/raihanstark/financy/blob/main/MOBILE.md).
