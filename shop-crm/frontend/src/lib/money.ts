/**
 * Kopeck-based money helpers for the UI. The server is the source of truth; these
 * functions only format what it already computed and mirror the backend arithmetic so
 * the receipt preview does not jump when the final sale comes back.
 */
export const KOP_PER_RUBLE = 100

const THIN_SPACE = ' '

/** Formats kopecks as the till shows them: 1 234,50 ₽. */
export function formatKop(kop: number): string {
  const negative = kop < 0
  const abs = Math.abs(Math.round(kop))
  const rubles = Math.floor(abs / KOP_PER_RUBLE)
  const fraction = abs % KOP_PER_RUBLE

  let out = groupThousands(rubles)
  if (fraction !== 0) {
    out += `,${String(fraction).padStart(2, '0')}`
  }
  out += ` ₽`
  return negative ? `-${out}` : out
}

/** Renders a bonus point count; one point is worth one ruble. */
export function formatBonuses(points: number): string {
  return `${groupThousands(Math.round(points))}`
}

function groupThousands(value: number): string {
  return String(value).replace(/\B(?=(\d{3})+(?!\d))/g, THIN_SPACE)
}

/**
 * Mirrors bonus.Calculate on the backend so the till can preview a sale. The server
 * still decides the real numbers, and the client refreshes from its response.
 */
export function previewSale(params: {
  subtotalKop: number
  bonusBalance: number
  bonusPercent: number
  cashbackPercent: number
  hasCard: boolean
}): { subtotalKop: number; bonusSpent: number; totalKop: number; bonusEarned: number } {
  const { subtotalKop, bonusBalance, bonusPercent, cashbackPercent, hasCard } = params
  if (!hasCard) {
    return { subtotalKop, bonusSpent: 0, totalKop: subtotalKop, bonusEarned: 0 }
  }

  let spendKop = Math.floor((subtotalKop * bonusPercent) / 100)
  const maxSpendKop = bonusBalance * KOP_PER_RUBLE
  if (spendKop > maxSpendKop) spendKop = maxSpendKop
  spendKop -= spendKop % KOP_PER_RUBLE

  const totalKop = subtotalKop - spendKop
  return {
    subtotalKop,
    bonusSpent: spendKop / KOP_PER_RUBLE,
    totalKop,
    bonusEarned: Math.floor((totalKop * cashbackPercent) / 100 / KOP_PER_RUBLE),
  }
}