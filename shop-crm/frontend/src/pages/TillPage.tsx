import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useCreateSale, useCustomers, useProducts, useShopConfig } from '../api/hooks'
import { useApiError } from '../api/useApiError'
import type { Customer, Product } from '../api/types'
import AsyncBoundary from '../components/AsyncBoundary'
import ProductTile from '../components/ProductTile'
import Receipt from '../components/Receipt'
import { formatBonuses, formatKop, previewSale } from '../lib/money'

interface CartLine {
  product: Product
  qty: number
}

export default function TillPage() {
  const { t } = useTranslation()
  const productsQuery = useProducts()
  const configQuery = useShopConfig()
  const checkout = useCreateSale()
  const describeError = useApiError((id) => productsQuery.data?.find((p) => p.id === id)?.name)

  const [cart, setCart] = useState<CartLine[]>([])
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [bonusPercent, setBonusPercent] = useState(0)
  const [notice, setNotice] = useState<string | null>(null)

  const cashbackPercent = configQuery.data?.cashbackPercent ?? 5
  const spendLimitPct = configQuery.data?.bonusSpendLimitPct ?? 50

  const subtotalKop = useMemo(
    () => cart.reduce((sum, line) => sum + line.product.priceKop * line.qty, 0),
    [cart],
  )

  const preview = useMemo(
    () =>
      previewSale({
        subtotalKop,
        bonusBalance: customer?.bonusBalance ?? 0,
        bonusPercent: customer ? bonusPercent : 0,
        cashbackPercent,
        hasCard: customer !== null,
      }),
    [subtotalKop, customer, bonusPercent, cashbackPercent],
  )

  function add(product: Product): void {
    setNotice(null)
    setCart((current) => {
      const existing = current.find((line) => line.product.id === product.id)
      if (existing) {
        if (existing.qty >= product.stock) return current
        return current.map((line) =>
          line.product.id === product.id ? { ...line, qty: line.qty + 1 } : line,
        )
      }
      return [...current, { product, qty: 1 }]
    })
  }

  // Quantity can never exceed what the warehouse holds; the tile is disabled too, but
  // the rule belongs here as well.
  function changeQty(productId: number, delta: number): void {
    setNotice(null)
    setCart((current) =>
      current
        .map((line) => {
          if (line.product.id !== productId) return line
          const qty = Math.min(line.qty + delta, line.product.stock)
          return { ...line, qty: Math.max(qty, 0) }
        })
        .filter((line) => line.qty > 0),
    )
  }

  function clearCart(): void {
    setCart([])
    setBonusPercent(0)
    setNotice(null)
  }

  async function checkoutSale(): Promise<void> {
    if (cart.length === 0) return
    try {
      const sale = await checkout.mutateAsync({
        customerId: customer?.id ?? null,
        bonusPercent: customer ? bonusPercent : 0,
        items: cart.map((line) => ({ productId: line.product.id, qty: line.qty })),
      })
      setNotice(
        t('till.checkDone', { id: sale.id, total: formatKop(sale.totalKop) }) +
          (sale.bonusEarned > 0 ? ` · ${t('till.checkDoneBonus', { bonus: formatBonuses(sale.bonusEarned) })}` : ''),
      )
      clearCart()
      setCustomer(null)
    } catch (error) {
      // The receipt stays intact so the cashier can correct it instead of rebuilding.
      setNotice(describeError(error))
    }
  }

  const receiptLines = [
    {
      key: 'items',
      title: '',
      rows: cart.map((line) => ({
        key: `line-${line.product.id}`,
        label: `${line.product.name} × ${line.qty}`,
        value: formatKop(line.product.priceKop * line.qty),
      })),
    },
    {
      key: 'totals',
      title: '',
      rows: [
        { key: 'subtotal', label: t('till.subtotal'), value: formatKop(preview.subtotalKop) },
        ...(customer
          ? [
              { key: 'spent', label: t('till.bonusToSpend'), value: `−${formatBonuses(preview.bonusSpent)}`, bonus: 0 },
              { key: 'earned', label: t('till.bonusEarned'), value: formatBonuses(preview.bonusEarned), bonus: 0 },
            ]
          : []),
        { key: 'total', label: t('till.payable'), value: formatKop(preview.totalKop), strong: true },
      ],
    },
  ]

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_360px]">
      <section aria-label={t('till.title')}>
        <AsyncBoundary
          isLoading={productsQuery.isLoading}
          error={productsQuery.error ? describeError(productsQuery.error) : null}
          onRetry={() => void productsQuery.refetch()}
        >
          {productsQuery.data && productsQuery.data.length === 0 ? (
            <p className="py-10 text-center text-sm text-slate-500">
              {t('products.empty')}
              <br />
              {t('products.emptyHint')}
            </p>
          ) : (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">
              {productsQuery.data?.map((product) => (
                <ProductTile
                  key={product.id}
                  product={product}
                  qtyInCart={cart.find((line) => line.product.id === product.id)?.qty ?? 0}
                  onAdd={() => add(product)}
                />
              ))}
            </div>
          )}
        </AsyncBoundary>
      </section>

      <aside className="flex flex-col gap-3 lg:sticky lg:top-20 lg:self-start">
        {notice && (
          <p
            role="status"
            className={[
              'rounded-lg px-3 py-2 text-sm',
              checkout.isError ? 'bg-brand-redLight text-brand-redDark' : 'bg-brand-greenLight text-brand-greenDark',
            ].join(' ')}
          >
            {notice}
          </p>
        )}

        {cart.length === 0 ? (
          <div className="card p-4 text-center text-sm text-slate-500">
            <p className="font-semibold text-slate-700">{t('till.cartEmpty')}</p>
            <p className="mt-1">{t('till.cartEmptyHint')}</p>
          </div>
        ) : (
          <>
            <div className="card divide-y divide-slate-100">
              {cart.map((line) => (
                <div key={line.product.id} className="flex items-center gap-2 p-3">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-semibold">{line.product.name}</p>
                    <p className="text-xs text-slate-500">{formatKop(line.product.priceKop)}</p>
                  </div>

                  <button
                    type="button"
                    className="btn-secondary h-10 min-h-[40px] w-10 p-0"
                    aria-label={`${t('till.decrease')}: ${line.product.name}`}
                    onClick={() => changeQty(line.product.id, -1)}
                  >
                    −
                  </button>
                  <span className="w-8 text-center text-sm font-bold">{line.qty}</span>
                  <button
                    type="button"
                    className="btn-secondary h-10 min-h-[40px] w-10 p-0"
                    aria-label={`${t('till.increase')}: ${line.product.name}`}
                    disabled={line.qty >= line.product.stock}
                    onClick={() => changeQty(line.product.id, 1)}
                  >
                    +
                  </button>
                  <button
                    type="button"
                    className="btn-ghost h-10 min-h-[40px] w-10 p-0 text-brand-red"
                    aria-label={`${t('till.remove')}: ${line.product.name}`}
                    onClick={() => changeQty(line.product.id, -line.qty)}
                  >
                    ×
                  </button>
                </div>
              ))}
            </div>

            <CustomerPicker customer={customer} onChange={setCustomer} />

            {customer && (
              <div className="card p-3">
                <div className="flex items-center justify-between gap-3">
                  <label htmlFor="bonus-percent" className="text-sm font-semibold">
                    {t('till.discountPercent')}
                  </label>
                  <span className="text-sm text-slate-500">{t('customers.bonusBalance')}: {formatBonuses(customer.bonusBalance)}</span>
                </div>
                <input
                  id="bonus-percent"
                  type="range"
                  min={0}
                  max={spendLimitPct}
                  step={5}
                  value={bonusPercent}
                  onChange={(event) => setBonusPercent(Number(event.target.value))}
                  className="mt-2 w-full accent-brand-red"
                />
                <p className="mt-1 text-xs text-slate-500">{t('till.discountHint', { max: spendLimitPct })}</p>
              </div>
            )}

            <Receipt lines={receiptLines} />

            <div className="flex gap-2">
              <button
                type="button"
                className="btn-primary flex-1"
                disabled={checkout.isPending}
                onClick={() => void checkoutSale()}
              >
                {checkout.isPending ? t('till.checkingOut') : t('till.checkout')}
              </button>
              <button type="button" className="btn-secondary" onClick={clearCart}>
                {t('till.clear')}
              </button>
            </div>
          </>
        )}
      </aside>
    </div>
  )
}

