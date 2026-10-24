import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ResponsiveContainer } from './ResponsiveContainer'

// jsdom has no layout engine, so give the container something to measure.
vi.stubGlobal(
  'ResizeObserver',
  class {
    observe() {}
    unobserve() {}
    disconnect() {}
  },
)

const container = (c: HTMLElement) => c.querySelector<HTMLElement>('.recharts-responsive-container')

describe('ResponsiveContainer', () => {
  it('fills the width of its parent by default', () => {
    const { container: root } = render(
      <ResponsiveContainer height={300} initialDimension={{ width: 500, height: 300 }}>
        <div />
      </ResponsiveContainer>,
    )
    expect(container(root)?.style.width).toBe('100%')
    expect(container(root)?.style.height).toBe('300px')
  })

  it('lets the caller choose another width', () => {
    const { container: root } = render(
      <ResponsiveContainer width="50%" height={300} initialDimension={{ width: 500, height: 300 }}>
        <div />
      </ResponsiveContainer>,
    )
    expect(container(root)?.style.width).toBe('50%')
  })
})
