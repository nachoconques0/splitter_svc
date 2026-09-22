import { ref } from 'vue'

import { toApiError } from '@/api/errors'
import { routes } from '@/api/routes'
import type { ApiError, Person, Result } from '@/types/api'

/**
 * The people who exist, so the page can offer them rather than asking for an
 * identifier. Not the Bill's: a Person outlives every Bill they appear on.
 */
export function usePeople() {
  const loading = ref(false)
  const error = ref<ApiError | null>(null)
  const data = ref<Person[]>([])

  async function load(): Promise<Result<Person[]>> {
    loading.value = true
    error.value = null

    try {
      const people = await routes.people()
      data.value = people
      return { ok: true, data: people }
    } catch (cause) {
      const apiError = toApiError(cause)
      error.value = apiError
      return { ok: false, error: apiError }
    } finally {
      loading.value = false
    }
  }

  /**
   * Someone already known by this name, ignoring case and space. Used before
   * creating, so typing a known name picks that person rather than copying them.
   */
  function findByName(name: string): Person | undefined {
    const wanted = name.trim().toLowerCase()
    return data.value.find((person) => person.name.toLowerCase() === wanted)
  }

  /** Creates a Person and keeps them, so they can be added straight away. */
  async function create(name: string): Promise<Result<Person>> {
    loading.value = true
    error.value = null

    try {
      const person = await routes.createPerson(name)
      // Appended, not re-sorted: the server orders this list, and JavaScript's
      // collation would quietly disagree. The next load puts them in place.
      data.value = [...data.value, person]
      return { ok: true, data: person }
    } catch (cause) {
      const apiError = toApiError(cause)
      error.value = apiError
      return { ok: false, error: apiError }
    } finally {
      loading.value = false
    }
  }

  return { loading, error, data, load, create, findByName }
}
