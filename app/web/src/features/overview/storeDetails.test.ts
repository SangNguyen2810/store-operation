import { describe, expect, it } from 'vitest'
import { STORES, makeStore } from '../../test/fixtures'
import { storeDetailLine } from './storeDetails'

describe('storeDetailLine', () => {
  const boksburg = makeStore(1, 'Boksburg, South Africa', {
    manager: 'Warner Hudson',
    phone: '490211944645',
    address: '569 Baicheng Lane',
    district: 'Gauteng',
    postalCode: '60304',
    inventoryItems: 2270,
    activeCustomers: 326,
  })

  it('describes the selected store', () => {
    expect(storeDetailLine([boksburg], 1)).toBe(
      'Boksburg, South Africa · Manager Warner Hudson · 569 Baicheng Lane, Gauteng 60304 · Phone 490211944645 · 2,270 inventory items · 326 active customers',
    )
  })

  it('leaves out what the store does not have', () => {
    const bare = makeStore(2, 'Hamilton, New Zealand', { manager: '', phone: '', district: '', postalCode: '' })
    expect(storeDetailLine([bare], 2)).toBe(
      'Hamilton, New Zealand · 1 Main Street · 1,002 inventory items · 102 active customers',
    )
    const noAddress = makeStore(3, 'X, Y', { address: '', district: 'D', postalCode: '' })
    expect(storeDetailLine([noAddress], 3)).toContain('· D ·')
  })

  it('summarises all stores when none is selected', () => {
    expect(storeDetailLine(STORES, null)).toBe('2 stores · 2,003 inventory items · 203 active customers')
  })

  it('says "1 store" for a single store', () => {
    expect(storeDetailLine([boksburg], null)).toBe('1 store · 2,270 inventory items · 326 active customers')
  })

  it('has nothing to say without stores, or for an unknown one', () => {
    expect(storeDetailLine([], null)).toBeNull()
    expect(storeDetailLine([], 1)).toBeNull()
    expect(storeDetailLine(STORES, 99)).toBeNull()
  })
})
