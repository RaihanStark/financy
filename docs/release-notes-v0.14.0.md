## Financy v0.14.0 — A redesigned desktop & recurring that looks ahead ✨

The desktop app got a full facelift — a modern sidebar shell, a refreshed
design system across both themes, and custom modals — and the Recurring
feature was rebuilt to actually help: it now shows what's coming and what it
costs, instead of just nagging about what's overdue. Your existing `.financy`
files open unchanged.

### 🔁 Recurring, forward-looking

- The Recurring screen now leads with your **commitments**: MONTHLY BILLS,
  MONTHLY INCOME, NET PER MONTH and ACTIVE stat cards, with every frequency
  (weekly, biweekly, monthly, quarterly, yearly) normalized to a monthly
  figure.
- A **Next 30 days timeline** projects every upcoming occurrence, bucketed
  into *Overdue & due* (amber, with an inline **Review & Post**), *This week*
  and *Later this month* — each row saying plainly "today", "in 5 days", or
  "3 days overdue".
- The **Accounts overview** gains an **Upcoming** card: anything due plus the
  next 14 days of bills, one click from the full Recurring screen.
- Financy now **re-checks the schedule while the app runs** (about once a
  minute, midnight rollover included) — not just when a file opens. The
  review prompt only reappears when something *new* becomes due, so **Later**
  actually means later.
- **Nothing changed about control:** every entry is still drafted for your
  review — recurring never posts silently, and duplicate detection still
  links occurrences to payments you already recorded.

### 🎨 Desktop redesign

- **A modern shell** — the icon-only toolbar and status bar are gone;
  navigation lives in a labeled left sidebar with a prominent **New
  Transaction** button, quick utilities, and a live net-worth summary pinned
  at the bottom (it steps aside entirely when no file is open).
- **A refreshed design system** in both themes: indigo accent, calmer
  emerald/rose money colors, a deeper dark palette, softer corners, airier
  headers and stat cards.
- **Every screen rebuilt on it** — Accounts is a scannable grouped table with
  share bars; Transactions groups each day into its own card with a one-line
  filter bar; Budget rows carry traffic-light usage bars and an overspend
  banner; Recurring as above.
- **Custom modals everywhere**: dialogs moved off Fyne's stock boxes onto
  centered rounded cards with proper headers and footer actions; delete
  confirmations use an explicit red **Delete** button.

### 💰 Budget: cover overspending in place

- An overspent envelope can be fixed where you see it: **Cover…** moves money
  from another envelope (or Ready to Assign), with sources sorted by what
  they can spare and the amount prefilled with the shortfall.

### 🧹 Housekeeping

- The release no longer ships an unsigned iOS `.ipa` (it couldn't be
  installed without re-signing weekly). Android, Linux, Windows and macOS
  assets are unchanged.

### 📥 Install

**Android:** download `Financy-v0.14.0-android-arm64.apk` below and sideload it
(you'll need to allow installs from your browser/file manager).

**Desktop:** grab the package for your OS below (`.deb` / `.rpm` / `.tar.xz` /
`.exe` / macOS `.zip`).

---

**Full changelog:** https://github.com/raihanstark/financy/compare/v0.13.0...v0.14.0
