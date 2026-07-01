# Financy on Mobile (Android)

Financy runs on Android from the **same codebase** as the desktop app. It is a
separate, touch-first UI (`internal/mobileui`) that shares only the domain/data
core (`internal/core`) with the desktop shell — the desktop UI is **not** shipped
in the APK.

> **Status: early / beta.** The core bookkeeping works end-to-end, but the mobile
> app does **not** yet have every desktop feature. This page tracks what you can
> expect today.

## What works today

The mobile app has four bottom tabs — **Home**, **Transactions**, **Budget**,
**Debts** — plus full drill-down detail pages. The floating **＋** button is
context-aware: it adds the thing that fits the current tab.

| Area | What you can do on mobile |
|------|---------------------------|
| **Home (dashboard)** | Net-worth hero, account cards, recent activity. ＋ adds an **account**. |
| **Accounts** | Tap an account → detail page with a running-balance **register**, **reconcile**, **edit**, **delete**, and add a transaction pre-filled to that account. |
| **Transactions** | Full list, **add / edit / delete**, and a detail page for each. ＋ adds a **transaction**. |
| **Budget** | Navigate **months** (◀ ▶), tap a category to **assign** money (with last-month / spent / average quick-fills), **auto-assign** last month's amounts. ＋ adds a **category**. |
| **Debts** | Add a debt (full schedule), tap a debt → detail with the installment **schedule** you can **pay / undo**, plus **edit** and **delete**. ＋ adds a **debt**. |
| **Documents** | One auto-saved document per device. **Open** a `.financy` file (incl. password-protected — you'll be prompted to unlock), **export a copy**, load **demo data**, or **start fresh**. |

### Cross-platform files & sync

A `.financy` file is **byte-identical** on desktop and mobile (a SQLite database,
or an encrypted blob), so you can keep one file in a cloud folder and open it on
both. When you open a file on the phone, edits are **written back to that file**
while the app is open. Because Android grants file access per session, you
**re-open the file after restarting** to resume syncing. It's last-write-wins,
not real-time multi-device sync.

## Not on mobile yet (available on desktop)

These desktop features are **not implemented** on mobile at this time:

- **Analytics** — the charts screen (net worth over time, income vs expense,
  category breakdown).
- **Recurring** — recurring-transaction templates and posting due entries.
- **Transaction filters & search** — the phone lists all transactions; there's no
  month/type/account filter or payee/memo search yet.
- **Bulk add** — the multi-row fast-entry form.
- **CSV import / export** — mobile only opens/exports `.financy` files.
- **Password management** — you can *open* an already-encrypted file, but you
  can't yet set / change / remove a passphrase on the phone's document.
- **Preferences** — currency, owner, number format, and category management
  (rename / delete / reorder).
- **Per-installment editing** — you can pay/undo installments but not edit an
  individual installment's due date or amount.

(Update checking is desktop-only by design — app stores handle mobile updates.)

## Installing the APK

Pushing a version tag (`v*`) builds a 16 KB-aligned arm64 APK in CI and attaches
it to the **[GitHub Release](../../releases)** (`.github/workflows/release.yml`,
the `android` job). Download `Financy-<version>-android-arm64.apk` and sideload
it (you'll need to allow installs from your browser/file manager). See the
workflow header for the signing secrets that make the APK upgradeable in place.

## Building the APK yourself

Requires the `fyne` CLI and the Android SDK + NDK (`ANDROID_HOME` /
`ANDROID_NDK_HOME` set). Then:

```bash
make apk ANDROID_ARCH=android/arm64      # builds a 16 KB-aligned, arm64 APK
adb install -r Financy.apk               # install on a connected device
```

To preview the mobile UI on the desktop without a device:

```bash
FINANCY_MOBILE=1 go run .
```

The APK auto-launches the touch UI via `fyne.CurrentDevice().IsMobile()`.
