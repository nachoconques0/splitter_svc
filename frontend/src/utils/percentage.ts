import { formatDecimal, parseDecimal, SCALE } from '@/utils/decimal'

/**
 * Percentages, held the way the API holds them: integer hundredths of a percent,
 * so 100.00% is 10000 and a Share Set's total is an exact integer comparison.
 */

/** What a Share Set's Percentages must add up to: 100.00. */
export const WHOLE = 100 * SCALE

// Matches the Go side's int32, so the two agree on what is a Percentage at all.
const MAX_HUNDREDTHS = 2147483647

/** Reads "33.33" into hundredths of a percent, or null if it is not a Percentage. */
export function parsePercentage(text: string): number | null {
  return parseDecimal(text, MAX_HUNDREDTHS)
}

/** Formats hundredths of a percent for display: 3333 → "33.33%". */
export function formatHundredths(hundredths: number): string {
  return `${formatDecimal(hundredths)}%`
}
