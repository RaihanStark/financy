# Changelog

All notable changes to Financy are recorded here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

How to use this file: add one line under `Unreleased` for every change that lands
on `main`. An entry under **Added** means the next release is a minor bump
(`0.x.0`); only **Changed** / **Fixed** entries mean a patch (`0.0.x`). At release
time, rename `Unreleased` to the new version + date and start a fresh empty block.
See `RELEASING.md`.

## [Unreleased]

### Added
- Refuse to open a `.financy` file whose schema is newer than the running app,
  with a clear "update Financy" dialog, instead of risking a misread. The file is
  left untouched.

## [0.1.0] - 2026-06-22

Initial release — a fast, local-first personal finance manager built on real
double-entry accounting, with data in a single `.financy` file you own.

### Added
- Double-entry accounting core: every transaction is balanced postings summing to
  zero; balances and net worth are derived, never stored.
- Integer-minor-unit (cents) money handling with formatting for Rp · $ · € · £ (no
  floating-point money).
- Account cards for assets & liabilities, a net-worth overview, and a per-account
  register with running balance.
- Date-grouped transaction journal with an Income / Expense / Transfer entry form,
  plus filters and search.
- Income & expense categories, managed from Preferences.
- Desktop experience: right-click context menus, hover tooltips, an icon toolbar,
  and a minimal File menu.
- First-run setup: load demo data (USD) or start from scratch with a chosen currency.
- Single-file `.financy` SQLite documents with always-on auto-save (ACID writes).
- New / Open / Open Recent / Save a Copy; reopens the last file on launch.
- Safe-by-design persistence: atomic writes, append-only schema migrations, and an
  automatic `.bak` before any file upgrade.
- Cross-platform release bundles (Linux / Windows / macOS) published from CI on tag.

[Unreleased]: https://github.com/raihanstark/financy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/raihanstark/financy/releases/tag/v0.1.0
