import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { breakpointFor, useBreakpoint } from './useBreakpoint'

describe('breakpointFor', () => {
  it('maps the two media queries to a breakpoint', () => {
    expect(breakpointFor(false, false)).toBe('mobile')
    expect(breakpointFor(true, false)).toBe('tablet')
    expect(breakpointFor(true, true)).toBe('desktop')
  })
})

describe('useBreakpoint', () => {
  afterEach(() => vi.unstubAllGlobals())

  /** A matchMedia whose answers can be changed, notifying listeners like a resize would. */
  function fakeMedia(initial: { tablet: boolean; desktop: boolean }) {
    const state = { ...initial }
    const listeners = new Set<() => void>()
    vi.stubGlobal('matchMedia', (query: string) => ({
      get matches() {
        return query.includes('1024') ? state.desktop : state.tablet
      },
      addEventListener: (_: string, fn: () => void) => listeners.add(fn),
      removeEventListener: (_: string, fn: () => void) => listeners.delete(fn),
    }))
    return {
      listeners,
      set(next: { tablet: boolean; desktop: boolean }) {
        Object.assign(state, next)
        listeners.forEach((fn) => fn())
      },
    }
  }

  it('falls back to tablet where matchMedia does not exist', () => {
    vi.stubGlobal('matchMedia', undefined)
    expect(renderHook(() => useBreakpoint()).result.current).toBe('tablet')
  })

  it.each([
    [{ tablet: false, desktop: false }, 'mobile'],
    [{ tablet: true, desktop: false }, 'tablet'],
    [{ tablet: true, desktop: true }, 'desktop'],
  ] as const)('reads %o as %s', (media, want) => {
    fakeMedia(media)
    expect(renderHook(() => useBreakpoint()).result.current).toBe(want)
  })

  it('follows a resize and stops listening on unmount', () => {
    const media = fakeMedia({ tablet: false, desktop: false })
    const { result, unmount } = renderHook(() => useBreakpoint())
    expect(result.current).toBe('mobile')
    act(() => media.set({ tablet: true, desktop: true }))
    expect(result.current).toBe('desktop')
    expect(media.listeners.size).toBeGreaterThan(0)
    unmount()
    expect(media.listeners.size).toBe(0)
  })
})
