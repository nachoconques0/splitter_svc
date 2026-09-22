import { computed, ref, watch, type Ref } from 'vue'

import type { Person, ShareInput, ShareSetResponse } from '@/types/api'
import { allocate } from '@/utils/allocation'
import { parseMoney } from '@/utils/currency'
import { formatDecimal } from '@/utils/decimal'
import { formatHundredths, parsePercentage, WHOLE } from '@/utils/percentage'

export interface DraftShare {
  person_id: string
  name: string
  percentage: string
}

export interface DraftRow {
  share: DraftShare
  /** null while it cannot be worked out. */
  amount: string | null
}

/**
 * The Share Set as it is being edited, kept apart from the one that is stored so
 * a refused save can leave the edits alone. It applies the API's own rules, so
 * the page never offers to send something that would come back a 422.
 */
export function useShareSetDraft(stored: Ref<ShareSetResponse | null>) {
  const draft = ref<DraftShare[]>([])

  watch(stored, (shareSet) => {
    draft.value = (shareSet?.shares ?? []).map((share) => ({
      person_id: share.person_id,
      name: share.name,
      percentage: share.percentage,
    }))
  })

  const parsed = computed(() => draft.value.map((share) => parsePercentage(share.percentage)))

  /** null when any Percentage is unreadable: there is no honest total to show. */
  const runningTotal = computed<number | null>(() => {
    if (parsed.value.some((hundredths) => hundredths === null)) return null
    return parsed.value.reduce<number>((total, hundredths) => total + (hundredths ?? 0), 0)
  })

  // Per Share, not left to the total: 110.00 and -10.00 still add up to 100.00.
  const outOfRange = computed(() =>
    parsed.value.some((hundredths) => hundredths !== null && !isInRange(hundredths)),
  )

  const isWhole = computed(() => runningTotal.value === WHOLE)

  const runningTotalLabel = computed(() =>
    runningTotal.value === null ? '—' : formatHundredths(runningTotal.value),
  )

  const totalMinor = computed(() => parseMoney(stored.value?.bill.total ?? ''))

  /**
   * What each Share comes to, recomputed as Percentages are typed. Shown for any
   * readable Percentage, even while the set does not account for the Bill —
   * which is most of the time someone is editing. What the column adds up to is
   * shown beside it, so one that falls short says so.
   */
  const rows = computed<DraftRow[]>(() => {
    const readable = totalMinor.value !== null && parsed.value.every((h) => h !== null)
    if (!readable) {
      return draft.value.map((share) => ({ share, amount: null }))
    }

    const amounts = allocate(totalMinor.value!, parsed.value as number[])
    return draft.value.map((share, i) => ({ share, amount: formatDecimal(amounts[i]) }))
  })

  /** What the Amount column adds up to, or null while it cannot be worked out. */
  const allocatedTotal = computed<string | null>(() => {
    if (rows.value.some((row) => row.amount === null)) return null
    return formatDecimal(
      rows.value.reduce((total, row) => total + Number(row.amount!.replace('.', '')), 0),
    )
  })

  /** Empty when the Share Set could be saved; otherwise why it could not. */
  const blockedReason = computed(() => {
    if (runningTotal.value === null) return 'One of the percentages is not a number.'
    if (outOfRange.value) {
      return `A percentage has to be between ${formatHundredths(0)} and ${formatHundredths(WHOLE)}.`
    }
    if (!isWhole.value) {
      return `The percentages add up to ${formatHundredths(runningTotal.value)}. A bill is only divided between people when they total ${formatHundredths(WHOLE)}.`
    }
    return ''
  })

  /**
   * Puts a Person on the Bill at 0.00% — a real Share, and one that does not
   * change what anyone else owes. Someone already on the Bill is not added
   * again: this is the one place a Share is appended, so the guard belongs here
   * rather than only in the picker.
   */
  function add(person: Person): boolean {
    if (isOnTheBill(person.id)) return false

    draft.value.push({ person_id: person.id, name: person.name, percentage: formatDecimal(0) })
    return true
  }

  function isOnTheBill(personID: string): boolean {
    return draft.value.some((share) => share.person_id === personID)
  }

  /** Removes the Share, never the Person: they stay available for other Bills. */
  function remove(index: number) {
    draft.value.splice(index, 1)
  }

  /** The people not already on the Bill, so nobody is offered twice. */
  function notOnTheBill(people: Person[]): Person[] {
    return people.filter((person) => !isOnTheBill(person.id))
  }

  function toShareInputs(): ShareInput[] {
    return draft.value.map((share) => ({ person_id: share.person_id, percentage: share.percentage }))
  }

  return {
    draft,
    rows,
    allocatedTotal,
    runningTotal,
    runningTotalLabel,
    isWhole,
    blockedReason,
    add,
    remove,
    isOnTheBill,
    notOnTheBill,
    toShareInputs,
  }
}

/** A Quasar form rule, so a bad Percentage is marked on the field holding it. */
export function isAPercentage(value: string): true | string {
  const hundredths = parsePercentage(value)
  if (hundredths === null) return 'Two decimal places, e.g. 33.33'
  if (!isInRange(hundredths)) return `Between ${formatHundredths(0)} and ${formatHundredths(WHOLE)}`
  return true
}

function isInRange(hundredths: number): boolean {
  return hundredths >= 0 && hundredths <= WHOLE
}
