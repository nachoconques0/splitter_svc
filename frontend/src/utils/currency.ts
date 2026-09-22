import { parseDecimal } from '@/utils/decimal'

/**
 * Money, in one place. Every component that shows an Amount or a Total comes
 * through here rather than reaching for Intl itself, so there is one answer to
 * what money looks like.
 */

/**
 * Reads money — "85.00" — into minor units, so the page can allocate Amounts for
 * unsaved edits. The bound is the safe-integer range rather than a Percentage's
 * narrower one, because the Go side holds money in an int64.
 */
export function parseMoney(decimal: string): number | null {
  return parseDecimal(decimal, Number.MAX_SAFE_INTEGER)
}

/**
 * Formats a decimal string as money in its currency: "85.00", "EUR" → "€85.00".
 *
 * The decimal string is handed to Intl as a *string*. That matters: Intl accepts
 * one and formats it exactly, where converting to a number first would put the
 * amount through a float. At 12345678901234567.89 the float route already answers
 * ...568.00, and the whole point of holding money as integers is that it never
 * gets the chance.
 */
export function formatMoney(decimal: string, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency,
    // To Intl's *string* overload, not to a number: TypeScript types it as a
    // template literal it cannot prove a plain string matches.
  }).format(decimal as Intl.StringNumericLiteral)
}
