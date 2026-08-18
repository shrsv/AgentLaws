import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [preact()],
  // Served from an embedded Go http.FileServer (internal/server), not
  // necessarily mounted at the domain root.
  base: './',
})
