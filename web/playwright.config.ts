import { defineConfig, devices } from '@playwright/test'

// The real stack, on ports of its own so it never collides with `make dev`:
// Pagila (already running) <- Go API :8180 <- the production web build served by `vite preview` :4173.
const API_PORT = 8180
const WEB_PORT = 4173
const DATABASE_URL = process.env.E2E_DATABASE_URL ?? 'postgres://postgres:123456@localhost:5432/postgres'

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : [['list']],
  use: {
    baseURL: `http://localhost:${WEB_PORT}`,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: [
    {
      command: 'go run ./cmd/api',
      cwd: '../api',
      env: { ADDR: `:${API_PORT}`, DATABASE_URL },
      url: `http://localhost:${API_PORT}/healthz`,
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: `npm run build && npx vite preview --port ${WEB_PORT} --strictPort`,
      env: { API_PROXY: `http://localhost:${API_PORT}` },
      url: `http://localhost:${WEB_PORT}`,
      reuseExistingServer: false,
      timeout: 180_000,
    },
  ],
})
