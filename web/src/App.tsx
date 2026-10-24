import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { OverviewPage } from './features/overview/OverviewPage'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 60_000, retry: 1, refetchOnWindowFocus: false } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <main>
          <OverviewPage />
        </main>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
