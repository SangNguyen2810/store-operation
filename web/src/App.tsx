import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { AppRoutes } from './routes'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 60_000, retry: 1, refetchOnWindowFocus: false } },
})

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <main className="mx-auto grid max-w-[1100px] grid-cols-1 gap-4 px-4 pt-4 pb-12 sm:gap-6 sm:px-6 sm:pt-6 sm:pb-16 lg:max-w-[1280px] lg:px-8">
          <AppRoutes />
        </main>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
