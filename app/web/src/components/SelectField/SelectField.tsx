import { Fragment } from 'react'
import { Field } from '../Field'

export interface SelectOption {
  value: string
  label: string
  /** Options with the same group that follow each other are listed under one heading. */
  group?: string
}

interface Props {
  label: string
  value: string
  options: SelectOption[]
  onChange: (value: string) => void
  /** Spans the full row when the fields are laid out two per row. */
  wide?: boolean
}

/** The options as runs of the same group, in order. Options without a group stand alone. */
function runs(options: SelectOption[]): { group?: string; options: SelectOption[] }[] {
  const out: { group?: string; options: SelectOption[] }[] = []
  for (const o of options) {
    const last = out.at(-1)
    if (last && o.group !== undefined && last.group === o.group) last.options.push(o)
    else out.push({ group: o.group, options: [o] })
  }
  return out
}

export function SelectField({ label, value, options, onChange, wide }: Props) {
  return (
    <Field label={label} wide={wide}>
      <select className="w-full min-w-0" value={value} onChange={(e) => onChange(e.target.value)}>
        {runs(options).map((run, i) => {
          const items = run.options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))
          return run.group === undefined ? (
            <Fragment key={i}>{items}</Fragment>
          ) : (
            <optgroup key={i} label={run.group}>
              {items}
            </optgroup>
          )
        })}
      </select>
    </Field>
  )
}
