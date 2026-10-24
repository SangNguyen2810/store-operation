import { useSyncExternalStore } from 'react'

export type Breakpoint = 'mobile' | 'tablet' | 'desktop'

// Keep in step with the breakpoints in index.css.
const TABLET = '(min-width: 640px)'
const DESKTOP = '(min-width: 1024px)'

function matches(query: string): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(query).matches
}

/** Pure, so it can be tested without a browser. */
export function breakpointFor(isTablet: boolean, isDesktop: boolean): Breakpoint {
  return isDesktop ? 'desktop' : isTablet ? 'tablet' : 'mobile'
}

function subscribe(onChange: () => void) {
  if (typeof window.matchMedia !== 'function') return () => {}
  const lists = [window.matchMedia(TABLET), window.matchMedia(DESKTOP)]
  lists.forEach((l) => l.addEventListener('change', onChange))
  return () => lists.forEach((l) => l.removeEventListener('change', onChange))
}

const snapshot = () => breakpointFor(matches(TABLET), matches(DESKTOP))

/**
 * For the few things CSS cannot size, such as the chart's pixel height and axis width.
 * Everything else should use media queries. Falls back to "tablet" where matchMedia is missing (jsdom).
 */
export function useBreakpoint(): Breakpoint {
  return useSyncExternalStore(subscribe, () => (typeof window.matchMedia === 'function' ? snapshot() : 'tablet'), () => 'tablet')
}
