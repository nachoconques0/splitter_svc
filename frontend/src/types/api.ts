/** The page branches on `error_code`, never on `message`. */
export interface ApiError {
  code: number
  error_code: string
  message: string
  detail?: string
}

/** Every API call resolves to one of these rather than throwing. */
export type Result<T> = { ok: true; data: T } | { ok: false; error: ApiError }

/** `total` is a decimal string, not a number. */
export interface Bill {
  id: string
  description: string
  total: string
  currency: string
  version: number
}

/** One Person's part of one Bill. `percentage` is a decimal string, as `total` is. */
export interface Share {
  person_id: string
  name: string
  percentage: string
}

export interface ShareSetResponse {
  bill: Bill
  shares: Share[]
}

export interface ShareInput {
  person_id: string
  percentage: string
}

export interface Person {
  id: string
  name: string
}

export interface PeopleResponse {
  people: Person[]
}

export interface ReplaceShareSetRequest {
  version: number
  shares: ShareInput[]
}
