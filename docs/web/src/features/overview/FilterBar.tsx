import type { Filters, Granularity, Store } from '../../api/types'
import { daysInclusive, isIsoDate } from '../../lib/dates'
import { GRANULARITIES, MAX_DAYS, PRESETS, presetFilters } from './filters'

interface Props {
  filters: Filters
  asOf: string
  stores: Store[]
  onChange: (next: Filters) => void
}

export function FilterBar({ filters, asOf, stores, onChange }: Props) {
  const activeDays = filters.to === asOf ? daysInclusive(filters.from, filters.to) : null

  function setDate(field: 'from' | 'to', value: string) {
    if (!isIsoDate(value)) return // a half-typed date; wait for a complete one
    const next = { ...filters, [field]: value }
    const span = daysInclusive(next.from, next.to)
    if (span < 1 || span > MAX_DAYS) return
    onChange(next)
  }

  return (
    <form className="filters" aria-label="Filters" onSubmit={(e) => e.preventDefault()}>
      <div role="group" aria-label="Date presets" className="presets">
        {PRESETS.map((p) => (
          <button
            key={p.days}
            type="button"
            aria-pressed={activeDays === p.days}
            onClick={() => onChange(presetFilters(filters, asOf, p.days))}
          >
            {p.label}
          </button>
        ))}
      </div>

      <label>
        From
        <input type="date" value={filters.from} max={filters.to} onChange={(e) => setDate('from', e.target.value)} />
      </label>
      <label>
        To
        <input type="date" value={filters.to} min={filters.from} onChange={(e) => setDate('to', e.target.value)} />
      </label>

      <label>
        Store
        <select
          value={filters.store ?? ''}
          onChange={(e) => onChange({ ...filters, store: e.target.value === '' ? null : Number(e.target.value) })}
        >
          <option value="">All stores</option>
          {stores.map((s) => (
            <option key={s.id} value={s.id}>
              {s.label}
            </option>
          ))}
        </select>
      </label>

      <label>
        Group by
        <select
          value={filters.granularity}
          onChange={(e) => onChange({ ...filters, granularity: e.target.value as Granularity })}
        >
          {GRANULARITIES.map((g) => (
            <option key={g} value={g}>
              {g[0].toUpperCase() + g.slice(1)}
            </option>
          ))}
        </select>
      </label>
    </form>
  )
}
