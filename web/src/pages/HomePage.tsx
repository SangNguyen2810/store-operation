import { OverviewPage } from '../features/overview/OverviewPage'

/** The "/" route. Route-level screens stay thin; the dashboard itself lives in features/overview. */
export function HomePage() {
  return <OverviewPage />
}
