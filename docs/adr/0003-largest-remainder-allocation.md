# Allocation uses largest remainder, and exists in both Go and TypeScript

Rounding each Share independently loses money: three Shares of 33.33/33.33/33.34
against a 10.00 Total round to 3.33 each and account for only 9.99. Allocation
floors every Share to whole cents and then hands the Remainder to the Shares with
the largest fractional parts, so the Amounts always sum to the Total exactly.

## Consequences

The rule lives in the Go domain and is returned by the API, but the UI has to
show Amounts for *unsaved* edits, before anything has been sent. So the same
algorithm is implemented a second time in TypeScript.

That duplication is deliberate, and it is the alternative we disliked least. A
preview endpoint would have kept one source of truth at the cost of a network
round trip per keystroke; letting the UI round naively until save would have made
the numbers visibly jump when the user hit save, which reads as a bug. The two
implementations are kept honest by running the same test cases against both.
