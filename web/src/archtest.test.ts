// Enforces the front-end layering on every `npm test` (so in CI too), the same idea as
// api/internal/archtest on the Go side. It reads import statements; it does not run the code.
/// <reference types="node" />
import { readFileSync, readdirSync, statSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const SRC = __dirname

/** Every source file under src, read from disk each run so newly added files are always checked. */
function sources(): Record<string, string> {
  const out: Record<string, string> = {}
  const walk = (dir: string) => {
    for (const name of readdirSync(dir)) {
      const full = path.join(dir, name)
      if (statSync(full).isDirectory()) walk(full)
      else if (/\.tsx?$/.test(name)) out[path.relative(SRC, full).split(path.sep).join('/')] = readFileSync(full, 'utf8')
    }
  }
  walk(SRC)
  delete out['archtest.test.ts']
  return out
}

// What each layer may import from inside src/. Dependencies point inward only.
//
//   api         -> nothing
//   lib         -> api (types)
//   hooks       -> api, lib
//   components  -> components, lib          (pure UI: props in, JSX out)
//   features    -> components, hooks, lib, api
//   pages       -> features, components, hooks, lib, api
//   (src root: App, routes, main compose everything and are not checked)
const ALLOWED: Record<string, string[]> = {
  api: ['api'],
  lib: ['lib', 'api'],
  hooks: ['hooks', 'lib', 'api'],
  components: ['components', 'lib'],
  features: ['features', 'components', 'hooks', 'lib', 'api'],
  pages: ['pages', 'features', 'components', 'hooks', 'lib', 'api'],
}

// Pure UI must not know about data fetching or routing.
const BANNED_IN_COMPONENTS = ['react-router-dom', 'react-router', '@tanstack/react-query']

const RECHARTS_HOME = 'components/libs/recharts/'

const isTest = (file: string) => /\.test\.tsx?$/.test(file)

/** Why `file` (relative to src, slash separated) may not import `spec`, or "" if it may. */
export function violation(file: string, spec: string): string {
  const layer = file.split('/')[0]
  const tests = isTest(file)

  if (spec === 'recharts' || spec.startsWith('recharts/')) {
    return file.startsWith(RECHARTS_HOME) ? '' : "only components/libs/recharts may import 'recharts'"
  }

  if (!spec.startsWith('.')) {
    const banned = layer === 'components' && !tests && BANNED_IN_COMPONENTS.some((b) => spec === b || spec.startsWith(b + '/'))
    return banned ? `components must stay pure UI and cannot import '${spec}'` : ''
  }

  const allowed = ALLOWED[layer]
  if (!allowed) return '' // src root and test helpers are not layered

  // `..` segments beyond the file's own folder depth would leave src; URL resolution would hide that by clamping at the root.
  const depth = file.split('/').length - 1
  if (spec.split('/').filter((part) => part === '..').length > depth) return 'imports outside src'
  const target = new URL(spec, `http://src/${file}`).pathname.slice(1)
  const targetLayer = target.split('/')[0]
  if (targetLayer === 'test') return tests ? '' : 'production code must not import test helpers'
  if (!target.includes('/') && !(targetLayer in ALLOWED)) return `${layer} must not import the app shell (${targetLayer})`
  return allowed.includes(targetLayer) ? '' : `${layer} must not import ${targetLayer}`
}

function importsOf(source: string): string[] {
  const specs = new Set<string>()
  const patterns = [/\bfrom\s*['"]([^'"]+)['"]/g, /\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)/g, /^\s*import\s+['"]([^'"]+)['"]/gm]
  for (const re of patterns) for (const m of source.matchAll(re)) specs.add(m[1])
  return [...specs]
}

describe('front-end layering', () => {
  it('no file imports something its layer may not', () => {
    const problems: string[] = []
    for (const [file, source] of Object.entries(sources())) {
      for (const spec of importsOf(source)) {
        const why = violation(file, spec)
        if (why) problems.push(`${file} imports '${spec}': ${why}`)
      }
    }
    expect(problems).toEqual([])
  })

  it('finds the files it is meant to check', () => {
    const files = Object.keys(sources())
    for (const expected of ['hooks/useKpis.ts', 'components/KpiGrid/KpiGrid.tsx', 'features/overview/OverviewPage.tsx', 'components/libs/recharts/Line/Line.tsx']) {
      expect(files).toContain(expected)
    }
  })
})

// The rules must actually bite: each case is a violation the walker has to catch.
describe('violation()', () => {
  const bad: [string, string][] = [
    ['components/KpiGrid/KpiGrid.tsx', '../../hooks/useKpis'],
    ['components/KpiGrid/KpiGrid.tsx', '../../features/overview/kpis'],
    ['components/KpiGrid/KpiGrid.tsx', '../../api/types'],
    ['components/KpiGrid/KpiGrid.tsx', '../../pages/HomePage'],
    ['components/AsyncSection/AsyncSection.tsx', '@tanstack/react-query'],
    ['components/PageHeader/PageHeader.tsx', 'react-router-dom'],
    ['components/TimeSeriesChart/TimeSeriesChart.tsx', 'recharts'],
    ['features/overview/OverviewPage.tsx', 'recharts'],
    ['hooks/useKpis.ts', '../features/overview/kpis'],
    ['hooks/useKpis.ts', '../components/Skeleton'],
    ['lib/format.ts', '../hooks'],
    ['lib/format.ts', '../components/Skeleton'],
    ['api/client.ts', '../lib/format'],
    ['features/overview/kpis.ts', '../../pages/HomePage'],
    ['components/Skeleton/Skeleton.tsx', '../../App'],
    ['hooks/useKpis.ts', '../test/utils'],
    ['components/KpiGrid/KpiGrid.tsx', '../../../outside'],
  ]
  it.each(bad)('rejects %s importing %s', (file, spec) => {
    expect(violation(file, spec)).not.toBe('')
  })

  const good: [string, string][] = [
    ['components/KpiGrid/KpiGrid.tsx', '../KpiCard'],
    ['components/TimeSeriesChart/TimeSeriesChart.tsx', '../libs/recharts'],
    ['components/TimeSeriesChart/TimeSeriesChart.tsx', '../../lib/useBreakpoint'],
    ['components/libs/recharts/Line/Line.tsx', 'recharts'],
    ['components/libs/recharts/Line/Line.test.tsx', 'recharts'],
    ['components/AsyncSection/AsyncSection.test.tsx', '@tanstack/react-query'],
    ['components/AsyncSection/AsyncSection.test.tsx', '../../test/utils'],
    ['hooks/useKpis.ts', '../api/analytics'],
    ['hooks/useFilters.ts', '../lib/filters'],
    ['hooks/useKpis.test.tsx', '../test/utils'],
    ['lib/filters.ts', '../api/types'],
    ['features/overview/OverviewPage.tsx', '../../hooks'],
    ['features/overview/OverviewPage.tsx', '../../components/KpiGrid'],
    ['pages/HomePage.tsx', '../features/overview/OverviewPage'],
    ['routes.tsx', './pages/HomePage'],
    ['App.tsx', './routes'],
  ]
  it.each(good)('allows %s importing %s', (file, spec) => {
    expect(violation(file, spec)).toBe('')
  })
})
