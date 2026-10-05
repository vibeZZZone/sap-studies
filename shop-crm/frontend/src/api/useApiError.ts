import { useTranslation } from 'react-i18next'

import { ApiError } from './client'

/**
 * Turns any error into a localized message. The server sends a stable code plus an
 * English fallback; the UI shows its own translation for known codes and falls back
 * to the server text otherwise.
 *
 * `getProductName` lets a 409 name the product that ran out of stock: the error only
 * carries the id, and the caller usually already has the catalogue in memory.
 */
export function useApiError(getProductName?: (id: number) => string | undefined) {
  const { t } = useTranslation()

  return (error: unknown): string => {
    if (!(error instanceof ApiError)) {
      return error instanceof Error ? error.message : t('errors.unknown')
    }
    if (error.code === 'NETWORK_ERROR') {
      return t('errors.network')
    }

    if (error.code === 'INSUFFICIENT_STOCK') {
      const rawId = error.params.productId
      const productId = rawId ? Number(rawId) : undefined
      const name = productId && getProductName ? getProductName(productId) : undefined
      if (name) return t('errors.insufficientStockFor', { name })
      if (productId) return t('errors.insufficientStockById', { id: productId })
    }

    const key = `errors.${error.code}`
    const translated = t(key)
    if (translated !== key) {
      return translated
    }
    return error.fallbackMessage || t('errors.unknown')
  }
}