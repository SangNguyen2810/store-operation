import { defineConfig, devices } from '@playwright/test'

// The real stack, on ports of its own so it never collides with `make dev`:
// Pagila (already running) <- stores :8181, ranges :8182, analytics :8183 <- gateway :8180 <- the production web build served by `vite preview` :4173.
// No cache here: the redis service is covered by its own tests.
const STORES_PORT = 8181
const RANGES_PORT = 8182
const ANALYTICS_PORT = 8183
const GATEWAY_PORT = 8180
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
      command: 'go run ./cmd/stores',
      cwd: '../stores',
      env: { ADDR: `:${STORES_PORT}`, DATABASE_URL },
      url: `http://localhost:${STORES_PORT}/healthz`,
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: 'go run ./cmd/ranges',
      cwd: '../ranges',
      env: { ADDR: `:${RANGES_PORT}` },
      url: `http://localhost:${RANGES_PORT}/healthz`,
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: 'go run ./cmd/analytics',
      cwd: '../analytics',
      env: { ADDR: `:${ANALYTICS_PORT}`, DATABASE_URL, RANGES_URL: `http://localhost:${RANGES_PORT}` },
      url: `http://localhost:${ANALYTICS_PORT}/healthz`,
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: 'go run ./cmd/gateway',
      cwd: '../gateway',
      // The limit is covered by the gateway's own tests; parallel browser tests share one client IP.
      env: {
        ADDR: `:${GATEWAY_PORT}`,
        UPSTREAM_URL: `http://localhost:${ANALYTICS_PORT}`,
        ROUTES: `/api/stores=http://localhost:${STORES_PORT},/api/presets=http://localhost:${RANGES_PORT}`,
        RATE_LIMIT_RPS: '0',
      },
      url: `http://localhost:${GATEWAY_PORT}/healthz`,
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: `npm run build && npx vite preview --port ${WEB_PORT} --strictPort`,
      env: { API_PROXY: `http://localhost:${GATEWAY_PORT}` },
      url: `http://localhost:${WEB_PORT}`,
      reuseExistingServer: false,
      timeout: 180_000,
    },
  ],
})
