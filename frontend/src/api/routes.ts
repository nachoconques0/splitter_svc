import { client } from '@/api/client'
import type { PeopleResponse, Person, ReplaceShareSetRequest, ShareSetResponse } from '@/types/api'

/**
 * Every endpoint the page knows about, in one place. Components and composables
 * call these functions and never build a URL or touch axios themselves.
 */
export const routes = {
  shareSet: (billId: string) =>
    client.get<ShareSetResponse>(`/bills/${encodeURIComponent(billId)}/shares`).then((response) => response.data),

  replaceShareSet: (billId: string, replacement: ReplaceShareSetRequest) =>
    client
      .put<ShareSetResponse>(`/bills/${encodeURIComponent(billId)}/shares`, replacement)
      .then((response) => response.data),

  people: () => client.get<PeopleResponse>('/people').then((response) => response.data.people),

  createPerson: (name: string) =>
    client.post<Person>('/people', { name }).then((response) => response.data),
}
