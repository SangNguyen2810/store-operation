import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ErrorState } from './ErrorState'

describe('ErrorState', () => {
  it('announces the message as an alert', () => {
    render(<ErrorState message="Couldn't load things: boom" onRetry={() => {}} />)
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't load things: boom")
  })

  it('calls onRetry when Retry is pressed', async () => {
    const onRetry = vi.fn()
    render(<ErrorState message="x" onRetry={onRetry} />)
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(onRetry).toHaveBeenCalledOnce()
  })
})
