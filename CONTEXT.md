# Bill Splitting

Splits a bill between people by percentage. One bill, a set of people, and the
rule that their percentages must account for the whole bill and nothing more.

## Language

### The bill

**Bill**:
Something owed in full, which several people account for between them.
_Avoid_: check, tab, expense, invoice

**Total**:
The whole amount of a Bill. The figure every Amount must add back up to.
_Avoid_: sum, price, cost

**Person**:
Someone who can appear on Bills. A Person exists in their own right: they are
not created by a Bill, and they outlive every Bill they appear on.
_Avoid_: user, account, participant, member, party

### Splitting

**Share**:
One Person's part of one Bill, expressed as a Percentage. A Share has no
meaning apart from the Bill it belongs to.
_Avoid_: split, portion, slice, entry, line

**Share Set**:
Every Share on a Bill, taken together. A Bill is only valid when its Share Set
accounts for exactly the whole Bill, so the Share Set is replaced as a whole and
never amended one Share at a time.
_Avoid_: shares list, participants, allocation list

**Percentage**:
How much of a Bill a Share claims. Percentages carry two decimal places, and a
Share Set's Percentages sum to exactly 100.00 or the Share Set is invalid.
_Avoid_: weight, ratio, fraction, proportion, split

**Amount**:
The money a Share comes to, once its Percentage is applied to the Total.
_Avoid_: owed, due, cost, value

### Resolving to money

**Allocation**:
Turning a Share Set and a Total into an Amount for each Share. Allocation is
constrained: the Amounts it produces must sum to exactly the Total, so it cannot
simply round each Share on its own.
_Avoid_: calculation, distribution, split, computation

**Remainder**:
The money left unaccounted for when every Share is rounded down to whole cents.
Allocation hands the Remainder out so that nothing is lost, which is why two
Shares with equal Percentages can differ by a cent.
_Avoid_: rounding error, leftover, drift, residue
