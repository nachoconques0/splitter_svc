import axios from 'axios'

/**
 * The API is at /api on the page's own origin — nginx serves the page and proxies
 * /api — so nothing is cross-origin and there is no CORS configuration anywhere.
 */
export const client = axios.create({
  baseURL: '/api',
  timeout: 10_000,
  headers: { 'Content-Type': 'application/json' },
})
