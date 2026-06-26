---
title: Getting Started
weight: 1
---

Financy is a desktop app. Your finances live in a single **`.financy`** file that
you own and can back up, move, or keep in a synced folder.

## Install

Download the build for your platform from the
[latest release](https://github.com/raihanstark/financy/releases/latest):

{{< tabs items="Linux,Windows,macOS" >}}
  {{< tab >}}Download `Financy-<version>-linux-amd64.tar.xz`, extract it, and run the app:
```sh
tar xf Financy-*-linux-amd64.tar.xz
./Financy
```{{< /tab >}}
  {{< tab >}}Run `Financy-<version>-windows-amd64.exe`. SmartScreen may warn (the build is unsigned): choose **More info → Run anyway**.{{< /tab >}}
  {{< tab >}}Unzip `Financy-<version>-macos.zip`, move `Financy.app` to Applications. First launch: **right-click → Open** (unsigned app).{{< /tab >}}
{{< /tabs >}}

{{< callout type="info" >}}
Builds are currently unsigned, so Windows and macOS may ask you to confirm before
the first run.
{{< /callout >}}

## First run

On first launch the **Setup Wizard** appears. Choose one:

- **Load demo data (USD)** — a complete worked example: ~6 months of transactions
  and a fully-assigned **zero-based budget**, so you can explore the Budget,
  Accounts, Transactions and Analytics immediately.
- **Start from scratch** — pick your **currency**, then choose where to save your
  new `.financy` file.

You can re-open the wizard any time from **File ▸ Setup Wizard…**, and create a
fresh document with **File ▸ New…** (which asks for the currency first).

## Build from source

You need **Go 1.25+** and a C toolchain + OpenGL/X11 headers (a
[Fyne](https://fyne.io) requirement). On Debian/Ubuntu:

```sh
sudo apt-get install gcc pkg-config libgl1-mesa-dev xorg-dev
go run .        # or: make run
```

## Next steps

- **[Budgeting]({{< relref "budget" >}})** — give every dollar a job with a zero-based budget.
- **[Core Concepts]({{< relref "concepts" >}})** — how double-entry keeps your books correct.
- **[Importing CSV]({{< relref "csv" >}})** — load real bank data instead of typing.
