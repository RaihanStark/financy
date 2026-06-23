# Sample bank CSVs

Example bank/card exports for trying **File ▸ Import CSV…** (or the *Import CSV*
button on the Transactions screen). Each file is shaped like a different real bank
export so you can see how the guided import adapts — different delimiters, date
formats and amount conventions.

Tip: amounts use **two decimal places**, so import into a `$` / `€` / `£` document
for matching results (a `Rp` document has no decimals).

| File | What it demonstrates | How to import |
| --- | --- | --- |
| `01_us_checking_signed.csv` | Comma, `MM/DD/YYYY`, one signed Amount column, quoted fields, **UTF-8 BOM** (as Excel saves) | Auto-detected — just confirm |
| `02_uk_current_debit_credit.csv` | Comma, `DD/MM/YYYY`, **separate Debit/Credit columns**, a Balance column | Auto-detected (debit/credit mode) |
| `03_euro_semicolon.csv` | **Semicolon** delimiter, `DD.MM.YYYY`, European numbers (`1.150,00`) | Auto-detected |
| `04_credit_card_positive_charges.csv` | Comma, `MM/DD/YYYY`, **positive = a charge** (spending), extra Category column | Tick **"My export uses + for money spent (flip the sign)"** |
| `05_parentheses_negatives.csv` | Comma, `YYYY-MM-DD`, **accounting-style `(123.45)`** for money out | Auto-detected |
| `06_iso_dates_with_memo.csv` | Comma, `YYYY-MM-DD`, signed Amount, a **Memo** column | Auto-detected (optionally map Memo) |
| `07_tab_delimited.csv` | **Tab**-delimited, `MM/DD/YYYY`, signed Amount | Auto-detected |
| `08_pipe_debit_credit.csv` | **Pipe** (`|`) delimited, `DD/MM/YYYY`, Debit/Credit columns | Auto-detected (debit/credit mode) |
| `09_no_header.csv` | **No header row** — raw data only, `YYYY-MM-DD`, signed | **Untick "First row is a header"**, then map Date/Payee/Amount by position |
| `10_pfm_extra_columns.csv` | Comma, `MM/DD/YYYY`, signed Amount among **extra columns** (Category, Account, Notes) | Auto-detected (extra columns ignored) |

## What import does

Each row becomes a balanced **double-entry** transaction in the account you pick:
money out → an expense category, money in → an income category (both default to
**Uncategorized**, which you can re-assign later in the journal). Rows that look
like an entry already in that account are flagged as **duplicates** and left
unchecked, so re-importing the same file won't double-add. Rows whose date or
amount can't be read are flagged and skipped.

These files are also used as fixtures by the import tests in
`internal/core/csvimport_test.go`.
