---
title: Importing CSV
weight: 1
---

Bring in bank and credit-card exports instead of typing transactions. Financy's
**guided import** adapts to almost any CSV shape.

## Start an import

**File ▸ Import CSV…**, or the **Import CSV** button on the Transactions screen.
Pick the `.csv` file your bank exported.

{{< callout type="info" >}}
Don't have a file handy? The repo ships ten example exports in
[`sample_data/csv/`](https://github.com/raihanstark/financy/tree/main/sample_data/csv)
covering different delimiters, date formats, and amount styles.
{{< /callout >}}

## Step 1 — Map the columns

Financy reads the header row and **auto-detects** which column is the date, payee,
and amount, plus the date format. Adjust anything in the mapping dialog:

- **Into account** — the money account this file belongs to.
- **Date** + **Date format** (e.g. `MM/DD/YYYY`, `DD.MM.YYYY`, ISO, …).
- **Payee** and optional **Memo**.
- **Amounts** — either a single signed column, or separate **Debit/Credit**
  columns. Tick **flip the sign** if your export uses `+` for money spent (common
  on credit-card exports).
- **Default income / expense category** for rows it can't auto-categorize.

Delimiters (comma, semicolon, tab, pipe), quoted fields, accounting-style
`(123.45)` negatives, and a UTF-8 BOM are all handled automatically.

## Step 2 — Review & import

A preview lists every row with its parsed date, payee, amount, **category**, and
status. Before committing you can:

- **Set the category per row** from a dropdown.
- See **duplicates** flagged (a row whose date + payee + amount already exists in
  that account) — excluded by default so re-importing a file won't double-add.
- See unparseable rows flagged and skipped.

Click **Import** — everything commits in one atomic batch.

## Auto-categorization

Rows are categorized by matching the payee against transactions you've **already
categorized**:

1. **Exact** payee match (case/spacing-insensitive).
2. **Token / substring** match — so `AMAZON.COM*A1B2C` still matches a learned
   `Amazon`, and `WHOLE FOODS MARKET, NYC` matches `Whole Foods`.

Anything it can't match falls to **Uncategorized**, which you set via the per-row
dropdown (or later in the journal). The more you categorize, the smarter future
imports get.

## What's not supported yet

OFX/QIF formats, multi-account transfer detection, and importing into a
multi-currency document are not available yet — see the
[FAQ]({{< relref "faq" >}}) for current limitations.
