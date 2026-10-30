import type { Store } from '../../api/types'
import { formatInteger } from '../../lib/format'

/** One line about the selected store, or a summary of all of them. Empty fields are left out. */
export function storeDetailLine(stores: Store[], storeId: number | null): string | null {
  if (stores.length === 0) return null

  if (storeId === null) {
    const items = stores.reduce((sum, s) => sum + s.inventoryItems, 0)
    const customers = stores.reduce((sum, s) => sum + s.activeCustomers, 0)
    return [
      stores.length === 1 ? '1 store' : `${stores.length} stores`,
      `${formatInteger(items)} inventory items`,
      `${formatInteger(customers)} active customers`,
    ].join(' · ')
  }

  const s = stores.find((x) => x.id === storeId)
  if (!s) return null
  const where = [s.address, [s.district, s.postalCode].filter(Boolean).join(' ')].filter(Boolean).join(', ')
  return [
    s.label,
    s.manager && `Manager ${s.manager}`,
    where,
    s.phone && `Phone ${s.phone}`,
    `${formatInteger(s.inventoryItems)} inventory items`,
    `${formatInteger(s.activeCustomers)} active customers`,
  ]
    .filter(Boolean)
    .join(' · ')
}
