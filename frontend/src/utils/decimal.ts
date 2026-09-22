/**
 * Decimal strings, which is how every number on this wire travels: "33.33" for a
 * Percentage, "85.00" for money. Both carry two places and are held as whole
 * counts of the smaller unit.
 *
 * These are JavaScript numbers, which are floats — but every value is a whole
 * count inside the safe-integer range and no decimal arithmetic happens on one.
 * 65.68 + 9.62 + 24.70 comes to 100.00000000000001; 6568 + 962 + 2470 does not.
 */

export const DECIMAL_PLACES = 2

/** One whole unit in the smaller unit: 100 hundredths, 100 minor units. */
export const SCALE = 100

/**
 * Reads a decimal string — "33.33", "25", "0.07" — into whole units of the
 * smaller unit, or null if it is not a decimal or is larger than `max`. A value
 * inside `max` but outside its own domain's range is left for the caller.
 *
 * `max` is the caller's: a Percentage and money are held in different widths on
 * the Go side, and each has to agree with its own counterpart.
 */
export function parseDecimal(text: string, max: number): number | null {
  const trimmed = text.trim()
  if (trimmed === '') return null

  const negative = trimmed.startsWith('-')
  const unsigned = negative ? trimmed.slice(1) : trimmed

  const parts = unsigned.split('.')
  if (parts.length > 2) return null

  const [whole, fraction] = parts
  if (!/^\d+$/.test(whole)) return null
  // More than two places is a value these types cannot hold.
  if (fraction !== undefined && !/^\d{1,2}$/.test(fraction)) return null

  const smaller = Number((fraction ?? '').padEnd(DECIMAL_PLACES, '0'))
  const value = Number(whole) * SCALE + smaller
  if (!Number.isSafeInteger(value) || value > max) return null

  return negative ? -value : value
}

/** Writes whole units of the smaller unit back as a decimal string: 3333 → "33.33". */
export function formatDecimal(scaled: number): string {
  const sign = scaled < 0 ? '-' : ''
  const digits = String(Math.abs(scaled)).padStart(DECIMAL_PLACES + 1, '0')
  return `${sign}${digits.slice(0, -DECIMAL_PLACES)}.${digits.slice(-DECIMAL_PLACES)}`
}
