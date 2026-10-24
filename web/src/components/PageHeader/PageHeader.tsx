import type { ReactNode } from 'react'

interface Props {
  title: string
  subtitle?: ReactNode
}

export function PageHeader({ title, subtitle }: Props) {
  return (
    <header>
      <h1 className="m-0 text-[1.35rem] font-bold sm:text-[1.6rem]">{title}</h1>
      {subtitle && <p className="mt-1 text-muted">{subtitle}</p>}
    </header>
  )
}
