import { AsyncSection } from '../../components/AsyncSection'
import { Skeleton } from '../../components/Skeleton'
import { Suspense, lazy } from 'react'
import { FilterBar } from './FilterBar'
import { KpiCards, KpiSkeleton } from './KpiCards'
import { TopFilms, TopFilmsSkeleton } from './TopFilms'
import { useFilters, useKpis, useMeta, useRevenue, useStores, useTopFilms } from './hooks'
import { formatBucket } from '../../lib/format'

// The chart library is the heaviest dependency, so it loads on demand.
const RevenueChart = lazy(() => import('./RevenueChart').then((m) => ({ default: m.RevenueChart })))

export function OverviewPage() {
  const meta = useMeta()
  if (meta.isError) {
    return (
      <div className="state state-error" role="alert">
        <p>Couldn't reach the API: {meta.error.message}</p>
        <button type="button" onClick={() => void meta.refetch()}>Retry</button>
      </div>
    )
  }
  if (!meta.data) return <div aria-busy="true"><Skeleton height={120} /></div>
  return <Dashboard asOf={meta.data.asOf} />
}

function Dashboard({ asOf }: { asOf: string }) {
  const [filters, setFilters] = useFilters(asOf)
  const stores = useStores()
  const kpis = useKpis(filters)
  const revenue = useRevenue(filters)
  const films = useTopFilms(filters)

  return (
    <>
      <header className="page-header">
        <h1>Store Operations</h1>
        <p className="muted">Data as of {formatBucket(asOf)}, {asOf.slice(0, 4)}</p>
      </header>

      <FilterBar filters={filters} asOf={asOf} stores={stores.data ?? []} onChange={setFilters} />

      <AsyncSection title="Key figures" query={kpis} skeleton={<KpiSkeleton />}>
        {(data) => <KpiCards kpis={data} />}
      </AsyncSection>

      <div className="split">
        <AsyncSection
          title="Revenue"
          query={revenue}
          isEmpty={(pts) => pts.every((p) => p.revenue === 0)}
          skeleton={<Skeleton height={300} />}
        >
          {(pts) => (
            <Suspense fallback={<Skeleton height={300} />}>
              <RevenueChart points={pts} stores={stores.data ?? []} filters={filters} />
            </Suspense>
          )}
        </AsyncSection>

        <AsyncSection title="Top films" query={films} isEmpty={(f) => f.length === 0} skeleton={<TopFilmsSkeleton />}>
          {(data) => <TopFilms films={data} />}
        </AsyncSection>
      </div>
    </>
  )
}
