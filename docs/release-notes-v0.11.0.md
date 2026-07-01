## Financy v0.11.0 — Mobile forms & budget polish 📱

This release is all about the **mobile app**: deeper Settings, faster and less
error-prone data entry, and a clearer Budget — plus two important form fixes.

### ⚙️ Settings, reorganized

Settings is now a menu that drills into focused sections, mirroring the desktop
Preferences with a mobile-native UI:

- **Document** — open / export a `.financy` file, load demo data, or start fresh.
- **Configuration** — currency & number format.
- **Categories** — add / edit / delete income & expense labels.
- **Data summary** — account/category/transaction counts and totals.
- **About** — version & info.

### ✍️ Better data entry

- **Live amount formatting** — every money field (add/edit & bulk transaction,
  assign, reconcile, add debt) formats with thousands grouping as you type,
  following your number format.
- **Forms scroll over their fields** — dragging on a text field or dropdown now
  scrolls the form instead of selecting text / opening the dropdown, so long
  forms like Bulk add no longer misfire an edit when you meant to scroll.
- **Focused fields stay above the keyboard** — keyboard-avoidance now works for
  any field on any form, so nothing hides behind the on-screen keyboard.

### 💰 Budget

- The Budget tab now splits **Spending** categories and **Debt payment**
  envelopes into their own sections, matching the desktop layout.

### 📥 Install

**Android:** download `Financy-v0.11.0-android-arm64.apk` below and sideload it
(you'll need to allow installs from your browser/file manager).

**Desktop:** grab the package for your OS below (`.deb` / `.rpm` / `.tar.xz` /
`.exe` / macOS `.zip`).

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.10.0...v0.11.0
