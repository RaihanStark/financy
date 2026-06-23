---
title: Transactions
weight: 2
---

The **Transactions** screen is a date-grouped journal of every entry, with filters
and search.

![Transactions screen](../../img/transactions.png)

## Adding a transaction

Click **Add Transaction** (or the toolbar **+**, or right-click an account →
*New Transaction*). Pick a **type** and Financy writes the correct double-entry
postings:

| Type | Money flow | Postings created |
| --- | --- | --- |
| **Expense** | money out of an account | category +amt, account −amt |
| **Income** | money into an account | account +amt, category −amt |
| **Transfer** | between two of your accounts | to +amt, from −amt |

Other fields: **Date** (with a calendar picker — type `YYYY-MM-DD` or click the
calendar button), **Amount** (formats with thousands separators as you type),
**Payee**, and an optional **Memo**.

## Filtering & search

Narrow the journal by **month**, **type**, **account**, or free-text **search**.
The summary cards update to show income, expense, and net for the current filter.

## Editing & deleting

Right-click any row (or use its menu) to **edit** or **delete**. Because balances
are derived, edits and deletes immediately and correctly update every total.

## Splits & transfers

The underlying model supports multi-posting transactions (e.g. one purchase across
several categories). The standard form covers Income/Expense/Transfer; richer split
editing may come in a future release.
