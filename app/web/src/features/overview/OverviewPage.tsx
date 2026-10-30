import { AsyncSection } from '../../components/AsyncSection'
import { ErrorState } from '../../components/ErrorState'
import { KpiGrid, KpiGridSkeleton } from '../../components/KpiGrid'
import { PageHeader } from '../../components/PageHeader'
import { RankedTable, RankedTableSkeleton } from '../../components/RankedTable'
import { Skeleton } from '../../components/Skeleton'
import { Suspense, lazy } from 'react'
import { OverviewFilters } from './OverviewFilters'
import { FILM_COLUMNS, toFilmRows } from './films'
import { toKpiItems } from './kpis'
import { useFilters, useKpis, useMeta, useRevenue, useStores, useTopFilms } from '../../hooks'
import { formatBucket } from '../../lib/format'

// The chart library is the heaviest dependency, so it loads on demand.
const RevenueChart = lazy(() => import('./RevenueChart').then((m) => ({ default: m.RevenueChart })))

export function OverviewPage() {
  const meta = useMeta()
  if (meta.isError) {
    return <ErrorState message={`Couldn't reach the API: ${meta.error.message}`} onRetry={() => void meta.refetch()} />
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
      <PageHeader title="Store Operations" subtitle={<>Data as of {formatBucket(asOf)}, {asOf.slice(0, 4)}</>} />

      <OverviewFilters filters={filters} asOf={asOf} stores={stores.data ?? []} onChange={setFilters} />

      <AsyncSection title="Key figures" query={kpis} skeleton={<KpiGridSkeleton />}>
        {(data) => <KpiGrid items={toKpiItems(data)} />}
      </AsyncSection>

      <div className="grid grid-cols-1 gap-4 sm:gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)] lg:items-start">
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

        <AsyncSection title="Top films" query={films} isEmpty={(f) => f.length === 0} skeleton={<RankedTableSkeleton />}>
          {(data) => <RankedTable labelHeader="Film" columns={FILM_COLUMNS} rows={toFilmRows(data)} />}
        </AsyncSection>
      </div>
    </>
  )
}
