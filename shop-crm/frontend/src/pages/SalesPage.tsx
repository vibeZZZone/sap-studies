import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useSales } from '../api/hooks'
import { useApiError } from '../api/useApiError'
import AsyncBoundary from '../components/AsyncBoundary'
import { formatBonuses } from '../lib/money'

export default function SalesPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const salesQuery = useSales(page)
  const describeError = useApiError()

  const data = salesQuery.data

  return (
    <div>
      <h1 className="mb-4 text-xl font-bold">{t('sales.title')}</h1>

      <AsyncBoundary
        isLoading={salesQuery.isLoading}
        error={salesQuery.error ? describeError(salesQuery.error) : null}
        onRetry={() => void salesQuery.refetch()}
      >
        {data && data.sales.length === 0 ? (
          <p className="py-12 text-center text-sm text-slate-500">
            {t('sales.empty')}
            <br />
            {t('sales.emptyHint')}
          </p>
        ) : (
          <>
            <ul className="space-y-3">
              {data?.sales.map((sale) => (
                <li key={sale.id} className="card p-3">
                  <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <h2 className="text-sm font-bold">{t('sales.receipt', { id: sale.id })}</h2>
                    <time dateTime={sale.createdAt} className="text-xs text-slate-500">
                      {new Date(sale.createdAt).toLocaleString()}
                    </time>
                  </div>

                  <p className="mt-1 text-xs text-slate-500">
                    {t('sales.customer')}:{' '}
                    {sale.customer ? `${sale.customer.name} · ${sale.customer.cardNumber}` : t('till.withoutCard')}
                  </p>

                  <dl className="mt-2 grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
                    <div>
                      <dt className="text-slate-500">{t('sales.sum')}</dt>
                      <dd className="font-semibold">{sale.subtotal}</dd>
                    </div>
                    <div>
                      <dt className="text-slate-500">{t('sales.discount')}</dt>
                      <dd className="font-semibold text-slate-700">{formatBonuses(sale.bonusSpent)}</dd>
                    </div>
                    <div>
                      <dt className="text-slate-500">{t('sales.earned')}</dt>
                      <dd className="font-semibold text-brand-green">{formatBonuses(sale.bonusEarned)}</dd>
                    </div>
                    <div>
                      <dt className="text-slate-500">{t('sales.total')}</dt>
                      <dd className="font-bold text-brand-red">{sale.total}</dd>
                    </div>
                  </dl>

                  <details className="mt-2">
                    <summary className="cursor-pointer text-xs font-semibold text-slate-600">
                      {t('sales.items')}
                    </summary>
                    <ul className="mt-2 space-y-1 border-t border-dashed border-slate-300 pt-2 font-receipt text-xs">
                      {sale.items.map((item) => (
                        <li key={`${sale.id}-${item.productId}`} className="flex justify-between gap-3">
                          <span>
                            {item.name} × {item.qty}
                          </span>
                          <span>{item.sum}</span>
                        </li>
                      ))}
                    </ul>
                  </details>
                </li>
              ))}
            </ul>

            {data && data.pages > 1 && (
              <nav className="mt-4 flex items-center justify-between gap-2" aria-label={t('sales.title')}>
                <button
                  type="button"
                  className="btn-secondary"
                  disabled={page <= 1}
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                >
                  {t('sales.prev')}
                </button>
                <span className="text-xs text-slate-600">{t('sales.page', { page, pages: data.pages })}</span>
                <button
                  type="button"
                  className="btn-secondary"
                  disabled={page >= data.pages}
                  onClick={() => setPage((current) => current + 1)}
                >
                  {t('sales.next')}
                </button>
              </nav>
            )}
          </>
        )}
      </AsyncBoundary>
    </div>
  )
}