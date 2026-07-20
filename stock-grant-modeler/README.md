# Stock Grant Modeler

A single-file, dependency-free tool for modeling annual private-company stock
grants that come with a vesting schedule and an employer loan. Open
`index.html` in any browser — no build step, no network access needed.

## What it models

Each grant card represents one year's offer:

- **Purchase**: N shares at that year's board-set price.
- **Vesting**: 25% per year on September 30, starting the year after the
  grant; tranches round down to whole shares, the final date picks up the
  remainder.
- **Financing**: pay in full, or a down payment plus an employer loan for the
  remainder. Loan interest accrues daily (actual/365) from September 30 of the
  grant year, is due each December 31 (first payment the following year), and
  can either be paid in cash or capitalized into principal at a separate rate.
  Principal is a single balloon due June 30, nine years after the grant.

Outputs: net walk-away value over time (vested shares at the modeled price,
unvested at cost, minus loan payoff), shares vesting per year, annual
out-of-pocket cash, and a year-by-year table.

The default numbers are placeholders — enter your own offer's figures.
Inputs persist in the browser's localStorage; nothing leaves the page.

Not modeled: taxes on gains at sale, payroll-deduct imputed interest, tax
loans, share-exchange down payments, and minimum-redemption rules.
