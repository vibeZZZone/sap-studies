import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useCreateCustomer, useCustomers } from '../api/hooks'
import { useApiError } from '../api/useApiError'
import AsyncBoundary from '../components/AsyncBoundary'
import { formatBonuses, formatKop } from '../lib/money'

export default function CustomersPage() {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const customersQuery = useCustomers(debouncedSearch)
  const create = useCreateCustomer()
  const describeError = useApiError()

  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [notice, setNotice] = useState<string | null>(null)

  function updateSearch(value: string): void {
    setSearch(value)
    setDebouncedSearch(value)
  }

  async function submit(event: React.FormEvent): Promise<void> {
    event.preventDefault()
    try {
      const customer = await create.mutateAsync({ name, phone })
      setNotice(t('customers.created', { card: customer.cardNumber }))
      setName('')
      setPhone('')
    } catch {
      setNotice(null)
    }
  }

  const isFiltered = debouncedSearch.trim().length > 0

  return (
    <div>
      <h1 className="mb-4 text-xl font-bold">{t('customers.title')}</h1>

      <div className="card mb-4 p-3">
        <label htmlFor="customer-filter" className="mb-1 block text-sm font-semibold">
          {t('common.search')}
        </label>
        <input
          id="customer-filter"
          className="field"
          placeholder={t('customers.searchPlaceholder')}
          value={search}
          onChange={(event) => updateSearch(event.target.value)}
        />
      </div>

      <div className="card mb-4 p-3">
        <h2 className="mb-2 text-sm font-semibold">{t('customers.issue')}</h2>
        <form onSubmit={(event) => void submit(event)} className="grid gap-3 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
          <div>
            <label htmlFor="new-name" className="mb-1 block text-xs text-slate-600">
              {t('customers.name')}
            </label>
            <input id="new-name" className="field" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div>
            <label htmlFor="new-phone" className="mb-1 block text-xs text-slate-600">
              {t('customers.phone')}
            </label>
            <input
              id="new-phone"
              className="field"
              inputMode="tel"
              placeholder="+7 900 000-00-00"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              required
            />
          </div>
          <button type="submit" className="btn-success" disabled={create.isPending}>
            {t('customers.issue')}
          </button>
        </form>

        {notice && (
          <p role="status" className="mt-2 rounded-lg bg-brand-greenLight px-3 py-2 text-sm text-brand-greenDark">
            {notice}
          </p>
        )}
        {create.error && (
          <p role="alert" className="mt-2 rounded-lg bg-brand-redLight px-3 py-2 text-sm text-brand-redDark">
            {describeError(create.error)}
          </p>
        )}
      </div>

      <AsyncBoundary
        isLoading={customersQuery.isLoading}
        error={customersQuery.error ? describeError(customersQuery.error) : null}
        onRetry={() => void customersQuery.refetch()}
      >
        {customersQuery.data && customersQuery.data.length === 0 ? (
          <p className="py-12 text-center text-sm text-slate-500">
            {isFiltered ? t('customers.notFound') : t('customers.empty')}
            <br />
            {isFiltered ? t('customers.notFoundHint') : t('customers.emptyHint')}
          </p>
        ) : (
          <ul className="space-y-2">
            {customersQuery.data?.map((customer) => (
              <li key={customer.id} className="card p-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold">{customer.name}</p>
                    <p className="mt-0.5 text-xs text-slate-500">
                      {customer.cardNumber} · {customer.phone}
                    </p>
                  </div>
                  <dl className="flex gap-4 text-sm">
                    <div>
                      <dt className="text-xs text-slate-500">{t('customers.bonusBalance')}</dt>
                      <dd className="font-semibold text-brand-green">{formatBonuses(customer.bonusBalance)}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-slate-500">{t('customers.totalSpent')}</dt>
                      <dd className="font-semibold">{formatKop(customer.totalSpentKop)}</dd>
                    </div>
                  </dl>
                </div>
              </li>
            ))}
          </ul>
        )}
      </AsyncBoundary>
    </div>
  )
}