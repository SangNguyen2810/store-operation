import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Skeleton } from './Skeleton'

describe('Skeleton', () => {
  it('is hidden from assistive tech and takes its size from props', () => {
    const { container } = render(<Skeleton height={120} width={80} />)
    const el = container.firstElementChild as HTMLElement
    expect(el).toHaveAttribute('aria-hidden', 'true')
    expect(el.style.height).toBe('120px')
    expect(el.style.width).toBe('80px')
  })

  it('defaults to a full-width line', () => {
    const { container } = render(<Skeleton />)
    const el = container.firstElementChild as HTMLElement
    expect(el.style.width).toBe('100%')
    expect(el.style.height).toBe('16px')
  })
})
