import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Kpis } from '../../api/types'
import { KpiCards } from './KpiCards'

const kpis: Kpis = {
  revenue: { value: 10707.22, previous: 10973.36, deltaPct: -2.43 },
  rentals: { value: 4158, previous: 2778, deltaPct: 49.68 },
  activeCustomers: { value: 598, previous: 0, deltaPct: null },
  avgPayment: { value: 4.15, previous: 4.15, deltaPct: 0 },
}

describe('KpiCards', () => {
  it('renders values and describes each delta in words', () => {
    render(<KpiCards kpis={kpis} />)
    expect(screen.getByText('$10,707.22')).toBeInTheDocument()
    expect(screen.getByText('4,158')).toBeInTheDocument()
    expect(screen.getByLabelText('down 2.4% versus previous period')).toBeInTheDocument()
    expect(screen.getByLabelText('up 49.7% versus previous period')).toBeInTheDocument()
    expect(screen.getByLabelText('unchanged 0.0% versus previous period')).toBeInTheDocument()
    expect(screen.getByLabelText('No previous period to compare')).toBeInTheDocument()
  })
})
