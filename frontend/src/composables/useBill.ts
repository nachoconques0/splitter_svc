import { computed, ref } from 'vue'

import { API_ERROR_CODES, CLIENT_ERROR_CODES, clientError, toApiError } from '@/api/errors'
import { routes } from '@/api/routes'
import type { ApiError, Result, ShareInput, ShareSetResponse } from '@/types/api'

/**
 * Reads a Bill and the Share Set it owns. Returns a result rather than throwing,
 * so the page handles the failure as a value and cannot leave it unhandled.
 */
export function useBill(billId: string) {
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<ApiError | null>(null)
  const data = ref<ShareSetResponse | null>(null)

  async function load(): Promise<Result<ShareSetResponse>> {
    loading.value = true
    error.value = null

    try {
      const shareSet = await routes.shareSet(billId)
      data.value = shareSet
      return { ok: true, data: shareSet }
    } catch (cause) {
      const apiError = toApiError(cause)
      error.value = apiError
      // Cleared, so a failed reload cannot leave a stale Bill looking current.
      data.value = null
      return { ok: false, error: apiError }
    } finally {
      loading.value = false
    }
  }

  /**
   * Replaces the whole Share Set at the version last read. On failure it leaves
   * `data` alone: the page still needs the Bill, and the edits are its to keep.
   */
  async function save(shares: ShareInput[]): Promise<Result<ShareSetResponse>> {
    const current = data.value
    if (current === null) {
      // No version to save against until the Bill has been read.
      const apiError = clientError(CLIENT_ERROR_CODES.unexpected, 'The bill has not been loaded yet.')
      error.value = apiError
      return { ok: false, error: apiError }
    }

    saving.value = true
    error.value = null

    try {
      const shareSet = await routes.replaceShareSet(billId, {
        version: current.bill.version,
        shares,
      })
      data.value = shareSet
      return { ok: true, data: shareSet }
    } catch (cause) {
      const apiError = toApiError(cause)
      error.value = apiError
      return { ok: false, error: apiError }
    } finally {
      saving.value = false
    }
  }

  /** The Bill moved on under the page, which is answered with a reload offer. */
  const conflicted = computed(() => error.value?.error_code === API_ERROR_CODES.billModified)

  return { loading, saving, error, conflicted, data, load, save }
}
