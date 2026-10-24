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
    <form
      aria-label={label}
      onSubmit={(e) => e.preventDefault()}
      className="grid gap-3 rounded-[10px] border border-line bg-surface p-3 sm:p-4 lg:grid-cols-[auto_minmax(0,1fr)] lg:items-end lg:gap-x-6 lg:gap-y-4"
    >
      <div className="flex items-center justify-between gap-2">
        {primary}
        <button
          type="button"
          className="inline-flex items-center gap-2 aria-expanded:border-accent sm:hidden"
          aria-expanded={open}
          aria-controls={fieldsId}
          onClick={() => setOpen((o) => !o)}
        >
          Filters
          {activeCount > 0 && (
            <span
              className="min-w-[1.4em] rounded-full bg-accent px-1.5 text-center text-[0.75rem] leading-normal text-page"
              aria-label={`${activeCount} active`}
            >
              {activeCount}
            </span>
          )}
        </button>
      </div>

      <div
        id={fieldsId}
        data-open={open}
        className="hidden grid-cols-2 gap-3 data-[open=true]:grid sm:grid lg:grid-cols-[minmax(9.5rem,1fr)_minmax(9.5rem,1fr)_minmax(0,1.8fr)_minmax(7rem,1fr)_auto] lg:items-end"
      >
        {children}
        {/* Always present so the layout never shifts; disabled when there is nothing to reset. */}
        <button
          type="button"
          onClick={onReset}
          disabled={activeCount === 0}
          className="col-span-full justify-self-start border-transparent bg-transparent px-0 text-accent underline disabled:cursor-not-allowed disabled:text-muted disabled:no-underline disabled:opacity-60 lg:col-auto lg:justify-self-end"
        >
          Reset filters
        </button>
      </div>
    </form>
  )
}
