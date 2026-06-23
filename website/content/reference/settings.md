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
