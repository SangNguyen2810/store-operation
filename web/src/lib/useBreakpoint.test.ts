import { describe, expect, it } from 'vitest'
import { breakpointFor } from './useBreakpoint'

describe('breakpointFor', () => {
  it('maps the two media queries to a breakpoint', () => {
    expect(breakpointFor(false, false)).toBe('mobile')
    expect(breakpointFor(true, false)).toBe('tablet')
    expect(breakpointFor(true, true)).toBe('desktop')
  })
})
