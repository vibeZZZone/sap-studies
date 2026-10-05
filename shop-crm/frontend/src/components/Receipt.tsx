import { useTranslation } from 'react-i18next'

import { formatBonuses } from '../lib/money'

export interface ReceiptLine {
  key: string
  title: string
  rows: { key: string; label: string; value: string; strong?: boolean; bonus?: number }[]
}

interface Props {
  lines: ReceiptLine[]
}

/**
 * Paper-styled receipt: monospaced font and dashed separators, like a printed till
 * roll. Purely presentational — every number arrives already calculated by the server
 * or mirrored from it.
 */
export default function Receipt({ lines }: Props) {
  const { t } = useTranslation()

  return (
    <div className="card bg-white p-4 font-receipt text-sm text-slate-800">
      <p className="mb-3 text-center text-xs font-bold uppercase tracking-widest text-slate-500">
        {t('till.cart')}
      </p>

      {lines.map((line) => (
        <div key={line.key} className="mb-4 last:mb-0">
          {line.title && <p className="mb-1 font-bold">{line.title}</p>}
          {line.rows.map((row) => (
            <div
              key={row.key}
              className="flex items-baseline justify-between gap-3 border-b border-dashed border-slate-300 py-1 last:border-b-0"
            >
              <span className="text-slate-600">{row.label}</span>
              <span className={row.strong ? 'text-base font-bold' : ''}>
                {row.value}
                {row.bonus !== undefined && row.bonus > 0 && (
                  <span className="ms-2 text-brand-green">+{formatBonuses(row.bonus)}</span>
                )}
              </span>
            </div>
          ))}
        </div>
      ))}

      <div className="mt-3 space-y-2 border-t-2 border-dashed border-slate-400 pt-3 font-bold">
        {lines
          .flatMap((line) => line.rows)
          .filter((row) => row.strong)
          .map((row) => (
            <div key={`total-${row.key}`} className="flex items-baseline justify-between gap-3 text-base">
              <span>{row.label}</span>
              <span className="text-brand-red">{row.value}</span>
            </div>
          ))}
      </div>
    </div>
  )
}