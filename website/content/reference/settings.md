---
title: Settings
weight: 1
---

Open **Preferences** (toolbar gear) → **Configuration** tab.

## Currency

Pick the document's currency: **Rp · $ · € · £** (or another symbol). The currency
sets the **symbol** and the **number of decimals** (e.g. `$` has 2, `Rp` has 0).
The symbol is used everywhere amounts are shown.

## Number format

Choose how thousands and decimals are separated, independently of the currency:

| Style | Example |
| --- | --- |
| Currency default | follows the currency |
| `1,234.56` | comma thousands, dot decimal |
| `1.234,56` | dot thousands, comma decimal |
| `1 234,56` | space thousands, comma decimal |
| `1234.56` | no grouping |

This affects every amount shown in the app, and the **live formatting** as you
type in money fields.

## Live amount formatting

When you type in a money field — a transaction amount, an opening balance —
Financy groups thousands as you go (e.g. typing `1000000` shows `1,000,000`),
using your chosen number format. Decimals are capped to the currency's places.

{{< callout type="info" >}}
Currency and number format are saved **per document**, so different `.financy`
files can use different conventions.
{{< /callout >}}

## Password & encryption

Encryption is managed from the **File** menu, not Preferences:

- **Set Password…** — encrypt the open document with a passphrase. It immediately
  becomes a [manual-save](../your-data/#saving) document.
- **Change Password…** — re-key an already-encrypted document.
- **Remove Password…** — convert an encrypted document back to a plain file.

You can also set a password up front when creating a file (**Setup Wizard** or **File
▸ New…**). See [Your Data → Password encryption](../your-data/#password-encryption)
for how it works and the **no-backdoor** warning: a forgotten passphrase can't be
recovered.

## Checking for updates

Financy checks **GitHub Releases** for a newer version — at most once a day on
launch, and on demand via **File → Check for Updates…**. The check is a single
anonymous request to GitHub; it sends no document data.

When a newer release exists, a popup offers:

- **Download** — opens the release page in your browser so you can grab the new
  build for your platform.
- **Skip This Version** — silences the automatic prompt for that one version
  (a manual check still shows it).
- **Later** — dismiss; you'll be reminded on a future launch.

If you're offline the launch check fails silently. Financy never downloads or
replaces itself — you stay in control of installing the update.
