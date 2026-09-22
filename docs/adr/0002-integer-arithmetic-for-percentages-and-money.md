# Percentages and money are held as integers

The one hard rule in the brief is that a Share Set's Percentages sum to *exactly*
100.00. We hold Percentages as integer hundredths of a percent and money as
integer minor units, so that rule is an exact integer comparison against 10000
with no tolerance value anywhere in the codebase.

## Considered options

`NUMERIC(5,2)` in Postgres is genuinely exact and would also have worked. We
passed on it because exactness would then hold only in the database: the value
still has to cross into Go, where it becomes a float or a decimal library, and
the guarantee is only as good as its weakest hop. Integers are exact at every
hop. `float64` was never viable: most percentages have no exact representation in
binary, so whether a Share Set sums to 100 comes down to how the error in each
one happens to fall. 65.68 + 9.62 + 24.70 is a valid Share Set and adds up to
100.00000000000001, while 33.33 + 33.33 + 33.34 lands on exactly 100 — and
nothing about the two tells you which will be which. As integers there is no
question to ask.

## Consequences

Basis points and minor units are internal. The API speaks decimal strings
(`"33.33"`, `"25.00"`) and parses inbound numbers as `json.Number`, digit by
digit, so no float ever touches a Percentage or an Amount even in transit.
