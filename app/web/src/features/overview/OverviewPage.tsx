import { AsyncSection } from '../../components/AsyncSection'
import { ErrorState } from '../../components/ErrorState'
import { KpiGrid, KpiGridSkeleton } from '../../components/KpiGrid'
import { PageHeader } from '../../components/PageHeader'
import { RankedTable, RankedTableSkeleton } from '../../components/RankedTable'
import { Skeleton } from '../../components/Skeleton'
import { Suspense, lazy } from 'react'
import { OverviewFilters } from './OverviewFilters'
import { storeDetailLine } from './storeDetails'
import { FILM_COLUMNS, toFilmRows } from './films'
import { toKpiItems } from './kpis'
import { useFilters, useKpis, useMeta, useRanges, useRevenue, useStores, useTopFilms } from '../../hooks'
import type { DateRange } from '../../api/types'
import { formatBucket } from '../../lib/format'

// The chart library is the heaviest dependency, so it loads on demand.
const RevenueChart = lazy(() => import('./RevenueChart').then((m) => ({ default: m.RevenueChart })))

export function OverviewPage() {
  const meta = useMeta()
  const ranges = useRanges()
  if (meta.isError) {
    return <ErrorState message={`Couldn't reach the API: ${meta.error.message}`} onRetry={() => void meta.refetch()} />
  }
  if (ranges.isError) {
    return <ErrorState message={`Couldn't load the date ranges: ${ranges.error.message}`} onRetry={() => void ranges.refetch()} />
  }
  if (!meta.data || !ranges.data) return <div aria-busy="true"><Skeleton height={120} /></div>
  if (ranges.data.length === 0) return <ErrorState message="The API offered no date ranges." onRetry={() => void ranges.refetch()} />
  return <Dashboard asOf={meta.data.asOf} ranges={ranges.data} />
}

function Dashboard({ asOf, ranges }: { asOf: string; ranges: DateRange[] }) {
  const [filters, setFilters] = useFilters(ranges)
  const stores = useStores()
  const kpis = useKpis(filters)
  const revenue = useRevenue(filters)
  const films = useTopFilms(filters)

  return (
    <>
      <PageHeader title="Store Operations" subtitle={<>Data as of {formatBucket(asOf)}, {asOf.slice(0, 4)}</>} />

      <OverviewFilters filters={filters} ranges={ranges} stores={stores.data ?? []} onChange={setFilters} />
      <StoreDetails line={storeDetailLine(stores.data ?? [], filters.store)} />

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

/** One quiet line under the filters. It is left out when the store list could not load. */
function StoreDetails({ line }: { line: string | null }) {
  if (!line) return null
  return (
    <p className="m-0 text-muted" aria-live="polite" data-testid="store-details">
      {line}
    </p>
  )
}
