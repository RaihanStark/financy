# 💰 Financy v0.5.1 — Maintenance release

A small housekeeping release on top of [v0.5.0](release-notes-v0.5.0.md). No
changes to the application itself, your data, or the `.financy` file format —
this is purely tooling and project metadata.

## 🔧 Changed

- **`make run-dev`** — run against an isolated dev profile so development never
  touches your real data. It repoints `$XDG_CONFIG_HOME` at `./.devdata/config`
  (gitignored), giving dev its own `prefs.json`, recent-files list, and `.financy`
  database; your `~/.config/financy` is left untouched. (Linux/BSD only — macOS
  ignores `XDG_CONFIG_HOME`.)
- **Funding & sponsorship** — added a GitHub Sponsors funding configuration and a
  "Support / sponsorship" section to the README.

## 📦 Install

Grab the build for your platform from the assets below.

- **Debian / Ubuntu:** `sudo apt install ./Financy-v0.5.1-linux-amd64.deb`
- **Fedora / RHEL / openSUSE:** `sudo dnf install ./Financy-v0.5.1-linux-x86_64.rpm`
- **Other Linux:** extract `Financy-v0.5.1-linux-amd64.tar.xz` and run the app.
- **Windows:** run the `.exe` (SmartScreen → *More info → Run anyway*; unsigned).
- **macOS:** unzip, move to Applications, first launch **right-click → Open** (unsigned).

The full list of changes is in [CHANGELOG.md](../CHANGELOG.md).
