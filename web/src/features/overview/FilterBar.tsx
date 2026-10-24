import { useState } from 'react'
import type { Filters, Granularity, Store } from '../../api/types'
import { daysInclusive, isIsoDate } from '../../lib/dates'
import { GRANULARITIES, MAX_DAYS, PRESETS, activePanelFilters, defaultFilters, presetFilters } from '../../lib/filters'

interface Props {
  filters: Filters
  asOf: string
  stores: Store[]
  onChange: (next: Filters) => void
}

export function FilterBar({ filters, asOf, stores, onChange }: Props) {
  // Only matters on narrow screens; from 640px up the CSS always shows the fields.
  const [open, setOpen] = useState(false)
  const activeDays = filters.to === asOf ? daysInclusive(filters.from, filters.to) : null
  const active = activePanelFilters(filters, asOf)

  function setDate(field: 'from' | 'to', value: string) {
    if (!isIsoDate(value)) return // a half-typed date; wait for a complete one
    const next = { ...filters, [field]: value }
    const span = daysInclusive(next.from, next.to)
    if (span < 1 || span > MAX_DAYS) return
    onChange(next)
  }

  return (
    <form className="filters" aria-label="Filters" onSubmit={(e) => e.preventDefault()}>
      <div className="filters-bar">
        <div className="field field-presets">
          <span className="field-label range-label" id="range-label">Range</span>
          <div role="group" aria-labelledby="range-label" className="segmented">
            {PRESETS.map((p) => (
              <button
                key={p.days}
                type="button"
                aria-label={p.label}
                aria-pressed={activeDays === p.days}
                onClick={() => onChange(presetFilters(filters, asOf, p.days))}
              >
                {p.days}d
              </button>
            ))}
          </div>
        </div>

        <button
          type="button"
          className="filters-toggle"
          aria-expanded={open}
          aria-controls="filter-fields"
          onClick={() => setOpen((o) => !o)}
        >
          Filters
          {active > 0 && (
            <span className="badge" aria-label={`${active} active`}>
              {active}
            </span>
          )}
        </button>
      </div>

      <div id="filter-fields" className="filter-fields" data-open={open}>
        <label className="field">
          <span className="field-label">From</span>
          <input type="date" value={filters.from} max={filters.to} onChange={(e) => setDate('from', e.target.value)} />
        </label>
        <label className="field">
          <span className="field-label">To</span>
          <input type="date" value={filters.to} min={filters.from} onChange={(e) => setDate('to', e.target.value)} />
        </label>

        <label className="field field-store">
          <span className="field-label">Store</span>
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

        <label className="field field-group">
          <span className="field-label">Group by</span>
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

        {active > 0 && (
          <button type="button" className="reset" onClick={() => onChange(defaultFilters(asOf))}>
            Reset filters
          </button>
        )}
      </div>
    </form>
  )
}
