import { fileURLToPath } from 'url'
import { dirname, resolve } from 'path'
import { readFileSync } from 'fs'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { viteSingleFile } from 'vite-plugin-singlefile'
import type { Plugin } from 'vite'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

/** Injects fixture analysis data as window.__MRI_DATA__ during vite dev. */
function mriDevDataPlugin(): Plugin {
  return {
    name: 'mri-dev-data',
    apply: 'serve',
    transformIndexHtml(html) {
      const fixturePath = resolve(__dirname, 'fixtures/analysis.json')
      const data = readFileSync(fixturePath, 'utf-8')
      return html.replace('</head>', `  <script>window.__MRI_DATA__ = ${data};</script>\n</head>`)
    },
  }
}

export default defineConfig({
  plugins: [tailwindcss(), react(), viteSingleFile(), mriDevDataPlugin()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
  },
})
