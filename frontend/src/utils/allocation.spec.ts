import { describe, expect, it } from 'vitest'

import { allocate } from '@/utils/allocation'
import { formatDecimal } from '@/utils/decimal'

/**
 * The same cases, with the same expected Amounts, as the Go table in
 * backend/internal/controller/bill/allocation_test.go. The algorithm is written
 * twice on purpose, so a case added to one table and not the other proves
 * nothing — keep them in step.
 */
const cases = [
  {
    name: 'shares that divide the total evenly leave nothing over',
    totalMinor: 10000,
    percentages: [2500, 2500, 2500, 2500],
    wantAmounts: ['25.00', '25.00', '25.00', '25.00'],
  },
  {
    // Rounding each Share alone gives 3.33 three times, or 9.99 of 10.00.
    name: 'a remainder is handed to the largest fractional part',
    totalMinor: 1000,
    percentages: [3333, 3333, 3334],
    wantAmounts: ['3.33', '3.33', '3.34'],
  },
  {
    name: 'a tie on the fractional parts goes to the earlier share',
    totalMinor: 3,
    percentages: [5000, 5000],
    wantAmounts: ['0.02', '0.01'],
  },
  {
    name: 'a single share at 100% takes the whole total',
    totalMinor: 8500,
    percentages: [10000],
    wantAmounts: ['85.00'],
  },
  {
    name: 'a share at 0% is never handed a cent',
    totalMinor: 10,
    percentages: [0, 3333, 3333, 3334],
    wantAmounts: ['0.00', '0.03', '0.03', '0.04'],
  },
  {
    name: 'the seeded bill',
    totalMinor: 8500,
    percentages: [3333, 3333, 3334],
    wantAmounts: ['28.33', '28.33', '28.34'],
  },
]

describe('allocate', () => {
  it.each(cases)('$name', ({ totalMinor, percentages, wantAmounts }) => {
    expect(allocate(totalMinor, percentages).map(formatDecimal)).toEqual(wantAmounts)
  })

  // The property the whole rule exists for: no cent invented, none lost.
  it.each(cases)('$name — the amounts sum to the total', ({ totalMinor, percentages }) => {
    const summed = allocate(totalMinor, percentages).reduce((total, amount) => total + amount, 0)
    expect(summed).toBe(totalMinor)
  })

  it('leaves two shares on equal percentages within a cent of each other', () => {
    const [first, second] = allocate(10007, [3333, 3333, 3334])
    expect(Math.abs(first - second)).toBeLessThanOrEqual(1)
  })

  // The page allocates a Share Set that is still being typed. Flooring it is
  // honest; handing out a Remainder that is not rounding would not be.
  it('floors a share set that does not account for the whole bill', () => {
    expect(allocate(8500, [5000, 2500]).map(formatDecimal)).toEqual(['42.50', '21.25'])
  })

  // A Total the Go side handles in an int64 must not be one this page gives up
  // on because a Percentage happens to be held in an int32.
  it('allocates a total far beyond a percentage\'s range', () => {
    expect(allocate(5_000_000_000, [3333, 3333, 3334]).map(formatDecimal)).toEqual([
      '16665000.00',
      '16665000.00',
      '16670000.00',
    ])
  })
})
