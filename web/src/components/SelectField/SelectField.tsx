import { Field } from '../Field'

export interface SelectOption {
  value: string
  label: string
}

interface Props {
  label: string
  value: string
  options: SelectOption[]
  onChange: (value: string) => void
  /** Spans the full row when the fields are laid out two per row. */
  wide?: boolean
}

export function SelectField({ label, value, options, onChange, wide }: Props) {
  return (
    <Field label={label} wide={wide}>
      <select className="w-full min-w-0" value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </Field>
  )
}
