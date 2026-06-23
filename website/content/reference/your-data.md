---
title: Your Data
weight: 2
---

Financy is **local-first**: there's no account, no cloud, no telemetry. Your data
is a file on your disk that you fully control.

## The `.financy` file

Each document is a single **SQLite database** with a `.financy` extension. Move it,
copy it, back it up, or keep it in a synced folder (while the app is closed). It's
a normal file.

## Always saved

There is **no Save button**. Every change is written through to the file
immediately with ACID guarantees, so a crash or power loss can't leave you with a
half-written document.

{{< callout type="warning" >}}
SQLite isn't designed for two machines writing the same file at once. Don't open
the same `.financy` from multiple devices simultaneously.
{{< /callout >}}

## Files menu

- **New…** — create a fresh document (asks for currency first).
- **Open…** / **Open Recent** — open existing files; the last file reopens on launch.
- **Save a Copy…** — write a clean snapshot elsewhere (e.g. a dated backup).
- **Setup Wizard…** — re-run the first-run wizard any time.

## Safety by design

- **Atomic writes** — the file is never left in a partial state.
- **Automatic backup on upgrade** — opening a file made by an older version writes
  a `.bak` copy *before* migrating it to the current schema.
- **Forward-compatibility guard** — Financy refuses to open a file written by a
  *newer* version than you're running (with a clear message), rather than risk
  misreading it. Update the app, then open the file.

## Backups

Because it's just a file: copy the `.financy` somewhere safe, or use **Save a
Copy…** periodically. That's a complete, portable backup of everything.
