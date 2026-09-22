import { fileURLToPath, URL } from 'node:url'

import { quasar, transformAssetUrls } from '@quasar/vite-plugin'
import vue from '@vitejs/plugin-vue'
// defineConfig from vitest rather than vite: it is vite's own, widened to accept
// the test block below.
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [
    vue({ template: { transformAssetUrls } }),
    quasar({ sassVariables: fileURLToPath(new URL('./src/quasar-variables.sass', import.meta.url)) }),
  ],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  // Node rather than a DOM: the thing under test is arithmetic.
  test: {
    environment: 'node',
    include: ['src/**/*.spec.ts'],
  },
  server: {
    // Only for `npm run dev`. In the shipped stack nginx serves the page and
    // proxies /api to the API on the same origin; this makes the dev server
    // behave the same way, so no code has to know which of the two it is under.
    proxy: { '/api': { target: 'http://localhost:8080', rewrite: (path) => path.replace(/^\/api/, '') } },
  },
})
