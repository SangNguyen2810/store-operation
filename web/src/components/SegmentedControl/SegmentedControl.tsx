import { useId } from 'react'
import { FieldLabel } from '../Field'

export interface SegmentedOption {
  value: string
  /** Short visible text, for example "30d". */
  label: string
  /** Full accessible name, for example "Last 30 days". Defaults to the label. */
  ariaLabel?: string
}

interface Props {
  label: string
  options: SegmentedOption[]
  /** The selected value, or null when none of the options is active. */
  value: string | null
  onChange: (value: string) => void
}

/** A row of connected toggle buttons. The visible label shows from 1024px up. */
export function SegmentedControl({ label, options, value, onChange }: Props) {
  const labelId = useId()
  return (
    <div className="grid min-w-0 gap-1.5">
      <FieldLabel id={labelId} className="hidden lg:block">
        {label}
      </FieldLabel>
      <div
        role="group"
        aria-labelledby={labelId}
        className="inline-flex overflow-hidden rounded-ui bg-surface -outline-offset-1 outline-1 outline-line"
      >
        {options.map((o) => (
          <button
            key={o.value}
            type="button"
            aria-label={o.ariaLabel ?? o.label}
            aria-pressed={o.value === value}
            onClick={() => onChange(o.value)}
            className="min-w-[3.25rem] rounded-none border-0 border-line bg-transparent tabular-nums not-first:border-l aria-pressed:bg-accent aria-pressed:text-page"
          >
            {o.label}
          </button>
        ))}
      </div>
    </div>
  )
}
