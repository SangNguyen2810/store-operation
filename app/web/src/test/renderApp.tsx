import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AppRoutes } from '../routes'
import { BackButton, Location } from './RouterProbes'

/** The real route table and pages, with a fresh query cache per test. */
export function renderApp(url = '/') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[url]}>
        <AppRoutes />
        <Location />
        <BackButton />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}
