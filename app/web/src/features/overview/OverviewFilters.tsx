import type { Filters, Granularity, Store } from '../../api/types'
import { DateField } from '../../components/DateField'
import { FilterPanel } from '../../components/FilterPanel'
import { SegmentedControl } from '../../components/SegmentedControl'
import { SelectField } from '../../components/SelectField'
import { daysInclusive, isIsoDate } from '../../lib/dates'
import { GRANULARITIES, MAX_DAYS, PRESETS, activePanelFilters, defaultFilters, presetFilters } from '../../lib/filters'

interface Props {
  filters: Filters
  asOf: string
  stores: Store[]
  onChange: (next: Filters) => void
}

const presetOptions = PRESETS.map((p) => ({ value: String(p.days), label: `${p.days}d`, ariaLabel: p.label }))
const groupOptions = GRANULARITIES.map((g) => ({ value: g, label: g[0].toUpperCase() + g.slice(1) }))

/** Knows what the overview filters mean (presets, ranges, limits) and hands plain props to the UI components. */
export function OverviewFilters({ filters, asOf, stores, onChange }: Props) {
  const activeDays = filters.to === asOf ? daysInclusive(filters.from, filters.to) : null

  function setDate(field: 'from' | 'to', value: string) {
    if (!isIsoDate(value)) return // a half-typed date; wait for a complete one
    const next = { ...filters, [field]: value }
    const span = daysInclusive(next.from, next.to)
    if (span < 1 || span > MAX_DAYS) return
    onChange(next)
  }

  return (
    <FilterPanel
      label="Filters"
      activeCount={activePanelFilters(filters, asOf)}
      onReset={() => onChange(defaultFilters(asOf))}
      primary={
        <SegmentedControl
          label="Range"
          options={presetOptions}
          value={PRESETS.some((p) => p.days === activeDays) ? String(activeDays) : null}
          onChange={(days) => onChange(presetFilters(filters, asOf, Number(days)))}
        />
      }
    >
      <DateField label="From" value={filters.from} max={filters.to} onChange={(v) => setDate('from', v)} />
      <DateField label="To" value={filters.to} min={filters.from} onChange={(v) => setDate('to', v)} />
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
