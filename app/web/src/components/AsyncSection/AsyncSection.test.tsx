import { useQuery } from '@tanstack/react-query'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test/utils'
import { AsyncSection } from './AsyncSection'

function Harness({ fn, empty }: { fn: () => Promise<string[]>; empty?: boolean }) {
  const query = useQuery({ queryKey: ['t'], queryFn: fn })
  return (
    <AsyncSection title="Things" query={query} isEmpty={empty ? (d) => d.length === 0 : undefined} skeleton={<p>skeleton</p>}>
      {(d) => <p>{d.join(',')}</p>}
    </AsyncSection>
  )
}

describe('AsyncSection', () => {
  it('shows a skeleton, then the data', async () => {
    renderWithProviders(<Harness fn={() => Promise.resolve(['a', 'b'])} />)
    expect(screen.getByText('skeleton')).toBeInTheDocument()
    expect(await screen.findByText('a,b')).toBeInTheDocument()
    expect(screen.queryByText('skeleton')).not.toBeInTheDocument()
  })

  it('shows the empty state', async () => {
    renderWithProviders(<Harness fn={() => Promise.resolve([])} empty />)
    expect(await screen.findByText(/no data for this period/i)).toBeInTheDocument()
  })

  it('shows an error and recovers on retry', async () => {
    const fn = vi.fn<() => Promise<string[]>>().mockRejectedValueOnce(new Error('boom')).mockResolvedValue(['ok'])
    renderWithProviders(<Harness fn={fn} />)
    expect(await screen.findByRole('alert')).toHaveTextContent('boom')

    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() => expect(screen.getByText('ok')).toBeInTheDocument())
    expect(fn).toHaveBeenCalledTimes(2)
  })
})

describe('AsyncSection without an error message', () => {
  it('still says what failed', () => {
    const query = { isError: true, isPending: false, error: null, data: undefined, refetch: vi.fn() } as never
    renderWithProviders(
      <AsyncSection title="Things" query={query} skeleton={<p>skeleton</p>}>
        {() => <p>never</p>}
      </AsyncSection>,
    )
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't load things: unknown error")
  })
})
