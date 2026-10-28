import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// The API address for the dev server and `vite preview` (the e2e tests point it at their own API).
const api = process.env.API_PROXY ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: { proxy: { '/api': api } },
  preview: { proxy: { '/api': api } },
  test: {
    // e2e/ holds Playwright specs, which Vitest must not try to run.
    include: ['src/**/*.test.{ts,tsx}'],
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    globals: false,
    coverage: {
      provider: 'v8',
      // Count every source file, not only the ones a test happens to import.
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/**/*.test.{ts,tsx}', 'src/test/**', 'src/main.tsx', 'src/**/index.ts', 'src/**/*.d.ts', 'src/api/types.ts'],
      reporter: ['text-summary', 'text'],
      thresholds: { lines: 85, statements: 85, functions: 80, branches: 80 },
    },
  },
})
