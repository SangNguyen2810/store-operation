import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PageHeader } from './PageHeader'

describe('PageHeader', () => {
  it('renders the title as the page heading, with an optional subtitle', () => {
    render(<PageHeader title="Store Operations" subtitle="Data as of Jul 27, 2022" />)
    expect(screen.getByRole('heading', { level: 1, name: 'Store Operations' })).toBeInTheDocument()
    expect(screen.getByText('Data as of Jul 27, 2022')).toBeInTheDocument()
  })

  it('omits the subtitle when there is none', () => {
    const { container } = render(<PageHeader title="Only a title" />)
    expect(container.querySelector('.muted')).toBeNull()
  })
})
