---
title: Core Concepts
weight: 2
---

Financy is built on **real double-entry accounting** — the same model used by
professional ledgers. Understanding a few ideas makes everything else click.

## Double-entry

Every transaction is made of **postings** — signed amounts against accounts — that
**must sum to zero**. Money always comes *from* somewhere and goes *to* somewhere.

For example, paying $84.20 for groceries from checking is two postings:

| Account | Amount |
| --- | ---: |
| Groceries (expense) | +84.20 |
| Checking (asset) | −84.20 |

They cancel out to zero, so the books always balance. Financy writes these
postings for you — you just pick **Income / Expense / Transfer** in the entry form.

{{< callout type="info" >}}
Financy rejects any transaction whose postings don't sum to zero. This is what
guarantees your books can never silently drift.
{{< /callout >}}

## Balances are derived, never stored

Financy never stores a "balance" field that could get out of sync. Every balance,
total, and net-worth figure is **recomputed by summing postings**. Delete or edit a
transaction and everything downstream updates automatically and correctly.

## Account types

| Type | Examples | Normal direction |
| --- | --- | --- |
| **Asset** | Checking, Savings, Cash | + increases |
| **Liability** | Credit card, Loan | + increases what you owe |
| **Income** | Salary, Interest | source of money |
| **Expense** | Groceries, Rent | use of money |
| **Equity** | Opening Balances | starting values |

Assets and liabilities are your **money accounts** (real balances). Income and
expense accounts are your **categories**. Net worth = total assets − total
liabilities.

## Money is stored as integer cents

Amounts are kept as **integer minor units** (cents for USD, whole rupiah for Rp),
never floating-point. This avoids rounding errors entirely. How they're displayed
(symbol, decimals, separators) is controlled by [Settings]({{< relref "settings" >}}).

## Opening balances

When you create an account with a starting balance, Financy posts a real
**Opening Balance** transaction against the Equity account — it doesn't just "set"
a number. That keeps the double-entry invariant intact from day one.
