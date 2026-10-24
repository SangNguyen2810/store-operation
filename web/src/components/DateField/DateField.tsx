import { Field } from '../Field'

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
    <Field label={label}>
      <input
        type="date"
        className="w-full min-w-0"
        value={value}
        min={min}
        max={max}
        onChange={(e) => onChange(e.target.value)}
      />
    </Field>
  )
}
