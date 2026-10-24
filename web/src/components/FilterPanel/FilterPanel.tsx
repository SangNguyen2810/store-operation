import { useId, useState } from 'react'
import type { ReactNode } from 'react'

interface Props {
  /** Accessible name of the form. */
  label: string
  /** Always-visible controls, for example a segmented range selector. */
  primary: ReactNode
  /** The remaining fields. Collapsed behind a button on small screens, always shown from 640px up. */
  children: ReactNode
  /** How many fields inside the panel differ from their default. Shown on the button. */
  activeCount: number
  onReset: () => void
}

/** Layout and open/closed state for a set of filters. It knows nothing about what the filters mean. */
export function FilterPanel({ label, primary, children, activeCount, onReset }: Props) {
  // Only matters on narrow screens; from 640px up the CSS always shows the fields.
  const [open, setOpen] = useState(false)
  const fieldsId = useId()
  return (
    <form className="filters" aria-label={label} onSubmit={(e) => e.preventDefault()}>
      <div className="filters-bar">
        {primary}
        <button
          type="button"
          className="filters-toggle"
          aria-expanded={open}
          aria-controls={fieldsId}
          onClick={() => setOpen((o) => !o)}
        >
          Filters
          {activeCount > 0 && (
            <span className="badge" aria-label={`${activeCount} active`}>
              {activeCount}
            </span>
          )}
        </button>
      </div>

      <div id={fieldsId} className="filter-fields" data-open={open}>
        {children}
        {activeCount > 0 && (
          <button type="button" className="reset" onClick={onReset}>
            Reset filters
          </button>
        )}
      </div>
    </form>
  )
}
