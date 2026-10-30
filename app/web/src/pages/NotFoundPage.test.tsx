import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { NotFoundPage } from './NotFoundPage'

describe('NotFoundPage', () => {
  it('explains the problem and links home', () => {
    render(
      <MemoryRouter>
        <NotFoundPage />
      </MemoryRouter>,
    )
    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to the dashboard' })).toHaveAttribute('href', '/')
  })

  it('sets the document title and restores it on unmount', () => {
    document.title = 'Store Operations'
    const { unmount } = render(
      <MemoryRouter>
        <NotFoundPage />
      </MemoryRouter>,
    )
    expect(document.title).toBe('Page not found · Store Operations')
    unmount()
    expect(document.title).toBe('Store Operations')
  })
})
