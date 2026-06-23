---
title: FAQ & Troubleshooting
weight: 1
---

## Why does the installer/app warn that it's unsigned?

Current builds aren't code-signed, so Windows SmartScreen and macOS Gatekeeper may
warn on first launch. On Windows choose **More info → Run anyway**; on macOS
**right-click → Open** the first time. See [Getting Started]({{< relref "getting-started" >}}).

## Can I use more than one currency in a document?

Not yet — each `.financy` file is single-currency. You can keep separate documents
for separate currencies. Multi-currency / FX within one file is a possible future
feature.

## My credit-card import has everything backwards (income vs expense).

Some card exports use **positive numbers for charges**. In the import mapping step,
tick **"My export uses + for money spent (flip the sign)."** See
[Importing CSV]({{< relref "csv" >}}).

## Re-importing a file — will it duplicate transactions?

No. Import flags rows that match an existing entry (same date + payee + amount in
that account) as **duplicates** and excludes them by default.

## Where is my data stored?

In the `.financy` file you chose when creating the document. On launch Financy
reopens your last file. See [Your Data]({{< relref "your-data" >}}).

## Can two people / machines edit the same file?

Not simultaneously — it's a single-writer SQLite file. Use one machine at a time.

## How do I report a bug or request a feature?

Open an issue on [GitHub](https://github.com/raihanstark/financy/issues).

## Where's the changelog?

See [Releases](https://github.com/raihanstark/financy/releases) and
[CHANGELOG.md](https://github.com/raihanstark/financy/blob/main/CHANGELOG.md).
