import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { useCreateProduct, useDeleteProduct, useProducts, useRestockProduct } from '../api/hooks'
import { useApiError } from '../api/useApiError'
import AsyncBoundary from '../components/AsyncBoundary'
import Modal from '../components/Modal'
import { formatKop } from '../lib/money'

export default function ProductsPage() {
  const { t } = useTranslation()
  const productsQuery = useProducts()
  const restock = useRestockProduct()
  const remove = useDeleteProduct()
  const create = useCreateProduct()
  const describeError = useApiError()

  const [showForm, setShowForm] = useState(false)
  const [name, setName] = useState('')
  const [price, setPrice] = useState('')
  const [initialQty, setInitialQty] = useState('0')
  const [pendingDelete, setPendingDelete] = useState<{ id: number; name: string } | null>(null)

  async function submit(event: React.FormEvent): Promise<void> {
    event.preventDefault()
    try {
      await create.mutateAsync({ name, price, initialQty: Number(initialQty) || 0 })
      setShowForm(false)
      setName('')
      setPrice('')
      setInitialQty('0')
    } catch {
      // The message is rendered from create.error below; the form stays open.
    }
  }

  async function confirmDelete(): Promise<void> {
    if (!pendingDelete) return
    try {
      await remove.mutateAsync(pendingDelete.id)
      setPendingDelete(null)
    } catch {
      setPendingDelete(null)
    }
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-xl font-bold">{t('products.title')}</h1>
        <button type="button" className="btn-primary" onClick={() => setShowForm(true)}>
          {t('products.add')}
        </button>
      </div>

      <AsyncBoundary
        isLoading={productsQuery.isLoading}
        error={productsQuery.error ? describeError(productsQuery.error) : null}
        onRetry={() => void productsQuery.refetch()}
      >
        {productsQuery.data && productsQuery.data.length === 0 ? (
          <p className="py-12 text-center text-sm text-slate-500">
            {t('products.empty')}
            <br />
            {t('products.emptyHint')}
          </p>
        ) : (
          <>
            {(restock.error || remove.error) && (
              <p role="alert" className="mb-3 rounded-lg bg-brand-redLight px-3 py-2 text-sm text-brand-redDark">
                {describeError(restock.error ?? remove.error)}
              </p>
            )}

            {/* Table on wide screens, stacked cards on a phone. */}
            <ul className="space-y-2">
              {productsQuery.data?.map((product) => (
                <li key={product.id} className="card p-3">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold">{product.name}</p>
                      <p className="mt-1 flex flex-wrap items-center gap-2 text-sm">
                        <span className="font-bold text-brand-red">{formatKop(product.priceKop)}</span>
                        <span
                          className={[
                            'badge',
                            product.stock <= 0
                              ? 'bg-slate-200 text-slate-600'
                              : product.lowStock
                                ? 'bg-amber-100 text-amber-800'
                                : 'bg-brand-greenLight text-brand-greenDark',
                          ].join(' ')}
                        >
                          {product.stock <= 0 ? t('products.outOfStock') : `${t('products.stock')}: ${product.stock}`}
                        </span>
                        {product.lowStock && product.stock > 0 && (
                          <span className="badge bg-amber-100 text-amber-800">{t('products.lowStock')}</span>
                        )}
                      </p>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        className="btn-secondary"
                        disabled={restock.isPending}
                        onClick={() => restock.mutate({ id: product.id, qty: 10 })}
                      >
                        {t('products.restock')}
                      </button>
                      <button
                        type="button"
                        className="btn-ghost text-brand-red"
                        onClick={() => setPendingDelete({ id: product.id, name: product.name })}
                      >
                        {t('common.delete')}
                      </button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </>
        )}
      </AsyncBoundary>

      <Modal open={showForm} title={t('products.add')} onClose={() => setShowForm(false)}>
        <form onSubmit={(event) => void submit(event)} className="space-y-3">
          <div>
            <label htmlFor="product-name" className="mb-1 block text-sm font-semibold">
              {t('products.name')}
            </label>
            <input
              id="product-name"
              className="field"
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
            />
          </div>

          <div>
            <label htmlFor="product-price" className="mb-1 block text-sm font-semibold">
              {t('products.price')}
            </label>
            <input
              id="product-price"
              className="field"
              inputMode="decimal"
              placeholder="349.50"
              value={price}
              onChange={(event) => setPrice(event.target.value)}
              required
            />
          </div>

          <div>
            <label htmlFor="product-qty" className="mb-1 block text-sm font-semibold">
              {t('products.initialQty')}
            </label>
            <input
              id="product-qty"
              className="field"
              type="number"
              min={0}
              value={initialQty}
              onChange={(event) => setInitialQty(event.target.value)}
            />
          </div>

          {create.error && (
            <p role="alert" className="rounded-lg bg-brand-redLight px-3 py-2 text-sm text-brand-redDark">
              {describeError(create.error)}
            </p>
          )}

          <div className="flex gap-2">
            <button type="submit" className="btn-success flex-1" disabled={create.isPending}>
              {t('common.save')}
            </button>
            <button type="button" className="btn-secondary" onClick={() => setShowForm(false)}>
              {t('common.cancel')}
            </button>
          </div>
        </form>
      </Modal>

      <Modal open={pendingDelete !== null} title={t('products.delete')} onClose={() => setPendingDelete(null)}>
        <p className="text-sm">{t('products.deleteConfirm', { name: pendingDelete?.name ?? '' })}</p>
        <div className="mt-4 flex gap-2">
          <button
            type="button"
            className="btn-primary flex-1"
            disabled={remove.isPending}
            onClick={() => void confirmDelete()}
          >
            {t('common.delete')}
          </button>
          <button type="button" className="btn-secondary" onClick={() => setPendingDelete(null)}>
            {t('common.cancel')}
          </button>
        </div>
      </Modal>
    </div>
  )
}