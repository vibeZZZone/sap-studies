import { useTranslation } from 'react-i18next'

import type { Product } from '../api/types'
import { formatKop } from '../lib/money'

interface Props {
  product: Product
  qtyInCart: number
  onAdd: () => void
}

/** A tappable product tile showing name, price and stock. */
export default function ProductTile({ product, qtyInCart, onAdd }: Props) {
  const { t } = useTranslation()
  const outOfStock = product.stock <= 0
  const atLimit = qtyInCart >= product.stock

  return (
    <button
      type="button"
      onClick={onAdd}
      disabled={outOfStock || atLimit}
      aria-label={`${product.name}, ${formatKop(product.priceKop)}`}
      className={[
        'card flex min-h-[104px] flex-col justify-between p-3 text-start transition',
        outOfStock || atLimit ? 'cursor-not-allowed opacity-50' : 'hover:border-brand-red hover:shadow-md',
      ].join(' ')}
    >
      <span className="text-sm font-semibold leading-snug text-slate-900">{product.name}</span>

      <span className="mt-2 flex flex-wrap items-center gap-2">
        <span className="text-base font-bold text-brand-red">{formatKop(product.priceKop)}</span>
        {outOfStock ? (
          <span className="badge bg-slate-200 text-slate-600">{t('products.outOfStock')}</span>
        ) : (
          <span
            className={[
              'badge',
              product.lowStock ? 'bg-amber-100 text-amber-800' : 'bg-brand-greenLight text-brand-greenDark',
            ].join(' ')}
          >
            {t('products.stock')}: {product.stock}
          </span>
        )}
        {qtyInCart > 0 && (
          <span className="badge bg-brand-redLight text-brand-redDark">× {qtyInCart}</span>
        )}
      </span>
    </button>
  )
}