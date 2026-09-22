import { WHOLE } from '@/utils/percentage'

/**
 * Allocation: turning a Share Set and a Total into an Amount for each Share.
 *
 * The same rule as Bill.Allocate in Go, written a second time on purpose: the
 * page shows what a Percentage comes to while it is being typed, and the
 * alternatives were a round trip per keystroke or numbers that jump on save. One
 * table of cases runs against both — see allocation.spec.ts.
 */

/**
 * Allocates a Total, in minor units, across Percentages in hundredths of a
 * percent, answering one Amount per Percentage in the same order.
 *
 * Every Share is floored, then the remainder handed out a unit at a time to the
 * largest fractional parts, so the Amounts sum to the Total exactly. Ties go to
 * the earlier Share, so the same Share Set allocates the same way every time.
 *
 * A Share Set that does not total WHOLE is floored and left there: there is no
 * Total for its Amounts to add up to. The page allocates one while it is still
 * being typed.
 */
export function allocate(totalMinor: number, hundredths: number[]): number[] {
  const amounts: number[] = []
  const fractions: { position: number; size: number }[] = []

  let floored = 0
  let claimed = 0
  hundredths.forEach((percentage, position) => {
    const [amount, fraction] = share(totalMinor, percentage)

    amounts.push(amount)
    fractions.push({ position, size: fraction })
    floored += amount
    claimed += percentage
  })

  if (claimed !== WHOLE) return amounts

  // Always smaller than the number of Shares.
  const remainder = totalMinor - floored

  fractions.sort((a, b) => b.size - a.size)
  for (let i = 0; i < remainder; i += 1) {
    amounts[fractions[i].position] += 1
  }

  return amounts
}

/**
 * What one Percentage of a Total comes to, and what was rounded away, scaled by
 * WHOLE. The multiplication is split around the division the same way the Go
 * side splits it: done first it leaves the safe-integer range above about nine
 * billion euros.
 */
function share(totalMinor: number, hundredths: number): [amount: number, fraction: number] {
  const units = Math.floor(totalMinor / WHOLE)
  const part = totalMinor % WHOLE

  const scaled = part * hundredths
  return [units * hundredths + Math.floor(scaled / WHOLE), scaled % WHOLE]
}
