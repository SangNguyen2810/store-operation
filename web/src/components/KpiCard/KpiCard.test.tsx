import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { KpiCard } from './KpiCard'

const base = { label: 'Revenue', value: '$10,707.22', caption: 'prev. $10,973.36' }

function renderCard(trend: Parameters<typeof KpiCard>[0]['trend']) {
  return render(
    <ul>
      <KpiCard {...base} trend={trend} />
    </ul>,
  )
}

describe('KpiCard', () => {
  it('shows the label, value and caption', () => {
    renderCard(null)
    expect(screen.getByText('Revenue')).toBeInTheDocument()
    expect(screen.getByText('$10,707.22')).toBeInTheDocument()
    expect(screen.getByText('prev. $10,973.36')).toBeInTheDocument()
  })

  it.each([
    ['up', '▲'],
    ['down', '▼'],
    ['flat', '■'],
  ] as const)('shows an arrow and text for a %s trend, with a spoken label', (direction, arrow) => {
    renderCard({ direction, text: '+2.4%', spokenLabel: `${direction} 2.4% versus previous period` })
    const trend = screen.getByLabelText(`${direction} 2.4% versus previous period`)
    expect(trend).toHaveClass(`delta-${direction}`)
    expect(trend).toHaveTextContent(`${arrow} +2.4%`)
  })

  it('says there is nothing to compare when there is no trend', () => {
    renderCard(null)
    expect(screen.getByLabelText('No previous period to compare')).toHaveClass('delta-none')
  })
})
