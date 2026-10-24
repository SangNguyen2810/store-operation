import type { ComponentProps, ReactNode } from 'react'

/** The small uppercase text above a control. */
export function FieldLabel({ className = '', ...props }: ComponentProps<'span'>) {
  return (
    <span
      className={`text-[0.72rem] font-semibold tracking-[0.04em] text-muted uppercase ${className}`}
      {...props}
    />
  )
}

interface FieldProps {
  label: string
  children: ReactNode
  /** Spans the full row when fields are laid out two per row. */
  wide?: boolean
}

/** A native control with its label. Clicking the label focuses the control. */
export function Field({ label, children, wide }: FieldProps) {
  return (
    <label data-wide={wide || undefined} className={`grid min-w-0 gap-1.5 ${wide ? 'col-span-full sm:col-auto' : ''}`}>
      <FieldLabel>{label}</FieldLabel>
      {children}
    </label>
  )
}
