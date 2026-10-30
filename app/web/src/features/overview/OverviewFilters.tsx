import type { DateRange, Filters, Granularity, Store } from '../../api/types'
import { FilterPanel } from '../../components/FilterPanel'
import { SegmentedControl } from '../../components/SegmentedControl'
import { SelectField } from '../../components/SelectField'
import { GRANULARITIES, QUICK_RANGES, activePanelFilters, defaultFilters, findRange, rangeFilters } from '../../lib/filters'

interface Props {
  filters: Filters
  /** The ranges the API offers; nothing else can be chosen. */
  ranges: DateRange[]
  stores: Store[]
  onChange: (next: Filters) => void
}

const groupOptions = GRANULARITIES.map((g) => ({ value: g, label: g[0].toUpperCase() + g.slice(1) }))

/** Knows what the overview filters mean (ranges, stores, grouping) and hands plain props to the UI components. */
export function OverviewFilters({ filters, ranges, stores, onChange }: Props) {
  const current = findRange(ranges, filters.from, filters.to)
  const choose = (id: string) => {
    const range = ranges.find((r) => r.id === id)
    if (range) onChange(rangeFilters(filters, range))
  }

  // A button only for a quick range the API actually offers.
  const quick = QUICK_RANGES.flatMap((q) => {
    const range = ranges.find((r) => r.id === q.id)
    return range ? [{ value: range.id, label: q.label, ariaLabel: range.label }] : []
  })

  return (
    <FilterPanel
      label="Filters"
      activeCount={activePanelFilters(filters, ranges)}
      onReset={() => onChange(defaultFilters(ranges))}
      primary={
        <SegmentedControl
          label="Range"
          options={quick}
          value={quick.some((q) => q.value === current?.id) ? (current?.id ?? null) : null}
          onChange={choose}
        />
      }
    >
      <SelectField
        label="More ranges"
        wide
        value={current?.id ?? ''}
        options={ranges.map((r) => ({ value: r.id, label: r.label, group: r.group }))}
        onChange={choose}
      />
      <SelectField
        label="Store"
        wide
        value={filters.store === null ? '' : String(filters.store)}
        options={[{ value: '', label: 'All stores' }, ...stores.map((s) => ({ value: String(s.id), label: s.label }))]}
        onChange={(v) => onChange({ ...filters, store: v === '' ? null : Number(v) })}
      />
      <SelectField
        label="Group by"
        value={filters.granularity}
        options={groupOptions}
        onChange={(v) => onChange({ ...filters, granularity: v as Granularity })}
      />
    </FilterPanel>
  )
}
