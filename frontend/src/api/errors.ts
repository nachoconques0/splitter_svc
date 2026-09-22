import axios from 'axios'

import type { ApiError } from '@/types/api'

/** Codes the page does more with than display. */
export const API_ERROR_CODES = {
  /** The Bill moved on since it was read, so the save was refused. */
  billModified: 'bill_modified',
} as const

/** Codes raised by the client itself, for failures that never reached the API. */
export const CLIENT_ERROR_CODES = {
  network: 'network_unavailable',
  unexpected: 'unexpected_error',
} as const

/**
 * An envelope for a failure the API never answered. It lives here because this
 * module owns the shape. `code` is 0: there was no response to take a status from.
 */
export function clientError(errorCode: string, message: string): ApiError {
  return { code: 0, error_code: errorCode, message }
}

/**
 * Normalises anything axios throws into one envelope, so callers branch on one
 * shape. The server's message is preferred where there is one; a raw axios
 * string is never rendered.
 */
export function toApiError(cause: unknown): ApiError {
  if (axios.isAxiosError(cause)) {
    const envelope = cause.response?.data as Partial<ApiError> | undefined

    if (envelope && typeof envelope.error_code === 'string' && typeof envelope.message === 'string') {
      return {
        code: envelope.code ?? cause.response?.status ?? 0,
        error_code: envelope.error_code,
        message: envelope.message,
        detail: envelope.detail,
      }
    }

    if (!cause.response) {
      return clientError(
        CLIENT_ERROR_CODES.network,
        'Could not reach the server. Check that it is running and try again.',
      )
    }

    return {
      ...clientError(CLIENT_ERROR_CODES.unexpected, 'The server returned an unexpected response.'),
      code: cause.response.status,
    }
  }

  return clientError(CLIENT_ERROR_CODES.unexpected, 'Something went wrong.')
}
