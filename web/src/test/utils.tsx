import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, renderHook } from '@testing-library/react'
import type { ReactElement, ReactNode } from 'react'
import { vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'

export function renderWithProviders(ui: ReactElement, url = '/') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[url]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

/** Renders a hook inside the same providers as the app (query client and router). */
export function renderHookWithProviders<T>(hook: () => T, url = '/') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[url]}>{children}</MemoryRouter>
    </QueryClientProvider>
  )
  return renderHook(hook, { wrapper })
}

/** Stubs fetch with one JSON body per path and records every requested URL. */
export function stubApi(routes: Record<string, unknown>) {
  const calls: string[] = []
  vi.stubGlobal('fetch', (input: string) => {
    calls.push(input)
    const body = routes[input.split('?')[0]]
    return Promise.resolve(
      body === undefined ? new Response('{}', { status: 404 }) : new Response(JSON.stringify(body), { status: 200 }),
    )
  })
  return calls
}
