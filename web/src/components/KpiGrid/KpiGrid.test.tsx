import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { KpiGrid, KpiGridSkeleton } from './KpiGrid'

const items = [
  { id: 'a', label: 'Revenue', value: '$100', caption: 'prev. $50', trend: { direction: 'up' as const, text: '+100.0%', spokenLabel: 'up 100.0% versus previous period' } },
  { id: 'b', label: 'Rentals', value: '10', caption: 'prev. 0', trend: null },
]

describe('KpiGrid', () => {
  it('renders one card per item in a list', () => {
    render(<KpiGrid items={items} />)
    expect(screen.getAllByRole('listitem')).toHaveLength(2)
    expect(screen.getByText('Revenue')).toBeInTheDocument()
    expect(screen.getByText('Rentals')).toBeInTheDocument()
  })

  it('renders the skeleton with the requested number of placeholders', () => {
    const { container } = render(<KpiGridSkeleton count={3} />)
    expect(container.querySelectorAll('li.card')).toHaveLength(3)
  })
})
