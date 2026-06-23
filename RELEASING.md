# Maintaining & Releasing Financy

## Project layout
- `main.go` — entry point; embeds `Icon.png`, calls `ui.Run`.
- `internal/core` — domain model, double-entry store, SQLite persistence, formatting. **No UI deps; this is where logic + tests live.**
- `internal/ui/style` — palette + theme.
- `internal/ui/component` — reusable widgets (cards, table, rows, app bar…).
- `internal/ui/view` — the screens (Accounts, Transactions) + Preferences + forms.
- `internal/ui` — app shell, controller, toolbar, File menu, document manager.

## Day-to-day
```
make run         # run the app
make check       # build + vet + test (run before every commit)
make shot        # regenerate UI screenshots into /tmp/financy-shots
```
Add changes on a branch; add a test in `internal/core` for any new logic there.
When a change lands on `main`, add a one-line entry under `## [Unreleased]` in
`CHANGELOG.md` — that running list is how you tell what the next release contains
(an **Added** entry ⇒ minor bump; only **Fixed**/**Changed** ⇒ patch).

## Two rules that protect user data
1. **Migrations are append-only.** Schema changes go in `internal/core/db.go` as a NEW entry in the `migrations` slice — never edit an existing one. `PRAGMA user_version` upgrades old `.financy` files on open (a `.bak` is written first). Test by opening a file made with the previous version.
2. **Money stays integer minor units (cents).** Never use floats for money.

## Releasing
Versioning is semantic (`MAJOR.MINOR.PATCH`). One command stamps both the in-app
version (`internal/core.Version`) and the packaging metadata (`FyneApp.toml`):

```
# In CHANGELOG.md: rename "## [Unreleased]" to "## [0.2.0] - YYYY-MM-DD",
#   add a fresh empty [Unreleased] above it, and update the compare links.
make release VERSION=0.2.0      # stamps, runs check, builds ./financy
# Write docs/release-notes-v0.2.0.md  (the polished, user-facing Release body —
#   expand the changelog's 0.2.0 section with install steps, etc.)
git add -A && git commit -m "Release v0.2.0"
git push
git tag v0.2.0 && git push origin v0.2.0
```

> Two files, two audiences: `CHANGELOG.md` is the terse running log you append to
> per change; `docs/release-notes-<tag>.md` is the polished GitHub Release body,
> written once at release time from the changelog's section for that version.

> The release workflow reads `docs/release-notes-<tag>.md` as the release body, so
> create that file before tagging (a missing file fails the publish step).

Pushing the tag triggers `.github/workflows/release.yml`, which packages
Linux / Windows / macOS bundles (native runners + the fyne CLI) and attaches
them to the GitHub Release.

### Packaging locally (optional)
```
go install fyne.io/fyne/v2/cmd/fyne@latest   # once
make package                                  # bundle for your current OS
```

## CI
- `.github/workflows/ci.yml` — vet + test + build on every push to `main` and PR.
- `.github/workflows/release.yml` — on `v*` tags, build & publish the bundles.
