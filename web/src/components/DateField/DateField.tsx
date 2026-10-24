interface Props {
  label: string
  /** YYYY-MM-DD. */
  value: string
  min?: string
  max?: string
  onChange: (value: string) => void
}

/** A native date input. It reports whatever the browser gives, including "" while a date is half typed. */
export function DateField({ label, value, min, max, onChange }: Props) {
  return (
    <label className="field">
      <span className="field-label">{label}</span>
      <input type="date" value={value} min={min} max={max} onChange={(e) => onChange(e.target.value)} />
    </label>
  )
}
