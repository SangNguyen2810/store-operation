import { useId } from 'react'

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

/** A row of connected toggle buttons. The visible label shows from 1024px up (see index.css). */
export function SegmentedControl({ label, options, value, onChange }: Props) {
  const labelId = useId()
  return (
    <div className="field field-segmented">
      <span className="field-label segmented-label" id={labelId}>
        {label}
      </span>
      <div role="group" aria-labelledby={labelId} className="segmented">
        {options.map((o) => (
          <button
            key={o.value}
            type="button"
            aria-label={o.ariaLabel ?? o.label}
            aria-pressed={o.value === value}
            onClick={() => onChange(o.value)}
          >
            {o.label}
          </button>
        ))}
      </div>
    </div>
  )
}