interface PickerProps {
  customer: Customer | null
  onChange: (customer: Customer | null) => void
}

function CustomerPicker({ customer, onChange }: PickerProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const customersQuery = useCustomersLazy(search, open)

  return (
    <div className="card p-3">
      <label htmlFor="customer-search" className="text-sm font-semibold">
        {t('till.chooseCustomer')}
      </label>

      {customer ? (
        <div className="mt-2 flex items-center justify-between gap-2">
          <span className="text-sm">
            {customer.name} · {customer.cardNumber}
          </span>
          <button type="button" className="btn-ghost h-10 min-h-[40px]" onClick={() => onChange(null)}>
            {t('common.cancel')}
          </button>
        </div>
      ) : (
        <>
          <input
            id="customer-search"
            className="field mt-2"
            placeholder={t('till.searchCustomer')}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            onFocus={() => setOpen(true)}
          />
          {open && (
            <ul className="mt-2 max-h-56 overflow-y-auto rounded-lg border border-slate-200">
              {customersQuery.isLoading && <li className="p-2 text-xs text-slate-500">{t('common.loading')}</li>}
              {customersQuery.data && customersQuery.data.length === 0 && (
                <li className="p-2 text-xs text-slate-500">{t('till.noCustomers')}</li>
              )}
              {customersQuery.data?.map((candidate) => (
                <li key={candidate.id}>
                  <button
                    type="button"
                    className="flex w-full flex-col px-3 py-2 text-start text-sm hover:bg-slate-50"
                    onClick={() => {
                      onChange(candidate)
                      setOpen(false)
                    }}
                  >
                    <span className="font-semibold">{candidate.name}</span>
                    <span className="text-xs text-slate-500">
                      {candidate.cardNumber} · {formatBonuses(candidate.bonusBalance)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </div>
  )
}

// Kept separate so the picker only queries once the cashier actually looks for a card.
function useCustomersLazy(search: string, enabled: boolean) {
  return useCustomers(search, { enabled })
}