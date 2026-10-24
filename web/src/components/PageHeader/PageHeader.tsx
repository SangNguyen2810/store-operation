import type { ReactNode } from 'react'

interface Props {
  title: string
  subtitle?: ReactNode
}

export function PageHeader({ title, subtitle }: Props) {
  return (
    <header className="page-header">
      <h1>{title}</h1>
      {subtitle && <p className="muted">{subtitle}</p>}
    </header>
  )
}
